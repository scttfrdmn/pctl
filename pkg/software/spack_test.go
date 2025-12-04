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

func TestSpackInstaller_GenerateInstallScript(t *testing.T) {
	tests := []struct {
		name   string
		config *SpackConfig
		checks []string
	}{
		{
			name:   "default config",
			config: nil,
			checks: []string{
				"#!/bin/bash",
				"set -e",
				"yum groupinstall -y \"Development Tools\"",
				"git clone",
				"https://github.com/spack/spack.git",
				"/opt/spack",
				"v0.23.0",
				"spack compiler find",
				"aws-binaries",
				"https://binaries.spack.io",
				"buildcache keys",
				"/etc/profile.d/z00_spack.sh",
			},
		},
		{
			name: "custom config",
			config: &SpackConfig{
				InstallPath: "/custom/spack",
				Version:     "v0.21.0",
			},
			checks: []string{
				"/custom/spack",
				"v0.21.0",
			},
		},
		{
			name: "with compilers",
			config: &SpackConfig{
				InstallPath:      "/opt/spack",
				Version:          "releases/latest",
				CompilerPackages: []string{"gcc@12.2.0", "llvm@15.0.0"},
			},
			checks: []string{
				"gcc@12.2.0",
				"llvm@15.0.0",
				"spack compiler find",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewSpackInstaller(tt.config)
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

func TestSpackInstaller_GeneratePackageInstallScript(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		checks   []string
	}{
		{
			name:     "no packages",
			packages: []string{},
			checks: []string{
				"#!/bin/bash",
				"No packages to install",
			},
		},
		{
			name: "regular packages",
			packages: []string{
				"python@3.10",
				"cmake@3.26.0",
				"samtools@1.17",
			},
			checks: []string{
				"#!/bin/bash",
				"python@3.10",
				"cmake@3.26.0",
				"samtools@1.17",
				"spack install",
				"--use-buildcache",
				"SPACK_PARALLEL_JOBS",
			},
		},
		{
			name: "compilers and packages",
			packages: []string{
				"gcc@11.3.0",
				"openmpi@4.1.4",
				"python@3.10",
			},
			checks: []string{
				"# Install compilers first",
				"gcc@11.3.0",
				"spack compiler find",
				"openmpi@4.1.4",
				"python@3.10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := NewSpackInstaller(nil)
			script := installer.GeneratePackageInstallScript(tt.packages)

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

func TestSpackInstaller_BuildcacheConfiguration(t *testing.T) {
	installer := NewSpackInstaller(nil)
	script := installer.GenerateInstallScript()

	// Verify AWS buildcache is configured
	requiredContent := []string{
		"aws-binaries",
		"https://binaries.spack.io",
		"spack buildcache keys",
		"--install --trust",
	}

	for _, content := range requiredContent {
		if !strings.Contains(script, content) {
			t.Errorf("Install script missing buildcache configuration: %q", content)
		}
	}

	// Verify package install uses buildcache
	packages := []string{"python@3.10"}
	packageScript := installer.GeneratePackageInstallScript(packages)

	if !strings.Contains(packageScript, "--use-buildcache") {
		t.Error("Package install script should use buildcache")
	}
}
