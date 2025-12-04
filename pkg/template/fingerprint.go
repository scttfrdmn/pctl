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

package template

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// AMIFingerprint represents a unique identifier for an AMI based on software configuration.
type AMIFingerprint struct {
	// BaseOS is the operating system (e.g., "amazonlinux2023")
	BaseOS string
	// SpackVersion is the Spack version (e.g., "releases/latest")
	SpackVersion string
	// LmodVersion is the Lmod version (e.g., "8.7.37")
	LmodVersion string
	// Packages is the sorted list of Spack packages
	Packages []string
	// Hash is the computed SHA256 hash
	Hash string
}

// ComputeFingerprint generates a unique fingerprint for a template based on its software configuration.
// This fingerprint is used to identify whether an existing AMI can be reused.
func (t *Template) ComputeFingerprint() *AMIFingerprint {
	// Default versions from pkg/software
	const (
		defaultBaseOS       = "amazonlinux2023"
		defaultSpackVersion = "releases/latest"
		defaultLmodVersion  = "8.7.37"
	)

	// Sort packages for consistent ordering
	packages := make([]string, len(t.Software.SpackPackages))
	copy(packages, t.Software.SpackPackages)
	sort.Strings(packages)

	fp := &AMIFingerprint{
		BaseOS:       defaultBaseOS,
		SpackVersion: defaultSpackVersion,
		LmodVersion:  defaultLmodVersion,
		Packages:     packages,
	}

	// Compute hash
	fp.Hash = fp.computeHash()

	return fp
}

// computeHash generates a SHA256 hash of the fingerprint components.
func (fp *AMIFingerprint) computeHash() string {
	// Create a canonical representation
	parts := []string{
		fp.BaseOS,
		fp.SpackVersion,
		fp.LmodVersion,
		strings.Join(fp.Packages, "|"),
	}
	canonical := strings.Join(parts, ":")

	// Compute SHA256 hash
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}

// String returns a human-readable representation of the fingerprint.
// Format: al2-spack-latest-lmod-8.7.37-<short-hash>
func (fp *AMIFingerprint) String() string {
	// Abbreviate base OS
	osAbbrev := strings.ReplaceAll(fp.BaseOS, "amazonlinux", "al")

	// Abbreviate Spack version
	spackAbbrev := strings.ReplaceAll(fp.SpackVersion, "releases/", "")

	// Use first 8 chars of hash
	shortHash := fp.Hash[:8]

	return fmt.Sprintf("%s-spack-%s-lmod-%s-%s",
		osAbbrev, spackAbbrev, fp.LmodVersion, shortHash)
}

// TagKey returns the AWS tag key for the fingerprint hash.
func (fp *AMIFingerprint) TagKey() string {
	return "pctl:fingerprint"
}

// TagValue returns the AWS tag value (the full hash).
func (fp *AMIFingerprint) TagValue() string {
	return fp.Hash
}

// Tags returns a map of all tags to apply to the AMI.
func (fp *AMIFingerprint) Tags() map[string]string {
	tags := map[string]string{
		"pctl:fingerprint":   fp.Hash,
		"pctl:base-os":       fp.BaseOS,
		"pctl:spack-version": fp.SpackVersion,
		"pctl:lmod-version":  fp.LmodVersion,
		"pctl:created-by":    "pctl",
	}

	// Add package count
	if len(fp.Packages) > 0 {
		tags["pctl:package-count"] = fmt.Sprintf("%d", len(fp.Packages))
		// Include first few packages in tags for searchability
		maxPkgs := 5
		if len(fp.Packages) < maxPkgs {
			maxPkgs = len(fp.Packages)
		}
		for i := 0; i < maxPkgs; i++ {
			tags[fmt.Sprintf("pctl:package-%d", i+1)] = fp.Packages[i]
		}
	}

	return tags
}

// Matches checks if this fingerprint matches another fingerprint.
func (fp *AMIFingerprint) Matches(other *AMIFingerprint) bool {
	return fp.Hash == other.Hash
}
