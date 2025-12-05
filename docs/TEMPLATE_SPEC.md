# pctl Template Specification

This document provides the complete specification for pctl cluster templates.

## Overview

A pctl template is a YAML file that describes a **complete, ready-to-use HPC cluster**. Unlike raw ParallelCluster configurations that only provision infrastructure, pctl templates specify:

- **Infrastructure**: Instances, networking, queues
- **Software**: Scientific packages, compilers, libraries, tools
- **Users**: Consistent UID/GID mapping across all nodes
- **Data**: S3, EFS, FSx mount points

The goal is to create a **usable cluster** where researchers can immediately submit jobs and run their workflows, not an empty cluster that needs days of manual configuration.

## Template Structure

A template consists of five main sections:

```yaml
cluster:      # Required - Cluster identification and region
compute:      # Required - Head node and compute queues
software:     # Optional - Software packages to install
users:        # Optional - User accounts and permissions
data:         # Optional - Data source mounts
```

## Cluster Section

**Required.** Defines cluster identification and AWS region.

```yaml
cluster:
  name: <string>    # Required
  region: <string>  # Required
```

### Fields

#### `name` (required)

**Type:** string
**Format:** Must start with a letter, contain only alphanumeric characters and hyphens, max 60 characters

The cluster identifier. This becomes:
- CloudFormation stack name
- Resource tag value
- SSH config identifier

**Examples:**
```yaml
name: research-cluster-01        # Good
name: bio-analysis               # Good
name: ml-gpu-prod                # Good
name: 123-invalid                # Bad - starts with number
name: My_Cluster                 # Bad - contains underscore and uppercase
```

#### `region` (required)

**Type:** string
**Valid values:** Any AWS region where ParallelCluster is supported

The AWS region where the cluster will be created.

**Examples:**
```yaml
region: us-east-1      # N. Virginia
region: us-west-2      # Oregon
region: eu-west-1      # Ireland
region: ap-northeast-1 # Tokyo
```

## Compute Section

**Required.** Defines the head node and compute queue configuration.

```yaml
compute:
  head_node: <string>        # Required
  queues:                    # Required - list of queues
    - name: <string>         # Required
      instance_types: <list> # Required
      min_count: <int>       # Required
      max_count: <int>       # Required
```

### Fields

#### `head_node` (required)

**Type:** string
**Format:** Valid EC2 instance type

The instance type for the head/master node that runs SLURM scheduler and job management.

**Recommendations:**
- General purpose: `t3.xlarge`, `t3.2xlarge`
- CPU intensive: `c5.2xlarge`
- Memory intensive: `r5.2xlarge`
- With GUI/Jupyter: `t3.2xlarge` or larger

**Examples:**
```yaml
head_node: t3.xlarge    # 4 vCPU, 16 GB RAM - good for most clusters
head_node: t3.2xlarge   # 8 vCPU, 32 GB RAM - for Jupyter/RStudio
head_node: c5.4xlarge   # 16 vCPU, 32 GB RAM - CPU intensive head node
```

#### `queues` (required)

**Type:** list of queue objects
**Minimum:** 1 queue

Defines compute queues with different instance types and scaling policies.

### Queue Object

Each queue has:

#### `name` (required)

**Type:** string
**Format:** Lowercase letters, numbers, hyphens; must start with letter

The queue identifier for SLURM job submission.

**Examples:**
```yaml
name: compute    # General purpose queue
name: gpu        # GPU queue
name: highmem    # High memory queue
name: spot       # Spot instance queue
```

#### `instance_types` (required)

**Type:** list of strings
**Minimum:** 1 instance type

EC2 instance types for this queue. If multiple types are specified, SLURM will choose based on availability and cost.

**Instance Type Families:**
- `t3.*` - Burstable general purpose (development, low-cost)
- `c5.*` - Compute optimized (CPU-intensive workloads)
- `m5.*` - General purpose (balanced compute/memory)
- `r5.*` - Memory optimized (large datasets, in-memory)
- `g4dn.*` - GPU instances (deep learning, rendering)
- `p3.*` - High-performance GPU (intensive deep learning)
- `i3.*` - Storage optimized (high IOPS)

**Examples:**
```yaml
# Single instance type
instance_types:
  - c5.4xlarge

# Multiple types for flexibility
instance_types:
  - c5.4xlarge
  - c5.9xlarge
  - c5.18xlarge

# GPU instances
instance_types:
  - g4dn.xlarge
  - g4dn.2xlarge
  - p3.2xlarge
```

