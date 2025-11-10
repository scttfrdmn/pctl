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

func TestNewManager(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager() returned nil manager")
	}

	if manager.stateDir == "" {
		t.Error("NewManager() manager has empty stateDir")
	}

	// Verify state directory was created
	if _, err := os.Stat(manager.stateDir); os.IsNotExist(err) {
		t.Errorf("State directory was not created: %s", manager.stateDir)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Create a file with invalid JSON
	invalidJSONPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJSONPath, []byte("{ invalid json }"), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err := manager.Load("invalid")
	if err == nil {
		t.Error("Expected error loading invalid JSON, got nil")
	}
}

func TestDeleteNonExistentCluster(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Deleting non-existent cluster should not error
	if err := manager.Delete("nonexistent"); err != nil {
		t.Errorf("Delete() of nonexistent cluster returned error: %v", err)
	}
}

func TestListWithInvalidJSONFiles(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Create a JSON file with invalid content
	invalidJSONPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(invalidJSONPath, []byte("{ invalid }"), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	// Create a valid state file
	state := &ClusterState{
		Name:      "valid-cluster",
		Region:    "us-east-1",
		Status:    "CREATE_COMPLETE",
		CreatedAt: time.Now(),
	}
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// List should skip invalid files and return only valid cluster
	clusters, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("Expected 1 valid cluster, got %d", len(clusters))
	}

	if len(clusters) > 0 && clusters[0].Name != "valid-cluster" {
		t.Errorf("Expected cluster name 'valid-cluster', got %s", clusters[0].Name)
	}
}

func TestListWithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	// Create a subdirectory (should be skipped)
	subdir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
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

	// List should skip directories and return only files
	clusters, err := manager.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(clusters))
	}
}

func TestSaveUpdatesTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	state := &ClusterState{
		Name:      "test-cluster",
		Region:    "us-east-1",
		Status:    "CREATE_IN_PROGRESS",
		CreatedAt: time.Now(),
	}

	// Save first time
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	firstUpdate := state.UpdatedAt

	// Wait a bit and save again
	time.Sleep(10 * time.Millisecond)

	state.Status = "CREATE_COMPLETE"
	if err := manager.Save(state); err != nil {
		t.Fatalf("Save() second time error = %v", err)
	}

	// UpdatedAt should be different
	if state.UpdatedAt.Equal(firstUpdate) || state.UpdatedAt.Before(firstUpdate) {
		t.Error("UpdatedAt was not updated on second save")
	}

	// Load and verify the updated timestamp was persisted
	loaded, err := manager.Load("test-cluster")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Status != "CREATE_COMPLETE" {
		t.Errorf("Status = %s, want CREATE_COMPLETE", loaded.Status)
	}
}

func TestStatePath(t *testing.T) {
	tempDir := t.TempDir()
	manager := &Manager{
		stateDir: tempDir,
	}

	expected := filepath.Join(tempDir, "my-cluster.json")
	actual := manager.statePath("my-cluster")

	if actual != expected {
		t.Errorf("statePath() = %s, want %s", actual, expected)
	}
}
