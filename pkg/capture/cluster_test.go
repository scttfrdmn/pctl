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

package capture

import (
	"strings"
	"testing"
)

func TestNewClusterCapturer(t *testing.T) {
	cc := NewClusterCapturer()
	if cc == nil {
		t.Fatal("NewClusterCapturer() returned nil")
	}

	if cc.moduleDB == nil {
		t.Error("moduleDB is nil")
	}

	if cc.analyzer == nil {
		t.Error("analyzer is nil")
	}
}

func TestCaptureFromCommands(t *testing.T) {
	cc := NewClusterCapturer()

	outputs := map[string]string{
		"module_avail": `
----------------------- /usr/share/modules/modulefiles -----------------------
gcc/11.2.0  openmpi/4.1.1  python/3.9.5  samtools/1.17
`,
		"module_list": `
Currently Loaded Modulefiles:
 1) gcc/11.2.0   2) openmpi/4.1.1
`,
		"scheduler_info": "/usr/bin/squeue\n/usr/bin/sbatch",
		"user_list": `root:x:0:0:root:/root:/bin/bash
ubuntu:x:1000:1000:Ubuntu:/home/ubuntu:/bin/bash
testuser:x:1001:1001:Test User:/home/testuser:/bin/bash
sysuser:x:999:999:System User:/var/lib/sysuser:/sbin/nologin
`,
		"which_commands": `gcc: /usr/bin/gcc
gfortran: /usr/bin/gfortran
python: /usr/bin/python
python3: /usr/bin/python3
R:
julia: /usr/local/bin/julia
`,
	}

	capture := cc.CaptureFromCommands(outputs)

	// Test available modules
	if len(capture.AvailableModules) == 0 {
		t.Error("Expected available modules, got none")
	}

	expectedModules := []string{"gcc/11.2.0", "openmpi/4.1.1", "python/3.9.5", "samtools/1.17"}
	if len(capture.AvailableModules) != len(expectedModules) {
		t.Errorf("Expected %d modules, got %d", len(expectedModules), len(capture.AvailableModules))
	}

	// Test loaded modules
	if len(capture.LoadedModules) != 2 {
		t.Errorf("Expected 2 loaded modules, got %d", len(capture.LoadedModules))
	}

	// Test scheduler detection
	if capture.Scheduler != "slurm" {
		t.Errorf("Expected scheduler slurm, got %s", capture.Scheduler)
	}

	// Test user parsing (should only include UIDs 1000-64999)
	if len(capture.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(capture.Users))
	}

	// Verify user details
	foundUbuntu := false
	foundTestuser := false
	for _, user := range capture.Users {
		if user.Name == "ubuntu" && user.UID == 1000 {
			foundUbuntu = true
		}
		if user.Name == "testuser" && user.UID == 1001 {
			foundTestuser = true
		}
	}

	if !foundUbuntu {
		t.Error("Expected to find ubuntu user")
	}

	if !foundTestuser {
		t.Error("Expected to find testuser user")
	}

	// Test installed software detection
	if len(capture.InstalledSoftware) == 0 {
		t.Error("Expected installed software, got none")
	}

	if capture.InstalledSoftware["gcc"] != "/usr/bin/gcc" {
		t.Errorf("Expected gcc at /usr/bin/gcc, got %s", capture.InstalledSoftware["gcc"])
	}

	if _, exists := capture.InstalledSoftware["R"]; exists {
		t.Error("R should not be in installed software (empty path)")
	}
}

