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
	"strings"
	"time"

	"github.com/scttfrdmn/petal/pkg/provisioner"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"garden"},
	Short:   "List all managed clusters",
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
	// Create provisioner
	prov, err := provisioner.NewProvisioner()
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}

	// List clusters
	clusters, err := prov.ListClusters()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Printf("📋 No managed clusters found.\n\n")
		fmt.Printf("Create a cluster with: pctl create -t template.yaml\n")
		return nil
	}

	// Print header
	fmt.Printf("📋 Managed Clusters (%d):\n\n", len(clusters))

	// Calculate column widths
	nameWidth := len("NAME")
	statusWidth := len("STATUS")
	regionWidth := len("REGION")

	for _, cluster := range clusters {
		if len(cluster.Name) > nameWidth {
			nameWidth = len(cluster.Name)
		}
		if len(cluster.Status) > statusWidth {
			statusWidth = len(cluster.Status)
		}
		if len(cluster.Region) > regionWidth {
			regionWidth = len(cluster.Region)
		}
	}

	// Print table header
	fmt.Printf("%-*s  %-*s  %-*s  %-15s  %s\n",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		regionWidth, "REGION",
		"CREATED", "HEAD NODE IP")
	fmt.Printf("%s  %s  %s  %s  %s\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", statusWidth),
		strings.Repeat("-", regionWidth),
		strings.Repeat("-", 15),
		strings.Repeat("-", 15))

	// Print cluster rows
	for _, cluster := range clusters {
		// Format creation time
		createdStr := formatTimeAgo(cluster.CreatedAt)

		// Format head node IP
		headNodeIP := cluster.HeadNodeIP
		if headNodeIP == "" {
			headNodeIP = "-"
		}

		// Add status emoji
		statusEmoji := getStatusEmoji(cluster.Status)

		fmt.Printf("%-*s  %-*s  %-*s  %-15s  %s\n",
			nameWidth, cluster.Name,
			statusWidth, statusEmoji+" "+cluster.Status,
			regionWidth, cluster.Region,
			createdStr, headNodeIP)
	}

	fmt.Printf("\nUse 'pctl status <cluster-name>' for detailed information.\n")

	return nil
}

// formatTimeAgo formats a time as a relative string (e.g., "2 hours ago")
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("2006-01-02")
	}
}

// getStatusEmoji returns an emoji for the cluster status
func getStatusEmoji(status string) string {
	status = strings.ToUpper(status)
	switch {
	case strings.Contains(status, "COMPLETE"):
		return "✅"
	case strings.Contains(status, "PROGRESS"), strings.Contains(status, "CREATING"):
		return "🔄"
	case strings.Contains(status, "FAILED"), strings.Contains(status, "ERROR"):
		return "❌"
	case strings.Contains(status, "DELETE"):
		return "🗑️"
	default:
		return "📦"
	}
}
