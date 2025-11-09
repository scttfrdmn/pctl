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
	"testing"
)

func TestBatchAnalyzer_AnalyzeScript(t *testing.T) {
	analyzer := NewBatchAnalyzer()

	script := `#!/bin/bash
#SBATCH --nodes=4
#SBATCH --ntasks-per-node=16
#SBATCH --time=24:00:00
#SBATCH --partition=compute

module load gcc/11.2.0
module load openmpi/4.1.1
module load python/3.9.5

mpirun -np 64 ./my_simulation
python analyze_results.py
`

	analysis := analyzer.AnalyzeScript(script)

	// Check scheduler detection
	if analysis.Scheduler != "slurm" {
		t.Errorf("Expected scheduler=slurm, got %s", analysis.Scheduler)
	}

	// Check modules loaded
	expectedModules := []string{"gcc/11.2.0", "openmpi/4.1.1", "python/3.9.5"}
	if len(analysis.ModulesLoaded) != len(expectedModules) {
		t.Errorf("Expected %d modules, got %d", len(expectedModules), len(analysis.ModulesLoaded))
	}

	// Check resource requirements
	if analysis.ResourceRequirements.Nodes != 4 {
		t.Errorf("Expected 4 nodes, got %d", analysis.ResourceRequirements.Nodes)
	}

	if analysis.ResourceRequirements.TasksPerNode != 16 {
		t.Errorf("Expected 16 tasks per node, got %d", analysis.ResourceRequirements.TasksPerNode)
	}

	if analysis.ResourceRequirements.Walltime != "24:00:00" {
		t.Errorf("Expected walltime=24:00:00, got %s", analysis.ResourceRequirements.Walltime)
	}

	if analysis.ResourceRequirements.Partition != "compute" {
		t.Errorf("Expected partition=compute, got %s", analysis.ResourceRequirements.Partition)
	}

	// Check commands
	if len(analysis.Commands) < 2 {
		t.Errorf("Expected at least 2 commands, got %d", len(analysis.Commands))
	}
}

func TestExtractModuleLoad(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{"module load", "module load gcc/11.2.0", []string{"gcc/11.2.0"}},
		{"module add", "module add openmpi", []string{"openmpi"}},
		{"ml shorthand", "ml python/3.9.5", []string{"python/3.9.5"}},
		{"multiple modules", "module load gcc openmpi", []string{"gcc", "openmpi"}},
		{"no module", "echo hello", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractModuleLoad(tt.line)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d modules, got %d", len(tt.expected), len(result))
				return
			}

			for i, mod := range result {
				if mod != tt.expected[i] {
					t.Errorf("Module %d: expected %s, got %s", i, tt.expected[i], mod)
				}
			}
		})
	}
}

func TestDetectScheduler(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"#SBATCH --nodes=4", "slurm"},
		{"#PBS -l nodes=4", "pbs"},
		{"#$ -pe mpi 64", "sge"},
		{"#!/bin/bash", ""},
	}

	for _, tt := range tests {
		result := detectScheduler(tt.line)
		if result != tt.expected {
			t.Errorf("detectScheduler(%s) = %s, want %s", tt.line, result, tt.expected)
		}
	}
}
