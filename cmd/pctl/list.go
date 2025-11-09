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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed clusters",
	Long: `List all HPC clusters managed by pctl.

Shows cluster name, status, region, and creation date for all clusters.`,
	Example: `  # List all clusters
  pctl list

  # List with verbose output
  pctl list --verbose`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// TODO: Implement cluster listing from state
	fmt.Printf("📋 Managed Clusters:\n\n")
	fmt.Printf("⚠️  Cluster listing not yet implemented (v0.2.0)\n")
	fmt.Printf("This will be implemented in the AWS Integration milestone.\n\n")
	fmt.Printf("Will show:\n")
	fmt.Printf("  - Cluster name\n")
	fmt.Printf("  - Status (creating, running, stopped, failed)\n")
	fmt.Printf("  - Region\n")
	fmt.Printf("  - Creation date\n")
	fmt.Printf("  - Head node IP\n")
	fmt.Printf("  - Number of compute nodes\n")

	return nil
}