func TestGenerateTemplate(t *testing.T) {
	cc := NewClusterCapturer()

	capture := &ClusterCapture{
		AvailableModules: []string{"gcc/11.2.0", "openmpi/4.1.1", "python/3.9.5"},
		LoadedModules:    []string{"gcc/11.2.0"},
		InstalledSoftware: map[string]string{
			"gcc":    "/usr/bin/gcc",
			"python": "/usr/bin/python",
		},
		Scheduler: "slurm",
		Users: []User{
			{Name: "user1", UID: 1000, GID: 1000},
			{Name: "user2", UID: 1001, GID: 1001},
		},
	}

	tmpl := cc.GenerateTemplate(capture, "my-cluster")

	// Check cluster config
	if tmpl.Cluster.Name != "my-cluster" {
		t.Errorf("Expected cluster name my-cluster, got %s", tmpl.Cluster.Name)
	}

	if tmpl.Cluster.Region != "us-east-1" {
		t.Errorf("Expected default region us-east-1, got %s", tmpl.Cluster.Region)
	}

	// Check compute config
	if tmpl.Compute.HeadNode != "t3.large" {
		t.Errorf("Expected head node t3.large, got %s", tmpl.Compute.HeadNode)
	}

	if len(tmpl.Compute.Queues) != 1 {
		t.Errorf("Expected 1 queue, got %d", len(tmpl.Compute.Queues))
	}

	// Check software (should have converted modules to spack packages)
	if len(tmpl.Software.SpackPackages) == 0 {
		t.Error("Expected spack packages from module conversion")
	}

	// Check users
	if len(tmpl.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(tmpl.Users))
	}

	if tmpl.Users[0].Name != "user1" || tmpl.Users[0].UID != 1000 {
		t.Errorf("User 0 mismatch: got %v", tmpl.Users[0])
	}
}

func TestParseModuleAvail(t *testing.T) {
	cc := NewClusterCapturer()

	output := `
----------------------- /usr/share/modules/modulefiles -----------------------
gcc/11.2.0      openmpi/4.1.1      python/3.9.5
samtools/1.17   bwa/0.7.17

--- /opt/modulefiles ---
cuda/11.8  intel/2022.0

Use "module spider" to find all possible modules and extensions.
`

	modules := cc.parseModuleAvail(output)

	expectedModules := []string{
		"gcc/11.2.0", "openmpi/4.1.1", "python/3.9.5",
		"samtools/1.17", "bwa/0.7.17", "cuda/11.8", "intel/2022.0",
	}

	if len(modules) != len(expectedModules) {
		t.Errorf("Expected %d modules, got %d", len(expectedModules), len(modules))
	}

	// Check that all expected modules are present
	moduleMap := make(map[string]bool)
	for _, mod := range modules {
		moduleMap[mod] = true
	}

	for _, expected := range expectedModules {
		if !moduleMap[expected] {
			t.Errorf("Expected module %s not found", expected)
		}
	}
}

