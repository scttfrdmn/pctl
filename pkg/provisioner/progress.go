// Copyright 2025 Scott Friedman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/schollz/progressbar/v3"
)

// ProgressMonitor monitors cluster creation progress via CloudFormation events
type ProgressMonitor struct {
	cfnClient   *cloudformation.Client
	stackName   string
	region      string
	clusterName string
	startTime   time.Time
}

// ResourceStatus tracks the status of a CloudFormation resource
type ResourceStatus struct {
	LogicalID  string
	Type       string
	Status     types.ResourceStatus
	StatusText string
	Timestamp  time.Time
}

// estimatedResourceTimes maps AWS resource types to their typical creation times
var estimatedResourceTimes = map[string]time.Duration{
	// Fast resources (< 30s)
	"AWS::IAM::InstanceProfile":            15 * time.Second,
	"AWS::EC2::SecurityGroupIngress":       10 * time.Second,
	"AWS::EC2::SecurityGroupEgress":        10 * time.Second,
	"AWS::EC2::VPCGatewayAttachment":       20 * time.Second,
	"AWS::EC2::SubnetRouteTableAssociation": 15 * time.Second,

	// Medium resources (30s - 2m)
	"AWS::EC2::VPC":                      30 * time.Second,
	"AWS::EC2::InternetGateway":          45 * time.Second,
	"AWS::EC2::Subnet":                   60 * time.Second,
	"AWS::EC2::RouteTable":               45 * time.Second,
	"AWS::EC2::Route":                    30 * time.Second,
	"AWS::EC2::SecurityGroup":            30 * time.Second,
	"AWS::IAM::Role":                     60 * time.Second,
	"AWS::IAM::Policy":                   45 * time.Second,
	"AWS::Lambda::Function":              90 * time.Second,
	"AWS::CloudWatch::Dashboard":         60 * time.Second,
	"AWS::CloudWatch::CompositeAlarm":    45 * time.Second,
	"AWS::CloudWatch::LogGroup":          30 * time.Second,
	"AWS::Logs::LogGroup":                30 * time.Second,
	"AWS::SNS::Topic":                    30 * time.Second,
	"AWS::SQS::Queue":                    30 * time.Second,

	// Slow resources (2m - 5m)
	"AWS::EC2::Instance":                 180 * time.Second, // 3 minutes
	"AWS::EC2::Volume":                   120 * time.Second,
	"AWS::CloudFormation::WaitCondition": 300 * time.Second, // 5 minutes
}

// NewProgressMonitor creates a new progress monitor
func NewProgressMonitor(ctx context.Context, stackName, region, clusterName string) (*ProgressMonitor, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &ProgressMonitor{
		cfnClient:   cloudformation.NewFromConfig(cfg),
		stackName:   stackName,
		region:      region,
		clusterName: clusterName,
		startTime:   time.Now(),
	}, nil
}

// MonitorCreation monitors cluster creation and displays real-time progress
func (pm *ProgressMonitor) MonitorCreation(ctx context.Context) error {
	fmt.Printf("\n🚀 Monitoring cluster creation: %s\n\n", pm.clusterName)

	// Wait for stack to be created (pcluster create-cluster is async)
	fmt.Printf("⏳ Waiting for CloudFormation stack to be created...\n")
	if err := pm.waitForStackToExist(ctx); err != nil {
		return fmt.Errorf("stack creation timeout: %w", err)
	}

	// Track seen events to avoid duplicates
	seenEvents := make(map[string]bool)

	// Track resources
	resources := make(map[string]*ResourceStatus)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Initial check
	if err := pm.checkAndDisplayProgress(ctx, seenEvents, resources); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := pm.checkAndDisplayProgress(ctx, seenEvents, resources); err != nil {
				return err
			}

			// Check if stack creation is complete
			stackStatus, err := pm.getStackStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to get stack status: %w", err)
			}

			if stackStatus == types.StackStatusCreateComplete {
				fmt.Printf("\n✅ Cluster creation complete!\n")
				return nil
			} else if stackStatus == types.StackStatusCreateFailed ||
				stackStatus == types.StackStatusRollbackInProgress ||
				stackStatus == types.StackStatusRollbackComplete {
				return fmt.Errorf("cluster creation failed with status: %s", stackStatus)
			}
		}
	}
}

