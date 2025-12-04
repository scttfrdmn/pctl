# Feature Request: Template Operating System Override

## Status: Planned

## Priority: Medium

## Overview
Allow users to override the operating system used for cluster creation and AMI builds, while keeping Amazon Linux 2023 as the sensible default. This enables flexibility for users who need specific OS versions (Ubuntu 24.04, RHEL 8/9, etc.) without compromising the simplicity of the default experience.

## Current Behavior
- pctl defaults to Amazon Linux 2023 (`al2023`) for all clusters
- No way to specify a different OS in the template
- Users who need Ubuntu, RHEL, or other OS versions cannot use those with pctl

## Proposed Behavior
Users can optionally specify an OS in their template:

```yaml
cluster:
  name: my-cluster
  region: us-west-2
  os: ubuntu2404  # Optional: defaults to al2023 if not specified

compute:
  head_node: c5.large
  queues:
    - name: compute
      instance_types: [c5.2xlarge]
      min_count: 0
      max_count: 10

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
```

## Supported Operating Systems

Based on ParallelCluster 3.14.0 support:

### Tier 1 - Primary Support (Recommended)
- `al2023` - Amazon Linux 2023 (default, EOL 2029)
- `ubuntu2404` - Ubuntu 24.04 LTS (EOL 2029)
- `ubuntu2204` - Ubuntu 22.04 LTS (EOL 2027)

### Tier 2 - Extended Support
- `alinux2` - Amazon Linux 2 (EOL 2025, legacy)
- `ubuntu2004` - Ubuntu 20.04 LTS (EOL 2025)
- `rhel8` - Red Hat Enterprise Linux 8
- `rocky8` - Rocky Linux 8
- `rhel9` - Red Hat Enterprise Linux 9
- `rocky9` - Rocky Linux 9

## Implementation Details

### 1. Template Schema Update

Add optional `os` field to cluster config:

```go
// pkg/template/template.go
type ClusterConfig struct {
    Name   string `yaml:"name" json:"name"`
    Region string `yaml:"region" json:"region"`
    OS     string `yaml:"os,omitempty" json:"os,omitempty"` // NEW: Optional OS override
}
```

### 2. OS Validation

```go
// pkg/template/validate.go
var supportedOSList = map[string]string{
    // Tier 1 - Primary
    "al2023":     "Amazon Linux 2023",
    "ubuntu2404": "Ubuntu 24.04 LTS",
    "ubuntu2204": "Ubuntu 22.04 LTS",

    // Tier 2 - Extended
    "alinux2":    "Amazon Linux 2",
    "ubuntu2004": "Ubuntu 20.04 LTS",
    "rhel8":      "Red Hat Enterprise Linux 8",
    "rocky8":     "Rocky Linux 8",
    "rhel9":      "Red Hat Enterprise Linux 9",
    "rocky9":     "Rocky Linux 9",
}

func validateOS(os string) error {
    if os == "" {
        return nil // Optional field
    }

    if _, exists := supportedOSList[os]; !exists {
        return fmt.Errorf("unsupported OS: %s. Supported: %v", os, getSupportedOSKeys())
    }

    // Warn about legacy OS versions
    if os == "alinux2" || os == "ubuntu2004" {
        fmt.Printf("⚠️  Warning: %s reaches EOL in 2025. Consider using al2023 or ubuntu2404 instead.\n", supportedOSList[os])
    }

    return nil
}
```

### 3. Fingerprint Update

Update AMI fingerprinting to include OS in hash:

```go
// pkg/template/fingerprint.go
func (t *Template) ComputeFingerprint() *AMIFingerprint {
    // Use template OS if specified, otherwise default
    baseOS := t.Cluster.OS
    if baseOS == "" {
        baseOS = "amazonlinux2023" // Default
    }

    // ... rest of fingerprint logic
    fp := &AMIFingerprint{
        BaseOS:       baseOS,
        SpackVersion: defaultSpackVersion,
        LmodVersion:  defaultLmodVersion,
        Packages:     packages,
    }

    // ... compute hash
}
```

### 4. Config Generator Update

Pass OS through to ParallelCluster config:

