# Phase 4: Advanced Features

**Labels**: `enhancement`, `ami`, `sharing`, `phase-4`
**Milestone**: v1.4.0+
**Priority**: Low
**Estimated effort**: 1-2 weeks

## Summary

Production-ready AMI management features: sharing AMIs publicly or with specific AWS accounts, visualizing AMI dependency trees, intelligent cleanup, and multi-region support.

## User Stories

1. **AMI Sharing**: As a cluster administrator, I want to share my foundation AMI publicly or with partner organizations, so others can build on my work without rebuilding from scratch.

2. **Dependency Visualization**: As a developer, I want to see the AMI dependency tree for my application, so I understand what will be rebuilt when templates change.

3. **Intelligent Cleanup**: As an ops engineer, I want pctl to warn me before deleting an AMI that other AMIs depend on, so I don't break derived AMI builds.

4. **Multi-region**: As a global team, I want to copy AMIs to multiple regions, so users worldwide can build clusters quickly.

## Features

### 1. AMI Sharing

#### Public Sharing
```bash
# Share AMI publicly (like AWS Marketplace)
$ pctl ami share ami-123 --public

✅ AMI ami-123 is now public
   Anyone can launch instances from this AMI
   Launch permissions: all

# Unshare
$ pctl ami unshare ami-123 --public
```

#### Account-specific Sharing
```bash
# Share with specific AWS accounts
$ pctl ami share ami-123 --accounts 123456789012,987654321098

✅ AMI ami-123 shared with 2 accounts
   123456789012: GRANTED
   987654321098: GRANTED

# Share with organization
$ pctl ami share ami-123 --organization o-abc123xyz

✅ AMI ami-123 shared with organization o-abc123xyz
   All accounts in org can access this AMI
```

#### Share entire chains
```bash
# Share foundation + gromacs AMIs
$ pctl ami share-chain ami-456 --public

📋 AMI chain for ami-456:
   1. foundation-v1 (ami-123) → PUBLIC
   2. gromacs-v1 (ami-456) → PUBLIC

✅ Shared 2 AMIs publicly
```

#### Using shared AMIs
```yaml
# seeds/my-research.yaml
extends:
  template: https://pctl-public.s3.amazonaws.com/seeds/foundation.yaml
  ami: ami-123  # Public AMI to use

software:
  spack_packages:
    - my-research-code
```

### 2. AMI Dependency Tree

```bash
$ pctl ami tree gromacs-v1

AMI Dependency Tree:
└─ gromacs-v1 (ami-456) [10 GB]
   └─ foundation-v1 (ami-123) [8 GB]
      └─ aws-parallelcluster-3.14.0-amzn2 (ami-088fb4...) [base]

Total storage: 18 GB (delta-compressed)
Build time saved: 60 minutes (via layering)
```

```bash
$ pctl ami tree foundation-v1 --dependents

AMI Dependents Tree:
foundation-v1 (ami-123) [8 GB]
├─ gromacs-v1 (ami-456) [+2 GB]
├─ namd-v1 (ami-789) [+2 GB]
└─ openmm-v1 (ami-abc) [+1.5 GB]

⚠️  Warning: 3 AMIs depend on foundation-v1
   Deleting ami-123 will break these builds
```

### 3. Intelligent AMI Cleanup

#### Safe delete
```bash
$ pctl ami delete ami-123

⚠️  Warning: Cannot delete ami-123 (foundation-v1)
   3 AMIs depend on this AMI:
   - gromacs-v1 (ami-456)
   - namd-v1 (ami-789)
   - openmm-v1 (ami-abc)

Options:
  1. Delete dependents first: pctl ami delete ami-456 ami-789 ami-abc
  2. Force delete (breaks builds): pctl ami delete ami-123 --force
  3. Cancel

Continue? [y/N]
```

#### Garbage collection
```bash
$ pctl ami gc

🧹 Scanning for unused AMIs...

Found 5 unused AMIs:
  - old-foundation-v1 (ami-old123) [30 days old, no dependents]
  - test-build-1 (ami-test001) [60 days old, no dependents]
  - test-build-2 (ami-test002) [60 days old, no dependents]
  - gromacs-beta (ami-beta456) [90 days old, no dependents]
  - debug-ami (ami-debug789) [120 days old, no dependents]

Total storage: 45 GB
Estimated savings: $2.25/month

Delete these AMIs? [y/N]
```

### 4. Multi-region Support

