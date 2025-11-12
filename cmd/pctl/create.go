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

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/pctl/pkg/ami"
	"github.com/scttfrdmn/pctl/pkg/provisioner"
	"github.com/scttfrdmn/pctl/pkg/template"
	"github.com/spf13/cobra"
)

var (
	createTemplate  string
	createName      string
	createRegion    string
	createKeyName   string
	createSubnetID  string
	createCustomAMI string
	createWait      bool
	rebuildAMI      bool
	dryRun          bool
	forceBootstrap  bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new HPC cluster",
	Long: `Create a new AWS ParallelCluster from a template.

This command:
1. Validates the template
2. Generates ParallelCluster configuration
3. Provisions the cluster infrastructure
4. Installs software packages (if specified)
5. Configures users and data mounts

The cluster name can be specified with --name, or will use the name from the template.

If --subnet-id is not provided, pctl will automatically create a VPC with public
and private subnets, internet gateway, route tables, and security groups.`,
	Example: `  # Create a cluster with automatic VPC/networking
  pctl create -t bioinformatics.yaml --key-name my-key

  # Create using existing VPC/subnet
  pctl create -t bioinformatics.yaml --key-name my-key --subnet-id subnet-abc123

  # Create with custom name
  pctl create -t my-cluster.yaml --name production-cluster --key-name my-key

  # Override region from template
  pctl create -t bioinformatics.yaml --key-name my-key --region us-west-2

  # Create with custom AMI
  pctl create -t my-cluster.yaml --key-name my-key --custom-ami ami-0123456789

  # Dry run to see what would be created (no AWS credentials needed)
  pctl create -t my-cluster.yaml --dry-run

  # Create and wait for completion
  pctl create -t my-cluster.yaml --key-name my-key --wait`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&createTemplate, "template", "t", "", "path to template file (required)")
	createCmd.Flags().StringVarP(&createName, "name", "n", "", "cluster name (overrides template)")
	createCmd.Flags().StringVarP(&createRegion, "region", "r", "", "AWS region (overrides template)")
	createCmd.Flags().StringVarP(&createKeyName, "key-name", "k", "", "EC2 key pair name for SSH access (required)")
	createCmd.Flags().StringVarP(&createSubnetID, "subnet-id", "s", "", "subnet ID (optional, auto-creates VPC if not provided)")
	createCmd.Flags().StringVar(&createCustomAMI, "custom-ami", "", "custom AMI ID to use")
	createCmd.Flags().BoolVar(&createWait, "wait", false, "wait for cluster creation to complete")
	createCmd.Flags().BoolVar(&rebuildAMI, "rebuild-ami", false, "force rebuild of AMI even if cached version exists")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and show plan without creating")
	createCmd.Flags().BoolVar(&forceBootstrap, "force-bootstrap", false, "bypass AMI requirement and use bootstrap scripts (not recommended for production)")
	createCmd.MarkFlagRequired("template")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Loading template: %s\n", createTemplate)
	}

	// Load and validate template
	tmpl, err := template.Load(createTemplate)
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Override cluster name if provided
	clusterName := tmpl.Cluster.Name
	if createName != "" {
		clusterName = createName
		if verbose {
			fmt.Printf("Using custom cluster name: %s\n", clusterName)
		}
	}

	if dryRun {
		fmt.Printf("\n🔍 Dry run mode - no resources will be created\n\n")
	}

	fmt.Printf("Cluster Configuration:\n")
	fmt.Printf("  Name: %s\n", clusterName)
	fmt.Printf("  Region: %s\n", tmpl.Cluster.Region)
	fmt.Printf("  Head Node: %s\n", tmpl.Compute.HeadNode)
	fmt.Printf("\nCompute Queues:\n")
	for _, queue := range tmpl.Compute.Queues {
		fmt.Printf("  - %s: %v (min: %d, max: %d)\n",
			queue.Name, queue.InstanceTypes, queue.MinCount, queue.MaxCount)
	}

	if len(tmpl.Software.SpackPackages) > 0 {
		fmt.Printf("\nSoftware Packages (%d):\n", len(tmpl.Software.SpackPackages))
		for _, pkg := range tmpl.Software.SpackPackages {
			fmt.Printf("  - %s\n", pkg)
		}
	}

	if len(tmpl.Users) > 0 {
		fmt.Printf("\nUsers (%d):\n", len(tmpl.Users))
		for _, user := range tmpl.Users {
			fmt.Printf("  - %s (UID: %d, GID: %d)\n", user.Name, user.UID, user.GID)
		}
	}

	if len(tmpl.Data.S3Mounts) > 0 {
		fmt.Printf("\nS3 Mounts (%d):\n", len(tmpl.Data.S3Mounts))
		for _, mount := range tmpl.Data.S3Mounts {
			fmt.Printf("  - %s → %s\n", mount.Bucket, mount.MountPoint)
		}
	}

	if dryRun {
		fmt.Printf("\n✅ Template validation passed - ready to create\n")
		fmt.Printf("\nTo create this cluster, run without --dry-run\n")
		return nil
	}

	// Validate required flags
	if createKeyName == "" {
		return fmt.Errorf("--key-name is required for SSH access to the cluster")
	}

	// subnet-id is now optional - will auto-create VPC if not provided
	if createSubnetID != "" {
		fmt.Printf("📍 Using existing subnet: %s\n", createSubnetID)
	} else {
		fmt.Printf("📍 Will auto-create VPC and networking\n")
	}

	// Override region in template if provided (needed for AMI lookup)
	region := tmpl.Cluster.Region
	if createRegion != "" {
		region = createRegion
	}

	// AMI lookup/building logic
	if createCustomAMI == "" && len(tmpl.Software.SpackPackages) > 0 {
		fmt.Printf("\n🔍 Checking for existing AMI with required software...\n")

		// Compute template fingerprint
		fingerprint := tmpl.ComputeFingerprint()
		if verbose {
			fmt.Printf("Template fingerprint: %s\n", fingerprint.String())
			fmt.Printf("Fingerprint hash: %s\n", fingerprint.Hash)
		}

		// Create AMI manager
		ctx := context.Background()
		amiManager, err := ami.NewManager(ctx, region)
		if err != nil {
			return fmt.Errorf("failed to create AMI manager: %w", err)
		}

		// Check for existing AMI (skip cache if rebuild flag is set)
		var amiID string
		if rebuildAMI {
			fmt.Printf("⚙️  Skipping cache lookup (--rebuild-ami flag set)\n")
		} else {
			amiID, err = amiManager.FindAMIByFingerprint(ctx, fingerprint)
			if err != nil {
				return fmt.Errorf("failed to lookup AMI: %w", err)
			}

			if amiID != "" {
				fmt.Printf("✅ Found existing AMI: %s\n", amiID)
				fmt.Printf("   Using cached AMI with pre-installed software\n")
				createCustomAMI = amiID
			}
		}

		// Build new AMI if not found or rebuild requested
		if amiID == "" {
			// Generate AMI name from fingerprint
			amiName := fmt.Sprintf("pctl-%s", fingerprint.String())

			fmt.Printf("\n❌ No AMI found for this software configuration\n\n")
			fmt.Printf("pctl requires a custom AMI with pre-installed software for fast cluster creation.\n\n")
			fmt.Printf("Why AMIs are required:\n")
			fmt.Printf("  • Without AMI: EVERY cluster takes 30-90 minutes (slow every time)\n")
			fmt.Printf("  • With AMI: Build once (30-90 min), then every cluster is 3-5 minutes\n")
			fmt.Printf("  • Production-ready: Software pre-installed, tested, and ready to use\n\n")

			fmt.Printf("Build an AMI for this template:\n")
			fmt.Printf("  pctl ami build -t %s --name %s --detach\n\n", createTemplate, amiName)

			fmt.Printf("The AMI will build in the background (~30-90 minutes). Monitor with:\n")
			fmt.Printf("  pctl ami status %s\n\n", amiName)

			if !forceBootstrap {
				fmt.Printf("To bypass this check (not recommended for production):\n")
				fmt.Printf("  pctl create -t %s --key-name %s --force-bootstrap\n\n", createTemplate, createKeyName)
				return fmt.Errorf("AMI required for software packages - build AMI first or use --force-bootstrap")
			}

			// forceBootstrap is set - allow continuing with bootstrap scripts
			fmt.Printf("⚠️  WARNING: Using --force-bootstrap\n")
			fmt.Printf("   This cluster will take 30-90 minutes to be ready.\n")
			fmt.Printf("   Not recommended for production use.\n\n")
		}
	} else if createCustomAMI != "" {
		fmt.Printf("📀 Using custom AMI: %s\n", createCustomAMI)
	}

	// Create provisioner
	fmt.Printf("\n🚀 Creating cluster: %s\n\n", clusterName)
	prov, err := provisioner.NewProvisioner()
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}

	// Prepare create options
	opts := &provisioner.CreateOptions{
		TemplatePath: createTemplate,
		KeyName:      createKeyName,
		SubnetID:     createSubnetID,
		CustomAMI:    createCustomAMI,
		DryRun:       false,
	}

	// Override cluster name in template if provided
	if createName != "" {
		tmpl.Cluster.Name = createName
	}

	// Override region in template if provided
	if createRegion != "" {
		tmpl.Cluster.Region = createRegion
	}

	// Create cluster
	ctx := context.Background()
	if createWait {
		// Set a reasonable timeout for cluster creation (30 minutes)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
	}

	fmt.Printf("📝 Generating ParallelCluster configuration...\n")
	fmt.Printf("🔧 Provisioning cluster infrastructure...\n")
	fmt.Printf("⏳ This may take 10-15 minutes...\n\n")

	if err := prov.CreateCluster(ctx, tmpl, opts); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	fmt.Printf("\n✅ Cluster creation initiated successfully!\n\n")
	fmt.Printf("Cluster: %s\n", clusterName)
	fmt.Printf("Region: %s\n", tmpl.Cluster.Region)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Check status: pctl status %s\n", clusterName)
	fmt.Printf("  2. List clusters: pctl list\n")
	fmt.Printf("  3. SSH access: ssh -i ~/.ssh/%s.pem ec2-user@<head-node-ip>\n\n", createKeyName)

	if len(tmpl.Software.SpackPackages) > 0 {
		fmt.Printf("📦 Software installation will complete in background.\n")
		fmt.Printf("   Check bootstrap script logs in CloudWatch or /var/log/cfn-init.log\n\n")
	}

	return nil
}
