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

// Package config provides configuration management for pctl.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Defaults struct {
		Region  string `mapstructure:"region"`
		KeyName string `mapstructure:"key_name"`
	} `mapstructure:"defaults"`

	Registry struct {
		Sources []RegistrySource `mapstructure:"sources"`
	} `mapstructure:"registry"`

	ParallelCluster struct {
		Version       string `mapstructure:"version"`
		InstallMethod string `mapstructure:"install_method"`
	} `mapstructure:"parallelcluster"`

	Preferences struct {
		AutoUpdateRegistry   bool `mapstructure:"auto_update_registry"`
		ValidateBeforeCreate bool `mapstructure:"validate_before_create"`
		ConfirmDestructive   bool `mapstructure:"confirm_destructive"`
	} `mapstructure:"preferences"`
}

// RegistrySource represents a template registry source.
type RegistrySource struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

// Load loads the configuration from the default locations.
func Load() (*Config, error) {
	v := viper.New()

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add config paths
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	// Set defaults
	v.SetDefault("defaults.region", "us-east-1")
	v.SetDefault("parallelcluster.version", "3.8.0")
	v.SetDefault("parallelcluster.install_method", "pipx")
	v.SetDefault("preferences.auto_update_registry", true)
	v.SetDefault("preferences.validate_before_create", true)
	v.SetDefault("preferences.confirm_destructive", true)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// GetConfigDir returns the configuration directory for pctl.
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".pctl"), nil
}

// GetStateDir returns the state directory for pctl.
func GetStateDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "state"), nil
}

// EnsureConfigDir ensures the configuration directory exists.
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0755)
}

// EnsureStateDir ensures the state directory exists.
func EnsureStateDir() error {
	stateDir, err := GetStateDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(stateDir, 0755)
}
