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

// Package config provides ParallelCluster configuration generation.
package config

import (
	"fmt"

	"github.com/scttfrdmn/pctl/pkg/software"
	"github.com/scttfrdmn/pctl/pkg/template"
	"gopkg.in/yaml.v3"
)

// Generator generates ParallelCluster configurations from pctl templates.
type Generator struct {
	// KeyName is the EC2 key pair name for SSH access
	KeyName string
	// SubnetID is the subnet ID for the cluster (if not auto-creating VPC)
	SubnetID string
	// CustomAMI is a custom AMI ID to use instead of default
	CustomAMI string
	// BootstrapScriptS3URI is the S3 URI for the bootstrap script
	BootstrapScriptS3URI string
}

// NewGenerator creates a new config generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate generates a ParallelCluster configuration from a pctl template.
func (g *Generator) Generate(tmpl *template.Template) (string, error) {
	pcConfig := g.buildParallelClusterConfig(tmpl)

	// Marshal to YAML
	data, err := yaml.Marshal(pcConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(data), nil
}

func (g *Generator) buildParallelClusterConfig(tmpl *template.Template) map[string]interface{} {
	config := map[string]interface{}{
		"Region": tmpl.Cluster.Region,
		"Image": map[string]interface{}{
			"Os": "alinux2",
		},
	}

	// Add custom AMI if specified
	if g.CustomAMI != "" {
		config["Image"].(map[string]interface{})["CustomAmi"] = g.CustomAMI
	}

	// Head node configuration
	headNode := map[string]interface{}{
		"InstanceType": tmpl.Compute.HeadNode,
		"Networking": map[string]interface{}{
			"SubnetId": g.SubnetID,
		},
		"Ssh": map[string]interface{}{
			"KeyName": g.KeyName,
		},
	}

	// Add Iam configuration for S3 access if there are S3 mounts or bootstrap script
	if len(tmpl.Data.S3Mounts) > 0 || g.BootstrapScriptS3URI != "" {
		headNode["Iam"] = map[string]interface{}{
			"AdditionalIamPolicies": []map[string]interface{}{
				{
					"Policy": "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
				},
			},
		}
	}

	config["HeadNode"] = headNode

	// Scheduling configuration
	scheduling := map[string]interface{}{
		"Scheduler": "slurm",
	}

	// Build compute queues
	var queues []map[string]interface{}
	for _, queue := range tmpl.Compute.Queues {
		pcQueue := map[string]interface{}{
			"Name": queue.Name,
			"ComputeResources": []map[string]interface{}{
				{
					"Name":                              queue.Name + "-nodes",
					"InstanceType":                      queue.InstanceTypes[0], // Use first instance type
					"MinCount":                          queue.MinCount,
					"MaxCount":                          queue.MaxCount,
					"DisableSimultaneousMultithreading": false,
				},
			},
			"Networking": map[string]interface{}{
				"SubnetIds": []string{g.SubnetID},
			},
		}

		// Add multiple instance types if specified
		if len(queue.InstanceTypes) > 1 {
			computeResources := []map[string]interface{}{}
			for i, instanceType := range queue.InstanceTypes {
				computeResources = append(computeResources, map[string]interface{}{
					"Name":                              fmt.Sprintf("%s-nodes-%d", queue.Name, i),
					"InstanceType":                      instanceType,
					"MinCount":                          queue.MinCount / len(queue.InstanceTypes),
					"MaxCount":                          queue.MaxCount / len(queue.InstanceTypes),
					"DisableSimultaneousMultithreading": false,
				})
			}
			pcQueue["ComputeResources"] = computeResources
		}

		// Add IAM for S3 access if needed for S3 mounts or bootstrap script
		if len(tmpl.Data.S3Mounts) > 0 || g.BootstrapScriptS3URI != "" {
			pcQueue["Iam"] = map[string]interface{}{
				"AdditionalIamPolicies": []map[string]interface{}{
					{
						"Policy": "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
					},
				},
			}
		}

		queues = append(queues, pcQueue)
	}

	scheduling["SlurmQueues"] = queues
	config["Scheduling"] = scheduling

	// Shared storage configuration
	if len(tmpl.Data.S3Mounts) > 0 {
		var sharedStorage []map[string]interface{}

		// Add shared EBS for home directories
		sharedStorage = append(sharedStorage, map[string]interface{}{
			"MountDir":    "/shared",
			"Name":        "shared-ebs",
			"StorageType": "Ebs",
			"EbsSettings": map[string]interface{}{
				"VolumeType": "gp3",
				"Size":       100, // 100GB
			},
		})

		config["SharedStorage"] = sharedStorage
	}

	// Custom bootstrap actions for software installation and user creation
	if g.BootstrapScriptS3URI != "" {
		config["HeadNode"].(map[string]interface{})["CustomActions"] = map[string]interface{}{
			"OnNodeConfigured": map[string]interface{}{
				"Script": g.BootstrapScriptS3URI,
			},
		}
	}

	return config
}

// GenerateBootstrapScript generates a bootstrap script for software installation and user setup.
// This now delegates to the software.Manager for a more robust implementation.
func (g *Generator) GenerateBootstrapScript(tmpl *template.Template) string {
	manager := software.NewManager()
	return manager.GenerateBootstrapScript(tmpl, true, true)
}
