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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() failed: %v", err)
	}

	// Should be non-empty
	if dir == "" {
		t.Error("GetConfigDir() returned empty string")
	}

	// Should end with .petal
	if filepath.Base(dir) != ".petal" {
		t.Errorf("GetConfigDir() = %s, want directory ending with .petal", dir)
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("GetConfigDir() = %s, want absolute path", dir)
	}
}

func TestGetStateDir(t *testing.T) {
	stateDir, err := GetStateDir()
	if err != nil {
		t.Fatalf("GetStateDir() failed: %v", err)
	}

	// Should be non-empty
	if stateDir == "" {
		t.Error("GetStateDir() returned empty string")
	}

	// Should end with state
	if filepath.Base(stateDir) != "state" {
		t.Errorf("GetStateDir() = %s, want directory ending with state", stateDir)
	}

	// Should be an absolute path
	if !filepath.IsAbs(stateDir) {
		t.Errorf("GetStateDir() = %s, want absolute path", stateDir)
	}

	// Should be under config dir
	configDir, _ := GetConfigDir()
	expected := filepath.Join(configDir, "state")
	if stateDir != expected {
		t.Errorf("GetStateDir() = %s, want %s", stateDir, expected)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Get original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Ensure config directory
	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}

	// Check that directory was created
	configDir := filepath.Join(tempDir, ".petal")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Config directory was not created: %v", err)
	}

	// Check it's a directory
	if !info.IsDir() {
		t.Error("Config path is not a directory")
	}

	// Check permissions
	if info.Mode().Perm() != 0755 {
		t.Errorf("Config directory has permissions %o, want 0755", info.Mode().Perm())
	}
}

func TestEnsureStateDir(t *testing.T) {
	// Get original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create temp directory for testing
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Ensure state directory (should also create config dir)
	err := EnsureStateDir()
	if err != nil {
		t.Fatalf("EnsureStateDir() failed: %v", err)
	}

	// Check that directory was created
	stateDir := filepath.Join(tempDir, ".petal", "state")
	info, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("State directory was not created: %v", err)
	}

	// Check it's a directory
	if !info.IsDir() {
		t.Error("State path is not a directory")
	}

	// Check permissions
	if info.Mode().Perm() != 0755 {
		t.Errorf("State directory has permissions %o, want 0755", info.Mode().Perm())
	}
}

func TestLoadDefaults(t *testing.T) {
	// Get original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create temp directory for testing (no config file)
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Load should succeed even without config file (use defaults)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults
	if cfg.Defaults.Region != "us-east-1" {
		t.Errorf("Default region = %s, want us-east-1", cfg.Defaults.Region)
	}

	if cfg.ParallelCluster.Version != "3.14.0" {
		t.Errorf("Default ParallelCluster version = %s, want 3.14.0", cfg.ParallelCluster.Version)
	}

	if cfg.ParallelCluster.InstallMethod != "pipx" {
		t.Errorf("Default install method = %s, want pipx", cfg.ParallelCluster.InstallMethod)
	}

	if !cfg.Preferences.AutoUpdateRegistry {
		t.Error("Default auto_update_registry should be true")
	}

	if !cfg.Preferences.ValidateBeforeCreate {
		t.Error("Default validate_before_create should be true")
	}

	if !cfg.Preferences.ConfirmDestructive {
		t.Error("Default confirm_destructive should be true")
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	// Get original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create temp directory and config file
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, ".petal")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write config file
	configContent := `defaults:
  region: us-west-2
  key_name: my-key

parallelcluster:
  version: 3.9.0
  install_method: pip

preferences:
  auto_update_registry: false
  validate_before_create: false
  confirm_destructive: false

registry:
  sources:
    - name: official
      url: https://github.com/example/templates
`
	configFile := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check loaded values
	if cfg.Defaults.Region != "us-west-2" {
		t.Errorf("Loaded region = %s, want us-west-2", cfg.Defaults.Region)
	}

	if cfg.Defaults.KeyName != "my-key" {
		t.Errorf("Loaded key_name = %s, want my-key", cfg.Defaults.KeyName)
	}

	if cfg.ParallelCluster.Version != "3.9.0" {
		t.Errorf("Loaded ParallelCluster version = %s, want 3.9.0", cfg.ParallelCluster.Version)
	}

	if cfg.ParallelCluster.InstallMethod != "pip" {
		t.Errorf("Loaded install method = %s, want pip", cfg.ParallelCluster.InstallMethod)
	}

	if cfg.Preferences.AutoUpdateRegistry {
		t.Error("Loaded auto_update_registry should be false")
	}

	if cfg.Preferences.ValidateBeforeCreate {
		t.Error("Loaded validate_before_create should be false")
	}

	if cfg.Preferences.ConfirmDestructive {
		t.Error("Loaded confirm_destructive should be false")
	}

	// Check registry sources
	if len(cfg.Registry.Sources) != 1 {
		t.Fatalf("Expected 1 registry source, got %d", len(cfg.Registry.Sources))
	}

	if cfg.Registry.Sources[0].Name != "official" {
		t.Errorf("Registry source name = %s, want official", cfg.Registry.Sources[0].Name)
	}

	if cfg.Registry.Sources[0].URL != "https://github.com/example/templates" {
		t.Errorf("Registry source URL = %s, want https://github.com/example/templates", cfg.Registry.Sources[0].URL)
	}
}

func TestRegistrySourceStruct(t *testing.T) {
	source := RegistrySource{
		Name: "test-source",
		URL:  "https://example.com/templates",
	}

	if source.Name != "test-source" {
		t.Errorf("RegistrySource.Name = %s, want test-source", source.Name)
	}

	if source.URL != "https://example.com/templates" {
		t.Errorf("RegistrySource.URL = %s, want https://example.com/templates", source.URL)
	}
}