#### Copy AMI to other regions
```bash
$ pctl ami copy ami-123 --regions us-east-1,eu-west-1,ap-southeast-1

📦 Copying ami-123 (foundation-v1) to 3 regions...

us-east-1:    ████████████████████ 100% (15 min)
              ✅ ami-useast-456

eu-west-1:    ████████████████████ 100% (18 min)
              ✅ ami-euwest-789

ap-southeast-1: ████████████████████ 100% (22 min)
              ✅ ami-apsea-abc

✅ AMI copied to 3 regions (55 minutes total)
```

#### Build with multi-region
```bash
$ pctl ami build -t foundation.yaml \
    --regions us-west-2,us-east-1,eu-west-1

📦 Building in us-west-2 (primary)...
   ... (60 minutes)
   ✅ ami-123

📦 Copying to 2 additional regions...
   us-east-1: ✅ ami-456
   eu-west-1: ✅ ami-789

✅ AMI available in 3 regions
```

### 5. Template Repository

#### Public template repository
```bash
# Install template from registry
$ pctl template install foundation

📦 Installing template 'foundation' from registry...
   Source: https://github.com/pctl-seeds/foundation
   Version: v1.2.0
   ✅ Installed to ~/.pctl/seeds/foundation.yaml

# List installed templates
$ pctl template list

Installed Templates:
  foundation (v1.2.0)     - Base HPC environment
  gromacs (v1.0.0)        - Molecular dynamics
  openmm (v1.1.0)         - OpenMM simulator

Registry Templates:
  namd (v2.0.0)           - NAMD molecular dynamics
  lammps (v3.1.0)         - LAMMPS simulator
  pytorch (v1.5.0)        - PyTorch ML environment
```

#### Sharing templates
```bash
# Publish template to registry
$ pctl template publish foundation.yaml

📦 Publishing template to registry...
   Name: foundation
   Version: v1.2.0 (auto-bumped)
   Dependencies: none
   ✅ Published!

Share with:
  pctl template install foundation
```

## Acceptance Criteria

### AMI Sharing
- [ ] `pctl ami share` command with --public flag
- [ ] `pctl ami share` command with --accounts flag
- [ ] `pctl ami share-chain` shares entire dependency chain
- [ ] `pctl ami unshare` revokes permissions
- [ ] Error handling: Non-existent account IDs
- [ ] Documentation on security implications

### Dependency Tree
- [ ] `pctl ami tree` command shows parent chain
- [ ] `pctl ami tree --dependents` shows child AMIs
- [ ] Visual tree with storage sizes
- [ ] Shows build time savings from layering

### Intelligent Cleanup
- [ ] `pctl ami delete` checks for dependents
- [ ] Warning message lists dependent AMIs
- [ ] `--force` flag to override safety check
- [ ] `pctl ami gc` finds unused AMIs
- [ ] `--dry-run` flag for gc
- [ ] `--older-than` filter for gc

### Multi-region
- [ ] `pctl ami copy` copies to specified regions
- [ ] Progress bars for each region copy
- [ ] Parallel copying (up to 3 concurrent)
- [ ] AMI metadata preserved across regions
- [ ] `--wait` vs background copying

## Implementation Notes

### AMI Sharing (AWS API)
```go
func (m *Manager) ShareAMI(ctx context.Context, amiID string, accounts []string) error {
    input := &ec2.ModifyImageAttributeInput{
        ImageId: aws.String(amiID),
        LaunchPermission: &types.LaunchPermissionModifications{
            Add: make([]types.LaunchPermission, len(accounts)),
        },
    }

    for i, account := range accounts {
        input.LaunchPermission.Add[i] = types.LaunchPermission{
            UserId: aws.String(account),
        }
    }

    _, err := m.ec2Client.ModifyImageAttribute(ctx, input)
    return err
}

func (m *Manager) ShareAMIPublic(ctx context.Context, amiID string) error {
    input := &ec2.ModifyImageAttributeInput{
        ImageId: aws.String(amiID),
        LaunchPermission: &types.LaunchPermissionModifications{
            Add: []types.LaunchPermission{
                {Group: types.PermissionGroupAll},
            },
        },
    }

    _, err := m.ec2Client.ModifyImageAttribute(ctx, input)
    return err
}
```

