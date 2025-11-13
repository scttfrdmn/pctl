# Phase 2: Template Inheritance

**Labels**: `enhancement`, `ami`, `templates`, `phase-2`
**Milestone**: v1.2.0
**Priority**: Medium
**Estimated effort**: 1 week

## Summary

Add `extends` keyword to template YAML syntax, enabling templates to inherit configuration from base templates. This provides a declarative way to define AMI layers without manually specifying `--base-ami`.

## User Story

As a cluster administrator, I want to define a gromacs template that extends a foundation template, so that the inheritance relationship is explicit and I don't need to manually track base AMI IDs.

## Current Behavior (After Phase 1)

Manual AMI layering requires tracking AMI IDs:
```bash
$ pctl ami build -t foundation.yaml --name foundation-v1
# Output: ami-123

$ pctl ami build -t gromacs.yaml --name gromacs-v1 --base-ami ami-123
```

Problems:
- AMI IDs must be tracked manually
- No clear relationship between templates
- Package list must be maintained separately

## Desired Behavior

```yaml
# templates/foundation.yaml
cluster:
  name: foundation
  region: us-west-2

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10
    - cmake@3.26.0
    - git@2.40.0
```

```yaml
# templates/gromacs.yaml
extends: foundation.yaml

cluster:
  name: gromacs-cluster

software:
  spack_packages:
    - gromacs@2023.1+mpi
```

```bash
$ pctl ami build -t gromacs.yaml --name gromacs-v1
📋 Template chain detected:
   1. foundation.yaml
   2. gromacs.yaml

🔍 Checking for existing AMIs...
❌ No foundation AMI found
🚀 Building foundation.yaml first...
   ... (60 minutes)
   ✅ Foundation AMI: ami-123

🚀 Building gromacs layer on ami-123...
   ... (10 minutes)
   ✅ Gromacs AMI: ami-456
```

## Acceptance Criteria

### Template Parsing
- [ ] Template loader resolves `extends` field
- [ ] Supports relative paths: `extends: ./foundation.yaml`
- [ ] Supports absolute paths: `extends: /path/to/foundation.yaml`
- [ ] Detects circular dependencies and fails with clear error
- [ ] Maximum inheritance depth: 5 levels (configurable)

### Template Merging
- [ ] Deep merge: Arrays are concatenated, maps are merged
- [ ] Child template overrides take precedence
- [ ] Package lists are combined (foundation + derived)
- [ ] Cluster config from child overrides base
- [ ] User lists are merged (no duplicates)

### Validation
- [ ] Validates base template exists before merge
- [ ] Detects package version conflicts (gcc@11 vs gcc@12)
- [ ] Validates merged template against schema
- [ ] Clear error messages for merge conflicts

### CLI Integration
- [ ] `pctl ami build -t gromacs.yaml` works without `--base-ami`
- [ ] `--base-ami` flag still works (overrides auto-detection)
- [ ] Template chain is displayed during build
- [ ] Both base and derived templates are stored in AMI metadata

## Implementation Notes

### New Template Fields

```go
// pkg/template/template.go
type Template struct {
    Extends  string `yaml:"extends"`  // NEW
    Cluster  ClusterConfig
    Software SoftwareConfig
    Users    []UserConfig
}
```

### Template Resolution

```go
// pkg/template/loader.go
func Load(path string) (*Template, error) {
    tmpl, err := loadRaw(path)
    if err != nil {
        return nil, err
    }

    if tmpl.Extends != "" {
        base, err := Load(resolveExtends(path, tmpl.Extends))
        if err != nil {
            return nil, err
        }
        tmpl = merge(base, tmpl)
    }

    return tmpl, nil
}
```

### Merge Strategy

```go
func merge(base, derived *Template) *Template {
    result := &Template{}

    // Cluster: derived overrides base
    result.Cluster = derived.Cluster
    if result.Cluster.Name == "" {
        result.Cluster = base.Cluster
    }

    // Software: concatenate packages
    result.Software.SpackPackages = append(
        base.Software.SpackPackages,
        derived.Software.SpackPackages...,
    )

    // Users: merge without duplicates
    result.Users = mergeUsers(base.Users, derived.Users)

    return result
}
```

### Circular Dependency Detection

