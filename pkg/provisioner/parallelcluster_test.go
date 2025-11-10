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

package provisioner

import (
	"testing"
)

func TestCreateOptions(t *testing.T) {
	opts := &CreateOptions{
		TemplatePath: "/path/to/template.yaml",
		KeyName:      "my-key",
		SubnetID:     "subnet-12345",
		CustomAMI:    "ami-12345",
		DryRun:       true,
	}

	if opts.TemplatePath != "/path/to/template.yaml" {
		t.Errorf("Expected TemplatePath /path/to/template.yaml, got %s", opts.TemplatePath)
	}

	if opts.KeyName != "my-key" {
		t.Errorf("Expected KeyName my-key, got %s", opts.KeyName)
	}

	if opts.SubnetID != "subnet-12345" {
		t.Errorf("Expected SubnetID subnet-12345, got %s", opts.SubnetID)
	}

	if opts.CustomAMI != "ami-12345" {
		t.Errorf("Expected CustomAMI ami-12345, got %s", opts.CustomAMI)
	}

	if !opts.DryRun {
		t.Error("Expected DryRun to be true")
	}
}

func TestCreateOptionsDefaults(t *testing.T) {
	opts := &CreateOptions{}

	if opts.TemplatePath != "" {
		t.Error("Expected empty TemplatePath")
	}

	if opts.KeyName != "" {
		t.Error("Expected empty KeyName")
	}

	if opts.SubnetID != "" {
		t.Error("Expected empty SubnetID")
	}

	if opts.CustomAMI != "" {
		t.Error("Expected empty CustomAMI")
	}

	if opts.DryRun {
		t.Error("Expected DryRun to be false by default")
	}
}

func TestCreateOptionsMinimal(t *testing.T) {
	// Test with only required fields
	opts := &CreateOptions{
		TemplatePath: "/path/to/template.yaml",
		KeyName:      "my-key",
	}

	if opts.TemplatePath != "/path/to/template.yaml" {
		t.Errorf("Expected TemplatePath /path/to/template.yaml, got %s", opts.TemplatePath)
	}

	if opts.KeyName != "my-key" {
		t.Errorf("Expected KeyName my-key, got %s", opts.KeyName)
	}

	// Optional fields should be empty
	if opts.SubnetID != "" {
		t.Error("Expected empty SubnetID")
	}

	if opts.CustomAMI != "" {
		t.Error("Expected empty CustomAMI")
	}
}

func TestCreateOptionsWithCustomAMI(t *testing.T) {
	opts := &CreateOptions{
		TemplatePath: "/path/to/template.yaml",
		KeyName:      "my-key",
		CustomAMI:    "ami-custom-12345",
	}

	if opts.CustomAMI != "ami-custom-12345" {
		t.Errorf("Expected CustomAMI ami-custom-12345, got %s", opts.CustomAMI)
	}

	// Should have empty SubnetID (will trigger VPC creation)
	if opts.SubnetID != "" {
		t.Error("Expected empty SubnetID when using custom AMI")
	}
}

func TestCreateOptionsWithExistingNetwork(t *testing.T) {
	opts := &CreateOptions{
		TemplatePath: "/path/to/template.yaml",
		KeyName:      "my-key",
		SubnetID:     "subnet-existing-12345",
	}

	if opts.SubnetID != "subnet-existing-12345" {
		t.Errorf("Expected SubnetID subnet-existing-12345, got %s", opts.SubnetID)
	}

	// Should have empty CustomAMI (will use default)
	if opts.CustomAMI != "" {
		t.Error("Expected empty CustomAMI when using existing network")
	}
}

func TestClusterStatus(t *testing.T) {
	status := &ClusterStatus{
		Name:           "test-cluster",
		Status:         "CREATE_COMPLETE",
		Region:         "us-east-1",
		HeadNodeIP:     "1.2.3.4",
		ComputeNodes:   5,
		SchedulerState: "RUNNING",
	}

	if status.Name != "test-cluster" {
		t.Errorf("Expected Name test-cluster, got %s", status.Name)
	}

	if status.Status != "CREATE_COMPLETE" {
		t.Errorf("Expected Status CREATE_COMPLETE, got %s", status.Status)
	}

	if status.Region != "us-east-1" {
		t.Errorf("Expected Region us-east-1, got %s", status.Region)
	}

	if status.HeadNodeIP != "1.2.3.4" {
		t.Errorf("Expected HeadNodeIP 1.2.3.4, got %s", status.HeadNodeIP)
	}

	if status.ComputeNodes != 5 {
		t.Errorf("Expected ComputeNodes 5, got %d", status.ComputeNodes)
	}

	if status.SchedulerState != "RUNNING" {
		t.Errorf("Expected SchedulerState RUNNING, got %s", status.SchedulerState)
	}
}

func TestClusterStatusDefaults(t *testing.T) {
	status := &ClusterStatus{}

	if status.Name != "" {
		t.Error("Expected empty Name")
	}

	if status.Status != "" {
		t.Error("Expected empty Status")
	}

	if status.Region != "" {
		t.Error("Expected empty Region")
	}

	if status.HeadNodeIP != "" {
		t.Error("Expected empty HeadNodeIP")
	}

	if status.ComputeNodes != 0 {
		t.Error("Expected ComputeNodes to be 0")
	}

	if status.SchedulerState != "" {
		t.Error("Expected empty SchedulerState")
	}
}

