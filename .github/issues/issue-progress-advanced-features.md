# Phase 3: Advanced Features (Time Estimates, Analytics, Error Detection)

**Labels**: `enhancement`, `monitoring`, `analytics`, `phase-3`
**Milestone**: v1.3.0
**Priority**: Low
**Estimated effort**: 1-2 weeks

## Summary

Add intelligent features to AMI builds: historical time estimates, build analytics, proactive error detection, and ETA calculations. Make builds predictable and catch issues before they cause failures.

## User Stories

1. **As a cluster administrator**, I want to see estimated build times based on historical data, so I can plan my work accordingly.

2. **As a developer**, I want to be warned before I run out of disk space, so I can stop the build and fix the issue.

3. **As a team lead**, I want to see build analytics (average times, success rates), so I can optimize our templates.

4. **As a first-time user**, I want accurate ETAs, so I know when to check back on my build.

## Current Behavior (After Phase 2)

```bash
$ pctl ami build -t starter-usw2.yaml --verbose

📦 Installing gcc@11.3.0 [1/5] 40% (18m elapsed)
```

Good, but:
- ❌ No time estimate ("how much longer?")
- ❌ No proactive warnings (disk, memory)
- ❌ No historical context ("is this normal?")
- ❌ No build analytics

## Desired Behavior

### Time Estimates Based on History
```bash
$ pctl ami build -t starter-usw2.yaml

📦 Installing software
  [1/5] gcc@11.3.0        ████████░░░░░░░░░░ 40% (18m elapsed, ~25m remaining)
        └─ Estimated 62 minutes (based on 3 previous builds)
        └─ Your builds: 58min, 65min, 61min (avg: 61.3min ± 3.5min)

  Disk: 12.3 GB used / 45 GB (27%) - growing at 0.8 GB/min
  💡 Estimated peak: 28 GB (safe, 17 GB margin)
```

### Proactive Error Detection
```bash
$ pctl ami build -t starter-usw2.yaml

📦 Installing gcc@11.3.0 [1/5] 32%

  Disk: 38.5 GB used / 45 GB (86%)
  ⚠️  WARNING: Disk space running low!
     Current rate: 1.2 GB/min
     Estimated peak: 47 GB (exceeds 45 GB limit)

     Recommended actions:
     1. Stop build now (Ctrl+C) and increase volume size
     2. Or continue and risk failure during gcc build
     3. Command to restart: pctl ami build --volume-size 60

  [?] Continue anyway? (not recommended) [y/N]
```

### Build Analytics Dashboard
```bash
$ pctl ami stats

Build Statistics
════════════════════════════════════════════════════════════

Recent Builds (last 30 days):
  Total builds: 15
  Successful:   12 (80%)
  Failed:       3 (20%)

Average Build Times:
  starter-usw2:    68.5 min (±5.2 min) [8 builds]
  foundation:      62.1 min (±4.8 min) [5 builds]
  gromacs:         12.3 min (±1.5 min) [2 builds]

Package Build Times (average):
  gcc@11.3.0:      61.5 min ± 3.5 min
  openmpi@4.1.4:   18.2 min ± 2.1 min
  python@3.10:     12.8 min ± 1.8 min
  cmake@3.26.0:     5.1 min ± 0.8 min
  git@2.40.0:       3.2 min ± 0.5 min

Resource Usage (peak):
  Disk:            28.3 GB (foundation template)
  Memory:          4.2 GB (during gcc compile)
  CPU:             95% utilization average

Failure Analysis:
  Timeout:         2 builds (gcc compilation)
  Disk full:       1 build (insufficient space)
  Network:         0 builds

Recommendations:
  ✓ Current volume size (45 GB) is adequate
  ⚠ Consider 60-minute timeout for gcc builds
  💡 c6a.8xlarge could reduce gcc time to ~35 min
```

## Acceptance Criteria

### Time Estimation
- [ ] Store historical build times in `~/.pctl/build-history.json`
- [ ] Calculate ETA based on package progress
- [ ] Show "X minutes remaining" estimate
- [ ] Update ETA as build progresses
- [ ] Show historical average and standard deviation