func (pm *ProgressMonitor) checkAndDisplayProgress(ctx context.Context, seenEvents map[string]bool, resources map[string]*ResourceStatus) error {
	// Get stack events
	events, err := pm.getStackEvents(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stack events: %w", err)
	}

	// Process new events
	newEvents := false
	for _, event := range events {
		eventKey := fmt.Sprintf("%s-%s-%s", *event.LogicalResourceId, event.ResourceStatus, event.Timestamp.String())
		if !seenEvents[eventKey] {
			seenEvents[eventKey] = true
			newEvents = true

			// Update resource tracking
			if event.LogicalResourceId != nil && *event.LogicalResourceId != pm.stackName {
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

	// Display progress if there are new events
	if newEvents {
		pm.displayProgress(resources)
	}

	return nil
}

func (pm *ProgressMonitor) displayProgress(resources map[string]*ResourceStatus) {
	// Clear previous output (simple version - just add spacing)
	fmt.Printf("\n")

	// Count resources by status
	var completed, inProgress, failed int
	resourcesToDisplay := pm.getResourcesToDisplay(resources)

	for _, res := range resources {
		switch res.Status {
		case types.ResourceStatusCreateComplete:
			completed++
		case types.ResourceStatusCreateInProgress:
			inProgress++
		case types.ResourceStatusCreateFailed:
			failed++
		}
	}

	total := len(resources)
	if total == 0 {
		fmt.Printf("⏳ Initiating cluster creation...\n")
		return
	}

	// Display active and important resources
	fmt.Printf("📦 Infrastructure Provisioning:\n")
	for _, res := range resourcesToDisplay {
		icon := pm.getStatusIcon(res.Status)
		resourceName := pm.getReadableResourceName(res.LogicalID, res.Type)
		fmt.Printf("  %s %-35s %s\n", icon, resourceName, res.Status)
	}

	// Calculate progress percentage (infrastructure phase: 0-70%)
	progressPct := 0
	if total > 0 {
		progressPct = (completed * 70) / total
	}

	// Display progress bar
	elapsed := time.Since(pm.startTime)

	fmt.Printf("\n")
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Progress"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetElapsedTime(true),
	)
	bar.Set(progressPct)
	fmt.Printf("\n")

	// Display summary with time estimates
	fmt.Printf("Resources: %d/%d created", completed, total)
	if failed > 0 {
		fmt.Printf(" (%d failed)", failed)
	}
	fmt.Printf(" | Elapsed: %s", formatDuration(elapsed))

	// Show remaining time estimate if there are incomplete resources
	if inProgress > 0 || (completed < total) {
		remainingTime := pm.calculateRemainingTime(resources)
		if remainingTime > 0 {
			etaTime := time.Now().Add(remainingTime)
			fmt.Printf(" | Remaining: ~%s | ETA: %s",
				formatDuration(remainingTime),
				etaTime.Format("15:04:05"))
		}
	}
	fmt.Printf("\n")

	if inProgress > 0 {
		fmt.Printf("⏳ %d resource(s) in progress...\n", inProgress)
	}
}

func (pm *ProgressMonitor) getResourcesToDisplay(resources map[string]*ResourceStatus) []*ResourceStatus {
	// Define critical resource types to prioritize (show first)
	criticalTypes := map[string]bool{
		"AWS::EC2::VPC":                true,
		"AWS::EC2::InternetGateway":    true,
		"AWS::EC2::Subnet":             true,
		"AWS::EC2::SecurityGroup":      true,
		"AWS::EC2::RouteTable":         true,
		"AWS::IAM::Role":               true,
		"AWS::IAM::Policy":             true,
		"AWS::EC2::Instance":           true,
		"AWS::EC2::Volume":             true,
		"AWS::CloudWatch::LogGroup":    true,
		"AWS::Lambda::Function":        true,
	}

	var critical []*ResourceStatus
	var inProgress []*ResourceStatus
	var otherRecent []*ResourceStatus

	for _, res := range resources {
		// Always show resources that are in progress
		if res.Status == types.ResourceStatusCreateInProgress || res.Status == types.ResourceStatusCreateFailed {
			if criticalTypes[res.Type] {
				critical = append(critical, res)
			} else {
				inProgress = append(inProgress, res)
			}
		} else if criticalTypes[res.Type] && res.Status == types.ResourceStatusCreateComplete {
			// Show recently completed critical resources
			critical = append(critical, res)
		} else if res.Status == types.ResourceStatusCreateComplete {
			// Track other completed resources (limit display to most recent)
			otherRecent = append(otherRecent, res)
		}
	}

	// Combine lists: critical resources first, then in-progress non-critical, then other recent
	result := append(critical, inProgress...)

	// Limit other recent resources to avoid clutter (show up to 10 most recent)
	if len(otherRecent) > 0 {
		// For now, don't show all completed non-critical resources to keep output clean
		// but DO show them if they're in progress
	}

	return result
}

func (pm *ProgressMonitor) getReadableResourceName(logicalID, resourceType string) string {
	// Simplify resource names for display
	name := logicalID

	// Remove common prefixes
	name = strings.TrimPrefix(name, pm.clusterName)
	name = strings.TrimPrefix(name, "Pctl")

	// Truncate if too long
	if len(name) > 30 {
		name = name[:27] + "..."
	}

	return name
}

func (pm *ProgressMonitor) getStatusIcon(status types.ResourceStatus) string {
	switch status {
	case types.ResourceStatusCreateComplete:
		return "✅"
	case types.ResourceStatusCreateInProgress:
		return "🔄"
	case types.ResourceStatusCreateFailed:
		return "❌"
	default:
		return "⏳"
	}
}

func (pm *ProgressMonitor) getStackEvents(ctx context.Context) ([]types.StackEvent, error) {
	input := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(pm.stackName),
	}

	result, err := pm.cfnClient.DescribeStackEvents(ctx, input)
	if err != nil {
		return nil, err
	}

	// Return events in chronological order (oldest first)
	events := result.StackEvents
	for i := len(events)/2 - 1; i >= 0; i-- {
		opp := len(events) - 1 - i
		events[i], events[opp] = events[opp], events[i]
	}

	return events, nil
}

func (pm *ProgressMonitor) getStackStatus(ctx context.Context) (types.StackStatus, error) {
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(pm.stackName),
	}

	result, err := pm.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		return "", err
	}

	if len(result.Stacks) == 0 {
		return "", fmt.Errorf("stack not found")
	}

	return result.Stacks[0].StackStatus, nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	minutes := d / time.Minute
	seconds := (d % time.Minute) / time.Second

	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// calculateRemainingTime estimates remaining time based on incomplete resources