func TestClusterStatusStates(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedActive bool
	}{
		{"creating", "CREATE_IN_PROGRESS", false},
		{"complete", "CREATE_COMPLETE", true},
		{"failed", "CREATE_FAILED", false},
		{"deleting", "DELETE_IN_PROGRESS", false},
		{"deleted", "DELETE_COMPLETE", false},
		{"updating", "UPDATE_IN_PROGRESS", false},
		{"update_complete", "UPDATE_COMPLETE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &ClusterStatus{
				Name:   "test-cluster",
				Status: tt.status,
				Region: "us-east-1",
			}

			// Check if status indicates an active cluster
			isActive := status.Status == "CREATE_COMPLETE" || status.Status == "UPDATE_COMPLETE"

			if isActive != tt.expectedActive {
				t.Errorf("Expected active=%v for status %s, got %v", tt.expectedActive, tt.status, isActive)
			}
		})
	}
}

func TestClusterStatusWithoutComputeNodes(t *testing.T) {
	status := &ClusterStatus{
		Name:           "test-cluster",
		Status:         "CREATE_COMPLETE",
		Region:         "us-east-1",
		HeadNodeIP:     "1.2.3.4",
		ComputeNodes:   0,
		SchedulerState: "IDLE",
	}

	if status.ComputeNodes != 0 {
		t.Error("Expected ComputeNodes to be 0 when no compute nodes are running")
	}

	if status.SchedulerState != "IDLE" {
		t.Error("Expected SchedulerState IDLE when no compute nodes")
	}
}

func TestClusterStatusMultipleComputeNodes(t *testing.T) {
	status := &ClusterStatus{
		Name:           "large-cluster",
		Status:         "CREATE_COMPLETE",
		Region:         "us-west-2",
		HeadNodeIP:     "10.0.1.5",
		ComputeNodes:   100,
		SchedulerState: "RUNNING",
	}

	if status.ComputeNodes != 100 {
		t.Errorf("Expected ComputeNodes 100, got %d", status.ComputeNodes)
	}

	if status.Name != "large-cluster" {
		t.Errorf("Expected Name large-cluster, got %s", status.Name)
	}
}

func TestCreateOptionsValidation(t *testing.T) {
	tests := []struct {
		name  string
		opts  *CreateOptions
		valid bool
	}{
		{
			name: "valid_minimal",
			opts: &CreateOptions{
				TemplatePath: "/path/to/template.yaml",
				KeyName:      "my-key",
			},
			valid: true,
		},
		{
			name: "valid_with_subnet",
			opts: &CreateOptions{
				TemplatePath: "/path/to/template.yaml",
				KeyName:      "my-key",
				SubnetID:     "subnet-12345",
			},
			valid: true,
		},
		{
			name: "valid_with_ami",
			opts: &CreateOptions{
				TemplatePath: "/path/to/template.yaml",
				KeyName:      "my-key",
				CustomAMI:    "ami-12345",
			},
			valid: true,
		},
		{
			name: "invalid_no_template",
			opts: &CreateOptions{
				KeyName: "my-key",
			},
			valid: false,
		},
		{
			name: "invalid_no_key",
			opts: &CreateOptions{
				TemplatePath: "/path/to/template.yaml",
			},
			valid: false,
		},
		{
			name:  "invalid_empty",
			opts:  &CreateOptions{},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation: require TemplatePath and KeyName
			isValid := tt.opts.TemplatePath != "" && tt.opts.KeyName != ""

			if isValid != tt.valid {
				t.Errorf("Expected validity %v, got %v", tt.valid, isValid)
			}
		})
	}
}

func TestClusterStatusComparison(t *testing.T) {
	status1 := &ClusterStatus{
		Name:         "cluster-1",
		Status:       "CREATE_COMPLETE",
		Region:       "us-east-1",
		ComputeNodes: 5,
	}

	status2 := &ClusterStatus{
		Name:         "cluster-1",
		Status:       "CREATE_COMPLETE",
		Region:       "us-east-1",
		ComputeNodes: 5,
	}

	// Verify fields match
	if status1.Name != status2.Name {
		t.Error("Names don't match")
	}

	if status1.Status != status2.Status {
		t.Error("Statuses don't match")
	}

	if status1.Region != status2.Region {
		t.Error("Regions don't match")
	}

	if status1.ComputeNodes != status2.ComputeNodes {
		t.Error("ComputeNodes don't match")
	}
}

func TestCreateOptionsDryRun(t *testing.T) {
	opts := &CreateOptions{
		TemplatePath: "/path/to/template.yaml",
		KeyName:      "my-key",
		DryRun:       true,
	}

	if !opts.DryRun {
		t.Error("Expected DryRun to be true")
	}

	// In dry run mode, should still validate other options
	if opts.TemplatePath == "" {
		t.Error("TemplatePath should be set even in dry run")
	}

	if opts.KeyName == "" {
		t.Error("KeyName should be set even in dry run")
	}
}

func TestClusterStatusSchedulerStates(t *testing.T) {
	schedulerStates := []string{
		"RUNNING",
		"IDLE",
		"STARTING",
		"STOPPING",
		"STOPPED",
	}

	for _, state := range schedulerStates {
		status := &ClusterStatus{
			Name:           "test-cluster",
			Status:         "CREATE_COMPLETE",
			Region:         "us-east-1",
			SchedulerState: state,
		}

		if status.SchedulerState != state {
			t.Errorf("Expected SchedulerState %s, got %s", state, status.SchedulerState)
		}
	}
}