### Build Analytics
- [ ] Track all builds (success/failure, duration, template)
- [ ] Store package-level timing data
- [ ] Calculate statistics (mean, std dev, percentiles)
- [ ] `pctl ami stats` command to view analytics
- [ ] Clean up old data (>90 days)

### Proactive Error Detection
- [ ] Monitor disk space growth rate
- [ ] Warn if approaching limit (>80%)
- [ ] Predict peak disk usage
- [ ] Suggest corrective actions
- [ ] Optional: pause build for user confirmation

### Smart ETAs
- [ ] Initial ETA based on template + historical data
- [ ] Refine ETA as build progresses
- [ ] Account for current package speed
- [ ] Show confidence level ("±5 min")
- [ ] Detect anomalies (build much slower than usual)

## Implementation Notes

### Build History Storage

```go
// pkg/ami/history.go
type BuildHistory struct {
    Builds []BuildRecord `json:"builds"`
}

type BuildRecord struct {
    ID                string              `json:"id"`
    Template          string              `json:"template"`
    StartTime         time.Time           `json:"start_time"`
    EndTime           time.Time           `json:"end_time"`
    Duration          time.Duration       `json:"duration"`
    Success           bool                `json:"success"`
    FailureReason     string              `json:"failure_reason,omitempty"`
    InstanceType      string              `json:"instance_type"`
    VolumeSize        int                 `json:"volume_size"`
    PackageTiming     map[string]Duration `json:"package_timing"`
    PeakDiskUsage     int64               `json:"peak_disk_usage"`
    PeakMemoryUsage   int64               `json:"peak_memory_usage"`
}

func LoadHistory() (*BuildHistory, error) {
    path := filepath.Join(os.UserHomeDir(), ".pctl", "build-history.json")
    data, err := os.ReadFile(path)
    if err != nil {
        return &BuildHistory{Builds: []BuildRecord{}}, nil // Empty history
    }

    var history BuildHistory
    if err := json.Unmarshal(data, &history); err != nil {
        return nil, err
    }

    return &history, nil
}

func (h *BuildHistory) AddBuild(record BuildRecord) error {
    h.Builds = append(h.Builds, record)

    // Cleanup old builds (>90 days)
    cutoff := time.Now().AddDate(0, 0, -90)
    var recent []BuildRecord
    for _, b := range h.Builds {
        if b.StartTime.After(cutoff) {
            recent = append(recent, b)
        }
    }
    h.Builds = recent

    return h.Save()
}

func (h *BuildHistory) Save() error {
    path := filepath.Join(os.UserHomeDir(), ".pctl", "build-history.json")
    data, err := json.MarshalIndent(h, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}
```

### ETA Calculation

```go
// pkg/ami/eta.go
type ETACalculator struct {
    history *BuildHistory
}

func (e *ETACalculator) CalculateETA(template string, currentPkg string, pkgProgress float64) (time.Duration, time.Duration) {
    // Get historical data for this template
    similar := e.history.FindSimilar(template)
    if len(similar) == 0 {
        // No history, use defaults
        return estimateDefault(template), time.Duration(0)
    }

    // Calculate average and std dev
    durations := make([]float64, len(similar))
    for i, build := range similar {
        durations[i] = build.Duration.Seconds()
    }

    mean := calculateMean(durations)
    stdDev := calculateStdDev(durations, mean)

    // Adjust for current progress
    avgPkgTime := e.getPackageAverage(currentPkg)
    elapsed := time.Since(buildStartTime)
    estimated := time.Duration(mean) * time.Second

    // Refine based on current speed
    if pkgProgress > 0 {
        currentSpeed := elapsed.Seconds() / pkgProgress
        adjusted := time.Duration(currentSpeed * (1.0 - pkgProgress)) * time.Second
        estimated = adjusted
    }

    confidence := time.Duration(stdDev) * time.Second
    return estimated, confidence
}

func (e *ETACalculator) PredictPeakDiskUsage(currentUsage int64, rate float64, remaining time.Duration) int64 {
    projected := float64(currentUsage) + (rate * remaining.Minutes())
    return int64(projected)
}
```

### Proactive Warnings