func (pm *ProgressMonitor) calculateRemainingTime(resources map[string]*ResourceStatus) time.Duration {
	const defaultResourceTime = 60 * time.Second
	var remainingTime time.Duration

	for _, res := range resources {
		if res.Status != types.ResourceStatusCreateComplete {
			// Get estimated time for this resource type
			estimatedTime, exists := estimatedResourceTimes[res.Type]
			if !exists {
				estimatedTime = defaultResourceTime
			}

			// If resource is in progress, reduce estimate by time elapsed
			if res.Status == types.ResourceStatusCreateInProgress {
				elapsed := time.Since(res.Timestamp)
				remaining := estimatedTime - elapsed
				if remaining > 0 {
					remainingTime += remaining
				} else {
					// Resource taking longer than expected, add minimal time
					remainingTime += 30 * time.Second
				}
			} else {
				// Resource not started yet
				remainingTime += estimatedTime
			}
		}
	}

	return remainingTime
}

// waitForStackToExist waits for the CloudFormation stack to be created
// The pcluster create-cluster command initiates stack creation asynchronously
func (pm *ProgressMonitor) waitForStackToExist(ctx context.Context) error {
	maxRetries := 20 // 20 retries * 5 seconds = 100 seconds max wait
	for i := 0; i < maxRetries; i++ {
		_, err := pm.getStackStatus(ctx)
		if err == nil {
			// Stack exists
			return nil
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue waiting
		}
	}

	return fmt.Errorf("stack %s was not created within expected time", pm.stackName)
}
