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

package ami

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildStatus(t *testing.T) {
	tests := []struct {
		status BuildStatus
		str    string
	}{
		{BuildStatusLaunching, "launching"},
		{BuildStatusInstalling, "installing"},
		{BuildStatusCreating, "creating"},
		{BuildStatusComplete, "complete"},
		{BuildStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.str {
			t.Errorf("Expected status string %s, got %s", tt.str, string(tt.status))
		}
	}
}

func TestNewStateManager(t *testing.T) {
	// Set HOME to temp directory for testing
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, err := NewStateManager()
	if err != nil {
		t.Fatalf("NewStateManager() failed: %v", err)
	}

	if sm == nil {
		t.Fatal("NewStateManager() returned nil")
	}

	expectedStateDir := filepath.Join(tmpHome, ".pctl", "ami-builds")
	if sm.stateDir != expectedStateDir {
		t.Errorf("Expected state dir %s, got %s", expectedStateDir, sm.stateDir)
	}

	// Verify directory was created
	info, err := os.Stat(sm.stateDir)
	if err != nil {
		t.Errorf("State directory was not created: %v", err)
	}

	if info != nil && !info.IsDir() {
		t.Error("State path is not a directory")
	}
}

func TestNewBuildState(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 10)

	if state.BuildID == "" {
		t.Error("BuildID should not be empty")
	}

	if state.TemplateName != "test-template" {
		t.Errorf("Expected template name test-template, got %s", state.TemplateName)
	}

	if state.AMIName != "test-ami" {
		t.Errorf("Expected AMI name test-ami, got %s", state.AMIName)
	}

	if state.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", state.Region)
	}

	if state.PackageCount != 10 {
		t.Errorf("Expected package count 10, got %d", state.PackageCount)
	}

	if state.Status != BuildStatusLaunching {
		t.Errorf("Expected status launching, got %s", state.Status)
	}

	if state.Progress != 0 {
		t.Errorf("Expected progress 0, got %d", state.Progress)
	}

	if state.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()
	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 5)
	state.InstanceID = "i-1234567890abcdef0"

	// Save state
	err := sm.SaveState(state)
	if err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Load state
	loadedState, err := sm.LoadState(state.BuildID)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Verify loaded state matches
	if loadedState.BuildID != state.BuildID {
		t.Errorf("BuildID mismatch: expected %s, got %s", state.BuildID, loadedState.BuildID)
	}

	if loadedState.TemplateName != state.TemplateName {
		t.Errorf("TemplateName mismatch")
	}

	if loadedState.AMIName != state.AMIName {
		t.Errorf("AMIName mismatch")
	}

	if loadedState.Region != state.Region {
		t.Errorf("Region mismatch")
	}

	if loadedState.PackageCount != state.PackageCount {
		t.Errorf("PackageCount mismatch")
	}

	if loadedState.InstanceID != state.InstanceID {
		t.Errorf("InstanceID mismatch")
	}
}

func TestLoadStateNotFound(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	_, err := sm.LoadState("nonexistent-build-id")
	if err == nil {
		t.Error("Expected error for nonexistent build ID, got nil")
	}
}

func TestListStates(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	// Create multiple build states
	state1 := sm.NewBuildState("template1", "ami1", "us-east-1", 5)
	state2 := sm.NewBuildState("template2", "ami2", "us-west-2", 10)
	state3 := sm.NewBuildState("template3", "ami3", "eu-west-1", 15)

	sm.SaveState(state1)
	sm.SaveState(state2)
	sm.SaveState(state3)

	// List states
	states, err := sm.ListStates()
	if err != nil {
		t.Fatalf("ListStates() failed: %v", err)
	}

	if len(states) != 3 {
		t.Errorf("Expected 3 states, got %d", len(states))
	}

	// Verify states are present
	buildIDs := make(map[string]bool)
	for _, state := range states {
		buildIDs[state.BuildID] = true
	}

	if !buildIDs[state1.BuildID] {
		t.Error("state1 not found in list")
	}

	if !buildIDs[state2.BuildID] {
		t.Error("state2 not found in list")
	}

	if !buildIDs[state3.BuildID] {
		t.Error("state3 not found in list")
	}
}

