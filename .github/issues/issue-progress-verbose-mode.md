# Phase 2: Verbose Mode & Spack Integration

**Labels**: `enhancement`, `monitoring`, `spack`, `phase-2`
**Milestone**: v1.2.0
**Priority**: Medium
**Estimated effort**: 1 week

## Summary

Add `--verbose` flag to show detailed real-time output from Spack builds, including dependency resolution, compilation phases, and real-time log streaming. Enables power users to debug build issues and understand what's happening under the hood.

## User Stories

1. **As a developer debugging a build failure**, I want to see detailed Spack output in real-time, so I can identify which dependency or compilation step failed.

2. **As a power user**, I want to see which files are being compiled during gcc build, so I know it's progressing even though it takes 60 minutes.

3. **As a cluster administrator**, I want to see the full dependency tree before installation, so I understand what will be built.

## Current Behavior (After Phase 1)

```
📦 Installing software
  [2/5] gcc@11.3.0        ████████░░░░░░░░░░ 40% (18m elapsed)
        └─ Building from source (this takes 60-90 minutes)

  Disk: 12.3 GB used / 45 GB total (27%)
```

Good, but:
- ❌ Can't see what's happening inside Spack
- ❌ Can't debug compilation errors
- ❌ Can't see dependency resolution
- ❌ Limited visibility into long operations

## Desired Behavior (Verbose Mode)

```bash
$ pctl ami build -t starter-usw2.yaml --verbose

📦 Installing software packages
════════════════════════════════════════════════════════════

[1/5] gcc@11.3.0

Dependencies (18 total):
  ✓ zlib@1.2.13
  ✓ gmp@6.2.1
  ✓ mpfr@4.1.0
  ✓ mpc@1.2.1
  ... (14 more)

Phases:
  ┌─ autoreconf  (2m 15s) ✓
  ├─ configure   (3m 42s) ✓
  ├─ build       (58m 12s) ← CURRENT
  │  └─ [4823/12450] Compiling gcc/tree-ssa-loop.c
  │     CC gcc/tree-ssa-loop.o
  │     gcc/tree-ssa-loop.c: 15234 lines, optimizing...
  │
  └─ install     (pending)

Real-time log: /var/log/spack/gcc-11.3.0-xyz.log
Build directory: /tmp/spack-stage/gcc-11.3.0-xyz/spack-build
Spack spec: gcc@11.3.0%gcc@7.5.0 arch=linux-amzn2-x86_64

Progress: 38.7% (4823 of 12450 source files compiled)
```

## Acceptance Criteria

### Verbose Flag
- [ ] Add `--verbose` flag to `ami build` command
- [ ] Default: off (shows Phase 1 compact progress)
- [ ] When enabled: show detailed Spack output
- [ ] Output is still parseable for progress tracking

### Dependency Tree Display
- [ ] Show all dependencies before installation
- [ ] Mark already-installed dependencies with ✓
- [ ] Show dependency count (e.g., "18 total")
- [ ] Indent to show hierarchy

### Build Phase Tracking
- [ ] Detect Spack phases: autoreconf, configure, build, install
- [ ] Show time for completed phases
- [ ] Show current phase with ← marker
- [ ] Show sub-phase details (file being compiled)

### Real-time Log Streaming
- [ ] Stream Spack output in real-time
- [ ] Show compilation progress (file X of Y)
- [ ] Show current file being compiled
- [ ] Preserve log file path for debugging

### Spack Integration
- [ ] Parse `spack install --verbose` output
- [ ] Detect build progress from make output
- [ ] Extract file counts from compilation logs
- [ ] Handle errors gracefully

## Implementation Notes

### CLI Flag Addition

```go
// cmd/pctl/ami.go
var amiVerbose bool

buildAMICmd.Flags().BoolVar(&amiVerbose, "verbose", false,
    "show detailed Spack output and real-time build progress")
```

### Userdata Script Modification

```go
// pkg/ami/userdata.go
func GenerateUserData(config *Config, verbose bool) (string, error) {
    // ...

    if verbose {
        // Use Spack's verbose mode
        script.WriteString("export SPACK_INSTALL_FLAGS='--verbose'\n")

        // Emit detailed progress markers
        script.WriteString("echo 'PCTL_VERBOSE_MODE:enabled'\n")

        // Redirect Spack output for parsing
        script.WriteString("/opt/spack/bin/spack install ${pkg} 2>&1 | tee -a /var/log/spack/build.log | while IFS= read -r line; do\n")
        script.WriteString("  echo \"PCTL_SPACK_OUTPUT: $line\"\n")
        script.WriteString("done\n")
    } else {
        // Compact mode (current behavior)
        script.WriteString("/opt/spack/bin/spack install ${pkg} > /dev/null 2>&1\n")
    }
}
```

