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
	"fmt"
	"strings"

	"github.com/scttfrdmn/pctl/pkg/template"
)

// ClusterCapture contains information captured from a remote cluster.
type ClusterCapture struct {
	// AvailableModules are all modules available on the cluster
	AvailableModules []string
	// LoadedModules are currently loaded modules
	LoadedModules []string
	// InstalledSoftware contains detected software
	InstalledSoftware map[string]string
	// Scheduler is the detected job scheduler
	Scheduler string
	// Users are detected users (UIDs 1000-65000)
	Users []User
}

// User represents a cluster user.
type User struct {
	Name string
	UID  int
	GID  int
}

// ClusterCapturer captures configuration from a remote cluster.
type ClusterCapturer struct {
	moduleDB *ModuleDatabase
	analyzer *BatchAnalyzer
}

// NewClusterCapturer creates a new cluster capturer.
func NewClusterCapturer() *ClusterCapturer {
	return &ClusterCapturer{
		moduleDB: NewModuleDatabase(),
		analyzer: NewBatchAnalyzer(),
	}
}

// CaptureFromCommands analyzes command outputs to extract cluster configuration.
// This is designed to work with outputs from SSH commands.
func (cc *ClusterCapturer) CaptureFromCommands(outputs map[string]string) *ClusterCapture {
	capture := &ClusterCapture{
		InstalledSoftware: make(map[string]string),
	}

	// Parse module avail output
	if moduleAvail, ok := outputs["module_avail"]; ok {
		capture.AvailableModules = cc.parseModuleAvail(moduleAvail)
	}

	// Parse module list output
	if moduleList, ok := outputs["module_list"]; ok {
		capture.LoadedModules = ParseModuleList(moduleList)
	}

	// Detect scheduler
	if schedulerInfo, ok := outputs["scheduler_info"]; ok {
		capture.Scheduler = cc.detectSchedulerType(schedulerInfo)
	}

	// Parse user list
	if userList, ok := outputs["user_list"]; ok {
		capture.Users = cc.parseUserList(userList)
	}

	// Detect installed software
	if which, ok := outputs["which_commands"]; ok {
		capture.InstalledSoftware = cc.detectInstalledSoftware(which)
	}

	return capture
}

// GenerateTemplate generates a pctl template from captured cluster configuration.
func (cc *ClusterCapturer) GenerateTemplate(capture *ClusterCapture, clusterName string) *template.Template {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   clusterName,
			Region: "us-east-1", // Default, user should update
		},
		Compute: template.ComputeConfig{
			HeadNode: "t3.large", // Default, user should update
			Queues: []template.Queue{
				{
					Name:          "compute",
					InstanceTypes: []string{"c5.2xlarge"},
					MinCount:      0,
					MaxCount:      10,
				},
			},
		},
	}

	// Convert available modules to Spack packages
	spackPackages, unmapped := cc.moduleDB.ConvertModules(capture.AvailableModules)

	tmpl.Software = template.SoftwareConfig{
		SpackPackages: spackPackages,
	}

	// Convert users
	var users []template.User
	for _, user := range capture.Users {
		users = append(users, template.User{
			Name: user.Name,
			UID:  user.UID,
			GID:  user.GID,
		})
	}
	tmpl.Users = users

	// Note unmapped modules in comments (we'll add a comment field later)
	if len(unmapped) > 0 {
		// For now, log them (in a real implementation, add to template metadata)
		fmt.Printf("Warning: %d modules could not be mapped to Spack packages:\n", len(unmapped))
		for _, mod := range unmapped {
			fmt.Printf("  - %s\n", mod)
		}
	}

	return tmpl
}

func (cc *ClusterCapturer) parseModuleAvail(output string) []string {
	var modules []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip headers and empty lines
		if line == "" || strings.Contains(line, "---") ||
			strings.Contains(line, "Use") || strings.Contains(line, "Where:") {
			continue
		}

		// Extract module names (they may be listed with spaces between them)
		fields := strings.Fields(line)
		for _, field := range fields {
			// Filter out non-module entries
			if !strings.Contains(field, "(") && !strings.HasPrefix(field, "-") {
				modules = append(modules, field)
			}
		}
	}

	return modules
}

func (cc *ClusterCapturer) detectSchedulerType(output string) string {
	output = strings.ToLower(output)

	if strings.Contains(output, "slurm") || strings.Contains(output, "squeue") {
		return "slurm"
	} else if strings.Contains(output, "pbs") || strings.Contains(output, "qstat") {
		return "pbs"
	} else if strings.Contains(output, "sge") || strings.Contains(output, "grid engine") {
		return "sge"
	}

	return "unknown"
}

func (cc *ClusterCapturer) parseUserList(output string) []User {
	var users []User
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Expected format: username:x:UID:GID:...
		fields := strings.Split(line, ":")
		if len(fields) >= 4 {
			username := fields[0]
			var uid, gid int
			fmt.Sscanf(fields[2], "%d", &uid)
			fmt.Sscanf(fields[3], "%d", &gid)

			// Only include non-system users (UID >= 1000 and < 65000)
			if uid >= 1000 && uid < 65000 {
				users = append(users, User{
					Name: username,
					UID:  uid,
					GID:  gid,
				})
			}
		}
	}

	return users
}

func (cc *ClusterCapturer) detectInstalledSoftware(output string) map[string]string {
	software := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Expected format: command_name: /path/to/command
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cmdName := strings.TrimSpace(parts[0])
				cmdPath := strings.TrimSpace(parts[1])
				if cmdPath != "" && !strings.Contains(cmdPath, "not found") {
					software[cmdName] = cmdPath
				}
			}
		}
	}

	return software
}

// GenerateCaptureCommands returns a map of commands to run on the remote cluster.
func GenerateCaptureCommands() map[string]string {
	return map[string]string{
		"module_avail":    "module avail 2>&1",
		"module_list":     "module list 2>&1",
		"scheduler_info":  "which squeue sbatch qstat qsub 2>&1 || squeue --version 2>&1 || qstat --version 2>&1",
		"user_list":       "getent passwd",
		"which_commands":  "for cmd in gcc gfortran python python3 R julia perl cmake; do echo \"$cmd: $(which $cmd 2>/dev/null)\"; done",
	}
}