#### `min_count` (required)

**Type:** integer
**Range:** 0-1000
**Recommendation:** Set to 0 for auto-scaling

Minimum number of compute nodes to keep running. Set to 0 to allow full auto-scaling (nodes launch on-demand and terminate when idle).

#### `max_count` (required)

**Type:** integer
**Range:** 0-1000 (must be >= min_count)

Maximum number of compute nodes. This is your scale limit.

**Examples:**
```yaml
# Full auto-scaling (recommended)
min_count: 0
max_count: 50

# Always-on cluster
min_count: 10
max_count: 10

# Hybrid
min_count: 5    # 5 nodes always running
max_count: 100  # Scale up to 100
```

## Software Section

**Optional but highly recommended.** Defines software packages to install on the cluster using Spack.

**This is what makes pctl powerful** - you get a cluster with your scientific software already installed, not just empty compute nodes.

```yaml
software:
  spack_packages: <list>  # Optional - list of package specs
```

### Fields

#### `spack_packages` (optional)

**Type:** list of strings
**Format:** Spack package specifications

List of software packages to install using Spack. Packages are installed on the head node and made available to compute nodes via NFS. Lmod modules are automatically generated.

**Package Specification Format:**
```
package_name[@version][%compiler][ other-specs]
```

**Common Package Categories:**

**Compilers:**
```yaml
- gcc@11.3.0
- gcc@12.2.0
- intel-oneapi-compilers@2023.1.0
```

**MPI Libraries:**
```yaml
- openmpi@4.1.4
- mpich@4.0.3
- intel-oneapi-mpi@2021.9.0
```

**Programming Languages:**
```yaml
- python@3.10
- python@3.11
- r@4.2.0
- julia@1.8.0
- perl@5.36.0
```

**Bioinformatics:**
```yaml
- samtools@1.17
- bwa@0.7.17
- gatk@4.3.0
- blast-plus@2.14.0
- bowtie2@2.4.5
- bedtools2@2.30.0
```

**Machine Learning:**
```yaml
- py-torch@2.0.0
- py-tensorflow@2.12.0
- py-scikit-learn@1.2.0
- py-numpy@1.24.0
- py-pandas@2.0.0
```

**Molecular Dynamics:**
```yaml
- gromacs@2023.1
- lammps@20230802
- namd@2.14
- vmd@1.9.4
```

**Quantum Chemistry:**
```yaml
- quantum-espresso@7.2
- nwchem@7.2.0
- orca@5.0.3
```

**Data Science:**
```yaml
- py-jupyter@1.0.0
- py-matplotlib@3.7.0
- py-scipy@1.10.0
- r-ggplot2@3.4.0
```

**Examples:**

**Bioinformatics Stack:**
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
    - bwa@0.7.17
    - gatk@4.3.0
    - python@3.10
    - r@4.2.0
```

**Machine Learning Stack:**
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - cuda@11.8.0
    - cudnn@8.6.0
    - python@3.10
    - py-torch@2.0.0
    - py-tensorflow@2.12.0
    - py-numpy@1.24.0
```

**Computational Chemistry Stack:**
```yaml
software:
  spack_packages:
    - gcc@11.3.0
    - intel-oneapi-compilers@2023.1.0
    - openmpi@4.1.4
    - gromacs@2023.1
    - lammps@20230802
    - python@3.10
```

**Using Modules:**

Once installed, software is available via Lmod modules:

```bash
# List available modules
module avail

# Load software
module load gcc/11.3.0
module load openmpi/4.1.4
module load samtools/1.17

# In SLURM batch scripts
#!/bin/bash
#SBATCH -J my-job
#SBATCH -p compute
#SBATCH -n 16

module load gcc/11.3.0
module load openmpi/4.1.4
module load myapp/1.0

mpirun -n 16 myapp input.dat
```

## Users Section

**Optional.** Defines user accounts with consistent UID/GID across all nodes.

```yaml
users:
  - name: <string>   # Required
    uid: <int>       # Required
    gid: <int>       # Required
```

### Why User Management Matters

For shared clusters, consistent UID/GID is critical:
- File ownership is preserved across nodes
- NFS mounted home directories work correctly
- Permission issues are avoided
- Users can be integrated with organizational LDAP/AD later

