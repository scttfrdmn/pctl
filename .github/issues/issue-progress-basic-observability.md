# Phase 1: Basic Observability

**Labels**: `enhancement`, `monitoring`, `progress`, `phase-1`
**Milestone**: v1.1.0
**Priority**: High
**Estimated effort**: 2-3 days

## Summary

Add disk usage reporting and package-level progress visibility to AMI builds. Users should see which package is installing, disk space usage, and helpful context about long-running operations.

## User Stories

1. **As a cluster administrator**, I want to see which Spack package is currently installing, so I know the build is progressing and not stuck.

2. **As a developer**, I want to see disk usage during builds, so I can detect if I'm running out of space before the build fails.

3. **As a first-time user**, I want to know that GCC compilation takes 60-90 minutes, so I don't think the build is frozen.

## Current Behavior

```
📦 Installing software  32% [===========>                            ]
```

Users see:
- ✅ Overall percentage
- ❌ Don't know which package is building
- ❌ Don't know disk usage
- ❌ Don't know if "stuck" at 32% is normal

## Desired Behavior

```
📦 Installing software
  [2/5] gcc@11.3.0        ████████░░░░░░░░░░ 40% (18m elapsed)
        └─ Building from source (this takes 60-90 minutes)

  Disk: 12.3 GB used / 45 GB total (27% full)
```

Users see:
- ✅ Current package (gcc@11.3.0)
- ✅ Progress within package list (2 of 5)
- ✅ Time elapsed
- ✅ Context about long operations
- ✅ Disk usage with absolute and percentage

## Acceptance Criteria

### Package-Level Progress
- [ ] Show current package name (e.g., "gcc@11.3.0")
- [ ] Show package index (e.g., "[2/5]")
- [ ] Show time elapsed for current package
- [ ] Update progress in real-time (every 10-15 seconds)

### Disk Usage Reporting
- [ ] Show before software installation starts
- [ ] Show during cleanup phase
- [ ] Show after cleanup completes
- [ ] Include: used space, total space, percentage
- [ ] Format: "12.3 GB used / 45 GB total (27%)"

### Context Messages
- [ ] Detect long-running packages (>30 min)
- [ ] Show helpful message: "Building from source (60-90 min)"
- [ ] Messages for: gcc, llvm, openmpi, python
- [ ] Generic message for unknown packages

### Error Detection
- [ ] Warn if disk usage >80% during build
- [ ] Warn if disk usage >90% before cleanup
- [ ] Suggest fixes: larger volume, cleanup, instance type

## Implementation Notes

### Package Progress Tracking

Modify `pkg/ami/userdata.go` to emit package markers:

```go
// Before each Spack install
script.WriteString(fmt.Sprintf("echo 'PCTL_PKG_START:%s:%d/%d'\n",
    pkg, index, total))
script.WriteString(fmt.Sprintf("echo 'PCTL_PKG_INFO:%s has %d dependencies'\n",
    pkg, depCount))

// After each Spack install
script.WriteString(fmt.Sprintf("echo 'PCTL_PKG_END:%s:%s'\n",
    pkg, status))
```

### Disk Usage Reporting

Add to `pkg/ami/cleanup.go` before cleanup:

```go
func GenerateCleanupScript(customScript string) string {
    var script strings.Builder

    // ... existing code ...

    // Add disk usage reporting
    script.WriteString("# Disk Usage Before Cleanup\n")
    script.WriteString("echo 'PCTL_DISK_USAGE:'\n")
    script.WriteString("df -h / | tail -1 | awk '{print \"USED:\"$3\" TOTAL:\"$2\" PCT:\"$5}'\n")
    script.WriteString("echo 'Top directories:'\n")
    script.WriteString("du -h /opt /tmp /var 2>/dev/null | sort -hr | head -5\n\n")

    // ... rest of cleanup ...

    script.WriteString("# Disk Usage After Cleanup\n")
    script.WriteString("echo 'PCTL_DISK_USAGE:'\n")
    script.WriteString("df -h / | tail -1 | awk '{print \"USED:\"$3\" TOTAL:\"$2\" PCT:\"$5}'\n\n")

    return script.String()
}
```

### Progress Parser Enhancement

Update `pkg/ami/builder.go` console parsing:

```go
func (b *Builder) parseProgress(output string) *ProgressInfo {
    // Existing percentage parsing...

    // Parse package markers
    if strings.Contains(output, "PCTL_PKG_START:") {
        parts := strings.Split(output, "PCTL_PKG_START:")
        if len(parts) > 1 {
            info := strings.TrimSpace(parts[1])
            // Parse: "gcc@11.3.0:2/5"
            progress.CurrentPackage = extractPackageName(info)
            progress.PackageIndex = extractIndex(info)
            progress.PackageTotal = extractTotal(info)
        }
    }

    // Parse disk usage
    if strings.Contains(output, "PCTL_DISK_USAGE:") {
        // Parse: "USED:12.3G TOTAL:45G PCT:27%"
        progress.DiskUsed = extractDiskUsed(output)
        progress.DiskTotal = extractDiskTotal(output)
        progress.DiskPercent = extractDiskPercent(output)
    }

    return progress
}
```

