# petal 🌸 - Grow HPC Clusters from Seeds

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/petal)](https://goreportcard.com/report/github.com/scttfrdmn/petal)
[![CI](https://github.com/scttfrdmn/petal/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/petal/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/scttfrdmn/petal/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/petal)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/scttfrdmn/petal)](https://github.com/scttfrdmn/petal)

**Plant a seed, watch your cluster bloom!** 🌱

A Go-based CLI tool that simplifies AWS ParallelCluster deployment using intuitive seed files (YAML). petal bridges the gap between ParallelCluster's power and what users actually need - a simple, repeatable way to deploy HPC clusters with software, users, and data pre-configured.

## Overview

**Deploy production-ready HPC clusters in minutes, not days.**

petal bridges the gap between AWS ParallelCluster's infrastructure provisioning and what researchers actually need: clusters with software installed, users configured, and data accessible.

### The Problem
- Days of manual software installation (compilers, MPI, scientific packages)
- Inconsistent environments across nodes and users
- Complex S3/EFS data access setup
- No easy way to recreate working environments

### The Solution
One YAML seed → Complete, working HPC cluster

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
petal ami build --seed bioinformatics.yaml --name bio-v1 --detach

# Monitor progress anytime
petal ami status <build-id> --watch

# Deploy clusters instantly (2-3 minutes)
petal create --seed bioinformatics.yaml --custom-ami ami-xxx

# Or use the fun alias: petal bloom! 🌸
petal bloom --seed bioinformatics.yaml --custom-ami ami-xxx
```

### 🌐 Zero AWS Networking Knowledge Required
Automatic VPC creation with proper subnets and security groups. Or use existing VPC with `--subnet-id`. Just run `petal create` - networking handled automatically.

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
36+ ready-to-use seeds for common workloads:
- **Bioinformatics**: samtools, bwa, gatk, blast+
- **Machine Learning**: PyTorch, TensorFlow, CUDA
- **Chemistry**: GROMACS, LAMMPS, Quantum ESPRESSO
- **And more**: astronomy, climate modeling, CFD, rendering

Browse all seeds in `seeds/library/` directory.

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
Capture existing cluster configurations and generate petal seeds automatically:
- SSH to existing clusters, extract config
- Parse SLURM/PBS/SGE batch scripts
- 50+ pre-configured module-to-Spack mappings
- Auto-generate migration seeds

### 📝 Simple, Intuitive Seeds
20-50 lines vs 100+ for raw ParallelCluster configs. Focus on what matters: instances, software, users, data.

## Quick Start

### Installation

**macOS (Homebrew)**
```bash
brew install scttfrdmn/tap/petal
```

**Linux/macOS (Direct Download)**
```bash
# Download the latest release for your platform
curl -LO https://github.com/scttfrdmn/petal/releases/latest/download/petal_linux_x86_64.tar.gz
tar xzf petal_linux_x86_64.tar.gz
sudo mv petal /usr/local/bin/
```

**From Source**
```bash
git clone https://github.com/scttfrdmn/petal.git
cd petal
make build
sudo make install
```

### System Requirements

**AWS ParallelCluster**
- Version: **3.14.0** (latest stable, auto-installed via `petal pcluster install`)
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
petal pcluster install

# Configure AWS credentials (if not already done)
aws configure

# Update seed registry
petal registry update
```

### Create Your First Cluster

```bash
# Browse available seeds
petal registry search bioinformatics

# Create a cluster from a seed (professional commands)
petal create --seed seeds/library/bioinformatics.yaml --name my-cluster --key-name your-key

# Or use fun aliases! 🌸
petal bloom --seed seeds/library/bioinformatics.yaml --name my-cluster --key-name your-key

# Check cluster status
petal status my-cluster  # or: petal inspect my-cluster

# List all clusters
petal list  # or: petal garden

# SSH to your cluster
petal ssh my-cluster  # or: petal stem my-cluster

# Delete cluster when done
petal delete my-cluster  # or: petal harvest my-cluster
```

## Example Seed

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
petal ami build --seed seed.yaml --name my-ami --detach

# Monitor progress
petal ami status <build-id> --watch

# List all builds
petal ami list-builds

# Deploy with custom AMI
petal create --seed seed.yaml --custom-ami ami-xxxxx
```

**Why?** Build once (30-90 min) → deploy unlimited clusters in 2-3 min. Perfect for CI/CD, testing, and production workloads.

## Documentation

- [Getting Started](docs/GETTING_STARTED.md)
- [User Personas & Walkthroughs](docs/PERSONAS.md) - See how different users benefit from pctl
- [Seed Specification](docs/SEED_SPEC.md)
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
│   ├── registry/         # Seed registry
│   ├── capture/          # Configuration capture
│   └── pclusterinstaller/ # ParallelCluster management
├── internal/              # Private packages
│   ├── version/          # Version information
│   └── config/           # Configuration management
├── seeds/library/     # Pre-built seeds
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