func TestDeleteState(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()
	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 5)

	// Save and then delete
	sm.SaveState(state)

	err := sm.DeleteState(state.BuildID)
	if err != nil {
		t.Fatalf("DeleteState() failed: %v", err)
	}

	// Verify state is deleted
	_, err = sm.LoadState(state.BuildID)
	if err == nil {
		t.Error("Expected error loading deleted state, got nil")
	}

	// Delete again (should be idempotent)
	err = sm.DeleteState(state.BuildID)
	if err != nil {
		t.Errorf("DeleteState() of already deleted state failed: %v", err)
	}
}

func TestUpdateProgress(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()
	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 5)
	sm.SaveState(state)

	// Update progress
	err := sm.UpdateProgress(state.BuildID, 50, "Installing packages")
	if err != nil {
		t.Fatalf("UpdateProgress() failed: %v", err)
	}

	// Load and verify
	updatedState, err := sm.LoadState(state.BuildID)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if updatedState.Progress != 50 {
		t.Errorf("Expected progress 50, got %d", updatedState.Progress)
	}

	if updatedState.ProgressMessage != "Installing packages" {
		t.Errorf("Expected message 'Installing packages', got %s", updatedState.ProgressMessage)
	}
}

func TestMarkComplete(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()
	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 5)
	sm.SaveState(state)

	// Mark complete
	amiID := "ami-1234567890abcdef0"
	err := sm.MarkComplete(state.BuildID, amiID)
	if err != nil {
		t.Fatalf("MarkComplete() failed: %v", err)
	}

	// Load and verify
	completedState, err := sm.LoadState(state.BuildID)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if completedState.Status != BuildStatusComplete {
		t.Errorf("Expected status complete, got %s", completedState.Status)
	}

	if completedState.AMIID != amiID {
		t.Errorf("Expected AMI ID %s, got %s", amiID, completedState.AMIID)
	}

	if completedState.Progress != 100 {
		t.Errorf("Expected progress 100, got %d", completedState.Progress)
	}

	if completedState.EndTime == nil {
		t.Error("EndTime should be set")
	}
}

func TestMarkFailed(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()
	state := sm.NewBuildState("test-template", "test-ami", "us-east-1", 5)
	sm.SaveState(state)

	// Mark failed
	errorMsg := "Failed to install packages"
	err := sm.MarkFailed(state.BuildID, errorMsg)
	if err != nil {
		t.Fatalf("MarkFailed() failed: %v", err)
	}

	// Load and verify
	failedState, err := sm.LoadState(state.BuildID)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if failedState.Status != BuildStatusFailed {
		t.Errorf("Expected status failed, got %s", failedState.Status)
	}

	if failedState.ErrorMessage != errorMsg {
		t.Errorf("Expected error message '%s', got '%s'", errorMsg, failedState.ErrorMessage)
	}

	if failedState.EndTime == nil {
		t.Error("EndTime should be set")
	}
}

