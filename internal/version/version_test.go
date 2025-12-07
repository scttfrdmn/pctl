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

package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	// Check that all fields are populated
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	// Check that runtime values are correct
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %s, want %s", info.GoVersion, runtime.Version())
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Platform = %s, want %s", info.Platform, expectedPlatform)
	}
}

func TestInfo_String(t *testing.T) {
	info := Info{
		Version:   "v1.2.3",
		GitCommit: "abc123",
		BuildTime: "2025-01-01",
		GoVersion: "go1.21.0",
		Platform:  "linux/amd64",
	}

	result := info.String()

	// Check that all fields are in the output
	if !strings.Contains(result, "v1.2.3") {
		t.Error("String output should contain version")
	}
	if !strings.Contains(result, "abc123") {
		t.Error("String output should contain git commit")
	}
	if !strings.Contains(result, "2025-01-01") {
		t.Error("String output should contain build time")
	}
	if !strings.Contains(result, "go1.21.0") {
		t.Error("String output should contain go version")
	}
	if !strings.Contains(result, "linux/amd64") {
		t.Error("String output should contain platform")
	}
	if !strings.Contains(result, "petal") {
		t.Error("String output should contain 'petal'")
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set
	if Version == "" {
		t.Error("Default Version should not be empty")
	}
	if GitCommit == "" {
		t.Error("Default GitCommit should not be empty")
	}
	if BuildTime == "" {
		t.Error("Default BuildTime should not be empty")
	}
}
