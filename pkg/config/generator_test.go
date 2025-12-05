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

package config

import (
	"strings"
	"testing"

	"github.com/scttfrdmn/petal/pkg/template"
	"gopkg.in/yaml.v3"
)

func TestGenerateBasicConfig(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Compute: template.ComputeConfig{
			HeadNode: "t3.xlarge",
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

	gen := NewGenerator()
	gen.KeyName = "my-key"
	gen.SubnetID = "subnet-12345"

	config, err := gen.Generate(tmpl)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("Failed to parse generated config: %v", err)
	}

	// Verify basic structure
	if parsed["Region"] != "us-east-1" {
		t.Errorf("Expected Region=us-east-1, got %v", parsed["Region"])
	}

	headNode, ok := parsed["HeadNode"].(map[string]interface{})
	if !ok {
		t.Fatal("HeadNode not found or wrong type")
	}

	if headNode["InstanceType"] != "t3.xlarge" {
		t.Errorf("Expected HeadNode.InstanceType=t3.xlarge, got %v", headNode["InstanceType"])
	}

	// Verify scheduling configuration
	scheduling, ok := parsed["Scheduling"].(map[string]interface{})
	if !ok {
		t.Fatal("Scheduling not found or wrong type")
	}

	if scheduling["Scheduler"] != "slurm" {
		t.Errorf("Expected Scheduler=slurm, got %v", scheduling["Scheduler"])
	}
}

func TestGenerateBootstrapScript(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Compute: template.ComputeConfig{
			HeadNode: "t3.xlarge",
			Queues: []template.Queue{
				{
					Name:          "compute",
					InstanceTypes: []string{"c5.2xlarge"},
					MinCount:      0,
					MaxCount:      10,
				},
			},
		},
		Software: template.SoftwareConfig{
			SpackPackages: []string{"gcc@11.3.0", "openmpi@4.1.4"},
		},
		Users: []template.User{
			{Name: "user1", UID: 5001, GID: 5001},
		},
		Data: template.DataConfig{
			S3Mounts: []template.S3Mount{
				{Bucket: "my-bucket", MountPoint: "/shared/data"},
			},
		},
	}

	gen := NewGenerator()
	script := gen.GenerateBootstrapScript(tmpl)

	// Verify script contains expected sections
	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("Script missing shebang")
	}

	if !strings.Contains(script, "USER CREATION") {
		t.Error("Script missing user creation section")
	}

	if !strings.Contains(script, "useradd -u 5001") {
		t.Error("Script missing user1 creation")
	}

	if !strings.Contains(script, "SOFTWARE INSTALLATION") {
		t.Error("Script missing software installation section")
	}

	if !strings.Contains(script, "gcc@11.3.0") {
		t.Error("Script missing GCC package")
	}

	if !strings.Contains(script, "openmpi@4.1.4") {
		t.Error("Script missing OpenMPI package")
	}

	if !strings.Contains(script, "Lmod Installation") {
		t.Error("Script missing Lmod installation")
	}

	if !strings.Contains(script, "s3fs my-bucket /shared/data") {
		t.Error("Script missing S3 mount")
	}
}

func TestGenerateWithMultipleInstanceTypes(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Compute: template.ComputeConfig{
			HeadNode: "t3.xlarge",
			Queues: []template.Queue{
				{
					Name:          "compute",
					InstanceTypes: []string{"c5.2xlarge", "c5.4xlarge", "c5.9xlarge"},
					MinCount:      0,
					MaxCount:      30,
				},
			},
		},
	}

	gen := NewGenerator()
	gen.KeyName = "my-key"
	gen.SubnetID = "subnet-12345"

	config, err := gen.Generate(tmpl)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Parse and verify multiple compute resources were created
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("Failed to parse generated config: %v", err)
	}

	scheduling := parsed["Scheduling"].(map[string]interface{})
	queues := scheduling["SlurmQueues"].([]interface{})
	queue := queues[0].(map[string]interface{})
	computeResources := queue["ComputeResources"].([]interface{})

	if len(computeResources) != 3 {
		t.Errorf("Expected 3 compute resources, got %d", len(computeResources))
	}
}

func TestGenerateWithCustomAMI(t *testing.T) {
	tmpl := &template.Template{
		Cluster: template.ClusterConfig{
			Name:   "test-cluster",
			Region: "us-east-1",
		},
		Compute: template.ComputeConfig{
			HeadNode: "t3.xlarge",
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

	gen := NewGenerator()
	gen.KeyName = "my-key"
	gen.SubnetID = "subnet-12345"
	gen.CustomAMI = "ami-0123456789"

	config, err := gen.Generate(tmpl)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("Failed to parse generated config: %v", err)
	}

	image := parsed["Image"].(map[string]interface{})
	if image["CustomAmi"] != "ami-0123456789" {
		t.Errorf("Expected CustomAmi=ami-0123456789, got %v", image["CustomAmi"])
	}
}