func TestCleanupOldStates(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	// Create old completed state
	oldState := sm.NewBuildState("old-template", "old-ami", "us-east-1", 5)
	oldEndTime := time.Now().Add(-48 * time.Hour) // 2 days ago
	oldState.Status = BuildStatusComplete
	oldState.EndTime = &oldEndTime
	sm.SaveState(oldState)

	// Create recent completed state
	recentState := sm.NewBuildState("recent-template", "recent-ami", "us-east-1", 5)
	recentEndTime := time.Now().Add(-12 * time.Hour) // 12 hours ago
	recentState.Status = BuildStatusComplete
	recentState.EndTime = &recentEndTime
	sm.SaveState(recentState)

	// Create in-progress state (should not be cleaned up regardless of age)
	inProgressState := sm.NewBuildState("inprogress-template", "inprogress-ami", "us-east-1", 5)
	inProgressState.Status = BuildStatusInstalling
	sm.SaveState(inProgressState)

	// Cleanup states older than 24 hours
	err := sm.CleanupOldStates(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldStates() failed: %v", err)
	}

	// Verify old state is deleted
	_, err = sm.LoadState(oldState.BuildID)
	if err == nil {
		t.Error("Old state should have been deleted")
	}

	// Verify recent state still exists
	_, err = sm.LoadState(recentState.BuildID)
	if err != nil {
		t.Error("Recent state should not have been deleted")
	}

	// Verify in-progress state still exists
	_, err = sm.LoadState(inProgressState.BuildID)
	if err != nil {
		t.Error("In-progress state should not have been deleted")
	}
}

func TestListStatesWithInvalidFiles(t *testing.T) {
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	// Create valid state
	validState := sm.NewBuildState("valid-template", "valid-ami", "us-east-1", 5)
	sm.SaveState(validState)

	// Create invalid JSON file
	invalidFile := filepath.Join(sm.stateDir, "invalid.json")
	os.WriteFile(invalidFile, []byte("not valid json"), 0644)

	// Create non-JSON file
	nonJSONFile := filepath.Join(sm.stateDir, "README.txt")
	os.WriteFile(nonJSONFile, []byte("This is a readme"), 0644)

	// List should skip invalid files and return only valid state
	states, err := sm.ListStates()
	if err != nil {
		t.Fatalf("ListStates() failed: %v", err)
	}

	if len(states) != 1 {
		t.Errorf("Expected 1 valid state, got %d", len(states))
	}

	if len(states) > 0 && states[0].BuildID != validState.BuildID {
		t.Error("Valid state not found in list")
	}
}

func TestBuildStateJSONSerialization(t *testing.T) {
	now := time.Now()
	endTime := now.Add(1 * time.Hour)

	state := &BuildState{
		BuildID:         "test-build-id",
		AMIID:           "ami-12345",
		InstanceID:      "i-12345",
		Status:          BuildStatusComplete,
		Progress:        100,
		ProgressMessage: "Build complete",
		StartTime:       now,
		EndTime:         &endTime,
		TemplateName:    "test-template",
		AMIName:         "test-ami",
		Region:          "us-east-1",
		PackageCount:    10,
		ErrorMessage:    "",
	}

	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	sm, _ := NewStateManager()

	// Save and load
	err := sm.SaveState(state)
	if err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	loadedState, err := sm.LoadState(state.BuildID)
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Verify all fields
	if loadedState.BuildID != state.BuildID {
		t.Error("BuildID mismatch")
	}

	if loadedState.AMIID != state.AMIID {
		t.Error("AMIID mismatch")
	}

	if loadedState.InstanceID != state.InstanceID {
		t.Error("InstanceID mismatch")
	}

	if loadedState.Status != state.Status {
		t.Error("Status mismatch")
	}

	if loadedState.Progress != state.Progress {
		t.Error("Progress mismatch")
	}

	if loadedState.ProgressMessage != state.ProgressMessage {
		t.Error("ProgressMessage mismatch")
	}

	if loadedState.TemplateName != state.TemplateName {
		t.Error("TemplateName mismatch")
	}

	if loadedState.AMIName != state.AMIName {
		t.Error("AMIName mismatch")
	}

	if loadedState.Region != state.Region {
		t.Error("Region mismatch")
	}

	if loadedState.PackageCount != state.PackageCount {
		t.Error("PackageCount mismatch")
	}

	// Times need special handling due to JSON serialization
	if !loadedState.StartTime.Equal(state.StartTime) {
		t.Error("StartTime mismatch")
	}

	if loadedState.EndTime == nil || !loadedState.EndTime.Equal(*state.EndTime) {
		t.Error("EndTime mismatch")
	}
}
