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

// Package provisioner provides cluster provisioning functionality.
package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/pctl/internal/config"
	"github.com/scttfrdmn/pctl/pkg/bootstrap"
	pcconfig "github.com/scttfrdmn/pctl/pkg/config"
	"github.com/scttfrdmn/pctl/pkg/network"
	"github.com/scttfrdmn/pctl/pkg/state"
	"github.com/scttfrdmn/pctl/pkg/template"
)

// Provisioner handles cluster provisioning using ParallelCluster.
type Provisioner struct {
	stateManager *state.Manager
	configGen    *pcconfig.Generator
}

// NewProvisioner creates a new provisioner.
func NewProvisioner() (*Provisioner, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &Provisioner{
		stateManager: stateMgr,
		configGen:    pcconfig.NewGenerator(),
	}, nil
}

// CreateCluster creates a new cluster from a template.
func (p *Provisioner) CreateCluster(ctx context.Context, tmpl *template.Template, opts *CreateOptions) error {
	// Check if cluster already exists in AWS (not just local state)
	awsStatus, err := p.runPClusterDescribe(ctx, tmpl.Cluster.Name, tmpl.Cluster.Region)
	if err == nil {
		// Cluster exists in AWS
		if awsStatus.Status == "CREATE_FAILED" || awsStatus.Status == "DELETE_FAILED" {
			return fmt.Errorf("cluster %s exists in AWS with status %s\n\nTo retry, first clean up the failed stack:\n  pctl delete %s\n\nOr use AWS CLI directly:\n  pcluster delete-cluster --cluster-name %s --region %s",
				tmpl.Cluster.Name, awsStatus.Status, tmpl.Cluster.Name, tmpl.Cluster.Name, tmpl.Cluster.Region)
		}
		return fmt.Errorf("cluster %s already exists in AWS with status: %s", tmpl.Cluster.Name, awsStatus.Status)
	}

	// Check if cluster exists in local state
	if p.stateManager.Exists(tmpl.Cluster.Name) {
		return fmt.Errorf("cluster %s exists in local state but not in AWS\n\nThe cluster may have been deleted outside of pctl. To clean up:\n  rm ~/.pctl/state/%s.json\n\nOr use:\n  pctl delete %s --local-only",
			tmpl.Cluster.Name, tmpl.Cluster.Name, tmpl.Cluster.Name)
	}

	// Validate template
	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Create network resources if not provided
	var networkResources *network.NetworkResources
	subnetID := opts.SubnetID
	if subnetID == "" {
		fmt.Printf("🌐 Creating VPC and networking resources...\n")
		netMgr, err := network.NewManager(ctx, tmpl.Cluster.Region)
		if err != nil {
			return fmt.Errorf("failed to create network manager: %w", err)
		}

		networkResources, err = netMgr.CreateNetwork(ctx, tmpl.Cluster.Name)
		if err != nil {
			return fmt.Errorf("failed to create network: %w", err)
		}
		subnetID = networkResources.PublicSubnetID
		fmt.Printf("✅ VPC created: %s\n", networkResources.VpcID)
		fmt.Printf("✅ Public subnet: %s\n", networkResources.PublicSubnetID)
		fmt.Printf("✅ Private subnet: %s\n", networkResources.PrivateSubnetID)
	}

	// Generate and upload bootstrap script if needed
	// Skip if CustomAMI is provided (software pre-installed in AMI)
	var bootstrapS3URI string
	if opts.CustomAMI == "" && (len(tmpl.Software.SpackPackages) > 0 || len(tmpl.Users) > 0 || len(tmpl.Data.S3Mounts) > 0) {
		fmt.Printf("📝 Generating bootstrap script...\n")

		// Generate bootstrap script content
		scriptContent := p.configGen.GenerateBootstrapScript(tmpl)

		// Upload to S3
		fmt.Printf("☁️  Uploading bootstrap script to S3...\n")
		s3Mgr, err := bootstrap.NewS3Manager(ctx, tmpl.Cluster.Region)
		if err != nil {
			return fmt.Errorf("failed to create S3 manager: %w", err)
		}

		bootstrapS3URI, err = s3Mgr.UploadBootstrapScript(ctx, tmpl.Cluster.Name, scriptContent)
		if err != nil {
			return fmt.Errorf("failed to upload bootstrap script: %w", err)
		}
		fmt.Printf("✅ Bootstrap script uploaded: %s\n", bootstrapS3URI)
	} else if opts.CustomAMI != "" {
		fmt.Printf("📀 Using custom AMI with pre-installed software (skipping bootstrap)\n")
	}

	// Generate ParallelCluster config
	p.configGen.KeyName = opts.KeyName
	p.configGen.SubnetID = subnetID
	p.configGen.CustomAMI = opts.CustomAMI
	p.configGen.BootstrapScriptS3URI = bootstrapS3URI

	pcConfig, err := p.configGen.Generate(tmpl)
	if err != nil {
		return fmt.Errorf("failed to generate ParallelCluster config: %w", err)
	}

	// Write config to temporary file
	configPath, err := p.writeConfigFile(tmpl.Cluster.Name, pcConfig)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	defer os.Remove(configPath)

	// Create initial state
	clusterState := &state.ClusterState{
		Name:                 tmpl.Cluster.Name,
		Region:               tmpl.Cluster.Region,
		Status:               "CREATE_IN_PROGRESS",
		StackName:            fmt.Sprintf("pctl-%s", tmpl.Cluster.Name),
		TemplatePath:         opts.TemplatePath,
		CreatedAt:            time.Now(),
		CustomAMI:            opts.CustomAMI,
		KeyName:              opts.KeyName,
		BootstrapScriptS3URI: bootstrapS3URI,
	}

	// Store network resources if we created them
	if networkResources != nil {
		clusterState.VpcID = networkResources.VpcID
		clusterState.PublicSubnetID = networkResources.PublicSubnetID
		clusterState.PrivateSubnetID = networkResources.PrivateSubnetID
		clusterState.SecurityGroupID = networkResources.SecurityGroupID
		clusterState.InternetGatewayID = networkResources.InternetGatewayID
		clusterState.RouteTableID = networkResources.RouteTableID
		clusterState.NetworkManagedByPctl = true
	}

	if err := p.stateManager.Save(clusterState); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// Create cluster using pcluster CLI (initiates async creation)
	fmt.Printf("🔧 Initiating cluster creation...\n")
	if err := p.runPClusterCreateAsync(ctx, tmpl.Cluster.Name, configPath, tmpl.Cluster.Region); err != nil {
		clusterState.Status = "CREATE_FAILED"
		p.stateManager.Save(clusterState)

		// Clean up network resources if we created them
		if networkResources != nil {
			fmt.Printf("\n🧹 Cleaning up network resources due to cluster creation failure...\n")
			netMgr, _ := network.NewManager(ctx, tmpl.Cluster.Region)
			if netMgr != nil {
				netMgr.DeleteNetwork(ctx, networkResources)
			}
		}

		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Monitor cluster creation progress
	stackName := clusterState.StackName
	monitor, err := NewProgressMonitor(ctx, stackName, tmpl.Cluster.Region, tmpl.Cluster.Name)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to create progress monitor: %v\n", err)
		fmt.Printf("⏳ Cluster is being created in the background. Check status with: pctl status %s\n", tmpl.Cluster.Name)
	} else {
		// Monitor with timeout (30 minutes max)
		monitorCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()

		if err := monitor.MonitorCreation(monitorCtx); err != nil {
			if monitorCtx.Err() == context.DeadlineExceeded {
				fmt.Printf("\n⚠️  Monitoring timeout reached (30 minutes). Cluster is still being created.\n")
				fmt.Printf("Check status with: pctl status %s\n", tmpl.Cluster.Name)
			} else {
				clusterState.Status = "CREATE_FAILED"
				p.stateManager.Save(clusterState)

				// Clean up network resources if we created them
				if networkResources != nil {
					fmt.Printf("\n🧹 Cleaning up network resources due to cluster creation failure...\n")
					netMgr, _ := network.NewManager(ctx, tmpl.Cluster.Region)
					if netMgr != nil {
						netMgr.DeleteNetwork(ctx, networkResources)
					}
				}

				return fmt.Errorf("cluster creation failed: %w", err)
			}
		}
	}

	// Update state
	clusterState.Status = "CREATE_COMPLETE"
	if err := p.stateManager.Save(clusterState); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	return nil
}