```go
// pkg/config/generator.go
func (g *Generator) buildParallelClusterConfig(tmpl *template.Template) map[string]interface{} {
    // Determine OS (template override or default)
    os := tmpl.Cluster.OS
    if os == "" {
        os = "al2023" // Default
    }

    config := map[string]interface{}{
        "Region": tmpl.Cluster.Region,
        "Image": map[string]interface{}{
            "Os": os,
        },
    }

    // ... rest of config generation
}
```

### 5. CLI Flags

Add optional CLI flags for overriding OS:

```bash
# Override OS at cluster creation time
pctl create -t template.yaml --os ubuntu2404

# Override OS for AMI build
pctl ami build -t template.yaml --os ubuntu2404 --name my-ubuntu-ami
```

```go
// cmd/pctl/create.go
createCmd.Flags().String("os", "", "Override operating system (al2023, ubuntu2404, etc.)")

// cmd/pctl/ami.go
amiBuildCmd.Flags().String("os", "", "Override operating system for AMI build")
```

### 6. Software Compatibility

Some software may have OS-specific requirements. Add compatibility checks:

```go
// pkg/software/compatibility.go
type OSCompatibility struct {
    RequiresOS   []string // If set, package only works on these OS versions
    IncompatibleOS []string // If set, package doesn't work on these OS versions
}

var packageCompatibility = map[string]OSCompatibility{
    // Example: Some packages may require specific OS versions
    "singularity": {
        RequiresOS: []string{"al2023", "ubuntu2204", "ubuntu2404"},
    },
}

func checkSoftwareCompatibility(packages []string, os string) []string {
    var warnings []string
    for _, pkg := range packages {
        if compat, exists := packageCompatibility[pkg]; exists {
            // Check incompatibilities
            for _, incompatibleOS := range compat.IncompatibleOS {
                if os == incompatibleOS {
                    warnings = append(warnings,
                        fmt.Sprintf("Package %s may not work on %s", pkg, os))
                }
            }

            // Check requirements
            if len(compat.RequiresOS) > 0 {
                compatible := false
                for _, requiredOS := range compat.RequiresOS {
                    if os == requiredOS {
                        compatible = true
                        break
                    }
                }
                if !compatible {
                    warnings = append(warnings,
                        fmt.Sprintf("Package %s may require one of: %v", pkg, compat.RequiresOS))
                }
            }
        }
    }
    return warnings
}
```

## User Experience

### Default Case (No Change)
```bash
# User doesn't specify OS - gets AL2023 (current behavior)
pctl create -t bioinformatics.yaml --name my-cluster
# Uses Amazon Linux 2023
```

### Explicit Override in Template
```yaml
# templates/my-ubuntu-cluster.yaml
cluster:
  name: my-cluster
  region: us-west-2
  os: ubuntu2404  # Explicitly use Ubuntu 24.04

compute:
  head_node: c5.large
  # ...
```

```bash
pctl create -t templates/my-ubuntu-cluster.yaml
# Uses Ubuntu 24.04 as specified in template
```

### CLI Override (Highest Priority)
```bash
# Template has os: al2023, but CLI overrides
pctl create -t template.yaml --os ubuntu2404
# Uses Ubuntu 24.04 from CLI flag
```

### Priority Order
1. CLI flag `--os` (highest priority)
2. Template `cluster.os` field
3. Default `al2023` (lowest priority)

## Acceptance Criteria

- [ ] Template schema supports optional `os` field
- [ ] OS validation with supported OS list
- [ ] Warnings for legacy OS versions (AL2, Ubuntu 20.04)
- [ ] AMI fingerprinting includes OS in hash
- [ ] Different OS versions produce different AMI fingerprints
- [ ] Config generator passes OS to ParallelCluster
- [ ] CLI flags `--os` for create and ami build commands
- [ ] CLI flag overrides template OS setting
- [ ] Documentation updated with OS override examples
- [ ] Tests for all supported OS values
- [ ] Default behavior unchanged (AL2023)

## Testing Plan

