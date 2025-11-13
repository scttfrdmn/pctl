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

// Package ami provides AMI building and management for pre-baked cluster images.
package ami

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/schollz/progressbar/v3"
	"github.com/scttfrdmn/pctl/pkg/software"
	"github.com/scttfrdmn/pctl/pkg/template"
)

// AMIMetadata contains information about a built AMI.
type AMIMetadata struct {
	// AMIID is the AMI ID
	AMIID string
	// Name is the AMI name
	Name string
	// Description is the AMI description
	Description string
	// Region is the AWS region
	Region string
	// CreatedAt is when the AMI was created
	CreatedAt time.Time
	// TemplateName is the source template name
	TemplateName string
	// SpackPackages lists installed Spack packages
	SpackPackages []string
	// Tags are AMI tags
	Tags map[string]string
}

// Builder builds custom AMIs with pre-installed software.
type Builder struct {
	ec2Client    *ec2.Client
	region       string
	stateManager *StateManager
}

// NewBuilder creates a new AMI builder.
func NewBuilder(ctx context.Context, region string) (*Builder, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stateManager, err := NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &Builder{
		ec2Client:    ec2.NewFromConfig(cfg),
		region:       region,
		stateManager: stateManager,
	}, nil
}

// BuildAMI creates a custom AMI from a template.
func (b *Builder) BuildAMI(ctx context.Context, tmpl *template.Template, opts *BuildOptions) (*AMIMetadata, error) {
	// Create build state
	buildState := b.stateManager.NewBuildState(
		tmpl.Cluster.Name,
		opts.Name,
		b.region,
		len(tmpl.Software.SpackPackages),
	)

	if err := b.stateManager.SaveState(buildState); err != nil {
		return nil, fmt.Errorf("failed to save initial build state: %w", err)
	}

	fmt.Printf("🚀 Starting AMI build process...\n")
	fmt.Printf("   Build ID: %s\n\n", buildState.BuildID)

	// Ensure cleanup on failure
	defer func() {
		if buildState.Status != BuildStatusComplete {
			b.stateManager.MarkFailed(buildState.BuildID, "Build did not complete successfully")
		}
	}()

	// Step 1: Launch temporary instance
	fmt.Printf("1️⃣  Launching temporary build instance...\n")
	instanceID, err := b.launchBuildInstance(ctx, tmpl, opts)
	if err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("Failed to launch instance: %v", err))
		return nil, fmt.Errorf("failed to launch build instance: %w", err)
	}
	buildState.InstanceID = instanceID
	b.stateManager.SaveState(buildState)
	fmt.Printf("   ✅ Instance launched: %s\n\n", instanceID)

	// Ensure cleanup
	defer func() {
		fmt.Printf("🧹 Cleaning up temporary instance...\n")
		b.terminateInstance(ctx, instanceID)
	}()

	// Step 2: Wait for instance to be ready
	fmt.Printf("2️⃣  Waiting for instance to be ready...\n")
	if err := b.waitForInstanceReady(ctx, instanceID); err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("Instance failed to become ready: %v", err))
		return nil, fmt.Errorf("instance failed to become ready: %w", err)
	}
	fmt.Printf("   ✅ Instance is ready\n\n")

	// If detached mode, return here with the build ID
	if opts.Detach {
		buildState.Status = BuildStatusInstalling
		b.stateManager.SaveState(buildState)

		fmt.Printf("🚀 Build started in detached mode\n\n")
		fmt.Printf("Build ID:     %s\n", buildState.BuildID)
		fmt.Printf("Instance ID:  %s\n", instanceID)
		fmt.Printf("Status:       %s\n\n", buildState.Status)
		fmt.Printf("The build will continue in AWS. Check progress with:\n")
		fmt.Printf("  pctl ami status %s\n\n", buildState.BuildID)
		fmt.Printf("Or watch progress continuously:\n")
		fmt.Printf("  pctl ami status %s --watch\n\n", buildState.BuildID)

		// Return partial metadata (AMI not created yet)
		return &AMIMetadata{
			Name:          opts.Name,
			Description:   opts.Description,
			Region:        b.region,
			CreatedAt:     time.Now(),
			TemplateName:  tmpl.Cluster.Name,
			SpackPackages: tmpl.Software.SpackPackages,
			Tags:          opts.Tags,
		}, nil
	}

	// Step 3: Wait for software installation to complete
	buildState.Status = BuildStatusInstalling
	b.stateManager.SaveState(buildState)
	fmt.Printf("3️⃣  Installing software (this may take 30-90 minutes)...\n")
	fmt.Printf("   📦 Installing %d Spack packages\n", len(tmpl.Software.SpackPackages))
	if err := b.waitForSoftwareInstallation(ctx, instanceID, buildState.BuildID, opts); err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("Software installation failed: %v", err))
		return nil, fmt.Errorf("software installation failed: %w", err)
	}
	fmt.Printf("   ✅ Software installation complete\n\n")

	// Step 4: Stop the instance
	fmt.Printf("4️⃣  Stopping instance for AMI creation...\n")
	if err := b.stopInstance(ctx, instanceID); err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("Failed to stop instance: %v", err))
		return nil, fmt.Errorf("failed to stop instance: %w", err)
	}
	fmt.Printf("   ✅ Instance stopped\n\n")

	// Step 5: Create AMI
	buildState.Status = BuildStatusCreating
	b.stateManager.SaveState(buildState)
	fmt.Printf("5️⃣  Creating AMI...\n")
	amiID, err := b.createAMI(ctx, instanceID, tmpl, opts)
	if err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("Failed to create AMI: %v", err))
		return nil, fmt.Errorf("failed to create AMI: %w", err)
	}
	fmt.Printf("   ✅ AMI created: %s\n\n", amiID)

	// Step 6: Wait for AMI to be available
	fmt.Printf("6️⃣  Waiting for AMI to be available...\n")
	if err := b.waitForAMIAvailable(ctx, amiID); err != nil {
		b.stateManager.MarkFailed(buildState.BuildID, fmt.Sprintf("AMI failed to become available: %v", err))
		return nil, fmt.Errorf("AMI failed to become available: %w", err)
	}
	fmt.Printf("   ✅ AMI is available\n\n")

	// Mark build as complete
	if err := b.stateManager.MarkComplete(buildState.BuildID, amiID); err != nil {
		// Log error but don't fail the build
		fmt.Printf("⚠️  Warning: Failed to update build state: %v\n", err)
	}

	metadata := &AMIMetadata{
		AMIID:         amiID,
		Name:          opts.Name,
		Description:   opts.Description,
		Region:        b.region,
		CreatedAt:     time.Now(),
		TemplateName:  tmpl.Cluster.Name,
		SpackPackages: tmpl.Software.SpackPackages,
		Tags:          opts.Tags,
	}

	fmt.Printf("🎉 AMI build complete!\n")
	fmt.Printf("   Build ID: %s\n", buildState.BuildID)
	fmt.Printf("   AMI ID: %s\n", amiID)
	fmt.Printf("   Region: %s\n", b.region)
	fmt.Printf("\nYou can now use this AMI with:\n")
	fmt.Printf("  pctl create -t template.yaml --key-name <key> --custom-ami %s\n\n", amiID)

	return metadata, nil
}