```go
func detectCircular(path string, seen map[string]bool) error {
    if seen[path] {
        return fmt.Errorf("circular dependency detected: %s", path)
    }
    seen[path] = true

    tmpl, _ := loadRaw(path)
    if tmpl.Extends != "" {
        return detectCircular(resolveExtends(path, tmpl.Extends), seen)
    }
    return nil
}
```

## Testing Plan

### Unit Tests

```go
func TestTemplateInheritance(t *testing.T) {
    tests := []struct{
        name string
        base Template
        derived Template
        expected Template
    }{
        {
            name: "simple inheritance",
            base: Template{
                Software: SoftwareConfig{
                    SpackPackages: []string{"gcc@11", "openmpi"},
                },
            },
            derived: Template{
                Software: SoftwareConfig{
                    SpackPackages: []string{"gromacs"},
                },
            },
            expected: Template{
                Software: SoftwareConfig{
                    SpackPackages: []string{"gcc@11", "openmpi", "gromacs"},
                },
            },
        },
        // ... more test cases
    }
}

func TestCircularDependency(t *testing.T) {
    // Create temp files: a.yaml extends b.yaml, b.yaml extends a.yaml
    // Assert: Load() returns circular dependency error
}
```

### Integration Tests

```bash
# Create test templates
$ cat > /tmp/foundation.yaml << EOF
cluster:
  name: foundation
  region: us-west-2
software:
  spack_packages:
    - gcc@11.3.0
EOF

$ cat > /tmp/derived.yaml << EOF
extends: /tmp/foundation.yaml
cluster:
  name: derived
software:
  spack_packages:
    - gromacs@2023.1
EOF

# Test template loading
$ pctl template validate /tmp/derived.yaml
✅ Template valid
📋 Inheritance chain:
   1. /tmp/foundation.yaml
   2. /tmp/derived.yaml
📦 Total packages: 2 (gcc@11.3.0, gromacs@2023.1)
```

## Error Handling

### Missing Base Template
```
Error: Base template not found
  Template: /tmp/gromacs.yaml
  Extends:  /tmp/foundation.yaml
  Error:    No such file or directory

Fix: Create /tmp/foundation.yaml or update 'extends' field
```

### Circular Dependency
```
Error: Circular dependency detected
  Chain: a.yaml → b.yaml → c.yaml → a.yaml

Fix: Remove 'extends' field from one of the templates
```

### Version Conflict
```
Error: Package version conflict
  Base template (foundation.yaml):    gcc@11.3.0
  Derived template (gromacs.yaml):   gcc@12.1.0

Fix: Remove gcc from derived template or use same version
```

## Documentation Updates

**File**: `docs/templates.md`

Add section:
```markdown
## Template Inheritance

Templates can extend other templates using the `extends` keyword:

### Basic Inheritance

```yaml
# base.yaml
cluster:
  name: base-cluster
  region: us-west-2

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
```

```yaml
# derived.yaml
extends: base.yaml

cluster:
  name: my-cluster

software:
  spack_packages:
    - gromacs@2023.1
```

The derived template will have:
- gcc@11.3.0
- openmpi@4.1.4
- gromacs@2023.1

### Multi-level Inheritance

```yaml
# foundation.yaml
software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
```

```yaml
# hpc-base.yaml
extends: foundation.yaml
software:
  spack_packages:
    - python@3.10
    - cmake@3.26.0
```

```yaml
# gromacs.yaml
extends: hpc-base.yaml
software:
  spack_packages:
    - gromacs@2023.1
```

### Merge Rules

1. **Arrays**: Concatenated (packages from all levels)
2. **Maps**: Deep merged (child overrides parent)
3. **Scalars**: Child value overrides parent

### Limitations

- Maximum inheritance depth: 5 levels
- No circular dependencies
- Package version conflicts are not allowed
```

## Future Work (Out of Scope)

- Auto-building base AMI (Phase 3)
- Template imports from URLs
- Package version resolution/conflict resolution

## Dependencies

- Phase 1: Manual Base AMI Support (complete)

## Risks & Mitigations

**Risk**: Deep inheritance chains are hard to debug
**Mitigation**: Limit depth to 5, add `pctl template show --resolved` command

**Risk**: Merge behavior is confusing for complex configs
**Mitigation**: Clear documentation, validation errors, test examples

## Related Issues

- #[Phase 1] Manual Base AMI Support
- #[Phase 3] Auto-chaining & Caching
