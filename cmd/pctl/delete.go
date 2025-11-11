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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/scttfrdmn/pctl/pkg/provisioner"
	"github.com/spf13/cobra"
)

var (
	deleteForce     bool
	deleteLocalOnly bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete CLUSTER_NAME",
	Short: "Delete a cluster",
	Long: `Delete an HPC cluster and all associated resources.

This is a destructive operation that will:
- Delete all compute nodes
- Delete the head node
- Delete associated networking resources (if created by pctl)
- Remove cluster state from pctl

Data in S3 buckets will NOT be deleted.`,
	Example: `  # Delete a cluster (with confirmation)
  pctl delete my-cluster

  # Force delete without confirmation
  pctl delete my-cluster --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "skip confirmation prompt")
	deleteCmd.Flags().BoolVar(&deleteLocalOnly, "local-only", false, "only delete local state (cluster already deleted from AWS)")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	// Create provisioner to check if cluster exists
	prov, err := provisioner.NewProvisioner()
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}

	// Check if cluster exists
	clusters, err := prov.ListClusters()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	found := false
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("cluster '%s' not found. Use 'pctl list' to see managed clusters", clusterName)
	}

	// Handle local-only deletion
	if deleteLocalOnly {
		fmt.Printf("🗑️  Deleting local state only for cluster: %s\n", clusterName)
		fmt.Printf("⚠️  WARNING: This will NOT delete AWS resources.\n\n")

		if !deleteForce {
			fmt.Printf("Type the cluster name to confirm: ")
			reader := bufio.NewReader(os.Stdin)
			confirmation, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			confirmation = strings.TrimSpace(confirmation)
			if confirmation != clusterName {
				fmt.Printf("\n❌ Deletion cancelled. Cluster name did not match.\n")
				return nil
			}
		}

		// Delete just the local state
		if err := prov.DeleteLocalState(clusterName); err != nil {
			return fmt.Errorf("failed to delete local state: %w", err)
		}

		fmt.Printf("✅ Local state deleted for cluster '%s'.\n", clusterName)
		fmt.Printf("⚠️  AWS resources (if any) were NOT affected.\n")
		return nil
	}

	// Prompt for confirmation if not forced
	if !deleteForce {
		fmt.Printf("⚠️  WARNING: This will permanently delete cluster '%s' and all associated resources.\n\n", clusterName)
		fmt.Printf("This will delete:\n")
		fmt.Printf("  - All compute nodes\n")
		fmt.Printf("  - Head node\n")
		fmt.Printf("  - CloudFormation stacks\n")
		fmt.Printf("  - Local state files\n\n")
		fmt.Printf("Note: Data in S3 buckets will NOT be deleted.\n\n")
		fmt.Printf("This operation cannot be undone.\n\n")
		fmt.Printf("Type the cluster name to confirm deletion: ")

		reader := bufio.NewReader(os.Stdin)
		confirmation, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		confirmation = strings.TrimSpace(confirmation)
		if confirmation != clusterName {
			fmt.Printf("\n❌ Deletion cancelled. Cluster name did not match.\n")
			return nil
		}
	}

	if verbose {
		fmt.Printf("\nDeleting cluster: %s\n", clusterName)
	}

	// Delete the cluster
	fmt.Printf("\n🗑️  Deleting cluster: %s\n\n", clusterName)
	fmt.Printf("Deleting compute nodes...\n")
	fmt.Printf("Deleting head node...\n")
	fmt.Printf("Deleting CloudFormation stack...\n")
	fmt.Printf("⏳ This may take 5-10 minutes...\n\n")

	ctx := context.Background()
	if err := prov.DeleteCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("✅ Cluster '%s' deleted successfully.\n", clusterName)
	fmt.Printf("\nNote: S3 bucket data was preserved.\n")

	return nil
}