// BuildOptions contains options for building an AMI.
type BuildOptions struct {
	// Name is the AMI name
	Name string
	// Description is the AMI description
	Description string
	// InstanceType for the build instance (default: t3.xlarge)
	InstanceType string
	// SubnetID for the build instance
	SubnetID string
	// KeyName for SSH access to the build instance (optional)
	KeyName string
	// BaseAMI is the base ParallelCluster AMI (auto-detected if not specified)
	BaseAMI string
	// Tags are additional tags for the AMI
	Tags map[string]string
	// WaitTimeout is the maximum time to wait for software installation
	WaitTimeout time.Duration
	// SkipCleanup disables automatic cleanup before AMI creation
	SkipCleanup bool
	// CustomCleanupScript runs in addition to default cleanup
	CustomCleanupScript string
	// Detach starts the build and returns immediately (build continues in AWS)
	Detach bool
}

// DefaultBuildOptions returns default build options.
func DefaultBuildOptions() *BuildOptions {
	return &BuildOptions{
		InstanceType: "c6a.4xlarge", // 16 vCPUs, compute-optimized for fast Spack builds
		WaitTimeout:  4 * time.Hour,  // 4 hours - generous timeout for Spack builds
		Tags: map[string]string{
			"ManagedBy": "pctl",
		},
	}
}