```go
// pkg/ami/warnings.go
type WarningSystem struct {
    diskMonitor    *DiskMonitor
    memoryMonitor  *MemoryMonitor
    history        *BuildHistory
}

func (w *WarningSystem) CheckForIssues(progress *ProgressInfo) []Warning {
    var warnings []Warning

    // Check disk space
    if progress.DiskPercent > 80 {
        rate := w.diskMonitor.GetGrowthRate()
        peak := w.estimatePeakDiskUsage(progress, rate)

        if peak > progress.DiskTotal {
            warnings = append(warnings, Warning{
                Level:   "ERROR",
                Message: fmt.Sprintf("Disk space will run out! Peak: %.1f GB, Available: %.1f GB",
                    float64(peak)/1e9, float64(progress.DiskTotal)/1e9),
                Suggestion: "Stop build and increase volume size with --volume-size",
                Actionable: true,
                PauseRequired: true,
            })
        } else if peak > progress.DiskTotal*0.95 {
            warnings = append(warnings, Warning{
                Level:   "WARN",
                Message: "Disk space will be tight",
                Suggestion: "Consider larger volume for future builds",
            })
        }
    }

    // Check build time anomalies
    if progress.CurrentPackage != "" {
        avgTime := w.history.GetPackageAverage(progress.CurrentPackage)
        if progress.Elapsed > avgTime*1.5 {
            warnings = append(warnings, Warning{
                Level:   "INFO",
                Message: fmt.Sprintf("%s taking longer than usual (avg: %s, current: %s)",
                    progress.CurrentPackage, avgTime, progress.Elapsed),
                Suggestion: "This may indicate system resource constraints",
            })
        }
    }

    return warnings
}

type Warning struct {
    Level         string // ERROR, WARN, INFO
    Message       string
    Suggestion    string
    Actionable    bool
    PauseRequired bool
}
```

### Stats Command

```go
// cmd/pctl/ami_stats.go
var statsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show AMI build statistics and analytics",
    RunE: func(cmd *cobra.Command, args []string) error {
        history, err := LoadHistory()
        if err != nil {
            return err
        }

        stats := calculateStatistics(history)
        displayStats(stats)

        return nil
    },
}

func displayStats(stats *Statistics) {
    fmt.Println("Build Statistics")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println()

    // Recent builds
    fmt.Printf("Recent Builds (last 30 days): %d\n", stats.TotalBuilds)
    fmt.Printf("  Successful: %d (%.0f%%)\n", stats.SuccessCount, stats.SuccessRate*100)
    fmt.Printf("  Failed:     %d (%.0f%%)\n", stats.FailureCount, (1-stats.SuccessRate)*100)
    fmt.Println()

    // Template timing
    fmt.Println("Average Build Times:")
    for template, timing := range stats.TemplateTiming {
        fmt.Printf("  %-20s %6.1f min (±%.1f min) [%d builds]\n",
            template, timing.Mean.Minutes(), timing.StdDev.Minutes(), timing.Count)
    }
    fmt.Println()

    // Package timing
    fmt.Println("Package Build Times (average):")
    for pkg, timing := range stats.PackageTiming {
        fmt.Printf("  %-20s %6.1f min ± %.1f min\n",
            pkg, timing.Mean.Minutes(), timing.StdDev.Minutes())
    }
    fmt.Println()

    // Resource usage
    fmt.Println("Resource Usage (peak):")
    fmt.Printf("  Disk:    %.1f GB\n", float64(stats.PeakDisk)/1e9)
    fmt.Printf("  Memory:  %.1f GB\n", float64(stats.PeakMemory)/1e9)
    fmt.Printf("  CPU:     %.0f%% utilization average\n", stats.AvgCPU*100)
    fmt.Println()

    // Failure analysis
    if len(stats.Failures) > 0 {
        fmt.Println("Failure Analysis:")
        for reason, count := range stats.Failures {
            fmt.Printf("  %-20s %d builds\n", reason, count)
        }
        fmt.Println()
    }

    // Recommendations
    if len(stats.Recommendations) > 0 {
        fmt.Println("Recommendations:")
        for _, rec := range stats.Recommendations {
            fmt.Printf("  %s %s\n", rec.Icon, rec.Message)
        }
    }
}
```

## Testing Plan

### Unit Tests

