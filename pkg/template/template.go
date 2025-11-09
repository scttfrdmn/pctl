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

// Package template provides template parsing and validation.
package template

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Template represents a pctl cluster template.
type Template struct {
	Cluster  ClusterConfig  `yaml:"cluster"`
	Compute  ComputeConfig  `yaml:"compute"`
	Software SoftwareConfig `yaml:"software,omitempty"`
	Users    []User         `yaml:"users,omitempty"`
	Data     DataConfig     `yaml:"data,omitempty"`
}

// ClusterConfig holds cluster-level configuration.
type ClusterConfig struct {
	Name   string `yaml:"name"`
	Region string `yaml:"region"`
}

// ComputeConfig holds compute resource configuration.
type ComputeConfig struct {
	HeadNode string  `yaml:"head_node"`
	Queues   []Queue `yaml:"queues"`
}

// Queue represents a compute queue configuration.
type Queue struct {
	Name          string   `yaml:"name"`
	InstanceTypes []string `yaml:"instance_types"`
	MinCount      int      `yaml:"min_count"`
	MaxCount      int      `yaml:"max_count"`
}

// SoftwareConfig holds software installation configuration.
type SoftwareConfig struct {
	SpackPackages []string `yaml:"spack_packages,omitempty"`
}

// User represents a cluster user.
type User struct {
	Name string `yaml:"name"`
	UID  int    `yaml:"uid"`
	GID  int    `yaml:"gid"`
}

// DataConfig holds data source configuration.
type DataConfig struct {
	S3Mounts []S3Mount `yaml:"s3_mounts,omitempty"`
}

// S3Mount represents an S3 bucket mount.
type S3Mount struct {
	Bucket     string `yaml:"bucket"`
	MountPoint string `yaml:"mount_point"`
}

// Load loads a template from a file.
func Load(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &tmpl, nil
}

// Validate validates the template.
func (t *Template) Validate() error {
	if t.Cluster.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	if t.Cluster.Region == "" {
		return fmt.Errorf("cluster region is required")
	}
	if t.Compute.HeadNode == "" {
		return fmt.Errorf("head node instance type is required")
	}
	if len(t.Compute.Queues) == 0 {
		return fmt.Errorf("at least one compute queue is required")
	}

	for i, queue := range t.Compute.Queues {
		if queue.Name == "" {
			return fmt.Errorf("queue %d: name is required", i)
		}
		if len(queue.InstanceTypes) == 0 {
			return fmt.Errorf("queue %s: at least one instance type is required", queue.Name)
		}
		if queue.MaxCount < queue.MinCount {
			return fmt.Errorf("queue %s: max_count must be >= min_count", queue.Name)
		}
	}

	return nil
}
