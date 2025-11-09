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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError represents a collection of validation errors.
type ValidationError struct {
	Errors []string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return fmt.Sprintf("%d validation errors:\n  - %s", len(e.Errors), strings.Join(e.Errors, "\n  - "))
}

// Add adds an error message to the validation error.
func (e *ValidationError) Add(msg string) {
	e.Errors = append(e.Errors, msg)
}

// HasErrors returns true if there are validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validator provides comprehensive template validation.
type Validator struct {
	// ValidRegions is a list of valid AWS regions
	ValidRegions map[string]bool
	// ValidInstanceTypes is a list of valid EC2 instance types (patterns)
	ValidInstanceTypes []*regexp.Regexp
}

// NewValidator creates a new validator with default rules.
func NewValidator() *Validator {
	return &Validator{
		ValidRegions: map[string]bool{
			"us-east-1":      true,
			"us-east-2":      true,
			"us-west-1":      true,
			"us-west-2":      true,
			"eu-west-1":      true,
			"eu-west-2":      true,
			"eu-west-3":      true,
			"eu-central-1":   true,
			"eu-north-1":     true,
			"ap-northeast-1": true,
			"ap-northeast-2": true,
			"ap-southeast-1": true,
			"ap-southeast-2": true,
			"ap-south-1":     true,
			"sa-east-1":      true,
			"ca-central-1":   true,
		},
		ValidInstanceTypes: []*regexp.Regexp{
			regexp.MustCompile(`^[a-z][0-9][a-z0-9]*\.[a-z0-9]+$`), // e.g., t3.medium, c5.xlarge, g4dn.xlarge
		},
	}
}

// ValidateTemplate performs comprehensive validation on a template.
func (v *Validator) ValidateTemplate(t *Template) error {
	errs := &ValidationError{}

	v.validateCluster(t, errs)
	v.validateCompute(t, errs)
	v.validateSoftware(t, errs)
	v.validateUsers(t, errs)
	v.validateData(t, errs)

	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (v *Validator) validateCluster(t *Template, errs *ValidationError) {
	// Name validation
	if t.Cluster.Name == "" {
		errs.Add("cluster.name is required")
	} else if len(t.Cluster.Name) > 60 {
		errs.Add("cluster.name must be 60 characters or less")
	} else if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`).MatchString(t.Cluster.Name) {
		errs.Add("cluster.name must start with a letter and contain only alphanumeric characters and hyphens")
	}

	// Region validation
	if t.Cluster.Region == "" {
		errs.Add("cluster.region is required")
	} else if !v.ValidRegions[t.Cluster.Region] {
		errs.Add(fmt.Sprintf("cluster.region '%s' is not a valid AWS region", t.Cluster.Region))
	}
}

func (v *Validator) validateCompute(t *Template, errs *ValidationError) {
	// Head node validation
	if t.Compute.HeadNode == "" {
		errs.Add("compute.head_node is required")
	} else if !v.isValidInstanceType(t.Compute.HeadNode) {
		errs.Add(fmt.Sprintf("compute.head_node '%s' is not a valid instance type format", t.Compute.HeadNode))
	}

	// Queues validation
	if len(t.Compute.Queues) == 0 {
		errs.Add("compute.queues must have at least one queue")
	}

	queueNames := make(map[string]bool)
	for i, queue := range t.Compute.Queues {
		// Queue name validation
		if queue.Name == "" {
			errs.Add(fmt.Sprintf("compute.queues[%d].name is required", i))
		} else {
			if queueNames[queue.Name] {
				errs.Add(fmt.Sprintf("compute.queues[%d].name '%s' is duplicate", i, queue.Name))
			}
			queueNames[queue.Name] = true

			if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(queue.Name) {
				errs.Add(fmt.Sprintf("compute.queues[%d].name '%s' must start with lowercase letter and contain only lowercase letters, numbers, and hyphens", i, queue.Name))
			}
		}

		// Instance types validation
		if len(queue.InstanceTypes) == 0 {
			errs.Add(fmt.Sprintf("compute.queues[%d].instance_types must have at least one instance type", i))
		} else {
			for j, instanceType := range queue.InstanceTypes {
				if !v.isValidInstanceType(instanceType) {
					errs.Add(fmt.Sprintf("compute.queues[%d].instance_types[%d] '%s' is not a valid instance type format", i, j, instanceType))
				}
			}
		}

		// Count validation
		if queue.MinCount < 0 {
			errs.Add(fmt.Sprintf("compute.queues[%d].min_count must be >= 0", i))
		}
		if queue.MaxCount < 0 {
			errs.Add(fmt.Sprintf("compute.queues[%d].max_count must be >= 0", i))
		}
		if queue.MaxCount < queue.MinCount {
			errs.Add(fmt.Sprintf("compute.queues[%d].max_count (%d) must be >= min_count (%d)", i, queue.MaxCount, queue.MinCount))
		}
		if queue.MaxCount > 1000 {
			errs.Add(fmt.Sprintf("compute.queues[%d].max_count (%d) exceeds maximum of 1000", i, queue.MaxCount))
		}
	}
}

func (v *Validator) validateSoftware(t *Template, errs *ValidationError) {
	if len(t.Software.SpackPackages) > 0 {
		for i, pkg := range t.Software.SpackPackages {
			if pkg == "" {
				errs.Add(fmt.Sprintf("software.spack_packages[%d] cannot be empty", i))
			}
			// Basic validation of package spec format (name@version or name)
			if !regexp.MustCompile(`^[a-zA-Z0-9_-]+(@[a-zA-Z0-9._-]+)?$`).MatchString(pkg) {
				errs.Add(fmt.Sprintf("software.spack_packages[%d] '%s' is not a valid package spec format", i, pkg))
			}
		}
	}
}

func (v *Validator) validateUsers(t *Template, errs *ValidationError) {
	if len(t.Users) > 0 {
		userNames := make(map[string]bool)
		uids := make(map[int]bool)

		for i, user := range t.Users {
			// Name validation
			if user.Name == "" {
				errs.Add(fmt.Sprintf("users[%d].name is required", i))
			} else {
				if userNames[user.Name] {
					errs.Add(fmt.Sprintf("users[%d].name '%s' is duplicate", i, user.Name))
				}
				userNames[user.Name] = true

				if !regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`).MatchString(user.Name) {
					errs.Add(fmt.Sprintf("users[%d].name '%s' must start with lowercase letter or underscore and contain only lowercase letters, numbers, underscores, and hyphens", i, user.Name))
				}
			}

			// UID validation
			if user.UID <= 0 {
				errs.Add(fmt.Sprintf("users[%d].uid must be > 0", i))
			} else if user.UID < 1000 {
				errs.Add(fmt.Sprintf("users[%d].uid %d is in system range (< 1000), recommend using 1000 or higher", i, user.UID))
			} else if user.UID > 60000 {
				errs.Add(fmt.Sprintf("users[%d].uid %d exceeds recommended maximum of 60000", i, user.UID))
			}
			if uids[user.UID] {
				errs.Add(fmt.Sprintf("users[%d].uid %d is duplicate", i, user.UID))
			}
			uids[user.UID] = true

			// GID validation
			if user.GID <= 0 {
				errs.Add(fmt.Sprintf("users[%d].gid must be > 0", i))
			} else if user.GID < 1000 {
				errs.Add(fmt.Sprintf("users[%d].gid %d is in system range (< 1000), recommend using 1000 or higher", i, user.GID))
			} else if user.GID > 60000 {
				errs.Add(fmt.Sprintf("users[%d].gid %d exceeds recommended maximum of 60000", i, user.GID))
			}
		}
	}
}

