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
	"encoding/json"
	"fmt"
	"os/exec"
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
	"AWS::IAM::InstanceProfile":             15 * time.Second,
	"AWS::EC2::SecurityGroupIngress":        10 * time.Second,
	"AWS::EC2::SecurityGroupEgress":         10 * time.Second,
	"AWS::EC2::VPCGatewayAttachment":        20 * time.Second,
	"AWS::EC2::SubnetRouteTableAssociation": 15 * time.Second,

	// Medium resources (30s - 2m)
	"AWS::EC2::VPC":                   30 * time.Second,
	"AWS::EC2::InternetGateway":       45 * time.Second,
	"AWS::EC2::Subnet":                60 * time.Second,
	"AWS::EC2::RouteTable":            45 * time.Second,
	"AWS::EC2::Route":                 30 * time.Second,
	"AWS::EC2::SecurityGroup":         30 * time.Second,
	"AWS::IAM::Role":                  60 * time.Second,
	"AWS::IAM::Policy":                45 * time.Second,
	"AWS::Lambda::Function":           90 * time.Second,
	"AWS::CloudWatch::Dashboard":      60 * time.Second,
	"AWS::CloudWatch::CompositeAlarm": 45 * time.Second,
	"AWS::CloudWatch::LogGroup":       30 * time.Second,
	"AWS::Logs::LogGroup":             30 * time.Second,
	"AWS::SNS::Topic":                 30 * time.Second,
	"AWS::SQS::Queue":                 30 * time.Second,

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

	// Phase 1: Monitor CloudFormation infrastructure (0-70%)
	if err := pm.monitorInfrastructure(ctx); err != nil {
		return err
	}

	// Phase 2: Monitor cluster configuration (70-100%)
	if err := pm.MonitorClusterConfiguration(ctx); err != nil {
		return err
	}

	return nil
}