func TestDetectSchedulerType(t *testing.T) {
	cc := NewClusterCapturer()

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "slurm_squeue",
			output:   "/usr/bin/squeue\n/usr/bin/sbatch",
			expected: "slurm",
		},
		{
			name:     "slurm_version",
			output:   "slurm 22.05.0",
			expected: "slurm",
		},
		{
			name:     "pbs",
			output:   "/usr/bin/qstat\n/usr/bin/qsub",
			expected: "pbs",
		},
		{
			name:     "sge",
			output:   "Grid Engine version info",
			expected: "sge",
		},
		{
			name:     "unknown",
			output:   "command not found",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cc.detectSchedulerType(tt.output)
			if result != tt.expected {
				t.Errorf("Expected scheduler %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseUserList(t *testing.T) {
	cc := NewClusterCapturer()

	output := `root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
ubuntu:x:1000:1000:Ubuntu:/home/ubuntu:/bin/bash
user1:x:1001:1001:User One:/home/user1:/bin/bash
user2:x:1002:1002:User Two:/home/user2:/bin/bash
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
`

	users := cc.parseUserList(output)

	// Should only include users with UID 1000-64999
	expectedCount := 3
	if len(users) != expectedCount {
		t.Errorf("Expected %d users, got %d", expectedCount, len(users))
	}

	// Verify specific users
	userMap := make(map[string]User)
	for _, user := range users {
		userMap[user.Name] = user
	}

	if user, ok := userMap["ubuntu"]; !ok || user.UID != 1000 || user.GID != 1000 {
		t.Errorf("ubuntu user incorrect: %v", user)
	}

	if user, ok := userMap["user1"]; !ok || user.UID != 1001 {
		t.Errorf("user1 incorrect: %v", user)
	}

	// Verify system users are excluded
	if _, ok := userMap["root"]; ok {
		t.Error("root user should be excluded")
	}

	if _, ok := userMap["nobody"]; ok {
		t.Error("nobody user should be excluded")
	}
}

func TestDetectInstalledSoftware(t *testing.T) {
	cc := NewClusterCapturer()

	output := `gcc: /usr/bin/gcc
gfortran: /usr/bin/gfortran
python: /usr/bin/python
python3: /usr/bin/python3
R:
julia: /usr/local/bin/julia
perl: /usr/bin/perl
cmake: not found
`

	software := cc.detectInstalledSoftware(output)

	// Should have gcc, gfortran, python, python3, julia, perl
	expectedSoftware := map[string]string{
		"gcc":      "/usr/bin/gcc",
		"gfortran": "/usr/bin/gfortran",
		"python":   "/usr/bin/python",
		"python3":  "/usr/bin/python3",
		"julia":    "/usr/local/bin/julia",
		"perl":     "/usr/bin/perl",
	}

	if len(software) != len(expectedSoftware) {
		t.Errorf("Expected %d software entries, got %d", len(expectedSoftware), len(software))
	}

	for name, path := range expectedSoftware {
		if software[name] != path {
			t.Errorf("Software %s: expected %s, got %s", name, path, software[name])
		}
	}

	// Verify empty and "not found" entries are excluded
	if _, ok := software["R"]; ok {
		t.Error("R should not be included (empty path)")
	}

	if _, ok := software["cmake"]; ok {
		t.Error("cmake should not be included (not found)")
	}
}

func TestGenerateCaptureCommands(t *testing.T) {
	commands := GenerateCaptureCommands()

	expectedKeys := []string{
		"module_avail",
		"module_list",
		"scheduler_info",
		"user_list",
		"which_commands",
	}

	if len(commands) != len(expectedKeys) {
		t.Errorf("Expected %d commands, got %d", len(expectedKeys), len(commands))
	}

	for _, key := range expectedKeys {
		if _, ok := commands[key]; !ok {
			t.Errorf("Expected command key %s not found", key)
		}

		if commands[key] == "" {
			t.Errorf("Command for %s is empty", key)
		}
	}

	// Verify specific command content
	if !strings.Contains(commands["module_avail"], "module avail") {
		t.Error("module_avail command should contain 'module avail'")
	}

	if !strings.Contains(commands["user_list"], "getent passwd") {
		t.Error("user_list command should contain 'getent passwd'")
	}
}

func TestCaptureFromCommandsEmptyOutputs(t *testing.T) {
	cc := NewClusterCapturer()

	// Test with empty outputs map
	capture := cc.CaptureFromCommands(map[string]string{})

	if capture == nil {
		t.Fatal("CaptureFromCommands returned nil")
	}

	if capture.InstalledSoftware == nil {
		t.Error("InstalledSoftware should be initialized")
	}

	if len(capture.AvailableModules) != 0 {
		t.Error("Expected no available modules")
	}

	if len(capture.LoadedModules) != 0 {
		t.Error("Expected no loaded modules")
	}

	if capture.Scheduler != "" {
		t.Error("Expected empty scheduler")
	}

	if len(capture.Users) != 0 {
		t.Error("Expected no users")
	}
}

func TestParseUserListMalformed(t *testing.T) {
	cc := NewClusterCapturer()

	output := `root:x:0:0:root:/root:/bin/bash
malformed_line
user:x:invalid:1001:User:/home/user:/bin/bash
gooduser:x:1001:1001:Good User:/home/gooduser:/bin/bash
incomplete:x:1002
`

	users := cc.parseUserList(output)

	// Should only parse the valid user
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	if users[0].Name != "gooduser" {
		t.Errorf("Expected gooduser, got %s", users[0].Name)
	}
}
