# pctl - ParallelCluster Templates

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/pctl)](https://goreportcard.com/report/github.com/scttfrdmn/pctl)
[![CI](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/pctl/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/scttfrdmn/pctl)](https://github.com/scttfrdmn/pctl)

A Go-based CLI tool that simplifies AWS ParallelCluster deployment using intuitive YAML templates. pctl bridges the gap between ParallelCluster's power and what users actually need - a simple, repeatable way to deploy HPC clusters with software, users, and data pre-configured.

## Overview

AWS ParallelCluster is powerful but has gaps between infrastructure provisioning and production readiness. Users need:
- Scientific software installed and configured
- Consistent user management (UID/GID)
- Data accessibility (S3, EFS, FSx mounting)
- Simple, reusable configurations

pctl provides a template-driven approach that addresses all these needs with simple YAML configurations.

## Features

- **Simple YAML Templates**: 20-50 lines vs 100+ for raw ParallelCluster configs
- **Software Management**: Automatic Spack installation with Lmod module system
- **User Management**: Consistent UID/GID across all cluster nodes
- **Data Integration**: Easy S3, EFS, and FSx mounting
- **Template Registry**: Share and discover templates via GitHub
- **Configuration Capture**: Migrate existing on-prem clusters to cloud
- **Self-Contained**: Manages its own ParallelCluster installation

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

## Documentation

- [Getting Started](docs/GETTING_STARTED.md)
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
