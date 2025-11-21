# Phase 3: Cluster Configuration Monitoring (70-100% Progress)

## Status: Ready to Implement

## Overview
Extend progress monitoring beyond CloudFormation infrastructure (0-70%) to include cluster-level configuration and readiness (70-100%). Currently, monitoring stops at 70% when CloudFormation completes, but the cluster may not be ready for use.

## Current State
- ✅ Phase 1: CloudFormation monitoring (0-70%)
- ✅ Phase 2: Time estimates (if implemented)
- ❌ No visibility into cluster initialization after stack creation
- ❌ Progress stops at 70%, users don't know when cluster is truly ready
- ❌ No monitoring of:
  - Head node initialization
  - Slurm scheduler setup
  - Compute fleet readiness
  - Software configuration completion

## Problem Statement
After CloudFormation stack reaches CREATE_COMPLETE (70% progress), the cluster still needs several minutes to:
1. Head node finishes cloud-init/cfn-init scripts
2. Slurm controller starts and becomes active
3. Compute nodes register (if min > 0)
4. Custom bootstrap scripts complete
5. Cluster reaches "CREATE_COMPLETE" status

Users currently see:
```
✅ Cluster creation complete!    # This is misleading - only CloudFormation is complete
```

## Proposed Solution
Monitor cluster status using ParallelCluster APIs to track initialization from 70% → 100%.

### Architecture

```
Phase 1 (0-70%): CloudFormation Resources
├─ Stack events monitoring
└─ Resource creation tracking

Phase 2 (70-100%): Cluster Configuration ← NEW
├─ Head node initialization
├─ Slurm controller startup
├─ Compute fleet readiness
└─ Cluster status: CREATE_COMPLETE
```

## Implementation Details

### 1. Cluster Status Polling
After CloudFormation completes, poll cluster status:

```go
// pkg/provisioner/progress.go

func (pm *ProgressMonitor) MonitorClusterConfiguration(ctx context.Context) error {
    fmt.Printf("\n🎯 Cluster Configuration:\n")

    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            status, err := pm.getClusterStatus(ctx)
            if err != nil {
                return fmt.Errorf("failed to get cluster status: %w", err)
            }

            progress := pm.calculateClusterProgress(status)
            pm.displayClusterProgress(status, progress)

            if status.ClusterStatus == "CREATE_COMPLETE" {
                fmt.Printf("\n✅ Cluster fully ready!\n")
                return nil
            }

            if status.ClusterStatus == "CREATE_FAILED" {
                return fmt.Errorf("cluster configuration failed")
            }
        }
    }
}

func (pm *ProgressMonitor) getClusterStatus(ctx context.Context) (*ClusterStatus, error) {
    // Use pcluster CLI to get status
    cmd := exec.CommandContext(ctx, "pcluster", "describe-cluster",
        "--cluster-name", pm.clusterName,
        "--region", pm.region,
        "--query", "clusterStatus,computeFleetStatus,scheduler",
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }

    var status ClusterStatus
    if err := json.Unmarshal(output, &status); err != nil {
        return nil, err
    }

    return &status, nil
}
```

### 2. Cluster Status Types

```go
type ClusterStatus struct {
    ClusterStatus      string `json:"clusterStatus"`
    ComputeFleetStatus string `json:"computeFleetStatus"`
    Scheduler          struct {
        Type string `json:"type"`
    } `json:"scheduler"`
}

type ClusterConfigPhase struct {
    Name        string
    Status      string
    Progress    int
    Description string
}
```

### 3. Progress Calculation (70-100%)

```go
func (pm *ProgressMonitor) calculateClusterProgress(status *ClusterStatus) int {
    baseProgress := 70 // Infrastructure complete

    // Cluster status mapping:
    // CREATE_IN_PROGRESS: 70-90%
    // Compute fleet status: +10%

    switch status.ClusterStatus {
    case "CREATE_IN_PROGRESS":
        // Partial progress based on compute fleet
        switch status.ComputeFleetStatus {
        case "STARTING":
            return baseProgress + 10 // 80%
        case "RUNNING":
            return baseProgress + 15 // 85%
        default:
            return baseProgress + 5 // 75%
        }
    case "CREATE_COMPLETE":
        return 100
    case "CREATE_FAILED":
        return baseProgress // Stay at 70% on failure
    default:
        return baseProgress
    }
}
```

### 4. Display Cluster Progress

