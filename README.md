# pctl - ParallelCluster Templates

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/pctl)](https://goreportcard.com/report/github.com/scttfrdmn/pctl)
[![CI](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/scttfrdmn/pctl)](https://github.com/scttfrdmn/pctl)

A Go-based CLI tool that simplifies AWS ParallelCluster deployment using intuitive YAML templates. pctl bridges the gap between ParallelCluster's power and what users actually need - a simple, repeatable way to deploy HPC clusters with software, users, and data pre-configured.

## Overview

**pctl delivers ready-to-use HPC clusters, not just empty infrastructure.**

AWS ParallelCluster provisions compute nodes, but researchers still face the gap between infrastructure and productivity:
- **Days of manual work** installing compilers, MPI, scientific software
- **Inconsistent environments** across users and nodes
- **Complex data access** setup for S3, EFS, FSx
- **No repeatability** - hard to recreate working environments

**pctl solves this:** One simple YAML template provisions a complete, working cluster with your scientific software pre-installed, users configured, and data accessible. Submit jobs immediately, not after days of setup.

From template to working cluster with software installed - that's pctl.

## Key Features

### ⚡ Lightning-Fast Deployment (v0.5.0+)
**97% faster cluster creation** - Deploy production HPC clusters in 2-3 minutes instead of hours:
- **Custom AMI building** - Pre-bake software into custom AMIs once (30-90 min)
- **Detached builds** (v0.5.1) - Build AMIs in background, disconnect and check later
- **Real-time progress** (v0.5.1) - Monitor builds with live progress updates
- **Instant clusters** - Launch clusters with all software already installed
- Perfect for testing, development, CI/CD, and rapid scaling
```bash
# Build AMI once (30-90 minutes, one time) - runs in background
pctl ami build -t bioinformatics.yaml --name bio-v1 --detach

# Check build progress anytime, from anywhere
pctl ami status <build-id> --watch

# List all builds
pctl ami list-builds

# Deploy clusters in 2-3 minutes (forever after)
pctl create -t bioinformatics.yaml --custom-ami ami-xxx
```

### 🌐 Zero Network Configuration (v0.2.0)
**Automatic VPC/networking** - No AWS networking knowledge required:
- Auto-creates VPC with proper subnets and security groups
- Or use existing VPC with `--subnet-id`
- Automatic cleanup on cluster deletion
- Just run `pctl create` - networking handled automatically

### 🚀 Ready-to-Use Clusters
**Not just nodes - complete working environments.** Your cluster comes with scientific software installed and configured:
- Bioinformatics: samtools, bwa, gatk, blast+
- Machine Learning: PyTorch, TensorFlow, CUDA
- Computational Chemistry: GROMACS, LAMMPS, Quantum ESPRESSO
- Or bring your own: 6000+ packages via Spack

### 📦 Automatic Software Installation (v0.3.0)
**No more days installing dependencies.** Specify packages in your template, pctl installs them with Spack and generates Lmod modules:
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
    - python@3.10
```
- **AWS Spack buildcache** - Pre-built binaries install in minutes vs hours
- **Lmod module system** - Hierarchical organization (Core, Compiler, MPI)
- **Graceful fallbacks** - Builds from source if binaries unavailable

Users do: `module load samtools` and start working immediately.

### 👥 User Management
**Consistent UID/GID across all nodes.** Define users once, they work everywhere:
```yaml
users:
  - name: researcher1
    uid: 5001
    gid: 5001
```

### 💾 Data Access
**S3 buckets mounted as filesystem paths.** No manual s3fs setup:
```yaml
data:
  s3_mounts:
    - bucket: my-research-data
      mount_point: /shared/data
```
Users access data like local files: `ls /shared/data/`

### 📝 Simple Templates
**20-50 lines vs 100+ for raw ParallelCluster configs.** Focus on what matters: instances, software, users, data.

### 🔄 Migration & Discovery (v0.4.0)
**Seamless on-prem to AWS migration:**
- **Configuration capture** - SSH to existing clusters, extract configuration
- **Batch script analysis** - Parse SLURM/PBS/SGE scripts for requirements
- **Module mapping** - 50+ pre-configured module-to-Spack mappings
- **Template generation** - Automatically create pctl templates from captured configs

**Template registry:**
- Share and discover templates via GitHub
- Search by workload type (bioinformatics, ML, chemistry)
- Community-contributed templates

## Quick Start

### Installation

```bash
# Download the latest release
curl -LO https://github.com/scttfrdmn/pctl/releases/latest/download/pctl
chmod +x pctl
sudo mv pctl /usr/local/bin/

# Or build from source
git clone https://github.com/scttfrdmn/pctl.git
cd pctl
make build
sudo make install
```

### Initial Setup

```bash
# Install ParallelCluster
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
pctl create -t templates/library/bioinformatics.yaml --name my-cluster

# Check status
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

pctl provides powerful AMI building and management capabilities for pre-installing software:

```bash
# Build a custom AMI (detached mode - runs in background)
pctl ami build -t template.yaml --name my-ami --detach

# Check build status
pctl ami status <build-id>

# Watch build progress in real-time
pctl ami status <build-id> --watch

# List all AMI builds
pctl ami list-builds

# Use custom AMI when creating clusters
pctl create -t template.yaml --custom-ami ami-xxxxx
```

### Why Build Custom AMIs?

**30-90 minute builds → 2-3 minute clusters forever:**
- Build AMI once with all software pre-installed
- Launch unlimited clusters in 2-3 minutes
- Perfect for CI/CD, testing, and rapid experimentation
- Detached mode: start build and come back later

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
├── templates/library/     # Pre-built templates
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