### Fields

#### `name` (required)

**Type:** string
**Format:** Lowercase letters, numbers, underscores, hyphens; must start with letter or underscore

The username for the account.

**Examples:**
```yaml
name: researcher1    # Good
name: jsmith         # Good
name: lab_user       # Good
name: admin_user     # Good
name: John           # Bad - uppercase
name: 123user        # Bad - starts with number
```

#### `uid` (required)

**Type:** integer
**Range:** 1-60000
**Recommended:** 5000-59999 (avoid system range 1-999)

The user ID. Must be unique across all users in the template.

#### `gid` (required)

**Type:** integer
**Range:** 1-60000
**Recommended:** 5000-59999

The primary group ID. Can be same as UID for per-user groups, or shared across users for group access.

**Examples:**

**Individual Users:**
```yaml
users:
  - name: alice
    uid: 5001
    gid: 5001    # Per-user group
  - name: bob
    uid: 5002
    gid: 5002
```

**Shared Group:**
```yaml
users:
  - name: researcher1
    uid: 5001
    gid: 5000    # Shared group
  - name: researcher2
    uid: 5002
    gid: 5000    # Same group for file sharing
  - name: researcher3
    uid: 5003
    gid: 5000
```

## Data Section

**Optional.** Defines S3 bucket mounts for data access.

```yaml
data:
  s3_mounts:
    - bucket: <string>      # Required
      mount_point: <string> # Required
```

### Why Data Mounting Matters

Researchers need access to:
- Reference datasets (genomes, models, parameters)
- Input files (raw data, configurations)
- Output storage (results, checkpoints, logs)

pctl automatically mounts S3 buckets to the cluster filesystem, making cloud data accessible like local storage.

### Fields

#### S3 Mount Object

#### `bucket` (required)

**Type:** string
**Format:** Valid S3 bucket name (3-63 characters, lowercase, numbers, hyphens, dots)

The S3 bucket to mount. Bucket must exist and IAM permissions must allow access.

#### `mount_point` (required)

**Type:** string
**Format:** Absolute path (must start with `/`)

Where to mount the bucket on the cluster filesystem. Must be unique across all mounts.

**Recommendations:**
- `/shared/data` - Raw input data
- `/shared/references` - Reference datasets
- `/shared/results` - Output and results
- `/shared/software` - Custom software binaries

**Examples:**

**Single Data Bucket:**
```yaml
data:
  s3_mounts:
    - bucket: my-research-data
      mount_point: /shared/data
```

**Multiple Buckets:**
```yaml
data:
  s3_mounts:
    # Reference genomes
    - bucket: lab-reference-genomes
      mount_point: /shared/references

    # Raw sequencing data
    - bucket: lab-sequencing-data
      mount_point: /shared/rawdata

    # Analysis results
    - bucket: lab-results
      mount_point: /shared/results
```

**Accessing Mounted Data:**

```bash
# On cluster nodes
ls /shared/data/
cp /shared/references/hg38.fa ./
./analyze --input /shared/data/sample.bam --output /shared/results/
```

## Complete Examples

### Example 1: Minimal Cluster

Simplest possible configuration:

```yaml
cluster:
  name: minimal-cluster
  region: us-east-1

compute:
  head_node: t3.medium
  queues:
    - name: compute
      instance_types:
        - c5.xlarge
      min_count: 0
      max_count: 10
```

**Use case:** Quick testing, development, learning pctl

### Example 2: Bioinformatics Cluster

Complete genomics analysis environment:

```yaml
cluster:
  name: genomics-lab
  region: us-east-1

compute:
  head_node: t3.xlarge
  queues:
    # Memory-optimized for assembly
    - name: assembly
      instance_types:
        - r5.4xlarge
        - r5.8xlarge
      min_count: 0
      max_count: 10

    # Compute-optimized for alignment
    - name: alignment
      instance_types:
        - c5.4xlarge
        - c5.9xlarge
      min_count: 0
      max_count: 30

software:
  spack_packages:
    # Toolchain
    - gcc@11.3.0
    - openmpi@4.1.4

    # Bioinformatics
    - samtools@1.17
    - bwa@0.7.17
    - gatk@4.3.0
    - blast-plus@2.14.0
    - bowtie2@2.4.5
    - bedtools2@2.30.0

    # Analysis
    - python@3.10
    - r@4.2.0
    - py-numpy@1.24.0
    - py-pandas@2.0.0

users:
  - name: pi_researcher
    uid: 5001
    gid: 5000
  - name: postdoc1
    uid: 5002
    gid: 5000
  - name: student1
    uid: 5003
    gid: 5000

data:
  s3_mounts:
    - bucket: lab-reference-genomes
      mount_point: /shared/references
    - bucket: lab-sequencing-runs
      mount_point: /shared/data
    - bucket: lab-analysis-results
      mount_point: /shared/results
```

