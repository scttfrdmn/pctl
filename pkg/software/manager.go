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

package software

import (
	"fmt"
	"strings"

	"github.com/scttfrdmn/pctl/pkg/template"
)

// Manager coordinates Spack and Lmod installation and configuration.
type Manager struct {
	spackInstaller *SpackInstaller
	lmodInstaller  *LmodInstaller
}

// NewManager creates a new software manager.
func NewManager() *Manager {
	spackConfig := DefaultSpackConfig()
	lmodConfig := DefaultLmodConfig()

	return &Manager{
		spackInstaller: NewSpackInstaller(spackConfig),
		lmodInstaller:  NewLmodInstaller(lmodConfig),
	}
}

// GenerateBootstrapScript generates a complete bootstrap script for software installation.
// This replaces the old bootstrap script generation in pkg/config/generator.go
func (m *Manager) GenerateBootstrapScript(tmpl *template.Template, includeUsers, includeS3Mounts bool) string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# pctl Bootstrap Script\n")
	script.WriteString(fmt.Sprintf("# Generated for cluster: %s\n", tmpl.Cluster.Name))
	script.WriteString("# Region: " + tmpl.Cluster.Region + "\n\n")

	script.WriteString("# Enable detailed logging\n")
	script.WriteString("exec 1> >(logger -s -t pctl-bootstrap) 2>&1\n")
	script.WriteString("echo \"Starting pctl bootstrap at $(date)\"\n\n")

	// Add progress tagging helper function
	script.WriteString("# Helper function to update progress tag\n")
	script.WriteString("update_progress_tag() {\n")
	script.WriteString("  local message=\"$1\"\n")
	script.WriteString("  local percent=\"$2\"\n")
	script.WriteString("  \n")
	script.WriteString("  # Get instance ID and region from metadata\n")
	script.WriteString("  INSTANCE_ID=$(ec2-metadata --instance-id | cut -d ' ' -f 2)\n")
	script.WriteString("  REGION=$(ec2-metadata --availability-zone | cut -d ' ' -f 2 | sed 's/[a-z]$//')\n")
	script.WriteString("  \n")
	script.WriteString("  # Update tag (don't fail build if tagging fails)\n")
	script.WriteString("  aws ec2 create-tags --resources \"$INSTANCE_ID\" --region \"$REGION\" \\\n")
	script.WriteString("    --tags \"Key=pctl-progress,Value=${percent}% - ${message}\" 2>/dev/null || \\\n")
	script.WriteString("    echo \"Warning: Failed to update progress tag\"\n")
	script.WriteString("  \n")
	script.WriteString("  # Also echo for console output\n")
	script.WriteString("  echo \"PCTL_PROGRESS: ${message} (${percent}%)\"\n")
	script.WriteString("}\n\n")

	script.WriteString("# Initialize progress\n")
	script.WriteString("update_progress_tag \"Bootstrap started\" 0\n\n")

	// User creation
	if includeUsers && len(tmpl.Users) > 0 {
		script.WriteString("#" + strings.Repeat("=", 78) + "\n")
		script.WriteString("# USER CREATION\n")
		script.WriteString("#" + strings.Repeat("=", 78) + "\n\n")
		script.WriteString("echo \"Creating users...\"\n")
		for _, user := range tmpl.Users {
			script.WriteString(fmt.Sprintf("groupadd -g %d %s 2>/dev/null || echo \"Group %s already exists\"\n",
				user.GID, user.Name, user.Name))
			script.WriteString(fmt.Sprintf("useradd -u %d -g %d -m -s /bin/bash %s 2>/dev/null || echo \"User %s already exists\"\n",
				user.UID, user.GID, user.Name, user.Name))
		}
		script.WriteString("echo \"User creation complete\"\n\n")
	}

	// S3 mount setup
	if includeS3Mounts && len(tmpl.Data.S3Mounts) > 0 {
		script.WriteString("#" + strings.Repeat("=", 78) + "\n")
		script.WriteString("# S3 MOUNT CONFIGURATION\n")
		script.WriteString("#" + strings.Repeat("=", 78) + "\n\n")
		script.WriteString("echo \"Setting up S3 mounts...\"\n")
		script.WriteString("yum install -y s3fs-fuse\n\n")
		for _, mount := range tmpl.Data.S3Mounts {
			script.WriteString(fmt.Sprintf("mkdir -p %s\n", mount.MountPoint))
			script.WriteString(fmt.Sprintf("s3fs %s %s -o iam_role=auto -o allow_other || echo \"Warning: Failed to mount %s\"\n",
				mount.Bucket, mount.MountPoint, mount.Bucket))
			script.WriteString(fmt.Sprintf("echo 's3fs#%s %s fuse _netdev,allow_other,iam_role=auto 0 0' >> /etc/fstab\n",
				mount.Bucket, mount.MountPoint))
		}
		script.WriteString("echo \"S3 mount setup complete\"\n\n")
	}

	// Software installation
	if len(tmpl.Software.SpackPackages) > 0 {
		script.WriteString("#" + strings.Repeat("=", 78) + "\n")
		script.WriteString("# SOFTWARE INSTALLATION\n")
		script.WriteString("#" + strings.Repeat("=", 78) + "\n\n")

		// Install Spack
		script.WriteString("update_progress_tag \"Installing Spack package manager\" 10\n")
		script.WriteString("# Install Spack\n")
		script.WriteString(m.spackInstaller.GenerateInstallScript())
		script.WriteString("\n")

		// Install Lmod
		script.WriteString("update_progress_tag \"Installing Lmod module system\" 15\n")
		script.WriteString("# Install Lmod\n")
		script.WriteString(m.lmodInstaller.GenerateInstallScript())
		script.WriteString("\n")

		// Install packages
		script.WriteString("update_progress_tag \"Starting package installation\" 20\n")
		script.WriteString("# Install Spack packages\n")
		script.WriteString(m.spackInstaller.GeneratePackageInstallScript(tmpl.Software.SpackPackages))
		script.WriteString("\n")

		// Integrate Spack with Lmod
		script.WriteString("update_progress_tag \"Integrating Spack with Lmod\" 85\n")
		script.WriteString("# Integrate Spack with Lmod\n")
		script.WriteString(m.lmodInstaller.GenerateSpackIntegrationScript())
		script.WriteString("\n")

		// Mark completion at 100%
		script.WriteString("update_progress_tag \"Finalizing installation\" 95\n")
		script.WriteString("echo \"Flushing data to disk...\"\n")
		script.WriteString("sync\n")
		script.WriteString("sleep 2\n")
		script.WriteString("sync\n\n")
	}

	script.WriteString("update_progress_tag \"Installation complete\" 100\n")
	script.WriteString("echo \"Bootstrap complete at $(date)\"\n")
	script.WriteString("echo \"Cluster is ready for use!\"\n")
	script.WriteString("sync\n") // Final sync to ensure all data is written

	return script.String()
}

// GenerateSoftwareOnlyScript generates a script that only installs software (no users/S3).
func (m *Manager) GenerateSoftwareOnlyScript(packages []string) string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# Software Installation Script\n")
	script.WriteString("# Generated by pctl\n\n")

	// Install Spack
	script.WriteString(m.spackInstaller.GenerateInstallScript())
	script.WriteString("\n")

	// Install Lmod
	script.WriteString(m.lmodInstaller.GenerateInstallScript())
	script.WriteString("\n")

	// Install packages
	if len(packages) > 0 {
		script.WriteString(m.spackInstaller.GeneratePackageInstallScript(packages))
		script.WriteString("\n")
	}

	// Integrate Spack with Lmod
	script.WriteString(m.lmodInstaller.GenerateSpackIntegrationScript())

	return script.String()
}
