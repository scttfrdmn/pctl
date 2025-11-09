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

	"github.com/scttfrdmn/pctl/pkg/template"
)

func TestManager_GenerateBootstrapScript(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Software: template.SoftwareConfig{
			SpackPackages: []string{
				"python@3.10",
				"gcc@11.3.0",
			},
		},
		Users: []template.User{
			{Name: "testuser", UID: 5001, GID: 5001},
		},
		Data: template.DataConfig{
			S3Mounts: []template.S3Mount{
				{Bucket: "test-bucket", MountPoint: "/mnt/data"},
			},
		},
	}

	manager := NewManager()
	script := manager.GenerateBootstrapScript(tmpl, true, true)

	// Check that script is not empty
	if script == "" {
		t.Fatal("Generated script is empty")
	}

	// Check for all major sections
	requiredSections := []string{
		"#!/bin/bash",
		"set -e",
		"pctl Bootstrap Script",
		"test-cluster",
		"USER CREATION",
		"testuser",
		"S3 MOUNT CONFIGURATION",
		"test-bucket",
		"SOFTWARE INSTALLATION",
		"Spack Installation",
		"Lmod Installation",
		"python@3.10",
		"gcc@11.3.0",
		"Bootstrap complete",
	}

	for _, section := range requiredSections {
		if !strings.Contains(script, section) {
			t.Errorf("Script missing expected section: %q", section)
		}
	}
}

func TestManager_GenerateBootstrapScript_NoUsers(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Software: template.SoftwareConfig{
			SpackPackages: []string{"python@3.10"},
		},
	}

	manager := NewManager()
	script := manager.GenerateBootstrapScript(tmpl, false, false)

	// Should not contain user creation section
	if strings.Contains(script, "USER CREATION") {
		t.Error("Script should not contain user creation section")
	}

	// Should not contain S3 mount section
	if strings.Contains(script, "S3 MOUNT") {
		t.Error("Script should not contain S3 mount section")
	}

	// Should still contain software section
	if !strings.Contains(script, "SOFTWARE INSTALLATION") {
		t.Error("Script should contain software installation section")
	}
}

func TestManager_GenerateSoftwareOnlyScript(t *testing.T) {
	manager := NewManager()
	packages := []string{"python@3.10", "gcc@11.3.0", "cmake@3.26.0"}
	script := manager.GenerateSoftwareOnlyScript(packages)

	// Check that script is not empty
	if script == "" {
		t.Fatal("Generated script is empty")
	}

	// Check for required content
	requiredContent := []string{
		"#!/bin/bash",
		"set -e",
		"Software Installation Script",
		"python@3.10",
		"gcc@11.3.0",
		"cmake@3.26.0",
	}

	for _, content := range requiredContent {
		if !strings.Contains(script, content) {
			t.Errorf("Script missing expected content: %q", content)
		}
	}

	// Should not contain user or S3 sections
	if strings.Contains(script, "USER CREATION") {
		t.Error("Software-only script should not contain user creation")
	}
	if strings.Contains(script, "S3 MOUNT") {
		t.Error("Software-only script should not contain S3 mounts")
	}
}

func TestManager_GenerateBootstrapScript_NoSoftware(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Users: []template.User{
			{Name: "testuser", UID: 5001, GID: 5001},
		},
	}

	manager := NewManager()
	script := manager.GenerateBootstrapScript(tmpl, true, true)

	// Should contain user creation
	if !strings.Contains(script, "USER CREATION") {
		t.Error("Script should contain user creation section")
	}

	// Should not contain software installation section
	if strings.Contains(script, "SOFTWARE INSTALLATION") {
		t.Error("Script should not contain software installation section when no packages specified")
	}
}
