# pctl - Templated AWS ParallelCluster Deployment

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/pctl)](https://goreportcard.com/report/github.com/scttfrdmn/pctl)
[![CI](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/scttfrdmn/pctl/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/pctl)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/scttfrdmn/pctl)](https://github.com/scttfrdmn/pctl)

A Go-based CLI tool that simplifies AWS ParallelCluster deployment using intuitive YAML templates. pctl bridges the gap between ParallelCluster's power and what users actually need - a simple, repeatable way to deploy HPC clusters with software, users, and data pre-configured.

## Overview

**Deploy production-ready HPC clusters in minutes, not days.**

pctl bridges the gap between AWS ParallelCluster's infrastructure provisioning and what researchers actually need: clusters with software installed, users configured, and data accessible.

### The Problem
- Days of manual software installation (compilers, MPI, scientific packages)
- Inconsistent environments across nodes and users
- Complex S3/EFS data access setup
- No easy way to recreate working environments

### The Solution
One YAML template → Complete, working HPC cluster

- **Software pre-installed** via Spack + Lmod modules
- **97% faster deployment** with custom AMIs (2-3 min vs 30-90 min)
- **Zero networking knowledge** required (automatic VPC setup)
- **Production-ready** from day one

## Key Features

### ⚡ Lightning-Fast Deployment
**97% faster** - Deploy clusters in 2-3 minutes instead of 30-90 minutes

Pre-build custom AMIs with all software installed, then deploy unlimited clusters instantly:
```bash
# Build AMI once (30-90 min, runs in background)
pctl ami build -t bioinformatics.yaml --name bio-v1 --detach

# Monitor progress anytime
pctl ami status <build-id> --watch

# Deploy clusters instantly (2-3 minutes)
pctl create -t bioinformatics.yaml --custom-ami ami-xxx
```

### 🌐 Zero AWS Networking Knowledge Required
Automatic VPC creation with proper subnets and security groups. Or use existing VPC with `--subnet-id`. Just run `pctl create` - networking handled automatically.

### 📦 Automatic Software Installation
Specify packages in YAML, get working modules on your cluster:
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
```
- Pre-built binaries via AWS Spack buildcache (minutes vs hours)
- Lmod modules automatically generated
- 6000+ packages available

Users run `module load samtools` and start working immediately.

### 🚀 Pre-Configured Software Stacks
36+ ready-to-use templates for common workloads:
- **Bioinformatics**: samtools, bwa, gatk, blast+
- **Machine Learning**: PyTorch, TensorFlow, CUDA
- **Chemistry**: GROMACS, LAMMPS, Quantum ESPRESSO
- **And more**: astronomy, climate modeling, CFD, rendering

### 👥 Consistent User Management
Define users once, consistent UID/GID across all nodes:
```yaml
users:
  - name: researcher1
    uid: 5001
```

### 💾 Simple Data Access
S3 buckets mounted as filesystem paths:
```yaml
data:
  s3_mounts:
    - bucket: my-research-data
      mount_point: /shared/data
```

### 🔄 On-Prem to AWS Migration
Capture existing cluster configurations and generate pctl templates automatically:
- SSH to existing clusters, extract config
- Parse SLURM/PBS/SGE batch scripts
- 50+ pre-configured module-to-Spack mappings
- Auto-generate migration templates

### 📝 Simple, Intuitive Templates
20-50 lines vs 100+ for raw ParallelCluster configs. Focus on what matters: instances, software, users, data.

## Quick Start

### Installation

**macOS (Homebrew)**
```bash
brew install scttfrdmn/tap/pctl
```

**Linux/macOS (Direct Download)**
```bash
# Download the latest release for your platform
curl -LO https://github.com/scttfrdmn/pctl/releases/latest/download/pctl_linux_x86_64.tar.gz
tar xzf pctl_linux_x86_64.tar.gz
sudo mv pctl /usr/local/bin/
```

**From Source**
```bash
git clone https://github.com/scttfrdmn/pctl.git
cd pctl
make build
sudo make install
```

### System Requirements

**AWS ParallelCluster**
- Version: **3.14.0** (latest stable, auto-installed via `pctl pcluster install`)
- Includes Slurm 24.05.7, NICE DCV support, and P6e instance compatibility

**Operating System**
- Default: **Amazon Linux 2023** (supported until 2029)
- Also supports: Ubuntu 24.04, Ubuntu 22.04, RHEL 8/9, Rocky 8/9
- AL2023 features: Kernel 6.12, improved hardware support, long-term stability

**AWS Account Requirements**
- Active AWS account with appropriate permissions
- AWS credentials configured (via `aws configure`)

### Initial Setup

```bash
# Install ParallelCluster (required dependency)
pctl pcluster install

# Configure AWS credentials (if not already done)
aws configure

# Update template registry
pctl registry update
```

### Create Your First Cluster

```bash
# Browse available templates
pctl registry search bioinformatics

# Create a cluster from a template
pctl create -t seeds/library/bioinformatics.yaml --name my-cluster

# Check cluster status
pctl status my-cluster

# List all clusters
pctl list
```

## Example Template

```yaml
cluster:
  name: research-cluster
  region: us-east-1

compute:
  head_node: t3.xlarge
  queues:
    - name: compute
      instance_types: [c5.4xlarge]
      min_count: 0
      max_count: 10

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10
    - samtools@1.17

users:
  - name: researcher1
    uid: 5001
    gid: 5001

data:
  s3_mounts:
    - bucket: my-research-data
      mount_point: /shared/data
```

## AMI Management

Build custom AMIs once, deploy clusters instantly forever:

```bash
# Build custom AMI (runs in background)
pctl ami build -t template.yaml --name my-ami --detach

# Monitor progress
pctl ami status <build-id> --watch

# List all builds
pctl ami list-builds

# Deploy with custom AMI
pctl create -t template.yaml --custom-ami ami-xxxxx
```

**Why?** Build once (30-90 min) → deploy unlimited clusters in 2-3 min. Perfect for CI/CD, testing, and production workloads.

## Documentation

- [Getting Started](docs/GETTING_STARTED.md)
- [User Personas & Walkthroughs](docs/PERSONAS.md) - See how different users benefit from pctl
- [Template Specification](docs/TEMPLATE_SPEC.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Project Analysis](PROJECT_ANALYSIS.md)
- [Design Document](DESIGN.md)

## Development

### Prerequisites

- Go 1.21 or higher
- make
- golangci-lint (optional, for linting)

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run linters
make lint

# Run all checks (formatting, vetting, linting, tests)
make check

# Install locally
make install
```

### Project Structure

```
pctl/
├── cmd/pctl/              # CLI entry point and commands
├── pkg/                   # Public packages
│   ├── template/         # Template parsing and validation
│   ├── provisioner/      # Cluster orchestration
│   ├── config/           # ParallelCluster config generation
│   ├── spack/            # Software installation
│   ├── registry/         # Template registry
│   ├── capture/          # Configuration capture
│   └── pclusterinstaller/ # ParallelCluster management
├── internal/              # Private packages
│   ├── version/          # Version information
│   └── config/           # Configuration management
├── seeds/library/     # Pre-built templates
├── tests/                 # Test suites
└── docs/                  # Documentation
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests with race detector
make test-race
```

### Code Quality

This project maintains an A+ rating on [Go Report Card](https://goreportcard.com/). We use:

- `go fmt` for consistent formatting
- `go vet` for suspicious constructs
- `golangci-lint` for comprehensive linting
- `gocyclo` for complexity checking
- Pre-commit hooks for quality checks

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and quality checks (`make check`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Versioning

This project uses [Semantic Versioning 2.0.0](https://semver.org/). Version numbers follow the format:

```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality additions
- **PATCH**: Backwards-compatible bug fixes

See [CHANGELOG.md](CHANGELOG.md) for release history.

## License

Copyright 2025 Scott Friedman

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Acknowledgments

- AWS ParallelCluster team for the foundational cluster management tool
- Spack community for the package management system
- Lmod project for the module system

## Support

- [Issue Tracker](https://github.com/scttfrdmn/pctl/issues)
- [Discussions](https://github.com/scttfrdmn/pctl/discussions)
- [Project Board](https://github.com/users/scttfrdmn/projects)

## Roadmap

See the [GitHub Project Board](https://github.com/users/scttfrdmn/projects) for planned features and current development status.