func (b *Builder) launchBuildInstance(ctx context.Context, tmpl *template.Template, opts *BuildOptions) (string, error) {
	// Determine architecture from the instance type
	// Use the template's head_node instance type, or fall back to opts.InstanceType
	instanceType := opts.InstanceType
	if tmpl.Compute.HeadNode != "" {
		instanceType = tmpl.Compute.HeadNode
	}
	architecture := getInstanceTypeArchitecture(instanceType)

	// Get base AMI if not specified
	baseAMI := opts.BaseAMI
	if baseAMI == "" {
		var err error
		baseAMI, err = b.getLatestParallelClusterAMI(ctx, architecture)
		if err != nil {
			return "", fmt.Errorf("failed to get base AMI for architecture %s: %w", architecture, err)
		}
		fmt.Printf("   Using base AMI %s (%s architecture)\n", baseAMI, architecture)
	}

	// Generate user data script for software installation
	manager := software.NewManager()
	userData := manager.GenerateBootstrapScript(tmpl, false, false) // Software only, no users/S3

	// Append cleanup script unless skipped
	if !opts.SkipCleanup {
		userData += "\n\n# AMI Cleanup Script\n"
		userData += "echo '========================================'\n"
		userData += "echo 'Running AMI cleanup for optimal size and security...'\n"
		userData += "echo '========================================'\n"
		userData += GenerateCleanupScript(opts.CustomCleanupScript)
	}

	// Base64 encode user data
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Launch instance
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(baseAMI),
		InstanceType: types.InstanceType(opts.InstanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		UserData:     aws.String(userDataEncoded),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("pctl-ami-builder")},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("Purpose"), Value: aws.String("AMI-Build")},
				},
			},
		},
		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(opts.SubnetID),
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
			},
		},
	}

	// Add SSH key if specified
	if opts.KeyName != "" {
		runInput.KeyName = aws.String(opts.KeyName)
	}

	runResult, err := b.ec2Client.RunInstances(ctx, runInput)

	if err != nil {
		return "", err
	}

	if len(runResult.Instances) == 0 {
		return "", fmt.Errorf("no instances launched")
	}

	return *runResult.Instances[0].InstanceId, nil
}

func (b *Builder) waitForInstanceReady(ctx context.Context, instanceID string) error {
	waiter := ec2.NewInstanceRunningWaiter(b.ec2Client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
}

func (b *Builder) waitForSoftwareInstallation(ctx context.Context, instanceID, buildID string, opts *BuildOptions) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(opts.WaitTimeout)
	startTime := time.Now()
	lastProgress := ""
	lastProgressInt := 0

	// Create progress bar
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("📦 Installing software"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("software installation timed out after %v", opts.WaitTimeout)
		case <-ticker.C:
			// Poll console output for progress markers
			progress, err := b.getConsoleProgress(ctx, instanceID)
			if err != nil {
				// If we can't get console output, just show elapsed time
				elapsed := time.Since(startTime)
				bar.Describe(fmt.Sprintf("📦 Installation in progress (%dm elapsed)", int(elapsed.Minutes())))
				continue
			}

			// If we got a new progress update, display it
			if progress != "" && progress != lastProgress {
				lastProgress = progress

				// Extract progress percentage and update state
				progressInt := extractProgressPercentage(progress)
				if progressInt > lastProgressInt {
					// Update progress bar
					delta := progressInt - lastProgressInt
					bar.Add(delta)

					// Calculate time estimate
					elapsed := time.Since(startTime)
					if progressInt > 0 {
						totalEstimate := time.Duration(float64(elapsed) / float64(progressInt) * 100)
						remaining := totalEstimate - elapsed

						// Update bar description with estimate
						if remaining > 0 {
							bar.Describe(fmt.Sprintf("📦 Installing (~%dm remaining)", int(remaining.Minutes())))
						}
					}

					b.stateManager.UpdateProgress(buildID, progressInt, progress)
					lastProgressInt = progressInt
				}

				// If we see cleanup complete (95%), we're almost done
				if strings.Contains(progress, "95%") || strings.Contains(progress, "cleanup complete") {
					bar.Add(100 - lastProgressInt) // Complete the bar
					// Give it another minute for final steps
					time.Sleep(1 * time.Minute)
					return nil
				}
			}
		}
	}
}

// extractProgressPercentage extracts the percentage from a progress message.
func extractProgressPercentage(message string) int {
	// Look for patterns like "(42%)" or "42%"
	if idx := strings.Index(message, "%"); idx > 0 {
		// Find the start of the number
		start := idx - 1
		for start >= 0 && message[start] >= '0' && message[start] <= '9' {
			start--
		}
		start++
		numStr := message[start:idx]
		var percent int
		fmt.Sscanf(numStr, "%d", &percent)
		return percent
	}
	return 0
}

