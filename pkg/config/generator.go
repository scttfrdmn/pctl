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

	// Add Iam configuration for S3 access if there are S3 mounts
	if len(tmpl.Data.S3Mounts) > 0 {
		headNode["Iam"] = map[string]interface{}{
			"AdditionalIamPolicies": []map[string]interface{}{
				{
					"Policy": "arn:aws:iam::aws:policy/AmazonS3FullAccess",
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

		// Add IAM for S3 access
		if len(tmpl.Data.S3Mounts) > 0 {
			pcQueue["Iam"] = map[string]interface{}{
				"AdditionalIamPolicies": []map[string]interface{}{
					{
						"Policy": "arn:aws:iam::aws:policy/AmazonS3FullAccess",
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
	if len(tmpl.Software.SpackPackages) > 0 || len(tmpl.Users) > 0 || len(tmpl.Data.S3Mounts) > 0 {
		// We'll add bootstrap scripts here in a later commit
		// For now, document what needs to happen
		config["HeadNode"].(map[string]interface{})["CustomActions"] = map[string]interface{}{
			"OnNodeConfigured": map[string]interface{}{
				"Script": "s3://pctl-bootstrap/install-software.sh",
			},
		}
	}

	return config
}

// GenerateBootstrapScript generates a bootstrap script for software installation and user setup.
func (g *Generator) GenerateBootstrapScript(tmpl *template.Template) string {
	script := "#!/bin/bash\n"
	script += "set -e\n\n"
	script += "# pctl bootstrap script\n"
	script += "# Generated for cluster: " + tmpl.Cluster.Name + "\n\n"

	// User creation
	if len(tmpl.Users) > 0 {
		script += "# Create users\n"
		for _, user := range tmpl.Users {
			script += fmt.Sprintf("groupadd -g %d %s || true\n", user.GID, user.Name)
			script += fmt.Sprintf("useradd -u %d -g %d -m -s /bin/bash %s || true\n", user.UID, user.GID, user.Name)
		}
		script += "\n"
	}

	// S3 mount setup
	if len(tmpl.Data.S3Mounts) > 0 {
		script += "# Install s3fs for S3 mounts\n"
		script += "yum install -y s3fs-fuse\n\n"
		script += "# Mount S3 buckets\n"
		for _, mount := range tmpl.Data.S3Mounts {
			script += fmt.Sprintf("mkdir -p %s\n", mount.MountPoint)
			script += fmt.Sprintf("s3fs %s %s -o iam_role=auto -o allow_other\n", mount.Bucket, mount.MountPoint)
			script += fmt.Sprintf("echo 's3fs#%s %s fuse _netdev,allow_other,iam_role=auto 0 0' >> /etc/fstab\n", mount.Bucket, mount.MountPoint)
		}
		script += "\n"
	}

	// Spack installation
	if len(tmpl.Software.SpackPackages) > 0 {
		script += "# Install Spack\n"
		script += "cd /opt\n"
		script += "git clone -c feature.manyFiles=true https://github.com/spack/spack.git\n"
		script += "cd spack\n"
		script += "git checkout releases/latest\n"
		script += ". share/spack/setup-env.sh\n\n"

		script += "# Install packages\n"
		for _, pkg := range tmpl.Software.SpackPackages {
			script += fmt.Sprintf("spack install %s\n", pkg)
		}
		script += "\n"

		script += "# Install Lmod\n"
		script += "yum install -y lua lua-devel lua-filesystem lua-posix\n"
		script += "cd /opt\n"
		script += "wget https://github.com/TACC/Lmod/archive/8.7.tar.gz\n"
		script += "tar xzf 8.7.tar.gz\n"
		script += "cd Lmod-8.7\n"
		script += "./configure --prefix=/opt/apps\n"
		script += "make install\n\n"

		script += "# Generate Lmod modules\n"
		script += "cd /opt/spack\n"
		script += ". share/spack/setup-env.sh\n"
		script += "spack module lmod refresh -y\n\n"

		script += "# Setup Lmod for all users\n"
		script += "cat > /etc/profile.d/z00_lmod.sh << 'EOF'\n"
		script += "export MODULEPATH=/opt/spack/share/spack/lmod/linux-amzn2-x86_64/Core\n"
		script += "source /opt/apps/lmod/lmod/init/bash\n"
		script += "EOF\n"
	}

	return script
}
