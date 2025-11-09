# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- None

### Changed
- None

## [0.1.0] - 2025-11-09

### Added
- Initial project structure with Go best practices
- Apache 2.0 license (Copyright 2025 Scott Friedman)
- Semantic versioning support with build-time version injection
- CI/CD workflows with GitHub Actions (CI and Release)
- Comprehensive Makefile with quality checks
- golangci-lint configuration for A+ Go Report Card compliance
- Basic CLI framework using cobra and viper
- Complete template parsing and validation system
  - Comprehensive validation for cluster, compute, software, users, and data sections
  - Multi-error reporting with detailed messages
  - Validation for AWS regions, instance types, user names, UIDs/GIDs, S3 buckets
  - Best practices warnings (UID ranges, resource limits)
- CLI commands:
  - `pctl validate` - Validate template files with verbose mode
  - `pctl create` - Create clusters (stub with dry-run support)
  - `pctl list` - List managed clusters (stub)
  - `pctl status` - Check cluster status (stub)
  - `pctl delete` - Delete clusters (stub with force mode)
  - `pctl version` - Show version information (text and JSON output)
- Example templates:
  - Minimal cluster (simplest configuration)
  - Starter cluster (with software and users)
  - Bioinformatics cluster (genomics tools: samtools, bwa, gatk, blast+)
  - Machine Learning cluster (GPU instances, PyTorch, TensorFlow)
  - Computational Chemistry cluster (GROMACS, LAMMPS, Quantum ESPRESSO)
- Comprehensive documentation:
  - README with feature highlights and quick start
  - Getting Started guide with tutorials and workflows
  - Template Specification (complete reference)
  - Contributing guidelines
  - Project setup documentation
  - Project analysis document
- GitHub project management:
  - 25 labels organized by type, priority, status, component
  - 5 milestones (v0.1.0 through v1.0.0)
  - 20 initial issues organized by milestone
  - Setup scripts for labels and issues
- Unit tests:
  - Template validation tests (cluster, compute, users, S3)
  - Table-driven test design
  - Tests for edge cases and error conditions
- Configuration management system
  - Support for ~/.pctl/config.yaml
  - Default values for common settings
  - Environment-aware configuration

### Changed
- README emphasizes ready-to-use clusters with software, not just infrastructure
- Documentation highlights software installation as key feature

### Fixed
- None

[Unreleased]: https://github.com/scttfrdmn/pctl/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/scttfrdmn/pctl/releases/tag/v0.1.0
