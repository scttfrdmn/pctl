# Phase 4: Error Handling & Polish for Progress Monitoring

## Status: Ready to Implement

## Overview
Add comprehensive error handling, rollback monitoring, and UX polish to the progress monitoring system. Currently, error cases are minimally handled and rollback scenarios provide limited visibility.

## Current State
- ✅ Phase 1: Basic CloudFormation monitoring
- ✅ Phase 2: Time estimates (if implemented)
- ✅ Phase 3: Cluster status monitoring (if implemented)
- ❌ Limited error handling
- ❌ No rollback progress visibility
- ❌ Generic error messages without actionable details
- ❌ No AWS Console links for troubleshooting
- ❌ No detailed failure diagnostics

## Problems to Solve

### 1. Failed Resource Identification
When CloudFormation creation fails, users need to know:
- Which resource failed
- Why it failed (error message)
- How to troubleshoot (AWS console link)
- What was the stack status reason

**Current behavior:**
```
Error: cluster creation failed with status: CREATE_FAILED
```

**Desired behavior:**
```
❌ Cluster creation failed!

Failed Resource: HeadNode (AWS::EC2::Instance)
Reason: The subnet ID 'subnet-123456' does not exist
Status: CREATE_FAILED
Timestamp: 2025-11-20 14:23:15

View in AWS Console:
https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/stackinfo?stackId=...

Troubleshooting:
• Verify the subnet ID exists in your AWS account
• Check VPC configuration in the template
• See: https://docs.aws.amazon.com/parallelcluster/...
```

### 2. Rollback Progress Monitoring
When stack creation fails, CloudFormation rolls back. Users need visibility:

```
❌ Resource creation failed, rolling back...

🔄 Rollback Progress:
  ✅ HeadNode                            DELETE_COMPLETE
  🔄 SecurityGroup                       DELETE_IN_PROGRESS
  ⏳ VPC                                 DELETE_PENDING

Rollback: 15/20 resources deleted | Elapsed: 2m 15s
```

### 3. Timeout Handling
Gracefully handle timeouts and hung resources:

```
⚠️  Warning: Resource creation timeout

Resource: HeadNodeWaitCondition
Status: CREATE_IN_PROGRESS (stuck for 15m)
Expected time: ~5m

This may indicate:
• EC2 instance failed to signal completion
• Network connectivity issues
• Custom bootstrap script hanging

Check CloudWatch Logs:
https://console.aws.amazon.com/cloudwatch/home?region=us-west-2#logStream:group=/aws/parallelcluster/my-cluster
```

## Implementation Details

### 1. Enhanced Error Detection

```go
// pkg/provisioner/progress.go

type FailureDetails struct {
    FailedResource   *ResourceStatus
    StackStatus      types.StackStatus
    StackStatusReason string
    FailureTime      time.Time
    ConsoleURL       string
}

func (pm *ProgressMonitor) detectAndReportFailure(ctx context.Context) (*FailureDetails, error) {
    // Get stack details
    stack, err := pm.getStackDetails(ctx)
    if err != nil {
        return nil, err
    }

    // Find failed resources
    failedResources := pm.getFailedResources(ctx)
    if len(failedResources) == 0 {
        return nil, nil
    }

    // Get most recent failure
    latestFailure := failedResources[0]

    details := &FailureDetails{
        FailedResource:    latestFailure,
        StackStatus:       stack.StackStatus,
        StackStatusReason: aws.ToString(stack.StackStatusReason),
        FailureTime:       latestFailure.Timestamp,
        ConsoleURL:        pm.getConsoleURL(),
    }

    return details, nil
}

func (pm *ProgressMonitor) displayFailureDetails(details *FailureDetails) {
    fmt.Printf("\n❌ Cluster creation failed!\n\n")

    fmt.Printf("Failed Resource: %s (%s)\n",
        details.FailedResource.LogicalID,
        details.FailedResource.Type)

    if details.StackStatusReason != "" {
        fmt.Printf("Reason: %s\n", details.StackStatusReason)
    }

    fmt.Printf("Status: %s\n", details.FailedResource.Status)
    fmt.Printf("Timestamp: %s\n", details.FailureTime.Format("2006-01-02 15:04:05"))

    fmt.Printf("\nView in AWS Console:\n%s\n", details.ConsoleURL)

    // Provide troubleshooting hints
    pm.displayTroubleshootingHints(details.FailedResource)
}

func (pm *ProgressMonitor) displayTroubleshootingHints(res *ResourceStatus) {
    fmt.Printf("\nTroubleshooting:\n")

    hints := getTroubleshootingHints(res.Type, res.StatusText)
    for _, hint := range hints {
        fmt.Printf("• %s\n", hint)
    }
}

func getTroubleshootingHints(resourceType, statusReason string) []string {
    hints := []string{}

    switch resourceType {
    case "AWS::EC2::Instance":
        hints = append(hints, "Check instance launch logs in EC2 console")
        hints = append(hints, "Verify AMI exists in the target region")
        hints = append(hints, "Check EC2 service quotas (vCPU limits)")
    case "AWS::EC2::Subnet":
        hints = append(hints, "Verify VPC exists and is accessible")
        hints = append(hints, "Check subnet CIDR doesn't overlap")
    case "AWS::IAM::Role":
        hints = append(hints, "Verify IAM permissions to create roles")
        hints = append(hints, "Check for IAM policy conflicts")
    }

    return hints
}
```

