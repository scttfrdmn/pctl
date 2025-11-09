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

func TestModuleDatabase_Lookup(t *testing.T) {
	db := NewModuleDatabase()

	tests := []struct {
		name           string
		moduleName     string
		expectFound    bool
		expectedSpack  string
	}{
		{"gcc exact", "gcc", true, "gcc@11.3.0"},
		{"gcc with version", "gcc/11.2.0", true, "gcc@11.3.0"},
		{"python lowercase", "python", true, "python@3.10"},
		{"Python capitalized", "Python/3.9.5", true, "python@3.10"},
		{"samtools", "samtools", true, "samtools@1.17"},
		{"openmpi", "openmpi", true, "openmpi@4.1.4"},
		{"unknown module", "unknownmodule", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, found := db.Lookup(tt.moduleName)

			if found != tt.expectFound {
				t.Errorf("Lookup(%s) found=%v, want %v", tt.moduleName, found, tt.expectFound)
			}

			if found && mapping.SpackPackage != tt.expectedSpack {
				t.Errorf("Lookup(%s) spack=%s, want %s", tt.moduleName, mapping.SpackPackage, tt.expectedSpack)
			}
		})
	}
}

func TestNormalizeModuleName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gcc/11.2.0", "gcc"},
		{"openmpi-4.1.1", "openmpi"},
		{"Python/3.9.5", "python"},
		{"samtools_1.17", "samtools"},
		{"gcc", "gcc"},
		{"OpenMPI-4.1.1-mpi", "openmpi"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeModuleName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeModuleName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestModuleDatabase_ConvertModules(t *testing.T) {
	db := NewModuleDatabase()

	modules := []string{"gcc/11.2.0", "openmpi-4.1.1", "python/3.9.5", "unknownmodule"}
	spackPackages, unmapped := db.ConvertModules(modules)

	if len(spackPackages) != 3 {
		t.Errorf("Expected 3 spack packages, got %d", len(spackPackages))
	}

	if len(unmapped) != 1 {
		t.Errorf("Expected 1 unmapped module, got %d", len(unmapped))
	}

	if unmapped[0] != "unknownmodule" {
		t.Errorf("Expected unmapped module 'unknownmodule', got %s", unmapped[0])
	}
}

func TestParseModuleList(t *testing.T) {
	output := `Currently Loaded Modules:
  1) gcc/11.2.0   2) openmpi/4.1.1   3) python/3.9.5
`

	modules := ParseModuleList(output)

	expected := []string{"gcc/11.2.0", "openmpi/4.1.1", "python/3.9.5"}

	if len(modules) != len(expected) {
		t.Errorf("Expected %d modules, got %d", len(expected), len(modules))
	}

	for i, mod := range modules {
		if mod != expected[i] {
			t.Errorf("Module %d: expected %s, got %s", i, expected[i], mod)
		}
	}
}