```go
func TestETACalculation(t *testing.T) {
    history := &BuildHistory{
        Builds: []BuildRecord{
            {Template: "starter", Duration: 60 * time.Minute},
            {Template: "starter", Duration: 65 * time.Minute},
            {Template: "starter", Duration: 58 * time.Minute},
        },
    }

    calc := NewETACalculator(history)
    eta, confidence := calc.CalculateETA("starter", "gcc", 0.4)

    assert.InDelta(t, 61*time.Minute, eta, 5*time.Minute)
    assert.Less(t, confidence, 10*time.Minute)
}

func TestDiskSpaceWarning(t *testing.T) {
    progress := &ProgressInfo{
        DiskUsed:    40 * 1e9, // 40 GB
        DiskTotal:   45 * 1e9, // 45 GB
        DiskPercent: 89,
    }

    warnings := checkForIssues(progress)

    assert.NotEmpty(t, warnings)
    assert.Contains(t, warnings[0].Message, "Disk space")
}
```

### Integration Tests

```bash
# Test 1: Build with history tracking
$ pctl ami build -t starter-usw2.yaml
$ ls ~/.pctl/build-history.json
# Verify: JSON file exists and contains build record

# Test 2: View stats after multiple builds
$ pctl ami build -t starter-usw2.yaml  # build 1
$ pctl ami build -t starter-usw2.yaml  # build 2
$ pctl ami stats
# Verify: Shows 2 builds, averages, timing

# Test 3: ETA accuracy
$ pctl ami build -t starter-usw2.yaml
# After 30% complete, verify ETA is within 20% of actual
```

## CLI Examples

```bash
# View build statistics
$ pctl ami stats

# View stats for specific template
$ pctl ami stats --template starter-usw2.yaml

# Clear history
$ pctl ami stats --clear

# Export stats to JSON
$ pctl ami stats --json > stats.json
```

## Documentation Updates

**File**: `docs/build-analytics.md` (new)

```markdown
# Build Analytics

pctl tracks build history and provides analytics to help optimize your AMI builds.

## Viewing Statistics

```bash
pctl ami stats
```

Shows:
- Recent build success/failure rates
- Average build times by template
- Package-level timing statistics
- Resource usage peaks
- Failure analysis
- Recommendations

## Time Estimates

pctl uses historical data to estimate build times:

```bash
$ pctl ami build -t starter-usw2.yaml

[1/5] gcc@11.3.0  40% (~25m remaining)
      Estimated 62 minutes (based on 3 previous builds)
```

**How it works:**
- Tracks every build duration
- Calculates average and standard deviation
- Adjusts estimate as build progresses
- Shows confidence interval (±X min)

**First build:**
Uses default estimates (no history yet)

**Subsequent builds:**
Estimates become more accurate with more data

## Proactive Warnings

pctl detects potential issues before they cause failures:

**Disk space:**
- Monitors disk growth rate
- Predicts peak usage
- Warns if approaching limit
- Suggests volume size increase

**Build time anomalies:**
- Detects builds taking longer than usual
- May indicate resource constraints
- Suggests instance type changes

## Privacy

Build history is stored locally in `~/.pctl/build-history.json`:
- Only stored on your machine
- Never sent to external servers
- Can be deleted with `pctl ami stats --clear`

## Data Retention

- Builds older than 90 days are automatically removed
- Keeps history file small (~100KB typically)
- Manual cleanup: `pctl ami stats --clear`
```

## Future Enhancements (Out of Scope)

- **Team Analytics**: Share build data across team
- **Cost Tracking**: Calculate AMI build costs
- **Performance Trends**: Track build speed over time
- **A/B Testing**: Compare instance types, Spack versions
- **CI/CD Integration**: Export metrics to monitoring systems

## Dependencies

- Phase 1: Basic Observability
- Phase 2: Verbose Mode (for accurate timing)

## Related Issues

- #[Phase 1] Basic Observability
- #[Phase 2] Verbose Mode
- #[AMI Layering] Phase 3 (shared analytics)

## Success Metrics

Post-implementation:
- Users understand typical build times (confidence)
- Disk space failures reduced by 80% (proactive warnings)
- Build time anomalies detected early (resource issues)
- Data-driven optimization (template improvements)
