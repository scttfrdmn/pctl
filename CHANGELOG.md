# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Template registry system** (pkg/registry, Issue #13)
  - GitHub-based template registry for sharing and discovery
  - Registry interface with pluggable backends
  - GitHub registry implementation with HTTP client
  - Template metadata with search and filtering
  - `pctl registry list` - List all available templates
  - `pctl registry search` - Search templates by keyword
  - `pctl registry pull` - Download templates locally
  - Support for multiple registries
  - Template versioning and author information
  - URL parsing for GitHub repositories
- **On-prem to AWS migration tools** (pkg/capture)
  - Module mapping database (Issue #16)
    - 50+ pre-configured module-to-Spack mappings
    - Compilers (gcc, intel, llvm)
    - MPI libraries (openmpi, mpich, intelmpi)
    - Bioinformatics tools (samtools, bwa, gatk, blast)
    - Computational chemistry (gromacs, lammps, quantum-espresso)
    - Machine learning (pytorch, tensorflow, cuda)
    - Math libraries (fftw, blas, lapack, hdf5)
    - Module name normalization (handles versions, suffixes)
    - Confidence scoring for mappings
    - Unmapped module suggestions
  - Batch script analysis (Issue #15)
    - SLURM, PBS, and SGE script support
    - Automatic scheduler detection
    - Module load extraction
    - Resource requirement parsing (nodes, tasks, CPUs, memory, walltime)
    - Executable command detection
    - Partition/queue identification
  - Remote cluster capture (Issue #14)
    - SSH-based configuration capture
    - Module availability detection
    - User account discovery (non-system users)
    - Installed software detection
    - Automatic template generation from captured config
    - Module-to-Spack conversion during capture
- **Production-ready software management system** (pkg/software)
  - Spack installation framework with configurable versions and paths
  - AWS Spack buildcache integration for faster package installations
  - Automatic buildcache mirror configuration (https://binaries.spack.io)
  - Trusted GPG key management for buildcache
  - Compiler detection and installation (gcc, llvm, intel-oneapi)
  - Parallel package installation with fallback to source builds
  - Lmod module system integration
  - Hierarchical module organization (Core, Compiler, MPI)
  - Automatic module generation from Spack packages
  - Custom module file creation with environment variables
  - Comprehensive bootstrap script generation
  - Detailed logging and progress reporting
  - Error handling with graceful fallbacks
- Enhanced bootstrap script generation
  - Modular software management (replaces monolithic script)
  - Section-based organization (users, S3, software)
  - Improved error handling and logging
  - Support for software-only installations
- **AWS SDK integration for automatic VPC/networking** (pkg/network)
  - Automatic VPC creation with 10.0.0.0/16 CIDR
  - Public subnet (10.0.1.0/24) for head node with auto-assign public IP
  - Private subnet (10.0.2.0/24) for compute nodes
  - Internet gateway and route tables
  - Security groups with SSH access and internal communication
  - Automatic cleanup on cluster deletion
  - All resources tagged with cluster name and "ManagedBy: pctl"
  - AWS SDK Go v2 dependencies added
- Enhanced cluster state management
  - Network resource tracking (VPC, subnets, security groups, etc.)
  - NetworkManagedByPctl flag to distinguish user-provided vs auto-created
- ParallelCluster configuration generator (pkg/config)
  - Converts pctl templates to ParallelCluster YAML configs
  - Supports multiple instance types per queue
  - Generates bootstrap scripts for software installation
  - S3 mount configuration with IAM policies
  - Custom AMI support
- Cluster state management system (pkg/state)
  - JSON-based state storage in ~/.pctl/state/
  - Track cluster status, creation time, configuration
  - List, load, save, delete state operations
- Provisioner with ParallelCluster CLI integration (pkg/provisioner)
  - Create, delete, describe cluster operations
  - Wraps pcluster CLI commands
  - State management integration
  - Context-aware operations
- Complete CLI command implementations:
  - `pctl create` - Full cluster provisioning with --key-name and --subnet-id flags
    - Bootstrap script generation and software installation
    - --wait flag for synchronous cluster creation
    - --custom-ami flag for custom AMI support
  - `pctl list` - Beautiful table output showing all managed clusters
    - Status emojis (✅ complete, 🔄 in progress, ❌ failed)
    - Relative time formatting (e.g., "2 hours ago")
    - Dynamic column widths
  - `pctl status` - Detailed cluster status with actionable next steps
    - Head node IP and SSH instructions
    - Compute node counts
    - Scheduler state
    - Context-aware action suggestions
  - `pctl delete` - Safe cluster deletion with confirmation prompt
    - Type cluster name to confirm (or use --force)
    - Checks cluster exists before attempting deletion
    - Preserves S3 bucket data

### Changed
- `pctl create` command now auto-creates VPC/networking if --subnet-id not provided
  - --subnet-id flag is now optional (was required)
  - Auto-creates VPC with proper networking if not provided
  - Displays network creation progress with resource IDs
  - Cleans up network resources on failure
  - Updated help text and examples to show automatic networking

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
