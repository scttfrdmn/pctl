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

package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	// Create temporary state directory
	tempDir := t.TempDir()

	manager := &Manager{
		stateDir: tempDir,
	}

	state := &ClusterState{
		Name:         "test-cluster",
		Region:       "us-east-1",
		Status:       "CREATE_IN_PROGRESS",
		StackName:    "test-cluster-stack",
		TemplatePath: "/path/to/template.yaml",
		TemplateHash: "abc123",
		CreatedAt:    time.Now(),
	}

	// Save state
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load state
	loaded, err := manager.Load("test-cluster")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify
	if loaded.Name != state.Name {
		t.Errorf("Name mismatch: got %s, want %s", loaded.Name, state.Name)
	}
	if loaded.Region != state.Region {
		t.Errorf("Region mismatch: got %s, want %s", loaded.Region, state.Region)
	}
	if loaded.Status != state.Status {
		t.Errorf("Status mismatch: got %s, want %s", loaded.Status, state.Status)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	_, err := manager.Load("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent cluster, got nil")
	}
}

func TestDelete(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	state := &ClusterState{
		Name:      "test-cluster",
		Region:    "us-east-1",
		Status:    "CREATE_COMPLETE",
		CreatedAt: time.Now(),
	}

	// Save and delete
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := manager.Delete("test-cluster"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	if manager.Exists("test-cluster") {
		t.Error("Cluster still exists after deletion")
	}
}

func TestList(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Create multiple clusters
	clusters := []string{"cluster1", "cluster2", "cluster3"}
	for _, name := range clusters {
		state := &ClusterState{
			Name:      name,
			Region:    "us-east-1",
			Status:    "CREATE_COMPLETE",
			CreatedAt: time.Now(),
		}
		if err := manager.Save(state); err != nil {
			t.Fatalf("Save(%s) error = %v", name, err)
		}
	}

	// List clusters
	listed, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(listed) != len(clusters) {
		t.Errorf("List() returned %d clusters, want %d", len(listed), len(clusters))
	}
}

func TestExists(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	state := &ClusterState{
		Name:      "test-cluster",
		Region:    "us-east-1",
		Status:    "CREATE_COMPLETE",
		CreatedAt: time.Now(),
	}

	// Should not exist initially
	if manager.Exists("test-cluster") {
		t.Error("Cluster should not exist initially")
	}

	// Save and check existence
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !manager.Exists("test-cluster") {
		t.Error("Cluster should exist after saving")
	}
}

func TestListEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	clusters, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(clusters) != 0 {
		t.Errorf("Expected empty list, got %d clusters", len(clusters))
	}
}

func TestListWithNonJSONFiles(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Create a non-JSON file
	if err := os.WriteFile(filepath.Join(tempDir, "README.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a valid state file
	state := &ClusterState{
		Name:      "test-cluster",
		Region:    "us-east-1",
		Status:    "CREATE_COMPLETE",
		CreatedAt: time.Now(),
	}
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// List should only return valid cluster
	clusters, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(clusters))
	}
}