**Use case:** Research lab with multiple users, standard genomics pipelines

**What you get:**
- Ready-to-use cluster with all bioinformatics tools installed
- Module system for loading software
- Shared user environment (GID 5000)
- Reference data, input data, and results accessible via S3

**Researcher workflow:**
```bash
# SSH to cluster
ssh ec2-user@<head-node>

# Load modules
module load samtools bwa

# Submit job
sbatch align_samples.sh

# Check results
ls /shared/results/
```

### Example 3: Machine Learning Cluster

GPU cluster for deep learning:

```yaml
cluster:
  name: ml-training
  region: us-west-2

compute:
  head_node: t3.2xlarge  # Larger for Jupyter

  queues:
    # GPU training
    - name: gpu
      instance_types:
        - g4dn.xlarge
        - g4dn.2xlarge
        - p3.2xlarge
      min_count: 0
      max_count: 10

    # CPU preprocessing
    - name: cpu
      instance_types:
        - c5.4xlarge
      min_count: 0
      max_count: 20

software:
  spack_packages:
    # Compilers
    - gcc@11.3.0

    # GPU support
    - cuda@11.8.0
    - cudnn@8.6.0

    # ML frameworks
    - python@3.10
    - py-torch@2.0.0
    - py-tensorflow@2.12.0
    - py-scikit-learn@1.2.0

    # Data science
    - py-numpy@1.24.0
    - py-pandas@2.0.0
    - py-matplotlib@3.7.0
    - py-jupyter@1.0.0

users:
  - name: mluser1
    uid: 5001
    gid: 5001
  - name: mluser2
    uid: 5002
    gid: 5002

data:
  s3_mounts:
    - bucket: ml-training-datasets
      mount_point: /shared/datasets
    - bucket: ml-model-checkpoints
      mount_point: /shared/models
    - bucket: ml-experiment-results
      mount_point: /shared/experiments
```

**Use case:** Deep learning research, model training

**What you get:**
- GPU nodes with CUDA/cuDNN pre-configured
- PyTorch and TensorFlow ready to use
- Jupyter for interactive development
- Datasets and model storage via S3

**Researcher workflow:**
```bash
# SSH to cluster
ssh ec2-user@<head-node>

# Start Jupyter (on head node)
module load py-jupyter
jupyter lab --ip=0.0.0.0

# Or submit training job
sbatch --partition=gpu train_model.sh
```

## Validation Rules

pctl validates templates comprehensively:

### Cluster Validation
- Name must start with letter, alphanumeric and hyphens only, max 60 chars
- Region must be valid AWS region

### Compute Validation
- Head node must be valid instance type format
- At least one queue required
- Queue names must be unique, lowercase, start with letter
- Instance types must be valid format
- Min count >= 0, Max count >= min count, Max count <= 1000

### Software Validation
- Package specs must follow Spack format: `name[@version]`
- No empty package names

### Users Validation
- Usernames must be unique
- UIDs must be unique and > 0
- Usernames must start with letter/underscore, lowercase only
- Warning if UID/GID < 1000 (system range)

### Data Validation
- S3 bucket names must be valid (3-63 chars, lowercase, numbers, hyphens, dots)
- Mount points must be absolute paths
- Mount points must be unique

## Best Practices

### 1. Start with Examples

Don't write templates from scratch. Start with an example and modify it:

```bash
cp seeds/library/bioinformatics.yaml my-cluster.yaml
vim my-cluster.yaml
```

### 2. Validate Early and Often

Validate after each change:

```bash
pctl validate -t my-cluster.yaml
```

### 3. Use Dry Run

Always review what will be created:

```bash
pctl create -t my-cluster.yaml --dry-run
```

### 4. Version Control Your Templates

Templates are code. Use git:

```bash
git add my-cluster.yaml
git commit -m "Add production genomics cluster template"
```

### 5. Software Selection

