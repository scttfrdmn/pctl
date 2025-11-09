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
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// BatchScriptAnalysis contains the results of analyzing a batch script.
type BatchScriptAnalysis struct {
	// ModulesLoaded are the modules loaded in the script
	ModulesLoaded []string
	// Commands are the executable commands found
	Commands []string
	// ResourceRequirements contains resource allocation info
	ResourceRequirements *ResourceRequirements
	// Scheduler is the detected scheduler (slurm, pbs, sge)
	Scheduler string
}

// ResourceRequirements contains resource allocation information.
type ResourceRequirements struct {
	// Nodes is the number of nodes requested
	Nodes int
	// TasksPerNode is tasks per node
	TasksPerNode int
	// CPUsPerTask is CPUs per task
	CPUsPerTask int
	// Memory is the memory request (e.g., "32GB")
	Memory string
	// Walltime is the walltime request (e.g., "24:00:00")
	Walltime string
	// Partition is the partition/queue name
	Partition string
}

// BatchAnalyzer analyzes batch scripts to extract software and resource requirements.
type BatchAnalyzer struct {
	moduleDB *ModuleDatabase
}

// NewBatchAnalyzer creates a new batch script analyzer.
func NewBatchAnalyzer() *BatchAnalyzer {
	return &BatchAnalyzer{
		moduleDB: NewModuleDatabase(),
	}
}

// AnalyzeScript analyzes a batch script and extracts requirements.
func (ba *BatchAnalyzer) AnalyzeScript(content string) *BatchScriptAnalysis {
	analysis := &BatchScriptAnalysis{
		ModulesLoaded: []string{},
		Commands: []string{},
		ResourceRequirements: &ResourceRequirements{},
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments (unless they're scheduler directives)
		if line == "" {
			continue
		}

		// Detect scheduler
		if analysis.Scheduler == "" {
			analysis.Scheduler = detectScheduler(line)
		}

		// Parse scheduler directives
		if strings.HasPrefix(line, "#SBATCH") {
			ba.parseSlurmDirective(line, analysis)
		} else if strings.HasPrefix(line, "#PBS") {
			ba.parsePBSDirective(line, analysis)
		} else if strings.HasPrefix(line, "#$") {
			ba.parseSGEDirective(line, analysis)
		}

		// Skip regular comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse module commands
		if modules := extractModuleLoad(line); len(modules) > 0 {
			analysis.ModulesLoaded = append(analysis.ModulesLoaded, modules...)
		}

		// Extract executable commands
		if cmd := extractCommand(line); cmd != "" {
			analysis.Commands = append(analysis.Commands, cmd)
		}
	}

	return analysis
}

// ConvertToSpackPackages converts the detected modules to Spack packages.
func (ba *BatchAnalyzer) ConvertToSpackPackages(analysis *BatchScriptAnalysis) ([]string, []string) {
	return ba.moduleDB.ConvertModules(analysis.ModulesLoaded)
}

func detectScheduler(line string) string {
	if strings.HasPrefix(line, "#SBATCH") {
		return "slurm"
	} else if strings.HasPrefix(line, "#PBS") {
		return "pbs"
	} else if strings.HasPrefix(line, "#$") {
		return "sge"
	}
	return ""
}

func (ba *BatchAnalyzer) parseSlurmDirective(line string, analysis *BatchScriptAnalysis) {
	// Remove #SBATCH prefix
	line = strings.TrimPrefix(line, "#SBATCH")
	line = strings.TrimSpace(line)

	// Parse common SLURM directives
	if strings.HasPrefix(line, "--nodes=") || strings.HasPrefix(line, "-N ") {
		analysis.ResourceRequirements.Nodes = extractInt(line)
	} else if strings.HasPrefix(line, "--ntasks-per-node=") {
		analysis.ResourceRequirements.TasksPerNode = extractInt(line)
	} else if strings.HasPrefix(line, "--cpus-per-task=") || strings.HasPrefix(line, "-c ") {
		analysis.ResourceRequirements.CPUsPerTask = extractInt(line)
	} else if strings.HasPrefix(line, "--mem=") {
		analysis.ResourceRequirements.Memory = extractValue(line)
	} else if strings.HasPrefix(line, "--time=") || strings.HasPrefix(line, "-t ") {
		analysis.ResourceRequirements.Walltime = extractValue(line)
	} else if strings.HasPrefix(line, "--partition=") || strings.HasPrefix(line, "-p ") {
		analysis.ResourceRequirements.Partition = extractValue(line)
	}
}

func (ba *BatchAnalyzer) parsePBSDirective(line string, analysis *BatchScriptAnalysis) {
	// Remove #PBS prefix
	line = strings.TrimPrefix(line, "#PBS")
	line = strings.TrimSpace(line)

	// Parse common PBS directives
	if strings.HasPrefix(line, "-l nodes=") {
		analysis.ResourceRequirements.Nodes = extractInt(line)
	} else if strings.HasPrefix(line, "-l walltime=") {
		analysis.ResourceRequirements.Walltime = extractValue(line)
	} else if strings.HasPrefix(line, "-q ") {
		analysis.ResourceRequirements.Partition = extractValue(line)
	}
}

func (ba *BatchAnalyzer) parseSGEDirective(line string, analysis *BatchScriptAnalysis) {
	// Remove #$ prefix
	line = strings.TrimPrefix(line, "#$")
	line = strings.TrimSpace(line)

	// Parse common SGE directives
	if strings.HasPrefix(line, "-pe ") {
		// Parallel environment
		analysis.ResourceRequirements.TasksPerNode = extractInt(line)
	} else if strings.HasPrefix(line, "-q ") {
		analysis.ResourceRequirements.Partition = extractValue(line)
	}
}

func extractModuleLoad(line string) []string {
	var modules []string

	// Patterns for module load commands
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`module\s+load\s+(.+)`),
		regexp.MustCompile(`module\s+add\s+(.+)`),
		regexp.MustCompile(`ml\s+(.+)`), // ml is shorthand for module load
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
			// Split by whitespace to get multiple modules
			mods := strings.Fields(matches[1])
			modules = append(modules, mods...)
		}
	}

	return modules
}

func extractCommand(line string) string {
	// Skip variable assignments
	if strings.Contains(line, "=") && !strings.Contains(line, "$(") {
		return ""
	}

	// Skip control structures
	if strings.HasPrefix(line, "if ") || strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "while ") || strings.HasPrefix(line, "fi") ||
		strings.HasPrefix(line, "done") {
		return ""
	}

	// Extract the first word (command name)
	fields := strings.Fields(line)
	if len(fields) > 0 {
		cmd := fields[0]
		// Filter out shell built-ins and common utilities
		builtins := map[string]bool{
			"cd": true, "echo": true, "export": true, "source": true,
			"mkdir": true, "rm": true, "mv": true, "cp": true,
			"ls": true, "pwd": true, "cat": true, "grep": true,
		}
		if !builtins[cmd] {
			return cmd
		}
	}

	return ""
}

func extractInt(s string) int {
	// Extract integer from string like "--nodes=4" or "-N 4"
	re := regexp.MustCompile(`\d+`)
	if match := re.FindString(s); match != "" {
		var val int
		fmt.Sscanf(match, "%d", &val)
		return val
	}
	return 0
}

func extractValue(s string) string {
	// Extract value after = or space
	if strings.Contains(s, "=") {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	} else {
		// Space-separated
		fields := strings.Fields(s)
		if len(fields) > 1 {
			return fields[1]
		}
	}
	return ""
}
