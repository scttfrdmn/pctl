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

// Package capture provides tools for capturing and analyzing existing cluster configurations.
package capture

import (
	"fmt"
	"regexp"
	"strings"
)

// ModuleMapping represents a mapping from an on-prem module to a Spack package.
type ModuleMapping struct {
	// OnPremName is the module name in the on-prem system
	OnPremName string
	// SpackPackage is the equivalent Spack package specification
	SpackPackage string
	// Confidence is the mapping confidence (0.0 to 1.0)
	Confidence float64
	// Notes contains additional information about the mapping
	Notes string
}

// ModuleDatabase stores mappings from on-prem modules to Spack packages.
type ModuleDatabase struct {
	mappings map[string]*ModuleMapping
}

// NewModuleDatabase creates a new module database with default mappings.
func NewModuleDatabase() *ModuleDatabase {
	db := &ModuleDatabase{
		mappings: make(map[string]*ModuleMapping),
	}
	db.loadDefaultMappings()
	return db
}

// loadDefaultMappings loads common module to Spack package mappings.
func (db *ModuleDatabase) loadDefaultMappings() {
	defaults := map[string]*ModuleMapping{
		// Compilers
		"gcc":   {OnPremName: "gcc", SpackPackage: "gcc@11.3.0", Confidence: 1.0},
		"intel": {OnPremName: "intel", SpackPackage: "intel-oneapi-compilers@2023.1.0", Confidence: 0.9},
		"llvm":  {OnPremName: "llvm", SpackPackage: "llvm@15.0.0", Confidence: 1.0},

		// MPI
		"openmpi":  {OnPremName: "openmpi", SpackPackage: "openmpi@4.1.4", Confidence: 1.0},
		"mpich":    {OnPremName: "mpich", SpackPackage: "mpich@4.0", Confidence: 1.0},
		"mvapich2": {OnPremName: "mvapich2", SpackPackage: "mvapich2@2.3.7", Confidence: 1.0},
		"intelmpi": {OnPremName: "intelmpi", SpackPackage: "intel-oneapi-mpi@2021.9.0", Confidence: 0.9},

		// Languages
		"python": {OnPremName: "python", SpackPackage: "python@3.10", Confidence: 1.0},
		"r":      {OnPremName: "r", SpackPackage: "r@4.2.0", Confidence: 1.0},
		"julia":  {OnPremName: "julia", SpackPackage: "julia@1.9.0", Confidence: 1.0},
		"perl":   {OnPremName: "perl", SpackPackage: "perl@5.36.0", Confidence: 1.0},

		// Bioinformatics
		"samtools": {OnPremName: "samtools", SpackPackage: "samtools@1.17", Confidence: 1.0},
		"bwa":      {OnPremName: "bwa", SpackPackage: "bwa@0.7.17", Confidence: 1.0},
		"gatk":     {OnPremName: "gatk", SpackPackage: "gatk@4.3.0", Confidence: 1.0},
		"blast":    {OnPremName: "blast", SpackPackage: "blast-plus@2.14.0", Confidence: 1.0},
		"bowtie2":  {OnPremName: "bowtie2", SpackPackage: "bowtie2@2.4.5", Confidence: 1.0},
		"bedtools": {OnPremName: "bedtools", SpackPackage: "bedtools2@2.30.0", Confidence: 1.0},

		// Computational Chemistry
		"gromacs":          {OnPremName: "gromacs", SpackPackage: "gromacs@2023.1", Confidence: 1.0},
		"lammps":           {OnPremName: "lammps", SpackPackage: "lammps@20230802", Confidence: 1.0},
		"quantum-espresso": {OnPremName: "quantum-espresso", SpackPackage: "quantum-espresso@7.2", Confidence: 1.0},
		"namd":             {OnPremName: "namd", SpackPackage: "namd@2.14", Confidence: 1.0},

		// Machine Learning
		"pytorch":    {OnPremName: "pytorch", SpackPackage: "py-torch@2.0.0", Confidence: 1.0},
		"tensorflow": {OnPremName: "tensorflow", SpackPackage: "py-tensorflow@2.12.0", Confidence: 1.0},
		"cuda":       {OnPremName: "cuda", SpackPackage: "cuda@11.8.0", Confidence: 1.0},
		"cudnn":      {OnPremName: "cudnn", SpackPackage: "cudnn@8.9.0", Confidence: 1.0},

		// Math Libraries
		"fftw":   {OnPremName: "fftw", SpackPackage: "fftw@3.3.10", Confidence: 1.0},
		"blas":   {OnPremName: "blas", SpackPackage: "openblas@0.3.23", Confidence: 0.9},
		"lapack": {OnPremName: "lapack", SpackPackage: "openblas@0.3.23", Confidence: 0.9},
		"hdf5":   {OnPremName: "hdf5", SpackPackage: "hdf5@1.14.0", Confidence: 1.0},
		"netcdf": {OnPremName: "netcdf", SpackPackage: "netcdf-c@4.9.2", Confidence: 1.0},

		// Build Tools
		"cmake":    {OnPremName: "cmake", SpackPackage: "cmake@3.26.0", Confidence: 1.0},
		"autoconf": {OnPremName: "autoconf", SpackPackage: "autoconf@2.71", Confidence: 1.0},
		"automake": {OnPremName: "automake", SpackPackage: "automake@1.16.5", Confidence: 1.0},
	}

	for k, v := range defaults {
		db.mappings[k] = v
	}
}

