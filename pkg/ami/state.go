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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// BuildStatus represents the status of an AMI build.
type BuildStatus string

const (
	// BuildStatusLaunching means the build instance is being launched
	BuildStatusLaunching BuildStatus = "launching"
	// BuildStatusInstalling means software is being installed
	BuildStatusInstalling BuildStatus = "installing"
	// BuildStatusCreating means the AMI is being created
	BuildStatusCreating BuildStatus = "creating"
	// BuildStatusComplete means the build finished successfully
	BuildStatusComplete BuildStatus = "complete"
	// BuildStatusFailed means the build failed
	BuildStatusFailed BuildStatus = "failed"
)

// BuildState tracks the state of an AMI build.
type BuildState struct {
	// BuildID is a unique identifier for this build
	BuildID string `json:"build_id"`
	// AMIID is populated when the AMI is created
	AMIID string `json:"ami_id,omitempty"`
	// InstanceID is the temporary build instance
	InstanceID string `json:"instance_id"`
	// Status is the current build status
	Status BuildStatus `json:"status"`
	// Progress is the current progress percentage (0-100)
	Progress int `json:"progress"`
	// ProgressMessage is the last progress message
	ProgressMessage string `json:"progress_message,omitempty"`
	// StartTime is when the build started
	StartTime time.Time `json:"start_time"`
	// EndTime is when the build completed or failed
	EndTime *time.Time `json:"end_time,omitempty"`
	// TemplateName is the source template name
	TemplateName string `json:"template_name"`
	// AMIName is the target AMI name
	AMIName string `json:"ami_name"`
	// Region is the AWS region
	Region string `json:"region"`
	// PackageCount is the number of packages being installed
	PackageCount int `json:"package_count"`
	// ErrorMessage is populated if the build fails
	ErrorMessage string `json:"error_message,omitempty"`
}

// StateManager manages AMI build state persistence.
type StateManager struct {
	stateDir string
}

// NewStateManager creates a new state manager.
func NewStateManager() (*StateManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".pctl", "ami-builds")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &StateManager{stateDir: stateDir}, nil
}

// NewBuildState creates a new build state with a unique ID.
func (sm *StateManager) NewBuildState(templateName, amiName, region string, packageCount int) *BuildState {
	return &BuildState{
		BuildID:      uuid.New().String(),
		Status:       BuildStatusLaunching,
		Progress:     0,
		StartTime:    time.Now(),
		TemplateName: templateName,
		AMIName:      amiName,
		Region:       region,
		PackageCount: packageCount,
	}
}

// SaveState saves the build state to disk.
func (sm *StateManager) SaveState(state *BuildState) error {
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", state.BuildID))

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadState loads a build state from disk.
func (sm *StateManager) LoadState(buildID string) (*BuildState, error) {
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", buildID))

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("build %s not found", buildID)
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state BuildState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// ListStates lists all build states.
func (sm *StateManager) ListStates() ([]*BuildState, error) {
	entries, err := os.ReadDir(sm.stateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	var states []*BuildState
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		buildID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		state, err := sm.LoadState(buildID)
		if err != nil {
			// Skip invalid state files
			continue
		}

		states = append(states, state)
	}

	return states, nil
}

// DeleteState removes a build state from disk.
func (sm *StateManager) DeleteState(buildID string) error {
	stateFile := filepath.Join(sm.stateDir, fmt.Sprintf("%s.json", buildID))

	if err := os.Remove(stateFile); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

// CleanupOldStates removes completed/failed build states older than the specified duration.
func (sm *StateManager) CleanupOldStates(maxAge time.Duration) error {
	states, err := sm.ListStates()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)

	for _, state := range states {
		// Only clean up completed or failed builds
		if state.Status != BuildStatusComplete && state.Status != BuildStatusFailed {
			continue
		}

		// Check if the build is old enough
		endTime := state.EndTime
		if endTime != nil && endTime.Before(cutoff) {
			if err := sm.DeleteState(state.BuildID); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Warning: Failed to delete old state %s: %v\n", state.BuildID, err)
			}
		}
	}

	return nil
}

// UpdateProgress updates the progress for a build state.
func (sm *StateManager) UpdateProgress(buildID string, progress int, message string) error {
	state, err := sm.LoadState(buildID)
	if err != nil {
		return err
	}

	state.Progress = progress
	state.ProgressMessage = message

	return sm.SaveState(state)
}

// MarkComplete marks a build as complete.
func (sm *StateManager) MarkComplete(buildID, amiID string) error {
	state, err := sm.LoadState(buildID)
	if err != nil {
		return err
	}

	now := time.Now()
	state.Status = BuildStatusComplete
	state.AMIID = amiID
	state.Progress = 100
	state.EndTime = &now

	return sm.SaveState(state)
}

// MarkFailed marks a build as failed.
func (sm *StateManager) MarkFailed(buildID string, errorMsg string) error {
	state, err := sm.LoadState(buildID)
	if err != nil {
		return err
	}

	now := time.Now()
	state.Status = BuildStatusFailed
	state.ErrorMessage = errorMsg
	state.EndTime = &now

	return sm.SaveState(state)
}