// monitorInfrastructure monitors CloudFormation stack creation (Phase 1: 0-70%)
func (pm *ProgressMonitor) monitorInfrastructure(ctx context.Context) error {
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
				fmt.Printf("\n✅ Infrastructure provisioning complete! (70%%)\n")
				return nil
			} else if stackStatus == types.StackStatusCreateFailed {
				// Display failure details
				pm.displayFailureDetails(ctx)
				return fmt.Errorf("cluster creation failed")
			} else if stackStatus == types.StackStatusRollbackInProgress {
				// Monitor rollback progress
				return pm.monitorRollback(ctx)
			} else if stackStatus == types.StackStatusRollbackComplete {
				// Rollback completed, show failure details
				pm.displayFailureDetails(ctx)
				return fmt.Errorf("cluster creation failed and rolled back")
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
		"AWS::EC2::VPC":             true,
		"AWS::EC2::InternetGateway": true,
		"AWS::EC2::Subnet":          true,
		"AWS::EC2::SecurityGroup":   true,
		"AWS::EC2::RouteTable":      true,
		"AWS::IAM::Role":            true,
		"AWS::IAM::Policy":          true,
		"AWS::EC2::Instance":        true,
		"AWS::EC2::Volume":          true,
		"AWS::CloudWatch::LogGroup": true,
		"AWS::Lambda::Function":     true,
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

// getClusterStatus retrieves the cluster status from pcluster describe-cluster
func (pm *ProgressMonitor) getClusterStatus(ctx context.Context) (*pclusterDescribeResponse, error) {
	cmd := exec.CommandContext(ctx, "pcluster", "describe-cluster",
		"--cluster-name", pm.clusterName,
		"--region", pm.region,
		"--output", "json",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster status: %w", err)
	}

	// Parse the response - pcluster wraps the result in a "cluster" object
	var response struct {
		Cluster pclusterDescribeResponse `json:"cluster"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse cluster status: %w", err)
	}

	return &response.Cluster, nil
}

// calculateClusterProgress calculates progress from 70-100% based on cluster status
func (pm *ProgressMonitor) calculateClusterProgress(status *pclusterDescribeResponse) int {
	baseProgress := 70 // Infrastructure complete

	switch status.ClusterStatus {
	case "CREATE_IN_PROGRESS":
		// Cluster is initializing - progress based on compute fleet
		switch status.ComputeFleetStatus {
		case "STARTING":
			return baseProgress + 10 // 80%
		case "RUNNING":
			return baseProgress + 15 // 85%
		case "ENABLED", "PROTECTED":
			return baseProgress + 20 // 90%
		default:
			return baseProgress + 5 // 75%
		}
	case "CREATE_COMPLETE":
		return 100 // Fully ready
	case "CREATE_FAILED":
		return baseProgress // Stay at 70% on failure
	default:
		return baseProgress
	}
}

// displayClusterProgress displays cluster configuration phase progress
func (pm *ProgressMonitor) displayClusterProgress(status *pclusterDescribeResponse, progress int) {
	fmt.Printf("\n🎯 Cluster Configuration:\n")

	// Head node status
	headNodeIcon := "⏳"
	headNodeStatus := "PENDING"
	if status.ClusterStatus == "CREATE_IN_PROGRESS" {
		headNodeIcon = "🔄"
		headNodeStatus = "INITIALIZING"
	} else if status.ClusterStatus == "CREATE_COMPLETE" {
		headNodeIcon = "✅"
		headNodeStatus = "READY"
	}
	fmt.Printf("  Head Node:        %s %s\n", headNodeIcon, headNodeStatus)

	// Scheduler status (Slurm)
	schedulerIcon := "⏳"
	schedulerStatus := "PENDING"
	if status.ClusterStatus == "CREATE_IN_PROGRESS" {
		schedulerIcon = "🔄"
		schedulerStatus = "STARTING"
	} else if status.ClusterStatus == "CREATE_COMPLETE" {
		schedulerIcon = "✅"
		schedulerStatus = "ACTIVE"
	}
	fmt.Printf("  Slurm Controller: %s %s\n", schedulerIcon, schedulerStatus)

	// Compute fleet status
	computeIcon := "⏳"
	computeStatus := status.ComputeFleetStatus
	if computeStatus == "" {
		computeStatus = "PENDING"
	} else if computeStatus == "RUNNING" || computeStatus == "ENABLED" {
		computeIcon = "✅"
	} else if computeStatus == "STARTING" {
		computeIcon = "🔄"
	}
	fmt.Printf("  Compute Fleet:    %s %s\n", computeIcon, computeStatus)

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

// MonitorClusterConfiguration monitors cluster initialization from 70-100%
func (pm *ProgressMonitor) MonitorClusterConfiguration(ctx context.Context) error {
	fmt.Printf("\n🎯 Cluster Configuration:\n")
	fmt.Printf("⏳ Monitoring cluster initialization...\n")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Initial check
	status, err := pm.getClusterStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial cluster status: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err = pm.getClusterStatus(ctx)
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

// FailureDetails contains information about a failed cluster creation
type FailureDetails struct {
	FailedResource    *ResourceStatus
	StackStatus       types.StackStatus
	StackStatusReason string
	FailureTime       time.Time
}

// getFailedResources returns all resources that failed during creation
func (pm *ProgressMonitor) getFailedResources(ctx context.Context) ([]*ResourceStatus, error) {
	events, err := pm.getStackEvents(ctx)
	if err != nil {
		return nil, err
	}

	var failedResources []*ResourceStatus
	for _, event := range events {
		if event.ResourceStatus == types.ResourceStatusCreateFailed {
			failedResources = append(failedResources, &ResourceStatus{
				LogicalID:  aws.ToString(event.LogicalResourceId),
				Type:       aws.ToString(event.ResourceType),
				Status:     event.ResourceStatus,
				StatusText: aws.ToString(event.ResourceStatusReason),
				Timestamp:  *event.Timestamp,
			})
		}
	}

	return failedResources, nil
}

// getConsoleURL returns the AWS Console URL for the CloudFormation stack
func (pm *ProgressMonitor) getConsoleURL() string {
	return fmt.Sprintf(
		"https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks/stackinfo?stackId=%s",
		pm.region,
		pm.stackName,
	)
}

// getCloudWatchLogsURL returns the AWS Console URL for CloudWatch logs
func (pm *ProgressMonitor) getCloudWatchLogsURL() string {
	return fmt.Sprintf(
		"https://console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group//aws/parallelcluster/%s",
		pm.region,
		pm.clusterName,
	)
}

// displayFailureDetails displays detailed information about a failed cluster creation
func (pm *ProgressMonitor) displayFailureDetails(ctx context.Context) error {
	fmt.Printf("\n❌ Cluster creation failed!\n\n")

	// Get failed resources
	failedResources, err := pm.getFailedResources(ctx)
	if err != nil {
		fmt.Printf("Unable to retrieve failure details: %v\n", err)
		return nil
	}

	if len(failedResources) == 0 {
		fmt.Printf("No specific resource failures found.\n")
		return nil
	}

	// Display the most recent failure
	latestFailure := failedResources[0]
	fmt.Printf("Failed Resource: %s (%s)\n",
		latestFailure.LogicalID,
		latestFailure.Type)

	if latestFailure.StatusText != "" {
		fmt.Printf("Reason: %s\n", latestFailure.StatusText)
	}

	fmt.Printf("Status: %s\n", latestFailure.Status)
	fmt.Printf("Timestamp: %s\n\n", latestFailure.Timestamp.Format("2006-01-02 15:04:05"))

	// Show AWS Console links
	fmt.Printf("View in AWS Console:\n")
	fmt.Printf("  CloudFormation: %s\n", pm.getConsoleURL())
	fmt.Printf("  CloudWatch Logs: %s\n\n", pm.getCloudWatchLogsURL())

	// Show troubleshooting hints
	fmt.Printf("Troubleshooting:\n")
	hints := getTroubleshootingHints(latestFailure.Type, latestFailure.StatusText)
	for _, hint := range hints {
		fmt.Printf("  • %s\n", hint)
	}

	return nil
}

// getTroubleshootingHints returns troubleshooting suggestions based on resource type and error
func getTroubleshootingHints(resourceType, statusReason string) []string {
	hints := []string{}

	switch resourceType {
	case "AWS::EC2::Instance":
		hints = append(hints, "Check EC2 instance launch logs in AWS Console")
		hints = append(hints, "Verify AMI exists in the target region")
		hints = append(hints, "Check EC2 service quotas (vCPU limits)")
		if strings.Contains(statusReason, "subnet") {
			hints = append(hints, "Verify subnet ID is correct and exists in your account")
		}
	case "AWS::EC2::Subnet":
		hints = append(hints, "Verify VPC exists and is accessible")
		hints = append(hints, "Check subnet CIDR doesn't overlap with existing subnets")
	case "AWS::IAM::Role", "AWS::IAM::Policy", "AWS::IAM::InstanceProfile":
		hints = append(hints, "Verify IAM permissions to create roles and policies")
		hints = append(hints, "Check for IAM policy conflicts or naming collisions")
	case "AWS::CloudFormation::WaitCondition":
		hints = append(hints, "EC2 instance may have failed to signal completion")
		hints = append(hints, "Check CloudWatch Logs for bootstrap script errors")
		hints = append(hints, "Verify instance can reach CloudFormation service endpoints")
	default:
		hints = append(hints, "Check CloudFormation stack events for more details")
		hints = append(hints, "Review CloudWatch Logs for cluster initialization errors")
	}

	hints = append(hints, fmt.Sprintf("See: https://docs.aws.amazon.com/parallelcluster/latest/ug/troubleshooting.html"))

	return hints
}

// monitorRollback monitors the rollback progress when stack creation fails
func (pm *ProgressMonitor) monitorRollback(ctx context.Context) error {
	fmt.Printf("\n🔄 Stack creation failed, rolling back...\n\n")

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
					eventKey := fmt.Sprintf("%s-%s", aws.ToString(event.LogicalResourceId), event.ResourceStatus)
					if !seenEvents[eventKey] {
						seenEvents[eventKey] = true
						resources[aws.ToString(event.LogicalResourceId)] = &ResourceStatus{
							LogicalID:  aws.ToString(event.LogicalResourceId),
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
				// Stack might be deleted
				if strings.Contains(err.Error(), "does not exist") {
					fmt.Printf("\n✅ Rollback complete (stack deleted)\n")
					return fmt.Errorf("cluster creation failed and rolled back")
				}
				return err
			}

			if stackStatus == types.StackStatusRollbackComplete ||
				stackStatus == types.StackStatusDeleteComplete {
				fmt.Printf("\n✅ Rollback complete\n")
				return fmt.Errorf("cluster creation failed and rolled back")
			}
		}
	}
}

// displayRollbackProgress displays rollback progress
func (pm *ProgressMonitor) displayRollbackProgress(resources map[string]*ResourceStatus) {
	fmt.Printf("\n🔄 Rollback Progress:\n")

	var deleted, inProgress, pending int
	var displayedCount int
	maxDisplay := 10

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

		// Only display up to maxDisplay resources
		if displayedCount < maxDisplay {
			fmt.Printf("  %s %-35s %s\n",
				icon,
				pm.getReadableResourceName(res.LogicalID, res.Type),
				res.Status)
			displayedCount++
		}
	}

	total := len(resources)
	elapsed := time.Since(pm.startTime)

	if total > maxDisplay {
		fmt.Printf("  ... and %d more resources\n", total-maxDisplay)
	}

	fmt.Printf("\nRollback: %d/%d resources deleted", deleted, total)
	if inProgress > 0 {
		fmt.Printf(" (%d in progress)", inProgress)
	}
	fmt.Printf(" | Elapsed: %s\n", formatDuration(elapsed))
}