```go
func (pm *ProgressMonitor) displayClusterProgress(status *ClusterStatus, progress int) {
    fmt.Printf("\n🎯 Cluster Configuration:\n")

    // Head node status
    headNodeStatus := "⏳ PENDING"
    if status.ClusterStatus == "CREATE_IN_PROGRESS" {
        headNodeStatus = "🔄 INITIALIZING"
    } else if status.ClusterStatus == "CREATE_COMPLETE" {
        headNodeStatus = "✅ READY"
    }
    fmt.Printf("  Head Node:        %s\n", headNodeStatus)

    // Scheduler status
    schedulerStatus := "⏳ PENDING"
    if status.ClusterStatus == "CREATE_IN_PROGRESS" {
        schedulerStatus = "🔄 STARTING"
    } else if status.ClusterStatus == "CREATE_COMPLETE" {
        schedulerStatus = "✅ ACTIVE"
    }
    fmt.Printf("  Slurm Controller: %s\n", schedulerStatus)

    // Compute fleet status
    computeIcon := "⏳"
    if status.ComputeFleetStatus == "RUNNING" {
        computeIcon = "✅"
    } else if status.ComputeFleetStatus == "STARTING" {
        computeIcon = "🔄"
    }
    fmt.Printf("  Compute Fleet:    %s %s\n", computeIcon, status.ComputeFleetStatus)

    // Progress bar
    elapsed := time.Since(pm.startTime)
    fmt.Printf("\n")
    bar := progressbar.NewOptions(100,
        progressbar.OptionSetDescription("Progress"),
        progressbar.OptionSetWidth(40),
        progressbar.OptionShowCount(),
    )
    bar.Set(progress)
    fmt.Printf("\n")

    fmt.Printf("Status: %s | Elapsed: %s\n", status.ClusterStatus, formatDuration(elapsed))
}
```

### 5. Integration with Main Monitor

```go
// pkg/provisioner/progress.go

func (pm *ProgressMonitor) MonitorCreation(ctx context.Context) error {
    fmt.Printf("\n🚀 Monitoring cluster creation: %s\n\n", pm.clusterName)

    // Phase 1: CloudFormation infrastructure (0-70%)
    if err := pm.monitorInfrastructure(ctx); err != nil {
        return err
    }

    // Phase 2: Cluster configuration (70-100%)
    if err := pm.MonitorClusterConfiguration(ctx); err != nil {
        return err
    }

    return nil
}

func (pm *ProgressMonitor) monitorInfrastructure(ctx context.Context) error {
    // Existing CloudFormation monitoring code
    // ... (current MonitorCreation implementation)
}
```

## Expected Output

### Complete Flow:
```
🚀 Monitoring cluster creation: my-cluster

📦 Infrastructure Provisioning:
  ✅ VPC                                 CREATE_COMPLETE
  ✅ SecurityGroup                       CREATE_COMPLETE
  ✅ HeadNode                            CREATE_COMPLETE
  ✅ CloudwatchDashboard                 CREATE_COMPLETE

Progress  70% |████████████████████████████            | (70/100)
Resources: 48/48 created | Elapsed: 8m 51s

🎯 Cluster Configuration:
  Head Node:        🔄 INITIALIZING
  Slurm Controller: 🔄 STARTING
  Compute Fleet:    🔄 STARTING

Progress  85% |██████████████████████████████████      | (85/100)
Status: CREATE_IN_PROGRESS | Elapsed: 10m 15s

🎯 Cluster Configuration:
  Head Node:        ✅ READY
  Slurm Controller: ✅ ACTIVE
  Compute Fleet:    ✅ RUNNING

Progress  100% |████████████████████████████████████████| (100/100)
Status: CREATE_COMPLETE | Elapsed: 11m 23s

✅ Cluster fully ready!
```

## Acceptance Criteria
- [ ] Progress continues from 70% after CloudFormation completion
- [ ] Cluster status polled every 10 seconds
- [ ] Head node initialization displayed
- [ ] Slurm controller status shown
- [ ] Compute fleet status tracked
- [ ] Progress reaches 100% when cluster is truly ready
- [ ] Clear indication of what's still initializing
- [ ] Total elapsed time includes full initialization
- [ ] Error handling for failed cluster configuration

## Edge Cases to Handle
1. **Compute fleet disabled**: Skip compute fleet monitoring
2. **Long initialization**: Some clusters take > 10 minutes for bootstrap
3. **Configuration timeout**: Set reasonable timeout (e.g., 30 minutes total)
4. **Status API failures**: Retry with exponential backoff
5. **Cluster in wrong state**: Handle edge cases where status is unexpected

## Testing Plan
1. Create cluster and verify 70% → 100% progression
2. Test with different configurations:
   - Minimal cluster (no compute nodes)
   - Cluster with compute fleet (min > 0)
   - Cluster with custom bootstrap scripts
3. Verify status transitions are logical
4. Test timeout scenarios
5. Verify final "ready" state is accurate

## Files to Create/Modify
- `pkg/provisioner/progress.go` - Add cluster status monitoring
- Update `MonitorCreation()` - Two-phase monitoring
- Add `monitorInfrastructure()` - Renamed from MonitorCreation
- Add `MonitorClusterConfiguration()` - New cluster status phase
- Add `ClusterStatus` type - Status data structure
- Add `getClusterStatus()` - Call pcluster describe-cluster
- Add `calculateClusterProgress()` - 70-100% calculation
- Add `displayClusterProgress()` - Cluster phase display

## Dependencies
- Phase 1 must be complete ✅
- pcluster CLI must be available (already required)
- No new external dependencies

## Estimated Implementation Time
- Implementation: 2-3 hours
- Testing & verification: 1-2 hours
- **Total: 3-5 hours**

## Related Issues
- Phase 1: Basic CloudFormation Event Monitoring (COMPLETE ✅)
- Phase 2: Time Estimates
- Phase 4: Error Handling & Polish

## Benefits
- Users know when cluster is truly ready
- No confusion about "creation complete" vs "ready to use"
- Better visibility into initialization process
- Accurate 0-100% progress tracking
- Professional UX matching other tools (kubectl, terraform, etc.)