### Spack Output Parser

```go
// pkg/ami/spack_parser.go
type SpackProgress struct {
    Package         string
    Phase           string
    PhaseDuration   time.Duration
    CurrentFile     string
    FilesCompiled   int
    TotalFiles      int
    Dependencies    []Dependency
    LogPath         string
}

type Dependency struct {
    Name      string
    Installed bool
}

func ParseSpackOutput(line string) (*SpackProgress, error) {
    progress := &SpackProgress{}

    // Parse dependency list
    // Example: "==> Installing gmp-6.2.1-xyz"
    if strings.Contains(line, "==> Installing") {
        progress.Package = extractPackageName(line)
        progress.Phase = "install"
    }

    // Parse build phase
    // Example: "==> gcc: Executing phase: 'build'"
    if strings.Contains(line, "Executing phase") {
        progress.Phase = extractPhase(line)
        progress.PhaseStart = time.Now()
    }

    // Parse compilation progress
    // Example: "[1234/5678] CC  gcc/tree-ssa.c"
    if matches := compileRegex.FindStringSubmatch(line); matches != nil {
        progress.FilesCompiled = parseInt(matches[1])
        progress.TotalFiles = parseInt(matches[2])
        progress.CurrentFile = matches[3]
    }

    // Parse log path
    // Example: "==> [2024-11-13] Log file: /var/log/spack/gcc-xyz.log"
    if strings.Contains(line, "Log file:") {
        progress.LogPath = extractLogPath(line)
    }

    return progress, nil
}

var compileRegex = regexp.MustCompile(`\[(\d+)/(\d+)\]\s+\w+\s+(.+)`)
```

### Verbose Display Handler

```go
// pkg/ami/builder.go
func (b *Builder) displayVerboseProgress(progress *SpackProgress) {
    if progress.Package != "" {
        fmt.Printf("\n[%d/%d] %s\n\n", progress.PackageIndex, progress.PackageTotal, progress.Package)
    }

    // Show dependencies
    if len(progress.Dependencies) > 0 {
        fmt.Printf("Dependencies (%d total):\n", len(progress.Dependencies))
        for _, dep := range progress.Dependencies {
            if dep.Installed {
                fmt.Printf("  ✓ %s\n", dep.Name)
            } else {
                fmt.Printf("  ○ %s (will build)\n", dep.Name)
            }
        }
        fmt.Println()
    }

    // Show phases
    fmt.Println("Phases:")
    for _, phase := range []string{"autoreconf", "configure", "build", "install"} {
        if phase == progress.Phase {
            fmt.Printf("  ├─ %-12s ← CURRENT\n", phase)

            // Show compilation details
            if progress.TotalFiles > 0 {
                pct := float64(progress.FilesCompiled) / float64(progress.TotalFiles) * 100
                fmt.Printf("  │  └─ [%d/%d] %.1f%% - Compiling %s\n",
                    progress.FilesCompiled,
                    progress.TotalFiles,
                    pct,
                    progress.CurrentFile)
            }
        } else if phaseCompleted(phase, progress) {
            fmt.Printf("  ├─ %-12s (%s) ✓\n", phase, progress.PhaseDuration)
        } else {
            fmt.Printf("  └─ %-12s (pending)\n", phase)
        }
    }

    // Show paths
    if progress.LogPath != "" {
        fmt.Printf("\nReal-time log: %s\n", progress.LogPath)
    }
}
```

### Progress Bar with Verbose Mode

```go
func (b *Builder) monitorBuild(ctx context.Context, instanceID string, verbose bool) error {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            output, err := b.getConsoleProgress(ctx, instanceID)
            if err != nil {
                continue
            }

            if verbose {
                // Parse Spack output for detailed progress
                if strings.Contains(output, "PCTL_SPACK_OUTPUT:") {
                    progress := parseSpackOutput(output)
                    b.displayVerboseProgress(progress)
                }
            } else {
                // Use existing compact progress display
                progress := parseStandardProgress(output)
                b.displayCompactProgress(progress)
            }
        }
    }
}
```