### 1. Template OS Override
```bash
# Create template with Ubuntu
cat > test-ubuntu.yaml <<EOF
cluster:
  name: test-ubuntu
  region: us-west-2
  os: ubuntu2404
compute:
  head_node: t3.small
  queues:
    - name: compute
      instance_types: [t3.small]
      min_count: 0
      max_count: 2
EOF

# Build AMI
pctl ami build -t test-ubuntu.yaml --name test-ubuntu-ami

# Create cluster
pctl create -t test-ubuntu.yaml --custom-ami <ami-id>

# Verify cluster uses Ubuntu 24.04
pctl ssh test-ubuntu
cat /etc/os-release  # Should show Ubuntu 24.04
```

### 2. CLI Override
```bash
# Template has AL2023, override with Ubuntu
pctl create -t al2023-template.yaml --os ubuntu2404

# Verify Ubuntu is used
```

### 3. Fingerprint Uniqueness
```bash
# Same packages, different OS should produce different fingerprints
pctl ami build -t template.yaml --os al2023 --name test-al2023
pctl ami build -t template.yaml --os ubuntu2404 --name test-ubuntu

# Should produce two different AMIs with different fingerprints
```

### 4. Validation
```bash
# Invalid OS should error
pctl create -t template.yaml --os invalid-os
# Error: unsupported OS: invalid-os. Supported: al2023, ubuntu2404, ...
```

## Edge Cases

1. **OS Change After AMI Built**
   - If template OS changes, old AMI fingerprint won't match
   - New AMI build required (expected behavior)

2. **CLI vs Template Conflict**
   - CLI `--os` flag always wins
   - Warn user if overriding template OS

3. **Legacy OS Migration**
   - Warn users moving from AL2 to AL2023
   - Provide migration guide

4. **Software Compatibility**
   - Some Spack packages may not build on all OS versions
   - Provide clear error messages if package incompatible

## Documentation Updates

### README.md
Add section on OS selection:
```markdown
### Operating System Selection

pctl defaults to Amazon Linux 2023 (supported until 2029) but supports multiple operating systems:

**Recommended:**
- `al2023` - Amazon Linux 2023 (default)
- `ubuntu2404` - Ubuntu 24.04 LTS
- `ubuntu2204` - Ubuntu 22.04 LTS

**Also supported:**
- `rhel8`, `rocky8`, `rhel9`, `rocky9`
- `alinux2`, `ubuntu2004` (legacy, EOL 2025)

Specify in template:
\```yaml
cluster:
  os: ubuntu2404
\```

Or override via CLI:
\```bash
pctl create -t template.yaml --os ubuntu2404
\```
```

### Template Spec (docs/TEMPLATE_SPEC.md)
Document the `os` field with examples and validation rules.

## Benefits

1. **Flexibility**: Users can choose OS based on their requirements
2. **No Breaking Changes**: Defaults to AL2023, existing templates work unchanged
3. **Future-Proof**: Easy to add new OS versions as ParallelCluster supports them
4. **Clear Warnings**: Users get warnings about legacy OS versions
5. **Proper Caching**: Different OS versions create different AMI fingerprints

## Alternatives Considered

### Alternative 1: Always require OS in template
**Rejected**: Breaks existing templates, adds complexity for simple use cases

### Alternative 2: Per-package OS compatibility
**Deferred**: Too complex for initial implementation, can be added later

### Alternative 3: Auto-detect OS from package requirements
**Rejected**: Too magical, users should be explicit about OS choice

## Related Issues

- Current work: Upgraded to AL2023 as default (commit a32e6e1)
- Future: Package OS compatibility validation
- Future: OS-specific template examples in registry

## Estimated Effort

- Schema updates: 1 hour
- Validation logic: 2 hours
- Fingerprint updates: 1 hour
- Config generator updates: 1 hour
- CLI flags: 2 hours
- Testing: 3 hours
- Documentation: 2 hours
- **Total: 12-15 hours**

## References

- [ParallelCluster Supported Operating Systems](https://docs.aws.amazon.com/parallelcluster/latest/ug/operating-systems-v3.html)
- [Amazon Linux 2023 FAQs](https://aws.amazon.com/linux/amazon-linux-2023/faqs/)
- [Ubuntu Cloud Images](https://cloud-images.ubuntu.com/)
