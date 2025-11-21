# Phase 2: Add Time Estimates to Cluster Creation Progress

## Status: Ready to Implement

## Overview
Enhance the cluster creation progress monitor with accurate time estimation based on CloudFormation resource creation patterns. Currently shows elapsed time but no remaining time estimate.

## Current State
- ✅ Phase 1 Complete: Basic progress monitoring with percentage and elapsed time
- ❌ No estimated remaining time
- ❌ No ETA for completion
- ❌ Generic progress bar without time context

## Proposed Solution
Implement dynamic time estimation based on:
1. Historical resource creation times
2. Real-time completion rate
3. Remaining resource types and counts

## Implementation Details

### 1. Resource Type Time Estimates
Create a map of typical AWS resource creation times:

```go
// pkg/provisioner/progress.go
var estimatedResourceTimes = map[string]time.Duration{
    // Fast resources (< 30s)
    "AWS::IAM::InstanceProfile":         15 * time.Second,
    "AWS::EC2::SecurityGroup":           20 * time.Second,
    "AWS::EC2::VPCGatewayAttachment":    20 * time.Second,
    "AWS::EC2::SubnetRouteTableAssociation": 15 * time.Second,

    // Medium resources (30s - 2m)
    "AWS::EC2::VPC":                     30 * time.Second,
    "AWS::EC2::InternetGateway":         45 * time.Second,
    "AWS::EC2::Subnet":                  60 * time.Second,
    "AWS::EC2::RouteTable":              45 * time.Second,
    "AWS::IAM::Role":                    60 * time.Second,
    "AWS::IAM::Policy":                  45 * time.Second,
    "AWS::Lambda::Function":             90 * time.Second,
    "AWS::CloudWatch::Dashboard":        60 * time.Second,
    "AWS::CloudWatch::CompositeAlarm":   45 * time.Second,

    // Slow resources (2m - 5m)
    "AWS::EC2::Instance":                180 * time.Second, // 3 minutes
    "AWS::EC2::Volume":                  120 * time.Second,
    "AWS::CloudFormation::WaitCondition": 300 * time.Second, // 5 minutes

    // Default for unknown resources
    "default": 60 * time.Second,
}
```

### 2. Dynamic Time Calculation
Update progress display to calculate remaining time:

```go
func (pm *ProgressMonitor) calculateRemainingTime(resources map[string]*ResourceStatus) time.Duration {
    var remainingTime time.Duration

    for _, res := range resources {
        if res.Status != types.ResourceStatusCreateComplete {
            // Get estimated time for this resource type
            estimatedTime, exists := estimatedResourceTimes[res.Type]
            if !exists {
                estimatedTime = estimatedResourceTimes["default"]
            }

            // If resource is in progress, reduce estimate by time elapsed
            if res.Status == types.ResourceStatusCreateInProgress {
                elapsed := time.Since(res.Timestamp)
                remaining := estimatedTime - elapsed
                if remaining > 0 {
                    remainingTime += remaining
                }
            } else {
                // Resource not started yet
                remainingTime += estimatedTime
            }
        }
    }

    return remainingTime
}
```

### 3. Adaptive Learning (Optional Enhancement)
Track actual creation times to improve estimates:

```go
type ResourceTiming struct {
    Type         string
    StartTime    time.Time
    CompleteTime time.Time
    Duration     time.Duration
}

func (pm *ProgressMonitor) recordResourceTiming(res *ResourceStatus) {
    // Store in metrics file for future runs
    // ~/.pctl/metrics/resource-timings.json
}
```

### 4. Update Progress Display
Modify `displayProgress()` to show time estimates:

```go
// Calculate remaining time
remainingTime := pm.calculateRemainingTime(resources)
completionRate := float64(completed) / float64(total)

// Display with ETA
fmt.Printf("\n")
bar := progressbar.NewOptions(100,
    progressbar.OptionSetDescription("Progress"),
    progressbar.OptionSetWidth(40),
    progressbar.OptionShowCount(),
    progressbar.OptionSetPredictTime(false), // We'll show our own
    progressbar.OptionShowElapsedTimeOnFinish(),
    progressbar.OptionSetElapsedTime(true),
)
bar.Set(progressPct)
fmt.Printf("\n")

// Display summary with time estimates
elapsed := time.Since(pm.startTime)
etaTime := time.Now().Add(remainingTime)

fmt.Printf("Resources: %d/%d created", completed, total)
if failed > 0 {
    fmt.Printf(" (%d failed)", failed)
}
fmt.Printf(" | Elapsed: %s", formatDuration(elapsed))

if inProgress > 0 {
    fmt.Printf(" | Remaining: ~%s", formatDuration(remainingTime))
    fmt.Printf(" | ETA: %s", etaTime.Format("15:04:05"))
}
fmt.Printf("\n")
```

## Expected Output

### Before (Current):
```
📦 Infrastructure Provisioning:
  ✅ HeadNode                            CREATE_COMPLETE
  🔄 CloudwatchDashboard88785441         CREATE_IN_PROGRESS

Progress  68% |███████████████████████████             | (68/100) [0s:0s]
Resources: 47/48 created | Elapsed: 5m 21s
⏳ 1 resource(s) in progress...
```

### After (With Time Estimates):
```
📦 Infrastructure Provisioning:
  ✅ HeadNode                            CREATE_COMPLETE
  🔄 CloudwatchDashboard88785441         CREATE_IN_PROGRESS

Progress  68% |███████████████████████████             | (68/100)
Resources: 47/48 created | Elapsed: 5m 21s | Remaining: ~1m 15s | ETA: 14:23:36
⏳ 1 resource(s) in progress...
```

## Acceptance Criteria
- [ ] Time estimates display for all resource types
- [ ] Remaining time shown when resources are in progress
- [ ] ETA (completion time) displayed
- [ ] Estimates reasonably accurate (within 30% of actual time)
- [ ] Edge cases handled:
  - [ ] Unknown resource types use default estimate
  - [ ] WaitCondition resources properly estimated
  - [ ] Multiple in-progress resources calculated correctly
- [ ] Time format is user-friendly (e.g., "2m 15s" not "135s")

## Testing Plan
1. Create test cluster and track time estimate accuracy
2. Compare estimated vs actual completion time
3. Test with different cluster configurations:
   - Small clusters (< 30 resources)
   - Large clusters (> 50 resources)
   - Clusters with WaitConditions
4. Verify estimates improve as resources complete

## Files to Modify
- `pkg/provisioner/progress.go` - Add time estimation functions
- Update `displayProgress()` - Show remaining time and ETA
- Add `calculateRemainingTime()` - Calculate remaining time
- Add `estimatedResourceTimes` map - Resource time estimates

## Dependencies
- Phase 1 must be complete ✅
- No new external dependencies required

## Estimated Implementation Time
- Implementation: 1-2 hours
- Testing & tuning: 1-2 hours
- **Total: 2-4 hours**

## Related Issues
- Phase 1: Basic CloudFormation Event Monitoring (COMPLETE ✅)
- Phase 3: Cluster Configuration Monitoring (70-100%)
- Phase 4: Error Handling & Polish
