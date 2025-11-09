#!/bin/bash
# Copyright 2025 Scott Friedman
# Create initial issues for the project

set -e

echo "Creating initial issues for pctl project..."

# v0.1.0 - Foundation
gh issue create \
  --title "Implement template validation framework" \
  --body "Create comprehensive validation for template schemas including type checking, required fields, and semantic validation." \
  --label "type: feature,component: template,priority: high" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Add create command for cluster provisioning" \
  --body "Implement the 'pctl create' command to deploy clusters from templates." \
  --label "type: feature,component: cli,priority: high" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Add validate command for template checking" \
  --body "Implement the 'pctl validate' command to check templates before deployment." \
  --label "type: feature,component: cli,priority: medium" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Create example templates library" \
  --body "Create a library of example templates for common HPC workloads (bioinformatics, ML, chemistry)." \
  --label "type: documentation,component: template,priority: medium" \
  --milestone "v0.1.0 - Foundation"

# v0.2.0 - AWS Integration
gh issue create \
  --title "Implement AWS ParallelCluster CLI integration" \
  --body "Integrate with AWS ParallelCluster CLI for cluster creation and management." \
  --label "type: feature,component: provisioner,priority: critical" \
  --milestone "v0.2.0 - AWS Integration"

gh issue create \
  --title "Implement ParallelCluster config generation" \
  --body "Generate ParallelCluster configuration files from pctl templates." \
  --label "type: feature,component: provisioner,priority: high" \
  --milestone "v0.2.0 - AWS Integration"

gh issue create \
  --title "Add VPC and networking auto-creation" \
  --body "Implement automatic VPC and networking setup for new clusters." \
  --label "type: feature,component: provisioner,priority: medium" \
  --milestone "v0.2.0 - AWS Integration"

gh issue create \
  --title "Implement cluster state management" \
  --body "Track cluster state locally for management and cleanup." \
  --label "type: feature,component: provisioner,priority: high" \
  --milestone "v0.2.0 - AWS Integration"

# v0.3.0 - Software Management
gh issue create \
  --title "Implement Spack installation framework" \
  --body "Create framework for installing Spack on cluster head nodes." \
  --label "type: feature,component: spack,priority: high" \
  --milestone "v0.3.0 - Software Management"

gh issue create \
  --title "Implement Lmod integration" \
  --body "Set up Lmod module system with hierarchical modules." \
  --label "type: feature,component: spack,priority: high" \
  --milestone "v0.3.0 - Software Management"

gh issue create \
  --title "Add Spack package installation" \
  --body "Implement package installation via Spack from templates." \
  --label "type: feature,component: spack,priority: high" \
  --milestone "v0.3.0 - Software Management"

gh issue create \
  --title "Add module file generation" \
  --body "Auto-generate Lmod module files from Spack packages." \
  --label "type: feature,component: spack,priority: medium" \
  --milestone "v0.3.0 - Software Management"

# v0.4.0 - Registry & Capture
gh issue create \
  --title "Implement template registry system" \
  --body "Create GitHub-based template registry for sharing and discovery." \
  --label "type: feature,component: registry,priority: high" \
  --milestone "v0.4.0 - Registry & Capture"

gh issue create \
  --title "Add remote cluster capture" \
  --body "Implement capture of existing cluster configurations via SSH." \
  --label "type: feature,component: capture,priority: medium" \
  --milestone "v0.4.0 - Registry & Capture"

gh issue create \
  --title "Add batch script analysis" \
  --body "Parse batch scripts to extract software requirements." \
  --label "type: feature,component: capture,priority: medium" \
  --milestone "v0.4.0 - Registry & Capture"

gh issue create \
  --title "Implement module mapping database" \
  --body "Create mapping from on-prem modules to Spack packages." \
  --label "type: feature,component: capture,priority: medium" \
  --milestone "v0.4.0 - Registry & Capture"

# General improvements
gh issue create \
  --title "Setup CI/CD with comprehensive testing" \
  --body "Ensure CI/CD pipeline runs tests, linting, and builds on all commits." \
  --label "type: enhancement,priority: high" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Achieve A+ Go Report Card grade" \
  --body "Ensure code quality meets A+ standards for Go Report Card." \
  --label "type: enhancement,priority: high" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Create comprehensive documentation" \
  --body "Write detailed documentation including getting started, API docs, and examples." \
  --label "type: documentation,priority: medium" \
  --milestone "v0.1.0 - Foundation"

gh issue create \
  --title "Setup automated releases" \
  --body "Configure GitHub Actions to build and release binaries on version tags." \
  --label "type: enhancement,priority: medium" \
  --milestone "v0.1.0 - Foundation"

echo "Initial issues created successfully!"
