# Phase 3: Auto-chaining & Caching

**Labels**: `enhancement`, `ami`, `caching`, `phase-3`
**Milestone**: v1.3.0
**Priority**: Medium
**Estimated effort**: 1-2 weeks

## Summary

Automatically detect, build, and reuse base AMIs when building derived templates. This is the "magic" that makes AMI layering transparent - users just build what they need, and pctl handles the chaining automatically.

## User Story

As a cluster administrator, I want to run `pctl ami build -t gromacs.yaml` and have it automatically find or build the foundation AMI, so that I don't need to manually orchestrate multi-step builds.

## Current Behavior (After Phase 2)

Template inheritance works but requires manual orchestration:
```bash
# User must manually build foundation first
$ pctl ami build -t foundation.yaml --name foundation-v1
✅ AMI: ami-123

# Then build derived
$ pctl ami build -t gromacs.yaml --name gromacs-v1
❌ Error: Base template 'foundation.yaml' has no AMI
   Fix: Build foundation.yaml first with --base-ami ami-123
```

## Desired Behavior

```bash
# User just builds what they want
$ pctl ami build -t gromacs.yaml --name gromacs-v1

📋 Template chain detected:
   1. foundation.yaml
   2. gromacs.yaml

🔍 Checking for existing AMIs...
❌ No foundation AMI found for hash abc123

🚀 Building foundation.yaml automatically...
   Build ID: build-001
   ... (60 minutes)
   ✅ Foundation AMI: ami-123

🚀 Building gromacs layer on ami-123...
   Build ID: build-002
   ... (10 minutes)
   ✅ Gromacs AMI: ami-456

✅ Build complete!
   Foundation: ami-123 (built)
   Gromacs:    ami-456 (built)
```

**Later, building NAMD (reuses foundation):**
```bash
$ pctl ami build -t namd.yaml --name namd-v1

📋 Template chain detected:
   1. foundation.yaml
   2. namd.yaml

🔍 Checking for existing AMIs...
✅ Found foundation AMI: ami-123 (hash abc123, built 2 days ago)

🚀 Building namd layer on ami-123...
   ... (10 minutes)
   ✅ NAMD AMI: ami-789

✅ Build complete! (10m 23s)
   Foundation: ami-123 (reused)
   NAMD:       ami-789 (built)
```

## Acceptance Criteria

### AMI Metadata & Hashing
- [ ] Each AMI stores template hash in tags
- [ ] Template hash includes content + all extends chain
- [ ] Hash changes when any template in chain changes
- [ ] AMI tags include: TemplateName, TemplateHash, BaseTemplate, BaseAMI

### AMI Lookup
- [ ] `FindAMIByTemplateHash(template, region)` function
- [ ] Searches AMIs by TemplateName + TemplateHash tags
- [ ] Returns newest AMI if multiple matches
- [ ] Filters by region
- [ ] Fast query (indexed by tags)

### Auto-building
- [ ] Detects when base template has no AMI
- [ ] Recursively builds base templates (foundation → hpc → gromacs)
- [ ] Uses existing AMIs when available (cache hit)
- [ ] Builds missing AMIs automatically (cache miss)
- [ ] Progress shows both base and derived builds

### Cache Invalidation
- [ ] `--rebuild` flag forces rebuild of current template
- [ ] `--rebuild-base` flag forces rebuild of entire chain
- [ ] Template content change invalidates cache (hash changes)
- [ ] Manual AMI deletion triggers rebuild on next use

### Error Handling
- [ ] Base build failure stops derived build with clear error
- [ ] Parallel builds of same template are coordinated (lock)
- [ ] Network errors during AMI lookup are retried
- [ ] Clear message when AMI exists but is in wrong state

## Implementation Notes

### Template Hashing

```go
// pkg/template/hash.go
func ComputeHash(tmpl *Template) string {
    // Hash must include:
    // 1. Template content (packages, users, etc.)
    // 2. All parent templates (recursive)

    h := sha256.New()

    // Hash template content
    h.Write([]byte(tmpl.Cluster.Name))
    for _, pkg := range tmpl.Software.SpackPackages {
        h.Write([]byte(pkg))
    }

    // Hash base template (recursive)
    if tmpl.Extends != "" {
        base, _ := Load(tmpl.Extends)
        h.Write([]byte(ComputeHash(base)))
    }

    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

### AMI Lookup

```go
// pkg/ami/cache.go
type AMICache struct {
    ec2Client *ec2.Client
    region    string
}

