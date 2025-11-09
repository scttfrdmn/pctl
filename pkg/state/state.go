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

// Package state provides cluster state management.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/pctl/internal/config"
)

// ClusterState represents the state of a managed cluster.
type ClusterState struct {
	// Name is the cluster name
	Name string `json:"name"`
	// Region is the AWS region
	Region string `json:"region"`
	// Status is the cluster status
	Status string `json:"status"`
	// StackName is the CloudFormation stack name
	StackName string `json:"stack_name"`
	// TemplatePath is the path to the template used
	TemplatePath string `json:"template_path"`
	// TemplateHash is a hash of the template content
	TemplateHash string `json:"template_hash"`
	// CreatedAt is when the cluster was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the state was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// HeadNodeIP is the head node public IP
	HeadNodeIP string `json:"head_node_ip,omitempty"`
	// HeadNodePrivateIP is the head node private IP
	HeadNodePrivateIP string `json:"head_node_private_ip,omitempty"`
	// PCVersion is the ParallelCluster version used
	PCVersion string `json:"pc_version,omitempty"`
	// CustomAMI is the custom AMI ID if used
	CustomAMI string `json:"custom_ami,omitempty"`
	// Network resources (if managed by pctl)
	VpcID                string `json:"vpc_id,omitempty"`
	PublicSubnetID       string `json:"public_subnet_id,omitempty"`
	PrivateSubnetID      string `json:"private_subnet_id,omitempty"`
	SecurityGroupID      string `json:"security_group_id,omitempty"`
	InternetGatewayID    string `json:"internet_gateway_id,omitempty"`
	RouteTableID         string `json:"route_table_id,omitempty"`
	NetworkManagedByPctl bool   `json:"network_managed_by_pctl,omitempty"`
}

// Manager manages cluster state.
type Manager struct {
	stateDir string
}

// NewManager creates a new state manager.
func NewManager() (*Manager, error) {
	stateDir, err := config.GetStateDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get state directory: %w", err)
	}

	if err := config.EnsureStateDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure state directory: %w", err)
	}

	return &Manager{
		stateDir: stateDir,
	}, nil
}

// Save saves cluster state.
func (m *Manager) Save(state *ClusterState) error {
	state.UpdatedAt = time.Now()

	path := m.statePath(state.Name)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Load loads cluster state.
func (m *Manager) Load(name string) (*ClusterState, error) {
	path := m.statePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cluster %s not found in state", name)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ClusterState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Delete deletes cluster state.
func (m *Manager) Delete(name string) error {
	path := m.statePath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete state file: %w", err)
	}
	return nil
}

// List lists all managed clusters.
func (m *Manager) List() ([]*ClusterState, error) {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ClusterState{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var clusters []*ClusterState
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		state, err := m.Load(name)
		if err != nil {
			// Skip invalid state files
			continue
		}
		clusters = append(clusters, state)
	}

	return clusters, nil
}

// Exists checks if a cluster exists in state.
func (m *Manager) Exists(name string) bool {
	path := m.statePath(name)
	_, err := os.Stat(path)
	return err == nil
}

func (m *Manager) statePath(name string) string {
	return filepath.Join(m.stateDir, name+".json")
}