// getConsoleProgress retrieves and parses progress markers from EC2 console output.
func (b *Builder) getConsoleProgress(ctx context.Context, instanceID string) (string, error) {
	output, err := b.ec2Client.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true), // Force retrieval of most recent console output
	})
	if err != nil {
		return "", err
	}

	if output.Output == nil {
		return "", nil
	}

	// Decode base64 console output
	decodedBytes, err := base64.StdEncoding.DecodeString(*output.Output)
	if err != nil {
		return "", err
	}

	consoleOutput := string(decodedBytes)

	// Find the last PCTL_PROGRESS marker
	lines := strings.Split(consoleOutput, "\n")
	var lastProgress string
	for _, line := range lines {
		if strings.Contains(line, "PCTL_PROGRESS:") {
			// Extract the progress message
			parts := strings.SplitN(line, "PCTL_PROGRESS:", 2)
			if len(parts) == 2 {
				lastProgress = strings.TrimSpace(parts[1])
			}
		}
	}

	return lastProgress, nil
}

func (b *Builder) stopInstance(ctx context.Context, instanceID string) error {
	_, err := b.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return err
	}

	waiter := ec2.NewInstanceStoppedWaiter(b.ec2Client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
}

func (b *Builder) createAMI(ctx context.Context, instanceID string, tmpl *template.Template, opts *BuildOptions) (string, error) {
	result, err := b.ec2Client.CreateImage(ctx, &ec2.CreateImageInput{
		InstanceId:  aws.String(instanceID),
		Name:        aws.String(opts.Name),
		Description: aws.String(opts.Description),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeImage,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(opts.Name)},
					{Key: aws.String("ManagedBy"), Value: aws.String("pctl")},
					{Key: aws.String("TemplateName"), Value: aws.String(tmpl.Cluster.Name)},
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return *result.ImageId, nil
}

func (b *Builder) waitForAMIAvailable(ctx context.Context, amiID string) error {
	waiter := ec2.NewImageAvailableWaiter(b.ec2Client)
	// Increase timeout to 60 minutes for large AMIs (EBS snapshot can take 30-45 min for 40-50GB volumes)
	return waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}, 60*time.Minute)
}

func (b *Builder) terminateInstance(ctx context.Context, instanceID string) error {
	_, err := b.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

func (b *Builder) getLatestParallelClusterAMI(ctx context.Context, architecture string) (string, error) {
	// Query for AWS ParallelCluster AMIs with matching architecture
	// This is a simplified version - in production, query AWS Systems Manager Parameter Store
	result, err := b.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"amazon"},
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"aws-parallelcluster-*-amzn2-*"},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{architecture},
			},
		},
	})

	if err != nil {
		return "", err
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no ParallelCluster AMIs found")
	}

	// Return the most recent AMI
	latest := result.Images[0]
	for _, img := range result.Images[1:] {
		if img.CreationDate != nil && latest.CreationDate != nil {
			if *img.CreationDate > *latest.CreationDate {
				latest = img
			}
		}
	}

	return *latest.ImageId, nil
}

// getInstanceTypeArchitecture determines the CPU architecture for an instance type.
// This allows AMI building to work correctly from any host (e.g., Mac ARM building x86 AMIs).
func getInstanceTypeArchitecture(instanceType string) string {
	// Extract instance family (e.g., "t3", "c5", "m6g")
	// ARM instance families end with 'g' or 'gd' (Graviton)
	// Examples: t4g, m6g, c6gd, r6gd

	// Common ARM (Graviton) prefixes
	armFamilies := map[string]bool{
		"t4g":   true,
		"m6g":   true,
		"m6gd":  true,
		"m7g":   true,
		"m7gd":  true,
		"c6g":   true,
		"c6gd":  true,
		"c6gn":  true,
		"c7g":   true,
		"c7gd":  true,
		"c7gn":  true,
		"r6g":   true,
		"r6gd":  true,
		"r7g":   true,
		"r7gd":  true,
		"x2gd":  true,
		"im4gn": true,
		"is4gen": true,
	}

	// Extract family from instance type (e.g., "t3.large" -> "t3")
	parts := strings.Split(instanceType, ".")
	if len(parts) < 2 {
		// Default to x86_64 if we can't parse
		return "x86_64"
	}

	family := parts[0]
	if armFamilies[family] {
		return "arm64"
	}

	// Default to x86_64 for all other instance types
	return "x86_64"
}
