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

	"github.com/spf13/cobra"
)

var (
	deleteForce bool
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
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	if !deleteForce {
		fmt.Printf("⚠️  WARNING: This will permanently delete cluster '%s' and all associated resources.\n", clusterName)
		fmt.Printf("Data in S3 buckets will NOT be deleted.\n\n")
		fmt.Printf("This operation cannot be undone.\n\n")
		fmt.Printf("To proceed, run with --force flag\n")
		return nil
	}

	if verbose {
		fmt.Printf("Deleting cluster: %s\n\n", clusterName)
	}

	// TODO: Implement cluster deletion
	fmt.Printf("🗑️  Deleting Cluster: %s\n\n", clusterName)
	fmt.Printf("⚠️  Cluster deletion not yet implemented (v0.2.0)\n")
	fmt.Printf("This will be implemented in the AWS Integration milestone.\n\n")
	fmt.Printf("Will delete:\n")
	fmt.Printf("  - All compute nodes\n")
	fmt.Printf("  - Head node\n")
	fmt.Printf("  - Associated networking (if created by pctl)\n")
	fmt.Printf("  - CloudFormation stacks\n")
	fmt.Printf("  - Local state files\n")

	return nil
}
