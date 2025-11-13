# Phase 1: Manual Base AMI Support

**Labels**: `enhancement`, `ami`, `phase-1`
**Milestone**: v1.1.0
**Priority**: High
**Estimated effort**: 2-3 days

## Summary

Enable users to manually specify a base AMI when building new AMIs, allowing them to stack AMIs without template inheritance. This is the foundational capability for the full AMI layering feature.

## User Story

As a cluster administrator, I want to build a new AMI starting from an existing base AMI (instead of from ParallelCluster base), so that I can avoid reinstalling common packages and reduce build time.

## Current Behavior

Every `pctl ami build` starts from the ParallelCluster base AMI and installs all packages from scratch:
```bash
$ pctl ami build -t gromacs.yaml --name gromacs-v1
# Installs: gcc, openmpi, python, cmake, git, gromacs (70 min)
```

## Desired Behavior

```bash
# First, build foundation
$ pctl ami build -t foundation.yaml --name foundation-v1
# Creates ami-123 with gcc, openmpi, python, cmake, git (60 min)

# Then, build gromacs starting from foundation
$ pctl ami build -t gromacs.yaml --name gromacs-v1 --base-ami ami-123
# Only installs gromacs on top of ami-123 (10 min)
```

## Acceptance Criteria

- [ ] CLI accepts `--base-ami <ami-id>` flag
- [ ] Builder uses specified base AMI instead of auto-detecting ParallelCluster AMI
- [ ] Template validation still works (checks packages don't conflict)
- [ ] Build logs clearly indicate base AMI being used
- [ ] Error handling: Invalid/non-existent AMI ID shows helpful error
- [ ] Error handling: Base AMI in wrong region shows clear error
- [ ] Documentation updated with example workflow
- [ ] Integration test: Build foundation → Build derived with `--base-ami`

## Implementation Notes

### Files to modify

**`cmd/pctl/ami.go`** (lines 130-145):
```go
// Already has flag defined, but may need to expose it:
buildAMICmd.Flags().StringVar(&amiBaseAMI, "base-ami", "",
    "base AMI to start from (default: auto-detect ParallelCluster AMI)")
```

**`pkg/ami/builder.go`** (line 268):
```go
// This section already checks if BaseAMI is set:
baseAMI := opts.BaseAMI
if baseAMI == "" {
    var err error
    baseAMI, err = b.getLatestParallelClusterAMI(ctx, architecture)
    // ...
}
```

**Changes needed:**
1. Ensure `--base-ami` flag is properly wired from CLI to BuildOptions
2. Add validation: Check AMI exists and is in correct region
3. Update success message to show base AMI used
4. Add to help text with example

### Validation logic
```go
func (b *Builder) validateBaseAMI(ctx context.Context, amiID, region string) error {
    // Check AMI exists
    // Check AMI is in correct region
    // Check AMI state is "available"
    // Return descriptive error if any check fails
}
```

## Testing Plan

### Manual testing
```bash
# Build foundation
./bin/pctl ami build -t templates/examples/starter-usw2.yaml \
  --name foundation-test --subnet-id subnet-xxx --key-name test-key

# Note the AMI ID (e.g., ami-123)

# Build derived AMI
./bin/pctl ami build -t templates/examples/minimal.yaml \
  --name derived-test --base-ami ami-123 \
  --subnet-id subnet-xxx --key-name test-key

# Verify: Should complete much faster, only install delta packages
```

### Error cases to test
- Invalid AMI ID: `--base-ami ami-invalid`
- AMI in wrong region
- AMI that doesn't exist
- AMI that is still building (state: pending)

## Documentation Updates

**File**: `docs/ami-building.md`

Add section:
```markdown
## Building Layered AMIs

You can build AMIs on top of existing AMIs to save time:

1. Build a foundation AMI with common packages:
   ```bash
   pctl ami build -t foundation.yaml --name my-foundation
   ```

2. Note the AMI ID from the output (e.g., `ami-0abcd1234`)

3. Build derived AMIs using the foundation:
   ```bash
   pctl ami build -t gromacs.yaml --name my-gromacs \
     --base-ami ami-0abcd1234
   ```

This approach can reduce build times by 40-60% when building multiple
applications that share common dependencies.
```

## Future Work (Out of Scope)

- Auto-detection of base AMI (Phase 3)
- Template inheritance with `extends` keyword (Phase 2)
- Multi-level AMI chains (Phase 4)

## Dependencies

- None (can be implemented immediately after v1.0.0)

## Risks & Mitigations

**Risk**: User provides incompatible base AMI (wrong OS, wrong ParallelCluster version)
**Mitigation**: Add validation checks, clear error messages

**Risk**: User builds on deleted base AMI
**Mitigation**: Store base AMI ID in derived AMI metadata for tracking

## Related Issues

- #[TBD] Phase 2: Template Inheritance
- #[TBD] Phase 3: Auto-chaining & Caching
