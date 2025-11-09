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
	"fmt"

	"github.com/scttfrdmn/pctl/pkg/template"
	"github.com/spf13/cobra"
)

var (
	createTemplate string
	createName     string
	dryRun         bool
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

The cluster name can be specified with --name, or will use the name from the template.`,
	Example: `  # Create a cluster from template
  pctl create -t bioinformatics.yaml

  # Create with custom name
  pctl create -t my-cluster.yaml --name production-cluster

  # Dry run to see what would be created
  pctl create -t my-cluster.yaml --dry-run`,
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&createTemplate, "template", "t", "", "path to template file (required)")
	createCmd.Flags().StringVarP(&createName, "name", "n", "", "cluster name (overrides template)")
	createCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and show plan without creating")
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

	// TODO: Implement actual cluster creation
	fmt.Printf("\n⚠️  Cluster creation not yet implemented (v0.2.0)\n")
	fmt.Printf("This will be implemented in the AWS Integration milestone.\n\n")
	fmt.Printf("For now, this command validates your template and shows what would be created.\n")

	return nil
}