func (v *Validator) validateData(t *Template, errs *ValidationError) {
	if len(t.Data.S3Mounts) > 0 {
		mountPoints := make(map[string]bool)

		for i, mount := range t.Data.S3Mounts {
			// Bucket validation
			if mount.Bucket == "" {
				errs.Add(fmt.Sprintf("data.s3_mounts[%d].bucket is required", i))
			} else if !v.isValidS3Bucket(mount.Bucket) {
				errs.Add(fmt.Sprintf("data.s3_mounts[%d].bucket '%s' is not a valid S3 bucket name", i, mount.Bucket))
			}

			// Mount point validation
			if mount.MountPoint == "" {
				errs.Add(fmt.Sprintf("data.s3_mounts[%d].mount_point is required", i))
			} else {
				if !filepath.IsAbs(mount.MountPoint) {
					errs.Add(fmt.Sprintf("data.s3_mounts[%d].mount_point '%s' must be an absolute path", i, mount.MountPoint))
				}
				if mountPoints[mount.MountPoint] {
					errs.Add(fmt.Sprintf("data.s3_mounts[%d].mount_point '%s' is duplicate", i, mount.MountPoint))
				}
				mountPoints[mount.MountPoint] = true
			}
		}
	}
}

func (v *Validator) isValidInstanceType(instanceType string) bool {
	for _, pattern := range v.ValidInstanceTypes {
		if pattern.MatchString(instanceType) {
			return true
		}
	}
	return false
}

func (v *Validator) isValidS3Bucket(bucket string) bool {
	// S3 bucket naming rules:
	// - 3-63 characters
	// - lowercase letters, numbers, hyphens, dots
	// - must start and end with letter or number
	// - no consecutive dots
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}
	if strings.Contains(bucket, "..") {
		return false
	}
	return regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`).MatchString(bucket)
}