func (c *AMICache) FindByTemplateHash(templateName, hash string) (*AMIMetadata, error) {
    input := &ec2.DescribeImagesInput{
        Filters: []types.Filter{
            {Name: aws.String("tag:TemplateName"), Values: []string{templateName}},
            {Name: aws.String("tag:TemplateHash"), Values: []string{hash}},
            {Name: aws.String("tag:ManagedBy"), Values: []string{"pctl"}},
            {Name: aws.String("state"), Values: []string{"available"}},
        },
        Owners: []string{"self"},
    }

    resp, err := c.ec2Client.DescribeImages(context.Background(), input)
    if err != nil {
        return nil, err
    }

    if len(resp.Images) == 0 {
        return nil, ErrAMINotFound
    }

    // Return newest AMI
    sort.Slice(resp.Images, func(i, j int) bool {
        return *resp.Images[i].CreationDate > *resp.Images[j].CreationDate
    })

    return parseAMIMetadata(resp.Images[0]), nil
}
```

### Auto-chaining Logic

```go
// pkg/ami/chain.go
func (b *Builder) BuildChain(ctx context.Context, tmpl *Template, opts *BuildOptions) (*AMIMetadata, error) {
    // 1. Resolve template chain
    chain := resolveChain(tmpl)

    // 2. Find or build each template in chain
    var baseAMI string
    for _, t := range chain {
        hash := ComputeHash(t)

        // Try to find existing AMI
        cached, err := b.cache.FindByTemplateHash(t.Cluster.Name, hash)
        if err == nil {
            fmt.Printf("✅ Found %s AMI: %s\n", t.Cluster.Name, cached.AMIID)
            baseAMI = cached.AMIID
            continue
        }

        // Build missing AMI
        fmt.Printf("🚀 Building %s automatically...\n", t.Cluster.Name)
        buildOpts := *opts
        buildOpts.BaseAMI = baseAMI
        buildOpts.Name = generateAMIName(t, hash)

        metadata, err := b.BuildAMI(ctx, t, &buildOpts)
        if err != nil {
            return nil, fmt.Errorf("failed to build %s: %w", t.Cluster.Name, err)
        }

        baseAMI = metadata.AMIID
    }

    return metadata, nil
}
```

### Enhanced AMI Metadata

```go
// pkg/ami/metadata.go
type AMIMetadata struct {
    AMIID          string
    Name           string
    Region         string
    TemplateName   string
    TemplateHash   string      // NEW
    BaseTemplate   string      // NEW
    BaseAMI        string      // NEW
    SpackPackages  []string
    BuildDate      time.Time
}

