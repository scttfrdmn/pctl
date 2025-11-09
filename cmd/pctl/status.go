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

	"github.com/scttfrdmn/pctl/pkg/provisioner"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status CLUSTER_NAME",
	Short: "Get cluster status",
	Long: `Get detailed status information for a cluster.

Shows:
- Cluster state (creating, running, stopped, failed)
- Head node status and IP address
- Compute node counts and status
- ParallelCluster version
- Software installation status
- Error messages (if any)`,
	Example: `  # Get cluster status
  pctl status my-cluster

  # Get status with verbose output
  pctl status my-cluster --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	if verbose {
		fmt.Printf("Checking status for cluster: %s\n\n", clusterName)
	}

	// Create provisioner
	prov, err := provisioner.NewProvisioner()
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}

	// Get cluster status
	ctx := context.Background()
	status, err := prov.GetClusterStatus(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster status: %w", err)
	}

	// Print status header
	statusEmoji := getStatusEmoji(status.Status)
	fmt.Printf("📊 Cluster Status: %s\n\n", clusterName)
	fmt.Printf("Status: %s %s\n", statusEmoji, status.Status)
	fmt.Printf("Region: %s\n", status.Region)

	// Print head node information if available
	if status.HeadNodeIP != "" {
		fmt.Printf("\nHead Node:\n")
		fmt.Printf("  Public IP:  %s\n", status.HeadNodeIP)
		fmt.Printf("  SSH:        ssh -i ~/.ssh/<key>.pem ec2-user@%s\n", status.HeadNodeIP)
	}

	// Print compute node information if available
	if status.ComputeNodes > 0 {
		fmt.Printf("\nCompute Nodes: %d\n", status.ComputeNodes)
	}

	// Print scheduler information if available
	if status.SchedulerState != "" {
		fmt.Printf("\nScheduler:\n")
		fmt.Printf("  Type:   SLURM\n")
		fmt.Printf("  State:  %s\n", status.SchedulerState)
	}

	// Print next steps based on status
	fmt.Printf("\nActions:\n")
	switch status.Status {
	case "CREATE_IN_PROGRESS":
		fmt.Printf("  ⏳ Cluster is being created. Check again in a few minutes.\n")
		fmt.Printf("  💡 Monitor progress: pctl status %s\n", clusterName)
	case "CREATE_COMPLETE":
		fmt.Printf("  ✅ Cluster is ready to use!\n")
		if status.HeadNodeIP != "" {
			fmt.Printf("  🔗 SSH to head node: ssh -i ~/.ssh/<key>.pem ec2-user@%s\n", status.HeadNodeIP)
		}
		fmt.Printf("  🗑️  Delete cluster: pctl delete %s\n", clusterName)
	case "CREATE_FAILED":
		fmt.Printf("  ❌ Cluster creation failed.\n")
		fmt.Printf("  🔍 Check CloudFormation console for detailed error messages.\n")
		fmt.Printf("  🗑️  Clean up: pctl delete %s\n", clusterName)
	case "DELETE_IN_PROGRESS":
		fmt.Printf("  🗑️  Cluster is being deleted.\n")
	default:
		fmt.Printf("  💡 Monitor: pctl status %s\n", clusterName)
		fmt.Printf("  🗑️  Delete:  pctl delete %s\n", clusterName)
	}

	if verbose {
		fmt.Printf("\nVerbose Details:\n")
		fmt.Printf("  Full Status: %+v\n", status)
	}

	return nil
}
