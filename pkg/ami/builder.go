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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	ec2Client *ec2.Client
	region    string
}

// NewBuilder creates a new AMI builder.
func NewBuilder(ctx context.Context, region string) (*Builder, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Builder{
		ec2Client: ec2.NewFromConfig(cfg),
		region:    region,
	}, nil
}

// BuildAMI creates a custom AMI from a template.
func (b *Builder) BuildAMI(ctx context.Context, tmpl *template.Template, opts *BuildOptions) (*AMIMetadata, error) {
	fmt.Printf("🚀 Starting AMI build process...\n\n")

	// Step 1: Launch temporary instance
	fmt.Printf("1️⃣  Launching temporary build instance...\n")
	instanceID, err := b.launchBuildInstance(ctx, tmpl, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to launch build instance: %w", err)
	}
	fmt.Printf("   ✅ Instance launched: %s\n\n", instanceID)

	// Ensure cleanup
	defer func() {
		fmt.Printf("🧹 Cleaning up temporary instance...\n")
		b.terminateInstance(ctx, instanceID)
	}()

	// Step 2: Wait for instance to be ready
	fmt.Printf("2️⃣  Waiting for instance to be ready...\n")
	if err := b.waitForInstanceReady(ctx, instanceID); err != nil {
		return nil, fmt.Errorf("instance failed to become ready: %w", err)
	}
	fmt.Printf("   ✅ Instance is ready\n\n")

	// Step 3: Wait for software installation to complete
	fmt.Printf("3️⃣  Installing software (this may take 30-90 minutes)...\n")
	fmt.Printf("   📦 Installing %d Spack packages\n", len(tmpl.Software.SpackPackages))
	if err := b.waitForSoftwareInstallation(ctx, instanceID, opts); err != nil {
		return nil, fmt.Errorf("software installation failed: %w", err)
	}
	fmt.Printf("   ✅ Software installation complete\n\n")

	// Step 4: Stop the instance
	fmt.Printf("4️⃣  Stopping instance for AMI creation...\n")
	if err := b.stopInstance(ctx, instanceID); err != nil {
		return nil, fmt.Errorf("failed to stop instance: %w", err)
	}
	fmt.Printf("   ✅ Instance stopped\n\n")

	// Step 5: Create AMI
	fmt.Printf("5️⃣  Creating AMI...\n")
	amiID, err := b.createAMI(ctx, instanceID, tmpl, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create AMI: %w", err)
	}
	fmt.Printf("   ✅ AMI created: %s\n\n")

	// Step 6: Wait for AMI to be available
	fmt.Printf("6️⃣  Waiting for AMI to be available...\n")
	if err := b.waitForAMIAvailable(ctx, amiID); err != nil {
		return nil, fmt.Errorf("AMI failed to become available: %w", err)
	}
	fmt.Printf("   ✅ AMI is available\n\n")

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
}

// DefaultBuildOptions returns default build options.
func DefaultBuildOptions() *BuildOptions {
	return &BuildOptions{
		InstanceType: "t3.xlarge",
		WaitTimeout:  90 * time.Minute,
		Tags: map[string]string{
			"ManagedBy": "pctl",
		},
	}
}

func (b *Builder) launchBuildInstance(ctx context.Context, tmpl *template.Template, opts *BuildOptions) (string, error) {
	// Get base AMI if not specified
	baseAMI := opts.BaseAMI
	if baseAMI == "" {
		var err error
		baseAMI, err = b.getLatestParallelClusterAMI(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get base AMI: %w", err)
		}
	}

	// Generate user data script for software installation
	manager := software.NewManager()
	userData := manager.GenerateBootstrapScript(tmpl, false, false) // Software only, no users/S3

	// Base64 encode user data
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Launch instance
	runResult, err := b.ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
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
	})

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

func (b *Builder) waitForSoftwareInstallation(ctx context.Context, instanceID string, opts *BuildOptions) error {
	// TODO: Implement actual monitoring of software installation
	// For now, use a simple timeout based on expected installation time
	// In production, this should check CloudWatch logs or SSH to check status

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(opts.WaitTimeout)
	elapsed := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("software installation timed out after %v", opts.WaitTimeout)
		case <-ticker.C:
			elapsed += 30
			fmt.Printf("   ⏳ Installation in progress... (%d minutes elapsed)\n", elapsed/60)
		}
	}
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
	return waiter.Wait(ctx, &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}, 15*time.Minute)
}

func (b *Builder) terminateInstance(ctx context.Context, instanceID string) error {
	_, err := b.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

func (b *Builder) getLatestParallelClusterAMI(ctx context.Context) (string, error) {
	// Query for AWS ParallelCluster AMIs
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
