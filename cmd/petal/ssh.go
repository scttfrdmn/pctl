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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/scttfrdmn/petal/pkg/provisioner"
	"github.com/spf13/cobra"
)

var (
	sshKeyPath string
	sshUser    string
)

var sshCmd = &cobra.Command{
	Use:     "ssh CLUSTER_NAME",
	Aliases: []string{"stem", "connect"},
	Short:   "SSH into cluster head node",
	Long: `Connect to the cluster head node via SSH.

Automatically uses the key specified during cluster creation and connects
to the head node IP address. You can override the key path and username
with flags if needed.`,
	Example: `  # SSH to cluster (uses key from cluster creation)
  pctl ssh my-cluster

  # SSH with custom key path
  pctl ssh my-cluster --key ~/.ssh/my-key.pem

  # SSH with custom username
  pctl ssh my-cluster --user ubuntu`,
	Args: cobra.ExactArgs(1),
	RunE: runSSH,
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.Flags().StringVarP(&sshKeyPath, "key", "i", "", "Path to SSH private key (overrides cluster default)")
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "ec2-user", "SSH username")
}

func runSSH(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	if verbose {
		fmt.Printf("Connecting to cluster: %s\n\n", clusterName)
	}

	// Create provisioner
	prov, err := provisioner.NewProvisioner()
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}

	// Get cluster status to retrieve head node IP
	ctx := context.Background()
	status, err := prov.GetClusterStatus(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster status: %w", err)
	}

	// Check if cluster is ready
	if status.Status != "CREATE_COMPLETE" {
		return fmt.Errorf("cluster is not ready for SSH (status: %s)\n\nRun 'pctl status %s' to check cluster state", status.Status, clusterName)
	}

	// Check if head node IP is available
	if status.HeadNodeIP == "" {
		return fmt.Errorf("head node IP address not available yet\n\nTry again in a few moments")
	}

	// Determine key path
	keyPath := sshKeyPath
	if keyPath == "" {
		// Try to load cluster state to get key name
		stateMgr, err := prov.GetStateManager()
		if err == nil {
			state, err := stateMgr.Load(clusterName)
			if err == nil && state.KeyName != "" {
				// Check common key locations
				homeDir, _ := os.UserHomeDir()
				possiblePaths := []string{
					filepath.Join(homeDir, ".ssh", state.KeyName+".pem"),
					filepath.Join(homeDir, ".ssh", state.KeyName),
					filepath.Join(homeDir, ".ssh", "id_rsa"),
				}

				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						keyPath = path
						break
					}
				}
			}
		}

		// If still no key found, provide helpful error
		if keyPath == "" {
			return fmt.Errorf("SSH key path not found\n\nPlease specify the key path with:\n  pctl ssh %s --key ~/.ssh/<key>.pem\n\nOr use the full SSH command:\n  ssh -i ~/.ssh/<key>.pem %s@%s",
				clusterName, sshUser, status.HeadNodeIP)
		}
	}

	// Verify key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s\n\nPlease provide the correct key path with --key flag", keyPath)
	}

	// Print connection info
	fmt.Printf("🔗 Connecting to %s...\n", clusterName)
	fmt.Printf("   Host: %s\n", status.HeadNodeIP)
	fmt.Printf("   User: %s\n", sshUser)
	fmt.Printf("   Key:  %s\n\n", keyPath)

	// Build SSH command
	sshCmd := exec.Command("ssh",
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", sshUser, status.HeadNodeIP),
	)

	// Connect stdin/stdout/stderr to allow interactive session
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	// Execute SSH command
	if err := sshCmd.Run(); err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	return nil
}
