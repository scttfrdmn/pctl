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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/pctl/internal/config"
	pcconfig "github.com/scttfrdmn/pctl/pkg/config"
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
	// Check if cluster already exists
	if p.stateManager.Exists(tmpl.Cluster.Name) {
		return fmt.Errorf("cluster %s already exists", tmpl.Cluster.Name)
	}

	// Validate template
	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Generate ParallelCluster config
	p.configGen.KeyName = opts.KeyName
	p.configGen.SubnetID = opts.SubnetID
	p.configGen.CustomAMI = opts.CustomAMI

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
		Name:         tmpl.Cluster.Name,
		Region:       tmpl.Cluster.Region,
		Status:       "CREATE_IN_PROGRESS",
		StackName:    fmt.Sprintf("pctl-%s", tmpl.Cluster.Name),
		TemplatePath: opts.TemplatePath,
		CreatedAt:    time.Now(),
		CustomAMI:    opts.CustomAMI,
	}

	if err := p.stateManager.Save(clusterState); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// Create cluster using pcluster CLI
	if err := p.runPClusterCreate(ctx, tmpl.Cluster.Name, configPath, tmpl.Cluster.Region); err != nil {
		clusterState.Status = "CREATE_FAILED"
		p.stateManager.Save(clusterState)
		return fmt.Errorf("failed to create cluster: %w", err)
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

func (p *Provisioner) runPClusterCreate(ctx context.Context, name, configPath, region string) error {
	cmd := exec.CommandContext(ctx, "pcluster", "create-cluster",
		"--cluster-name", name,
		"--cluster-configuration", configPath,
		"--region", region,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (p *Provisioner) runPClusterDelete(ctx context.Context, name, region string) error {
	cmd := exec.CommandContext(ctx, "pcluster", "delete-cluster",
		"--cluster-name", name,
		"--region", region,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (p *Provisioner) runPClusterDescribe(ctx context.Context, name, region string) (*ClusterStatus, error) {
	cmd := exec.CommandContext(ctx, "pcluster", "describe-cluster",
		"--cluster-name", name,
		"--region", region,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pcluster describe-cluster failed: %w: %s", err, output)
	}

	// TODO: Parse JSON output from pcluster describe-cluster
	// For now, return basic status
	return &ClusterStatus{
		Name:   name,
		Status: "RUNNING", // Placeholder
		Region: region,
	}, nil
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