Include the software your users need:
- **Core toolchain**: gcc, openmpi, python
- **Domain tools**: samtools for bio, pytorch for ML, gromacs for chem
- **Analysis tools**: python, r, jupyter
- **Utilities**: git, cmake, hdf5

### 6. User Management

Use shared GIDs for lab groups:

```yaml
users:
  - name: pi
    uid: 5001
    gid: 5000    # Lab group
  - name: postdoc1
    uid: 5002
    gid: 5000    # Same group
  - name: postdoc2
    uid: 5003
    gid: 5000    # Same group
```

This allows file sharing within the group.

### 7. Data Organization

Organize S3 mounts by purpose:

```yaml
data:
  s3_mounts:
    - bucket: lab-references
      mount_point: /shared/references  # Read-only reference data
    - bucket: lab-inputs
      mount_point: /shared/inputs      # Job input files
    - bucket: lab-results
      mount_point: /shared/results     # Job outputs
    - bucket: lab-scratch
      mount_point: /shared/scratch     # Temporary data
```

### 8. Instance Type Selection

Match instance types to workload:
- **CPU-bound**: c5 family
- **Memory-bound**: r5 family
- **GPU required**: g4dn or p3 family
- **Storage I/O**: i3 family
- **Development/low-cost**: t3 family

### 9. Auto-Scaling Configuration

For cost efficiency, use full auto-scaling:

```yaml
queues:
  - name: compute
    instance_types: [c5.4xlarge]
    min_count: 0    # No nodes when idle
    max_count: 50   # Scale up as needed
```

Nodes launch on-demand and terminate when idle.

### 10. Template Documentation

Add comments to your templates:

```yaml
# Production genomics cluster
# Updated: 2025-01-15
# Contact: pi@university.edu

cluster:
  name: genomics-prod
  region: us-east-1

compute:
  head_node: t3.xlarge

  # Memory-optimized queue for de novo assembly
  queues:
    - name: assembly
      instance_types: [r5.8xlarge]
      min_count: 0
      max_count: 10
```

## Troubleshooting Templates

### Validation Fails

Read error messages carefully:

```
❌ Template validation failed:

3 validation errors:
  - cluster.name must start with a letter
  - compute.queues[0].max_count (5) must be >= min_count (10)
  - users[0].uid 500 is in system range (< 1000)
```

Fix each error and re-validate.

### YAML Syntax Errors

Use proper YAML syntax:
- Use 2-space indentation
- No tabs (use spaces)
- Proper list format with `-`
- Quoted strings if they contain special characters

### Package Not Found

If a Spack package doesn't exist:

```bash
# Search for package
spack list <package-name>

# Check package info
spack info <package-name>
```

Use exact Spack package names.

## Reference

### Valid AWS Regions

- us-east-1 (N. Virginia)
- us-east-2 (Ohio)
- us-west-1 (N. California)
- us-west-2 (Oregon)
- eu-west-1 (Ireland)
- eu-west-2 (London)
- eu-west-3 (Paris)
- eu-central-1 (Frankfurt)
- eu-north-1 (Stockholm)
- ap-northeast-1 (Tokyo)
- ap-northeast-2 (Seoul)
- ap-southeast-1 (Singapore)
- ap-southeast-2 (Sydney)
- ap-south-1 (Mumbai)
- sa-east-1 (São Paulo)
- ca-central-1 (Canada)

### Instance Type Patterns

- `t3.*` - Burstable general purpose
- `m5.*` - General purpose
- `c5.*` - Compute optimized
- `r5.*` - Memory optimized
- `g4dn.*` - GPU (NVIDIA T4)
- `p3.*` - GPU (NVIDIA V100)
- `i3.*` - Storage optimized

### Common Software Packages

See [Spack Package List](https://packages.spack.io/) for complete list.

**Frequently Used:**
- Compilers: gcc, intel-oneapi-compilers
- MPI: openmpi, mpich, intel-oneapi-mpi
- Languages: python, r, julia, perl
- Bio: samtools, bwa, gatk, blast-plus
- ML: py-torch, py-tensorflow
- Chem: gromacs, lammps, quantum-espresso
- Libraries: hdf5, netcdf, fftw, openblas

## See Also

- [Getting Started Guide](GETTING_STARTED.md)
- [Architecture Documentation](ARCHITECTURE.md)
- [Example Templates](../seeds/)
- [Spack Documentation](https://spack.readthedocs.io/)
