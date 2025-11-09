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

	// TODO: Implement cluster status checking
	fmt.Printf("📊 Cluster Status: %s\n\n", clusterName)
	fmt.Printf("⚠️  Status checking not yet implemented (v0.2.0)\n")
	fmt.Printf("This will be implemented in the AWS Integration milestone.\n\n")
	fmt.Printf("Will show:\n")
	fmt.Printf("  - Cluster state\n")
	fmt.Printf("  - Head node: IP, instance type, status\n")
	fmt.Printf("  - Compute nodes: count, types, status\n")
	fmt.Printf("  - SLURM scheduler status\n")
	fmt.Printf("  - Software installation progress\n")
	fmt.Printf("  - Recent events and errors\n")

	return nil
}
