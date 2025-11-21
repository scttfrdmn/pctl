# Progress Monitoring & Observability

**Status**: Planned for v1.1.0+
**Milestone**: v1.1.0 - v1.3.0
**Design Philosophy**: **Clear visibility without overwhelming users**

## Vision

Provide users with clear, actionable progress information during AMI builds - from high-level phase progress to detailed package-level insights - without requiring manual log inspection.

## Problem Statement

Current challenges with AMI build monitoring:
1. **Limited visibility**: Users see percentages but not what's happening
2. **No time estimates**: Unknown how long builds will take
3. **Silent phases**: GCC compilation appears stuck for 60+ minutes
4. **No diagnostics**: Can't tell if build is progressing or hung
5. **Missing disk metrics**: Can't see if space is running low
6. **No package details**: Don't know which Spack package is building

## User Experience Goals

### Beginner User (Default)
```bash
$ pctl ami build -t gromacs.yaml

📦 Installing software (estimated 45-60 minutes)
  [1/5] gcc@11.3.0         ████████░░░░░░░░░░ 40% (18m elapsed, ~25m remaining)
        └─ Phase: compile (this takes a while, building from source)

  Disk: 12.3 GB used / 45 GB total (27% full)
```

### Power User (Verbose Mode)
```bash
$ pctl ami build -t gromacs.yaml --verbose

📦 Installing software packages
════════════════════════════════════════════════════════════

[1/5] gcc@11.3.0 (18 dependencies)
  Dependencies: gmp@6.2.1 ✓, mpfr@4.1.0 ✓, mpc@1.2.1 ✓, ...
  ┌─ Phase: configure (2m 15s)
  ├─ Phase: build     (58m 42s) ← CURRENT
  │  └─ Compiling: gcc/tree-ssa-loop.c (4823 of 12450 files)
  └─ Phase: install   (pending)

Real-time log: /var/log/spack/gcc-11.3.0-build.log
Build directory: /tmp/spack-stage/gcc-11.3.0-abc123

Disk usage:
  /opt/spack:  8.2 GB (Spack installations)
  /tmp:        4.1 GB (build artifacts)
  Free space: 32.7 GB
```

## Implementation Phases

### Phase 1: Basic Observability (v1.1.0) - 2-3 days
**Goal**: Disk metrics + package-level progress

**Files**: [issue-progress-basic-observability.md](issues/issue-progress-basic-observability.md)

---

### Phase 2: Verbose Mode & Spack Integration (v1.2.0) - 1 week
**Goal**: Real-time Spack log parsing + detailed progress

**Files**: [issue-progress-verbose-mode.md](issues/issue-progress-verbose-mode.md)

---

### Phase 3: Advanced Features (v1.3.0) - 1-2 weeks
**Goal**: Time estimates, build analytics, error detection

**Files**: [issue-progress-advanced-features.md](issues/issue-progress-advanced-features.md)

---

## Design Principles

### 1. Progressive Detail
**Show relevant info at each level:**

**Level 0** (default): Phase + percentage
```
📦 Installing software  56% [=====================>  ] (12m:8m)
```

**Level 1** (--progress): Package names
```
📦 Installing openmpi@4.1.4 [3/5] 56% (12m:8m)
```

**Level 2** (--verbose): Full details
```
📦 Installing openmpi@4.1.4 [3/5]
   Dependencies: 8 installed, 0 pending
   Phase: configure → build → install
   Disk: 15.2 GB / 45 GB (34%)
```

### 2. Actionable Information
**Always tell users what to do or what's happening:**

❌ **Bad**: `Building... 32%`
✅ **Good**: `Building gcc@11.3.0 (compile phase, ~25 min remaining)`

❌ **Bad**: `Error: Build failed`
✅ **Good**: `Error: Build failed in gcc@11.3.0 configure phase
  Log: /var/log/spack/gcc-build.log:145
  Fix: Check that required dependencies are available`