func (b *Builder) tagAMI(amiID string, metadata *AMIMetadata) error {
    tags := []types.Tag{
        {Key: aws.String("Name"), Value: aws.String(metadata.Name)},
        {Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
        {Key: aws.String("TemplateName"), Value: aws.String(metadata.TemplateName)},
        {Key: aws.String("TemplateHash"), Value: aws.String(metadata.TemplateHash)},
        {Key: aws.String("Region"), Value: aws.String(metadata.Region)},
        {Key: aws.String("BuildDate"), Value: aws.String(metadata.BuildDate.Format(time.RFC3339))},
    }

    if metadata.BaseTemplate != "" {
        tags = append(tags, types.Tag{
            Key: aws.String("BaseTemplate"),
            Value: aws.String(metadata.BaseTemplate),
        })
    }

    if metadata.BaseAMI != "" {
        tags = append(tags, types.Tag{
            Key: aws.String("BaseAMI"),
            Value: aws.String(metadata.BaseAMI),
        })
    }

    // ... create tags
}
```

## Testing Plan

### Unit Tests

```go
func TestTemplateHash(t *testing.T) {
    // Test: Same template = same hash
    t1 := &Template{Software: SoftwareConfig{SpackPackages: []string{"gcc"}}}
    t2 := &Template{Software: SoftwareConfig{SpackPackages: []string{"gcc"}}}
    assert.Equal(t, ComputeHash(t1), ComputeHash(t2))

    // Test: Different template = different hash
    t3 := &Template{Software: SoftwareConfig{SpackPackages: []string{"llvm"}}}
    assert.NotEqual(t, ComputeHash(t1), ComputeHash(t3))
}

func TestAMICache(t *testing.T) {
    // Test: Find existing AMI by hash
    cache := NewAMICache(mockEC2Client, "us-west-2")
    ami, err := cache.FindByTemplateHash("foundation", "abc123")
    assert.NoError(t, err)
    assert.Equal(t, "ami-123", ami.AMIID)

    // Test: Return error when not found
    _, err = cache.FindByTemplateHash("nonexistent", "xyz789")
    assert.ErrorIs(t, err, ErrAMINotFound)
}

func TestAutoBuild(t *testing.T) {
    // Test: Auto-build missing base template
    // Mock: foundation AMI doesn't exist
    // Call: BuildChain(gromacsTemplate)
    // Assert: Builds foundation first, then gromacs

    // Test: Reuse existing base template
    // Mock: foundation AMI exists
    // Call: BuildChain(namdTemplate)
    // Assert: Skips foundation build, only builds namd
}
```

### Integration Tests

```bash
# Test 1: Auto-build chain
$ rm -rf ~/.pctl/state/*.json  # Clear state
$ pctl ami build -t gromacs.yaml --name test-gromacs
# Should build: foundation → gromacs

# Test 2: Cache reuse
$ pctl ami build -t namd.yaml --name test-namd
# Should reuse foundation AMI, only build namd

# Test 3: Cache invalidation
$ echo "    - cmake@3.27.0" >> foundation.yaml
$ pctl ami build -t gromacs.yaml --name test-gromacs-v2
# Should rebuild foundation (hash changed), then gromacs

# Test 4: Force rebuild
$ pctl ami build -t gromacs.yaml --name test-gromacs-v3 --rebuild-base
# Should rebuild entire chain even if cached
```

## CLI Flags

```go
// New flags for ami build command
buildAMICmd.Flags().BoolVar(&amiRebuild, "rebuild", false,
    "rebuild this AMI even if it exists")
buildAMICmd.Flags().BoolVar(&amiRebuildBase, "rebuild-base", false,
    "rebuild entire template chain from scratch")
buildAMICmd.Flags().BoolVar(&amiNoCache, "no-cache", false,
    "alias for --rebuild-base")
```

## Documentation Updates

**File**: `docs/ami-caching.md` (new)

```markdown
# AMI Caching

pctl automatically caches AMIs and reuses them when possible.

## How Caching Works

Each AMI is tagged with a template hash that includes:
- Template content (packages, users, config)
- All parent templates (inheritance chain)

When you build an AMI, pctl:
1. Computes template hash
2. Searches for existing AMI with same hash
3. Reuses AMI if found (cache hit)
4. Builds new AMI if not found (cache miss)

## Cache Invalidation

Cache is automatically invalidated when:
- Template content changes
- Any parent template changes
- Base AMI is deleted

## Manual Cache Control

Force rebuild:
```bash
# Rebuild just this template
pctl ami build -t gromacs.yaml --rebuild

# Rebuild entire chain
pctl ami build -t gromacs.yaml --rebuild-base
```

## Viewing Cache

```bash
# List all cached AMIs
pctl ami list

# Show AMI details
pctl ami describe ami-123
```
```

## Future Work (Out of Scope)

- Distributed cache (shared across team/org)
- Cache warming (pre-build common bases)
- Cache garbage collection (auto-delete unused AMIs)

## Dependencies

- Phase 1: Manual Base AMI Support (complete)
- Phase 2: Template Inheritance (complete)

## Risks & Mitigations

**Risk**: Hash collisions (unlikely but possible)
**Mitigation**: Use SHA-256 truncated to 16 chars (collision probability: ~10^-19)

**Risk**: Stale cache (AMI exists but is corrupted)
**Mitigation**: Add `--rebuild` flag, validate AMI state before use

**Risk**: Race condition (two builds of same template)
**Mitigation**: Use build state file as lock, detect concurrent builds

## Related Issues

- #[Phase 1] Manual Base AMI Support
- #[Phase 2] Template Inheritance
- #[Phase 4] Advanced Features