// DeleteCluster deletes a cluster.
func (p *Provisioner) DeleteCluster(ctx context.Context, name string) error {
	// Load cluster state
	clusterState, err := p.stateManager.Load(name)
	if err != nil {
		return fmt.Errorf("failed to load cluster state: %w", err)
	}

	// Delete cluster using pcluster CLI
	if err := p.runPClusterDelete(ctx, name, clusterState.Region); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Delete network resources if managed by pctl
	if clusterState.NetworkManagedByPctl {
		fmt.Printf("🧹 Deleting VPC and networking resources...\n")
		netMgr, err := network.NewManager(ctx, clusterState.Region)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to create network manager: %v\n", err)
		} else {
			networkResources := &network.NetworkResources{
				VpcID:             clusterState.VpcID,
				PublicSubnetID:    clusterState.PublicSubnetID,
				PrivateSubnetID:   clusterState.PrivateSubnetID,
				SecurityGroupID:   clusterState.SecurityGroupID,
				InternetGatewayID: clusterState.InternetGatewayID,
				RouteTableID:      clusterState.RouteTableID,
				Region:            clusterState.Region,
				ClusterName:       name,
				ManagedByPctl:     true,
			}
			if err := netMgr.DeleteNetwork(ctx, networkResources); err != nil {
				fmt.Printf("⚠️  Warning: failed to delete network resources: %v\n", err)
			} else {
				fmt.Printf("✅ Network resources deleted\n")
			}
		}
	}

	// Remove state
	if err := p.stateManager.Delete(name); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// GetClusterStatus gets the status of a cluster.