## Testing Plan

### Unit Tests

```go
func TestSpackOutputParsing(t *testing.T) {
    tests := []struct{
        input    string
        expected *SpackProgress
    }{
        {
            input: "==> Installing gmp-6.2.1-xyz",
            expected: &SpackProgress{Package: "gmp@6.2.1", Phase: "install"},
        },
        {
            input: "[1234/5678] CC  gcc/tree-ssa.c",
            expected: &SpackProgress{FilesCompiled: 1234, TotalFiles: 5678, CurrentFile: "gcc/tree-ssa.c"},
        },
    }

    for _, tt := range tests {
        result := ParseSpackOutput(tt.input)
        assert.Equal(t, tt.expected.Package, result.Package)
        assert.Equal(t, tt.expected.FilesCompiled, result.FilesCompiled)
    }
}
```

### Integration Tests

```bash
# Test 1: Verbose mode shows detailed output
$ pctl ami build -t starter-usw2.yaml --verbose

# Expected: Detailed Spack output with:
# - Dependency lists
# - Phase tracking
# - Compilation progress
# - Log file paths

# Test 2: Normal mode still works
$ pctl ami build -t starter-usw2.yaml

# Expected: Compact progress (Phase 1 behavior)

# Test 3: Verbose mode with build errors
$ pctl ami build -t broken-template.yaml --verbose

# Expected: Clear error with log file location
```

## CLI Examples

```bash
# Compact mode (default)
$ pctl ami build -t starter-usw2.yaml
📦 Installing gcc@11.3.0 [1/5] 40% (18m)

# Verbose mode
$ pctl ami build -t starter-usw2.yaml --verbose
[Shows detailed Spack output as shown in "Desired Behavior"]

# Verbose mode with log file inspection
$ pctl ami build -t starter-usw2.yaml --verbose
# ... build fails ...
$ ssh user@instance cat /var/log/spack/gcc-xyz.log
```

## Documentation Updates

**File**: `docs/verbose-mode.md` (new)

```markdown
# Verbose Mode

Use `--verbose` to see detailed build output during AMI creation.

## Basic Usage

```bash
pctl ami build -t template.yaml --verbose
```

Shows:
- Full Spack installation output
- Dependency trees
- Compilation phases (configure, build, install)
- File-by-file compilation progress
- Real-time log paths

## When to Use Verbose Mode

**Debugging build failures:**
- See exact error messages from Spack
- Locate log files for manual inspection
- Understand dependency resolution issues

**Understanding long operations:**
- See which files gcc is compiling
- Track progress through large builds
- Verify builds aren't stuck

**Learning Spack behavior:**
- Understand how packages are built
- See dependency relationships
- Learn Spack internals

## Output Format

Verbose mode shows three levels of detail:

1. **Package level**: Which package is installing
2. **Phase level**: Configure, build, install phases
3. **File level**: Individual files being compiled

Example:
```
[2/5] gcc@11.3.0

Dependencies (18 total):
  ✓ gmp@6.2.1
  ✓ mpfr@4.1.0
  ... (16 more)

Phases:
  ├─ configure  (3m 42s) ✓
  ├─ build      (58m 12s) ← CURRENT
  │  └─ [4823/12450] Compiling gcc/tree-ssa-loop.c
  └─ install    (pending)
```

## Performance Impact

Verbose mode:
- **Does not** slow down builds
- **Does** increase log output size (~2-5x)
- **Does** make progress tracking more accurate

## Troubleshooting

**Logs scrolling too fast:**
Use `tee` to capture output:
```bash
pctl ami build -t template.yaml --verbose | tee build.log
```

**Want to see logs after build:**
Logs are saved on build instance:
- `/var/log/spack/*.log` - Individual package logs
- `/var/log/cloud-init-output.log` - Full build log
```

## Future Enhancements (Out of Scope)

- Filter verbose output by package (--verbose=gcc)
- Syntax highlighting for compiler output
- Interactive mode to pause/resume builds
- Live log tailing from CLI

## Dependencies

- Phase 1: Basic Observability (for fallback display)

## Related Issues

- #[Phase 1] Basic Observability
- #[Phase 3] Advanced Features

## Success Metrics

Post-implementation:
- Build failures easier to debug (fewer support requests)
- Power users have visibility they need
- Log files easily located
- Compilation progress visible for long builds
