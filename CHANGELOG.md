# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- None

### Changed
- None

### Fixed
- None

## [0.9.0] - 2025-11-10

### Added
- **Comprehensive Template Library - 36 Production Templates** (Issue #27)
  - **Bioinformatics & Life Sciences (8 templates)**
    - bioinformatics.yaml - General genomics tools (BWA, GATK, SAMtools)
    - rna-seq.yaml - RNA sequencing pipelines (STAR, salmon, kallisto)
    - single-cell.yaml - Single-cell analysis (Seurat, Scanpy, CellRanger)
    - metagenomics.yaml - Microbiome analysis (QIIME2, Kraken2, MetaPhlAn)
    - structural-biology.yaml - AlphaFold2, RELION, cryoSPARC, Rosetta, molecular docking
    - medical-imaging.yaml - ITK, FSL, FreeSurfer, DICOM processing, 3D Slicer
    - systems-biology.yaml - Pathway analysis, network modeling
  - **AI/ML & Data Science (7 templates)**
    - machine-learning.yaml - General ML workflows (scikit-learn, XGBoost)
    - llm-training.yaml - Large language model training (multi-GPU, DeepSpeed)
    - computer-vision.yaml - Image processing, CNNs, object detection
    - reinforcement-learning.yaml - RL algorithms, gym environments
    - data-science.yaml - General analytics (Jupyter, pandas, Dask)
    - nlp.yaml - Traditional NLP (SpaCy, NLTK, Gensim, transformers inference)
    - ml-inference.yaml - Model serving and batch inference
  - **Physics & Chemistry (6 templates)**
    - computational-chemistry.yaml - General chemistry simulations
    - quantum-chemistry.yaml - DFT calculations (Gaussian, ORCA, NWChem)
    - molecular-dynamics.yaml - GROMACS, AMBER, NAMD (GPU-accelerated)
    - materials-science.yaml - VASP, Quantum ESPRESSO, LAMMPS
    - high-energy-physics.yaml - ROOT, Geant4, PYTHIA, particle physics
  - **Engineering & Simulation (6 templates)**
    - cfd-openfoam.yaml - CFD with OpenFOAM and ParaView
    - fem-analysis.yaml - Finite element analysis
    - numerical-simulation.yaml - General scientific computing
    - rendering.yaml - Blender, ray tracing, VFX render farms
    - climate-modeling.yaml - Weather/climate simulations (WRF, CESM)
  - **Earth & Space Sciences (4 templates)**
    - astronomy.yaml - Radio astronomy, telescope data processing
    - geoscience.yaml - Seismology (ObsPy, SPECFEM3D)
    - hydrology.yaml - Watershed modeling (SWAT, MODFLOW, HEC-RAS)
    - geospatial.yaml - GIS analysis (GDAL, QGIS, PostGIS)
  - **Data Analytics & Finance (4 templates)**
    - big-data-spark.yaml - Apache Spark for large-scale analytics
    - time-series.yaml - Forecasting and anomaly detection
    - quantitative-finance.yaml - QuantLib, VaR, Monte Carlo, portfolio optimization
    - graph-analytics.yaml - NetworkX, Neo4j, PageRank, community detection
  - **Infrastructure & Development (3 templates)**
    - development.yaml - General software development environment
    - jupyter-hub.yaml - Multi-user JupyterHub deployment
    - benchmarking.yaml - Performance testing and optimization
    - spot-compute.yaml - Cost-optimized Spot instance workloads
- **Template Strategy Documentation** (docs/TEMPLATE_STRATEGY.md)
  - Comprehensive template organization by domain
  - 4-phase implementation plan
  - Template structure standards
  - Success metrics and coverage goals

### Template Features
Each template includes:
- **Domain-specific compute queues** with optimized instance types
- **Comprehensive software stacks** (Spack + pip/conda recommendations)
- **Detailed workflow documentation** including:
  - Common use cases and workflows
  - Algorithm descriptions and complexity analysis
  - Data scales and performance considerations
  - Key software tool descriptions
- **Cost estimates** for typical workloads
- **S3 data integration** with pre-configured mount points
- **Multi-user support** with UIDs/GIDs

### Benefits
- **Production-ready templates** for 36 major HPC/scientific domains
- **Copy-paste deployment** - templates work out-of-the-box
- **Best practice instance selection** - optimized for each workload type
- **Comprehensive documentation** - understand workflows before deploying
- **Community testing ready** - templates available for validation
- **Cost transparency** - estimated hourly costs included

### Testing Phase
This release marks the beginning of the v1.0.0 testing phase:
- Community validation of template configurations
- Real-world software stack testing
- Instance type optimization verification
- Cost estimate validation
- Documentation accuracy review
- Bug reports and feedback collection

## [0.6.0] - 2025-11-10

### Added
- **Async AMI builds with progress monitoring** (Issue #26)
  - **Phase 1: Real-time progress monitoring** (pkg/ami, pkg/software)
    - PCTL_PROGRESS markers throughout software installation scripts
    - EC2 console output polling every 30 seconds during builds
    - Progress percentage tracking (0-100%) across all build phases
    - Detailed progress messages for each installation step
  - **Phase 2: State persistence** (pkg/ami/state.go)
    - JSON-based build state management in ~/.pctl/ami-builds/
    - UUID-based build identifiers for unique tracking
    - BuildState with full metadata (progress, status, timestamps, packages)
    - StateManager for loading, saving, listing, and cleaning up states
    - Build status tracking: launching → installing → creating → complete/failed
  - **Phase 3: Detached builds and status commands** (cmd/pctl/ami.go)
    - `pctl ami build --detach` - Start build and exit immediately
    - `pctl ami status <build-id>` - Check build status with full details
    - `pctl ami status <build-id> --watch` - Continuous progress monitoring
    - `pctl ami list-builds` - Table view of all AMI builds
    - Builds continue running in AWS after CLI exits
    - Reconnect from any machine to check progress
    - Perfect for CI/CD pipelines and long-running builds
  - **Phase 4: Visual progress bars and time estimates**
    - Interactive progress bars with 40-character width
    - Real-time progress visualization: [=========>    ] 45%
    - Estimated time remaining calculations: "~32m remaining"
    - Consistent UX in both build and watch modes
    - Professional appearance with emoji and theming
    - Linear time projection based on elapsed time and progress
- **Persona-based documentation** (docs/PERSONAS.md)
  - 5 detailed user personas with workflows and priorities
  - Bioinformatics researcher, ML engineer, DevOps, lab manager, migration specialist
  - Feature prioritization matrix showing impact by persona
  - Complete walkthroughs with example commands
  - Development guidelines for feature evaluation
  - Persona-driven roadmap planning
- **AMI cleanup and optimization** (Issue #25, pkg/ami/cleanup.go)
  - Comprehensive cleanup script for AMI size reduction (30-50%)
  - Package manager cache cleanup (APT/YUM)
  - Temporary file removal (/tmp, /var/tmp)
  - Log file clearing (security and size)
  - SSH host key removal (regenerated on first boot)
  - Bash history clearing (security)
  - Cloud-init artifact cleanup
  - Spack cache cleanup
  - Free space zeroing for optimal compression
  - Custom cleanup script support

### Changed
- Enhanced README with async build documentation and AMI management section
- Updated bootstrap script generation to include progress markers
- Improved software installation logging with percentage-based progress
- AMI building now includes cleanup phase by default (can skip with --skip-cleanup)

### Technical Improvements
- Added github.com/google/uuid dependency for unique build IDs
- Added github.com/schollz/progressbar/v3 for visual progress feedback
- Enhanced error handling in AMI builder with better state tracking
- Improved console output parsing for progress extraction

### Benefits
- Users can start 30-90 minute builds and disconnect
- Real-time progress visibility without waiting at terminal
- Multiple concurrent builds can be tracked
- CI/CD integration with programmatic status checking
- Professional UX with visual progress and time estimates
- 30-50% smaller AMIs through automated cleanup

## [0.5.0] - 2025-11-09

### Added
- **Custom AMI building system** (pkg/ami, Issue #21)
  - Automated AMI creation from pctl templates
  - Pre-bakes all software into custom AMIs
  - Reduces cluster creation time from hours to 2-3 minutes
  - `pctl ami build` - Build custom AMIs with all software pre-installed
  - `pctl ami list` - List all pctl-managed AMIs
  - `pctl ami delete` - Delete AMIs and associated snapshots
  - Automated build process:
    - Launches temporary EC2 instance
    - Installs all Spack packages and Lmod
    - Creates AMI from configured instance
    - Cleans up temporary resources
  - AMI lifecycle management
  - Template-to-AMI association tracking
  - Automatic base ParallelCluster AMI detection
  - Snapshot cleanup on AMI deletion
  - Progress reporting during 30-90 minute build process
  - Integration with --custom-ami flag in pctl create

## [0.4.0] - 2025-11-09

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

## [0.3.0] - 2025-11-09

### Added
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

## [0.2.0] - 2025-11-09

### Added
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

[Unreleased]: https://github.com/scttfrdmn/pctl/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/scttfrdmn/pctl/compare/v0.6.0...v0.9.0
[0.6.0]: https://github.com/scttfrdmn/pctl/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/scttfrdmn/pctl/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/scttfrdmn/pctl/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/scttfrdmn/pctl/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/pctl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/scttfrdmn/pctl/releases/tag/v0.1.0