### 2. Rollback Monitoring

```go
func (pm *ProgressMonitor) monitorRollback(ctx context.Context) error {
    fmt.Printf("\n❌ Resource creation failed, rolling back...\n\n")

    seenEvents := make(map[string]bool)
    resources := make(map[string]*ResourceStatus)

    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            events, err := pm.getStackEvents(ctx)
            if err != nil {
                return err
            }

            // Track DELETE events
            for _, event := range events {
                if strings.Contains(string(event.ResourceStatus), "DELETE") {
                    eventKey := fmt.Sprintf("%s-%s", *event.LogicalResourceId, event.ResourceStatus)
                    if !seenEvents[eventKey] {
                        seenEvents[eventKey] = true
                        resources[*event.LogicalResourceId] = &ResourceStatus{
                            LogicalID:  *event.LogicalResourceId,
                            Type:       aws.ToString(event.ResourceType),
                            Status:     event.ResourceStatus,
                            StatusText: string(event.ResourceStatus),
                            Timestamp:  *event.Timestamp,
                        }
                    }
                }
            }

            pm.displayRollbackProgress(resources)

            // Check if rollback complete
            stackStatus, err := pm.getStackStatus(ctx)
            if err != nil {
                return err
            }

            if stackStatus == types.StackStatusRollbackComplete ||
                stackStatus == types.StackStatusDeleteComplete {
                fmt.Printf("\n🔄 Rollback complete\n")
                return fmt.Errorf("cluster creation failed and rolled back")
            }
        }
    }
}

func (pm *ProgressMonitor) displayRollbackProgress(resources map[string]*ResourceStatus) {
    fmt.Printf("\n🔄 Rollback Progress:\n")

    var deleted, inProgress, pending int
    for _, res := range resources {
        icon := "⏳"
        switch res.Status {
        case types.ResourceStatusDeleteComplete:
            icon = "✅"
            deleted++
        case types.ResourceStatusDeleteInProgress:
            icon = "🔄"
            inProgress++
        default:
            pending++
        }

        fmt.Printf("  %s %-35s %s\n",
            icon,
            pm.getReadableResourceName(res.LogicalID, res.Type),
            res.Status)
    }

    total := deleted + inProgress + pending
    elapsed := time.Since(pm.startTime)
    fmt.Printf("\nRollback: %d/%d resources deleted | Elapsed: %s\n",
        deleted, total, formatDuration(elapsed))
}
```

### 3. Timeout Detection

```go
type StuckResource struct {
    Resource      *ResourceStatus
    StuckDuration time.Duration
    ExpectedTime  time.Duration
}

func (pm *ProgressMonitor) detectStuckResources(resources map[string]*ResourceStatus) []StuckResource {
    var stuck []StuckResource

    for _, res := range resources {
        if res.Status != types.ResourceStatusCreateInProgress {
            continue
        }

        elapsed := time.Since(res.Timestamp)
        expected := getExpectedResourceTime(res.Type)

        // If resource is taking 3x longer than expected, consider it stuck
        if elapsed > expected*3 {
            stuck = append(stuck, StuckResource{
                Resource:      res,
                StuckDuration: elapsed,
                ExpectedTime:  expected,
            })
        }
    }

    return stuck
}

func (pm *ProgressMonitor) displayStuckResourceWarning(stuck StuckResource) {
    fmt.Printf("\n⚠️  Warning: Resource creation timeout\n\n")
    fmt.Printf("Resource: %s\n", stuck.Resource.LogicalID)
    fmt.Printf("Status: %s (stuck for %s)\n",
        stuck.Resource.Status,
        formatDuration(stuck.StuckDuration))
    fmt.Printf("Expected time: ~%s\n", formatDuration(stuck.ExpectedTime))

    fmt.Printf("\nThis may indicate:\n")
    if stuck.Resource.Type == "AWS::CloudFormation::WaitCondition" {
        fmt.Printf("• EC2 instance failed to signal completion\n")
        fmt.Printf("• Network connectivity issues\n")
        fmt.Printf("• Custom bootstrap script hanging\n")
        fmt.Printf("\nCheck CloudWatch Logs:\n")
        fmt.Printf("%s\n", pm.getCloudWatchLogsURL())
    }
}
```

### 4. AWS Console Links

```go
func (pm *ProgressMonitor) getConsoleURL() string {
    return fmt.Sprintf(
        "https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks/stackinfo?stackId=%s",
        pm.region,
        url.QueryEscape(pm.stackARN),
    )
}

func (pm *ProgressMonitor) getCloudWatchLogsURL() string {
    return fmt.Sprintf(
        "https://console.aws.amazon.com/cloudwatch/home?region=%s#logStream:group=/aws/parallelcluster/%s",
        pm.region,
        pm.clusterName,
    )
}

func (pm *ProgressMonitor) getEC2ConsoleURL(instanceID string) string {
    return fmt.Sprintf(
        "https://console.aws.amazon.com/ec2/home?region=%s#Instances:instanceId=%s",
        pm.region,
        instanceID,
    )
}
```