### 3. No Surprises
**Warn about slow operations before they happen:**

```
📦 Installing gcc@11.3.0 [1/5]
   ⚠️  This package builds from source (60-90 minutes)
   💡 Tip: Use --verbose to see detailed compilation progress
```

### 4. Fail Fast, Fail Clear
**Detect problems early and report them clearly:**

```
❌ Error: Disk space low during gcc@11.3.0 build
   Current: 2.1 GB free
   Required: ~8 GB for gcc compilation

   Suggested fixes:
   1. Use larger instance type (c6a.4xlarge → c6a.8xlarge)
   2. Clean up build artifacts: pctl ami build --no-cache
   3. Increase EBS volume size: --volume-size 80
```

## Metrics We Track

### Build Metrics
- Package install progress (X of Y packages)
- Current package phase (configure/build/install)
- Time per package (historical + current)
- Estimated time remaining (based on historical data)

### System Metrics
- Disk usage (total, Spack, /tmp, free)
- Memory usage (optional, for debugging)
- CPU utilization (optional, for debugging)
- Network I/O (for package downloads)

### Error Metrics
- Failed packages (with logs)
- Timeout warnings
- Disk space warnings
- Dependency resolution issues

## Success Metrics

Post-implementation, we expect:

- **Clarity**: Users understand build progress without checking logs
- **Confidence**: Users know if build is progressing or stuck
- **Speed**: Issues detected early (disk full, timeouts)
- **Satisfaction**: "I always know what's happening" feedback
- **Reduced support**: Fewer "is my build stuck?" questions

## Documentation Roadmap

### For v1.1.0 (Phase 1)
- [ ] `docs/ami-building.md` - Add disk metrics section
- [ ] `docs/troubleshooting.md` - Add progress debugging guide

### For v1.2.0 (Phase 2)
- [ ] `docs/verbose-mode.md` - Document --verbose flag
- [ ] `docs/spack-integration.md` - Explain Spack log parsing

### For v1.3.0 (Phase 3)
- [ ] `docs/build-analytics.md` - Historical build data
- [ ] `docs/monitoring.md` - Advanced monitoring guide

## Future Enhancements (Out of Scope)

- **CI/CD Integration**: JSON progress API for automation
- **WebSocket Streaming**: Real-time web dashboard
- **Build Caching Analytics**: Show cache hit rates
- **Distributed Builds**: Progress across multiple instances
- **Custom Hooks**: User-defined progress callbacks

## Getting Started (Post-Implementation)

After Phase 1 is complete, users will immediately see:

```bash
$ pctl ami build -t starter-usw2.yaml

📋 Template: starter-usw2.yaml
   Packages: gcc, openmpi, python, cmake, git (5 total)

📦 Installing software
  [1/5] gcc@11.3.0         ████░░░░░░░░░░░░░░ 20% (15m:45m)

  Disk: 12.3 GB used / 45 GB (27%)
  💡 Building gcc from source, this will take 60-90 minutes
```

After Phase 2 (verbose mode):
```bash
$ pctl ami build -t starter-usw2.yaml --verbose

[Shows detailed Spack output, dependency trees, real-time logs]
```

After Phase 3 (analytics):
```bash
$ pctl ami build -t starter-usw2.yaml

📦 Installing gcc@11.3.0 [1/5]
   Estimated: 62 minutes (based on 3 previous builds)
   Average build time: 61.5 min (±5.2 min)

   Similar builds:
   - foundation-v1: 58 minutes (3 days ago)
   - test-build: 65 minutes (1 week ago)
```

## Questions?

See individual phase documents for implementation details:
- [Phase 1: Basic Observability](.github/issues/issue-progress-basic-observability.md)
- [Phase 2: Verbose Mode](.github/issues/issue-progress-verbose-mode.md)
- [Phase 3: Advanced Features](.github/issues/issue-progress-advanced-features.md)