func (p *Provisioner) GetClusterStatus(ctx context.Context, name string) (*ClusterStatus, error) {
	// Load cluster state
	clusterState, err := p.stateManager.Load(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster state: %w", err)
	}

	// Get status from ParallelCluster
	status, err := p.runPClusterDescribe(ctx, name, clusterState.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	return status, nil
}

// ListClusters lists all managed clusters.
func (p *Provisioner) ListClusters() ([]*state.ClusterState, error) {
	return p.stateManager.List()
}

// DeleteLocalState deletes only the local state for a cluster without touching AWS resources.
func (p *Provisioner) DeleteLocalState(name string) error {
	return p.stateManager.Delete(name)
}

// GetStateManager returns the state manager for direct state access.
func (p *Provisioner) GetStateManager() (*state.Manager, error) {
	return p.stateManager, nil
}

func (p *Provisioner) writeConfigFile(name, content string) (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(configDir, fmt.Sprintf("%s-config.yaml", name))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}

	return path, nil
}

func (p *Provisioner) getPClusterBinary() (string, error) {
	// Use only the private venv pcluster installation
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	venvPCluster := filepath.Join(homeDir, ".pctl", "venv", "bin", "pcluster")
	if _, err := os.Stat(venvPCluster); err != nil {
		return "", fmt.Errorf("pcluster not found in private venv (%s)\n\nThe pctl installation may be corrupted. Please reinstall pctl.", venvPCluster)
	}

	return venvPCluster, nil
}

func (p *Provisioner) runPClusterCreate(ctx context.Context, name, configPath, region string) error {
	pclusterBin, err := p.getPClusterBinary()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, pclusterBin, "create-cluster",
		"--cluster-name", name,
		"--cluster-configuration", configPath,
		"--region", region,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runPClusterCreateAsync initiates cluster creation without blocking on output
// The pcluster create-cluster command is already async, but we suppress stdout/stderr
// and let the progress monitor handle the display
func (p *Provisioner) runPClusterCreateAsync(ctx context.Context, name, configPath, region string) error {
	pclusterBin, err := p.getPClusterBinary()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, pclusterBin, "create-cluster",
		"--cluster-name", name,
		"--cluster-configuration", configPath,
		"--region", region,
	)

	// Capture output but don't display it (progress monitor will handle display)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pcluster create-cluster failed: %w: %s", err, output)
	}

	return nil
}

func (p *Provisioner) runPClusterDelete(ctx context.Context, name, region string) error {
	pclusterBin, err := p.getPClusterBinary()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, pclusterBin, "delete-cluster",
		"--cluster-name", name,
		"--region", region,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (p *Provisioner) runPClusterDescribe(ctx context.Context, name, region string) (*ClusterStatus, error) {
	pclusterBin, err := p.getPClusterBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, pclusterBin, "describe-cluster",
		"--cluster-name", name,
		"--region", region,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pcluster describe-cluster failed: %w: %s", err, output)
	}

	// Parse JSON output from pcluster describe-cluster
	var pcResponse pclusterDescribeResponse
	if err := json.Unmarshal(output, &pcResponse); err != nil {
		return nil, fmt.Errorf("failed to parse pcluster output: %w", err)
	}

	status := &ClusterStatus{
		Name:           name,
		Status:         pcResponse.ClusterStatus,
		Region:         region,
		SchedulerState: pcResponse.ComputeFleetStatus,
	}

	// Extract head node info if available
	if pcResponse.HeadNode != nil {
		status.HeadNodeIP = pcResponse.HeadNode.PublicIPAddress
	}

	return status, nil
}

// CreateOptions contains options for cluster creation.
type CreateOptions struct {
	TemplatePath string
	KeyName      string
	SubnetID     string
	CustomAMI    string
	DryRun       bool
}

// ClusterStatus represents the status of a cluster.
type ClusterStatus struct {
	Name           string
	Status         string
	Region         string
	HeadNodeIP     string
	ComputeNodes   int
	SchedulerState string
}

// pclusterDescribeResponse represents the JSON response from pcluster describe-cluster
type pclusterDescribeResponse struct {
	ClusterStatus             string            `json:"clusterStatus"`
	CloudFormationStackStatus string            `json:"cloudFormationStackStatus"`
	ComputeFleetStatus        string            `json:"computeFleetStatus"`
	HeadNode                  *pclusterHeadNode `json:"headNode"`
}

// pclusterHeadNode represents head node information from pcluster
type pclusterHeadNode struct {
	PublicIPAddress  string `json:"publicIpAddress"`
	PrivateIPAddress string `json:"privateIpAddress"`
	InstanceType     string `json:"instanceType"`
	State            string `json:"state"`
}