### 5. Integration with Main Monitor

```go
func (pm *ProgressMonitor) MonitorCreation(ctx context.Context) error {
    fmt.Printf("\n🚀 Monitoring cluster creation: %s\n\n", pm.clusterName)

    // Monitor infrastructure
    if err := pm.monitorInfrastructure(ctx); err != nil {
        // Check if it's a failure that needs rollback monitoring
        stackStatus, _ := pm.getStackStatus(ctx)
        if stackStatus == types.StackStatusRollbackInProgress {
            return pm.monitorRollback(ctx)
        }

        // Detect and report failure details
        if details, err := pm.detectAndReportFailure(ctx); err == nil && details != nil {
            pm.displayFailureDetails(details)
        }

        return err
    }

    // Monitor cluster configuration (Phase 3)
    if err := pm.MonitorClusterConfiguration(ctx); err != nil {
        return err
    }

    return nil
}

func (pm *ProgressMonitor) monitorInfrastructure(ctx context.Context) error {
    // Existing CloudFormation monitoring with timeout detection
    for {
        // ... existing monitoring code ...

        // Check for stuck resources
        stuck := pm.detectStuckResources(resources)
        for _, s := range stuck {
            pm.displayStuckResourceWarning(s)
        }

        // ... continue monitoring ...
    }
}
```

## Expected Output Examples

### Failure with Details:
```
❌ Cluster creation failed!

Failed Resource: HeadNode (AWS::EC2::Instance)
Reason: The subnet ID 'subnet-abc123' does not exist
Status: CREATE_FAILED
Timestamp: 2025-11-20 14:23:15

View in AWS Console:
https://console.aws.amazon.com/cloudformation/home?region=us-west-2#/stacks/...

Troubleshooting:
• Verify the subnet ID exists in your AWS account
• Check VPC configuration in the template
• Check AWS account permissions for EC2
```

### Rollback Monitoring:
```
❌ Resource creation failed, rolling back...

🔄 Rollback Progress:
  ✅ HeadNode                            DELETE_COMPLETE
  🔄 SecurityGroup                       DELETE_IN_PROGRESS
  🔄 IAMRole                             DELETE_IN_PROGRESS
  ⏳ VPC                                 DELETE_PENDING

Rollback: 15/20 resources deleted | Elapsed: 2m 15s
```

### Timeout Warning:
```
⚠️  Warning: Resource creation timeout

Resource: HeadNodeWaitCondition
Status: CREATE_IN_PROGRESS (stuck for 15m 23s)
Expected time: ~5m

This may indicate:
• EC2 instance failed to signal completion
• Network connectivity issues
• Custom bootstrap script hanging

Check CloudWatch Logs:
https://console.aws.amazon.com/cloudwatch/home?region=us-west-2#logStream:group=/aws/parallelcluster/my-cluster
```

## Acceptance Criteria
- [ ] Failed resources identified and displayed
- [ ] Error messages include reason and troubleshooting hints
- [ ] AWS Console links provided for debugging
- [ ] Rollback progress monitored and displayed
- [ ] Stuck resources detected (3x expected time)
- [ ] Timeout warnings displayed with diagnostic info
- [ ] CloudWatch Logs links for debugging
- [ ] EC2 Console links for instance issues
- [ ] Graceful handling of API errors
- [ ] User-friendly error messages (not raw exceptions)

## Testing Plan
1. **Test failure scenarios:**
   - Invalid subnet ID
   - Insufficient IAM permissions
   - Service quota exceeded
   - Invalid AMI ID
2. **Test rollback:**
   - Trigger failure mid-creation
   - Verify rollback monitoring
3. **Test timeouts:**
   - Simulate hung WaitCondition
   - Verify timeout warning appears
4. **Test console links:**
   - Verify all URLs are correctly formed
   - Test in different regions

## Files to Modify
- `pkg/provisioner/progress.go` - Add error handling functions
- Add `detectAndReportFailure()` - Failure detection
- Add `displayFailureDetails()` - Failure display
- Add `monitorRollback()` - Rollback monitoring
- Add `detectStuckResources()` - Timeout detection
- Add `getConsoleURL()` - AWS console links
- Add troubleshooting hints map

## Dependencies
- Phase 1 must be complete ✅
- No new external dependencies

## Estimated Implementation Time
- Implementation: 3-4 hours
- Testing failure scenarios: 2-3 hours
- **Total: 5-7 hours**

## Related Issues
- Phase 1: Basic CloudFormation Event Monitoring (COMPLETE ✅)
- Phase 2: Time Estimates
- Phase 3: Cluster Configuration Monitoring

## Benefits
- Users can quickly diagnose failures
- Reduced support burden (clear error messages)
- AWS Console integration for advanced debugging
- Rollback visibility prevents confusion
- Professional error handling UX
- Actionable troubleshooting guidance
