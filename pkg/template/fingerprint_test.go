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
	"testing"
)

func TestComputeFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		template *Template
		wantOS   string
		wantSpack string
		wantLmod  string
		wantPkgs  int
	}{
		{
			name: "empty software",
			template: &Template{
				Software: SoftwareConfig{
					SpackPackages: []string{},
				},
			},
			wantOS:    "amazonlinux2",
			wantSpack: "releases/latest",
			wantLmod:  "8.7.37",
			wantPkgs:  0,
		},
		{
			name: "single package",
			template: &Template{
				Software: SoftwareConfig{
					SpackPackages: []string{"gcc@11.3.0"},
				},
			},
			wantOS:    "amazonlinux2",
			wantSpack: "releases/latest",
			wantLmod:  "8.7.37",
			wantPkgs:  1,
		},
		{
			name: "multiple packages",
			template: &Template{
				Software: SoftwareConfig{
					SpackPackages: []string{
						"gcc@11.3.0",
						"openmpi@4.1.4",
						"python@3.10",
					},
				},
			},
			wantOS:    "amazonlinux2",
			wantSpack: "releases/latest",
			wantLmod:  "8.7.37",
			wantPkgs:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := tt.template.ComputeFingerprint()

			if fp.BaseOS != tt.wantOS {
				t.Errorf("BaseOS = %v, want %v", fp.BaseOS, tt.wantOS)
			}
			if fp.SpackVersion != tt.wantSpack {
				t.Errorf("SpackVersion = %v, want %v", fp.SpackVersion, tt.wantSpack)
			}
			if fp.LmodVersion != tt.wantLmod {
				t.Errorf("LmodVersion = %v, want %v", fp.LmodVersion, tt.wantLmod)
			}
			if len(fp.Packages) != tt.wantPkgs {
				t.Errorf("Packages count = %v, want %v", len(fp.Packages), tt.wantPkgs)
			}
			if fp.Hash == "" {
				t.Error("Hash is empty")
			}
			if len(fp.Hash) != 64 { // SHA256 hex is 64 chars
				t.Errorf("Hash length = %v, want 64", len(fp.Hash))
			}
		})
	}
}

func TestFingerprintConsistency(t *testing.T) {
	template := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{
				"gcc@11.3.0",
				"python@3.10",
				"openmpi@4.1.4",
			},
		},
	}

	// Compute fingerprint twice
	fp1 := template.ComputeFingerprint()
	fp2 := template.ComputeFingerprint()

	// Should produce identical hashes
	if fp1.Hash != fp2.Hash {
		t.Errorf("Fingerprints not consistent: %v != %v", fp1.Hash, fp2.Hash)
	}
}

func TestFingerprintPackageOrdering(t *testing.T) {
	// Templates with same packages in different order should produce same hash
	template1 := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{
				"gcc@11.3.0",
				"python@3.10",
				"openmpi@4.1.4",
			},
		},
	}

	template2 := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{
				"python@3.10",
				"gcc@11.3.0",
				"openmpi@4.1.4",
			},
		},
	}

	fp1 := template1.ComputeFingerprint()
	fp2 := template2.ComputeFingerprint()

	if fp1.Hash != fp2.Hash {
		t.Errorf("Package ordering affects hash: %v != %v", fp1.Hash, fp2.Hash)
	}

	if !fp1.Matches(fp2) {
		t.Error("Fingerprints should match")
	}
}

func TestFingerprintDifferences(t *testing.T) {
	// Different packages should produce different hashes
	template1 := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{"gcc@11.3.0"},
		},
	}

	template2 := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{"gcc@12.2.0"},
		},
	}

	fp1 := template1.ComputeFingerprint()
	fp2 := template2.ComputeFingerprint()

	if fp1.Hash == fp2.Hash {
		t.Error("Different packages produced same hash")
	}

	if fp1.Matches(fp2) {
		t.Error("Fingerprints should not match")
	}
}

func TestFingerprintString(t *testing.T) {
	template := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{
				"gcc@11.3.0",
				"openmpi@4.1.4",
			},
		},
	}

	fp := template.ComputeFingerprint()
	str := fp.String()

	// Should start with "al2-spack-latest-lmod-8.7.37-"
	expectedPrefix := "al2-spack-latest-lmod-8.7.37-"
	if len(str) < len(expectedPrefix) {
		t.Errorf("String too short: %v", str)
	}

	prefix := str[:len(expectedPrefix)]
	if prefix != expectedPrefix {
		t.Errorf("String prefix = %v, want %v", prefix, expectedPrefix)
	}

	// Should end with 8-character hash
	hashPart := str[len(expectedPrefix):]
	if len(hashPart) != 8 {
		t.Errorf("Hash part length = %v, want 8", len(hashPart))
	}
}

func TestFingerprintTags(t *testing.T) {
	template := &Template{
		Software: SoftwareConfig{
			SpackPackages: []string{
				"gcc@11.3.0",
				"openmpi@4.1.4",
				"python@3.10",
			},
		},
	}

	fp := template.ComputeFingerprint()
	tags := fp.Tags()

	// Check required tags
	if tags["pctl:fingerprint"] != fp.Hash {
		t.Error("Missing or incorrect pctl:fingerprint tag")
	}
	if tags["pctl:base-os"] != "amazonlinux2" {
		t.Error("Missing or incorrect pctl:base-os tag")
	}
	if tags["pctl:spack-version"] != "releases/latest" {
		t.Error("Missing or incorrect pctl:spack-version tag")
	}
	if tags["pctl:lmod-version"] != "8.7.37" {
		t.Error("Missing or incorrect pctl:lmod-version tag")
	}
	if tags["pctl:created-by"] != "pctl" {
		t.Error("Missing or incorrect pctl:created-by tag")
	}
	if tags["pctl:package-count"] != "3" {
		t.Errorf("Package count = %v, want 3", tags["pctl:package-count"])
	}

	// Check package tags (should have first 3 packages)
	if tags["pctl:package-1"] == "" {
		t.Error("Missing pctl:package-1 tag")
	}
	if tags["pctl:package-2"] == "" {
		t.Error("Missing pctl:package-2 tag")
	}
	if tags["pctl:package-3"] == "" {
		t.Error("Missing pctl:package-3 tag")
	}
}

func TestFingerprintTagKey(t *testing.T) {
	fp := &AMIFingerprint{}
	if fp.TagKey() != "pctl:fingerprint" {
		t.Errorf("TagKey() = %v, want pctl:fingerprint", fp.TagKey())
	}
}

func TestFingerprintTagValue(t *testing.T) {
	fp := &AMIFingerprint{
		Hash: "abc123",
	}
	if fp.TagValue() != "abc123" {
		t.Errorf("TagValue() = %v, want abc123", fp.TagValue())
	}
}