### Dependency Graph
```go
type AMINode struct {
    AMIID    string
    Name     string
    BaseAMI  string
    Children []*AMINode
    Size     int64
}

func (m *Manager) BuildDependencyTree(ctx context.Context, amiID string) (*AMINode, error) {
    ami, err := m.GetAMI(ctx, amiID)
    if err != nil {
        return nil, err
    }

    node := &AMINode{
        AMIID: ami.AMIID,
        Name:  ami.Name,
        BaseAMI: ami.BaseAMI,
        Size:  ami.Size,
    }

    if ami.BaseAMI != "" {
        parent, err := m.BuildDependencyTree(ctx, ami.BaseAMI)
        if err == nil {
            node.Children = append(node.Children, parent)
        }
    }

    return node, nil
}
```

### Multi-region Copy
```go
func (m *Manager) CopyAMIToRegions(ctx context.Context, amiID string, regions []string) (map[string]string, error) {
    results := make(map[string]string)
    var wg sync.WaitGroup
    errs := make(chan error, len(regions))

    // Limit to 3 concurrent copies
    sem := make(chan struct{}, 3)

    for _, region := range regions {
        wg.Add(1)
        go func(r string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            newAMI, err := m.copyAMI(ctx, amiID, r)
            if err != nil {
                errs <- fmt.Errorf("region %s: %w", r, err)
                return
            }
            results[r] = newAMI
        }(region)
    }

    wg.Wait()
    close(errs)

    // Check for errors
    if len(errs) > 0 {
        return results, <-errs
    }

    return results, nil
}
```

## "Just Works" Design Philosophy

**Default Behavior** (no flags needed):
```bash
# User just builds what they want
$ pctl ami build -t gromacs.yaml

# pctl automatically:
# ✅ Detects base template
# ✅ Finds or builds base AMI
# ✅ Chains builds transparently
# ✅ Caches for reuse
```

**Power User Controls** (optional flags):
```bash
# Force rebuild
$ pctl ami build -t gromacs.yaml --rebuild

# Use specific base AMI
$ pctl ami build -t gromacs.yaml --base-ami ami-123

# Disable caching
$ pctl ami build -t gromacs.yaml --no-cache

# Multi-region build
$ pctl ami build -t gromacs.yaml --regions us-west-2,us-east-1

# Share result publicly
$ pctl ami build -t gromacs.yaml --share-public
```

## Documentation Updates

**File**: `docs/ami-sharing.md` (new)

```markdown
# Sharing AMIs

## Making AMIs Public

Share your AMI so anyone can use it:
```bash
pctl ami share ami-123 --public
```

**Security considerations**:
- Anyone can launch instances from public AMIs
- Your AMI content is visible to all AWS users
- Recommended for:
  - Open source software stacks
  - Community-contributed tools
  - Public research environments

## Sharing with Specific Accounts

Share with trusted partners:
```bash
pctl ami share ami-123 --accounts 123456789012,987654321098
```

## Sharing AMI Chains

When sharing a derived AMI, share the base too:
```bash
pctl ami share-chain ami-456 --public
```

This shares both the base (foundation) and derived (gromacs) AMIs.

## Using Shared AMIs

Reference public AMIs in templates:
```yaml
extends:
  ami: ami-123  # Public foundation AMI
  template: https://example.com/seeds/foundation.yaml

software:
  spack_packages:
    - my-app
```
```

## Testing Plan

### AMI Sharing
```bash
# Test public share
pctl ami share ami-123 --public
aws ec2 describe-image-attribute --image-id ami-123 --attribute launchPermission
# Should show: {"Group": "all"}

# Test account share
pctl ami share ami-123 --accounts 123456789012
# Verify in partner account: can see AMI

# Test unshare
pctl ami unshare ami-123 --public
# Verify: public access revoked
```

### Dependency Tree
```bash
# Build chain: foundation → gromacs
pctl ami tree ami-gromacs
# Should show: gromacs → foundation → parallelcluster

# Check dependents
pctl ami tree ami-foundation --dependents
# Should show: gromacs, namd, etc.
```

## Future Enhancements

- **AMI Versioning**: Semantic versioning for AMIs
- **AMI Diffing**: Show what changed between AMI versions
- **Cost Tracking**: Track AMI storage costs
- **Build Analytics**: Time saved via caching, build duration trends

## Dependencies

- Phase 1: Manual Base AMI Support
- Phase 2: Template Inheritance
- Phase 3: Auto-chaining & Caching

## Related Issues

- #[TBD] AMI Registry/Marketplace
- #[TBD] Cross-account AMI Access
- #[TBD] Template Versioning