// Lookup finds a Spack package for an on-prem module name.
func (db *ModuleDatabase) Lookup(moduleName string) (*ModuleMapping, bool) {
	// Normalize module name (lowercase, remove version)
	normalized := normalizeModuleName(moduleName)

	if mapping, ok := db.mappings[normalized]; ok {
		return mapping, true
	}

	return nil, false
}

// AddMapping adds a custom mapping to the database.
func (db *ModuleDatabase) AddMapping(onPremName, spackPackage string, confidence float64) {
	normalized := normalizeModuleName(onPremName)
	db.mappings[normalized] = &ModuleMapping{
		OnPremName:   onPremName,
		SpackPackage: spackPackage,
		Confidence:   confidence,
	}
}

// ConvertModules converts a list of on-prem module names to Spack packages.
func (db *ModuleDatabase) ConvertModules(modules []string) ([]string, []string) {
	var spackPackages []string
	var unmapped []string

	for _, mod := range modules {
		if mapping, ok := db.Lookup(mod); ok {
			spackPackages = append(spackPackages, mapping.SpackPackage)
		} else {
			unmapped = append(unmapped, mod)
		}
	}

	return spackPackages, unmapped
}

// normalizeModuleName normalizes a module name for lookup.
// Examples:
//   - "gcc/11.2.0" -> "gcc"
//   - "openmpi-4.1.1" -> "openmpi"
//   - "Python/3.9.5" -> "python"
func normalizeModuleName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove version patterns
	// Pattern: /version or -version or _version
	versionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`/\d+\.\d+.*$`),     // /1.2.3
		regexp.MustCompile(`-\d+\.\d+.*$`),     // -1.2.3
		regexp.MustCompile(`_\d+\.\d+.*$`),     // _1.2.3
		regexp.MustCompile(`\d+\.\d+\.\d+.*$`), // 1.2.3 at end
	}

	for _, pattern := range versionPatterns {
		name = pattern.ReplaceAllString(name, "")
	}

	// Remove common suffixes
	name = strings.TrimSuffix(name, "-mpi")
	name = strings.TrimSuffix(name, "_mpi")

	return strings.TrimSpace(name)
}

// ParseModuleList parses output from 'module list' command.
func ParseModuleList(output string) []string {
	var modules []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and headers
		if line == "" || strings.HasPrefix(line, "Currently") ||
			strings.HasPrefix(line, "No modules") {
			continue
		}

		// Split by whitespace and extract module names
		fields := strings.Fields(line)
		for _, field := range fields {
			// Skip numeric indices (1), 2), etc.)
			if strings.HasSuffix(field, ")") {
				continue
			}
			// Skip paths
			if strings.Contains(field, "/") && !strings.Contains(field, "modulefiles") {
				modules = append(modules, field)
			}
		}
	}

	return modules
}

// SuggestAlternatives suggests alternative Spack packages for unmapped modules.
func (db *ModuleDatabase) SuggestAlternatives(moduleName string) []string {
	normalized := normalizeModuleName(moduleName)
	var suggestions []string

	// Search for similar names in the database
	for key, mapping := range db.mappings {
		// Check if the key contains the search term or vice versa
		if strings.Contains(key, normalized) || strings.Contains(normalized, key) {
			suggestions = append(suggestions, fmt.Sprintf("%s -> %s (confidence: %.0f%%)",
				mapping.OnPremName, mapping.SpackPackage, mapping.Confidence*100))
		}
	}

	return suggestions
}