### Long-Running Package Detection

```go
var longRunningPackages = map[string]string{
    "gcc":     "Building from source (60-90 minutes)",
    "llvm":    "Building from source (90-120 minutes)",
    "openmpi": "Compiling parallel libraries (15-25 minutes)",
    "python":  "Building Python from source (10-20 minutes)",
}

func (b *Builder) getPackageContext(pkgName string) string {
    for pattern, message := range longRunningPackages {
        if strings.Contains(pkgName, pattern) {
            return message
        }
    }
    return ""
}
```

### Display Format

Update `pkg/ami/builder.go` progress bar display:

```go
func (b *Builder) displayProgress(progress *ProgressInfo) {
    if progress.CurrentPackage != "" {
        fmt.Printf("\n  [%d/%d] %-20s %s %d%% (%s elapsed)\n",
            progress.PackageIndex,
            progress.PackageTotal,
            progress.CurrentPackage,
            renderProgressBar(progress.Percent),
            progress.Percent,
            progress.Elapsed)

        if context := b.getPackageContext(progress.CurrentPackage); context != "" {
            fmt.Printf("        └─ %s\n", context)
        }
    }

    if progress.DiskUsed != "" {
        fmt.Printf("\n  Disk: %s used / %s total (%s%%)\n",
            progress.DiskUsed,
            progress.DiskTotal,
            progress.DiskPercent)

        if progress.DiskPercent > 80 {
            fmt.Printf("  ⚠️  Warning: Disk usage high, cleanup will free space\n")
        }
    }
}
```

## Testing Plan

### Unit Tests

```go
func TestPackageProgressParsing(t *testing.T) {
    output := "PCTL_PKG_START:gcc@11.3.0:2/5"
    progress := parseProgress(output)

    assert.Equal(t, "gcc@11.3.0", progress.CurrentPackage)
    assert.Equal(t, 2, progress.PackageIndex)
    assert.Equal(t, 5, progress.PackageTotal)
}

func TestDiskUsageParsing(t *testing.T) {
    output := "PCTL_DISK_USAGE:\nUSED:12.3G TOTAL:45G PCT:27%"
    progress := parseProgress(output)

    assert.Equal(t, "12.3 GB", progress.DiskUsed)
    assert.Equal(t, "45 GB", progress.DiskTotal)
    assert.Equal(t, 27, progress.DiskPercent)
}

func TestLongRunningPackageDetection(t *testing.T) {
    context := getPackageContext("gcc@11.3.0")
    assert.Contains(t, context, "60-90 minutes")

    context = getPackageContext("unknown-package@1.0")
    assert.Equal(t, "", context)
}
```

### Integration Tests

```bash
# Test 1: Build with monitoring
$ pctl ami build -t starter-usw2.yaml --name test-monitoring

# Expected output:
📦 Installing software
  [1/5] gcc@11.3.0        ██░░░░░░░░░░░░░░░░ 10% (5m elapsed)
        └─ Building from source (60-90 minutes)

  Disk: 8.2 GB used / 45 GB total (18%)

# Test 2: Verify disk warnings
# Manually fill disk to 85% before build
$ pctl ami build -t starter-usw2.yaml

# Expected:
  Disk: 38.3 GB used / 45 GB total (85%)
  ⚠️  Warning: Disk usage high, cleanup will free space
```

## Documentation Updates

**File**: `docs/ami-building.md`

Add section:

```markdown
### Monitoring Build Progress

pctl shows real-time progress during AMI builds:

**Package-level progress:**
- Current package being installed
- Package number (e.g., 2 of 5)
- Time elapsed for current package

**Disk usage:**
- Shown before installation, during cleanup, and after
- Warns if disk space is running low

**Long-running operations:**
- Packages like gcc show helpful context
- "Building from source (60-90 minutes)" for gcc
- "This takes a while" messages prevent confusion

Example output:
```
📦 Installing software
  [2/5] gcc@11.3.0        ████████░░░░░░░░░░ 40% (18m elapsed)
        └─ Building from source (this takes 60-90 minutes)

  Disk: 12.3 GB used / 45 GB total (27% full)
```

### Troubleshooting

**Build appears stuck at same percentage:**
- Check which package is installing
- Some packages (gcc, llvm) take 60-90 minutes
- Look for context messages about long operations

**Disk space warnings:**
- pctl warns if disk usage >80%
- Cleanup will free 5-10 GB after build
- If needed: use larger volume with --volume-size
```

## Future Enhancements (Out of Scope)

- Historical time estimates ("gcc typically takes 62 min")
- Package dependency tree visualization
- Real-time log tailing (--verbose)
- JSON progress output for CI/CD

## Dependencies

None - standalone feature

## Related Issues

- #[TBD] Phase 2: Verbose Mode & Spack Integration
- #[TBD] Phase 3: Advanced Features

## Success Metrics

Post-implementation:
- Users don't ask "is my build stuck?" (confidence)
- Disk space issues caught early (fewer failures)
- Clear understanding of long operations (satisfaction)
