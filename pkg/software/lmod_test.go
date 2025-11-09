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

package software

import (
	"strings"
	"testing"
)

func TestLmodInstaller_GenerateInstallScript(t *testing.T) {
	tests := []struct {
		name   string
		config *LmodConfig
		checks []string
	}{
		{
			name:   "default config",
			config: nil,
			checks: []string{
				"#!/bin/bash",
				"set -e",
				"yum install -y lua",
				"https://github.com/TACC/Lmod",
				"/opt/apps",
				"8.7.37",
				"./configure --prefix",
				"make install",
				"/etc/profile.d/z00_lmod.sh",
				"MODULEPATH",
				"LMOD_CMD",
			},
		},
		{
			name: "custom config",
			config: &LmodConfig{
				InstallPath: "/custom/apps",
				Version:     "8.7.30",
				ModulePath:  "/custom/modules",
			},
			checks: []string{
				"/custom/apps",
				"8.7.30",
				"/custom/modules",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewLmodInstaller(tt.config)
			script := installer.GenerateInstallScript()

			// Check that script is not empty
			if script == "" {
				t.Fatal("Generated script is empty")
			}

			// Check for required content
			for _, check := range tt.checks {
				if !strings.Contains(script, check) {
					t.Errorf("Script missing expected content: %q", check)
				}
			}
		})
	}
}

func TestLmodInstaller_GenerateSpackIntegrationScript(t *testing.T) {
	installer := NewLmodInstaller(nil)
	script := installer.GenerateSpackIntegrationScript()

	// Check that script is not empty
	if script == "" {
		t.Fatal("Generated script is empty")
	}

	// Check for required content
	requiredContent := []string{
		"#!/bin/bash",
		"set -e",
		"Spack-Lmod Integration",
		"modules.yaml",
		"spack module lmod refresh",
		"module avail",
	}

	for _, content := range requiredContent {
		if !strings.Contains(script, content) {
			t.Errorf("Script missing expected content: %q", content)
		}
	}
}

func TestLmodInstaller_GenerateModuleFile(t *testing.T) {
	installer := NewLmodInstaller(nil)

	env := map[string]string{
		"PYTHON_ROOT": "/opt/python/3.10",
		"PYTHONPATH":  "/opt/python/3.10/lib",
	}

	moduleFile := installer.GenerateModuleFile("python", "3.10", "/opt/python/3.10", env)

	// Check that module file is not empty
	if moduleFile == "" {
		t.Fatal("Generated module file is empty")
	}

	// Check for required Lua module file content
	requiredContent := []string{
		"-- -*- lua -*-",
		"help([[",
		"python",
		"3.10",
		"whatis(",
		"setenv(\"PYTHON_ROOT\"",
		"setenv(\"PYTHONPATH\"",
		"prepend_path(\"PATH\"",
		"prepend_path(\"LD_LIBRARY_PATH\"",
		"prepend_path(\"MANPATH\"",
	}

	for _, content := range requiredContent {
		if !strings.Contains(moduleFile, content) {
			t.Errorf("Module file missing expected content: %q", content)
		}
	}
}
