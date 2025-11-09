# Getting Started with pctl

This guide will help you get started with pctl (ParallelCluster Templates), a tool for simplified AWS ParallelCluster deployment.

## Prerequisites

Before using pctl, ensure you have:

1. **AWS Account** with appropriate permissions
2. **AWS CLI** configured with credentials
3. **pctl** installed (see Installation below)

### AWS Permissions Required

Your AWS user/role needs permissions for:
- EC2 (instance management, VPC, security groups)
- CloudFormation (stack creation/deletion)
- S3 (for ParallelCluster configuration)
- IAM (for instance roles)

The AWS managed policy `AdministratorAccess` has all required permissions. For production, create a custom policy with minimal required permissions.

## Installation

### Option 1: Download Binary (Recommended)

```bash
# Download the latest release for your platform
curl -LO https://github.com/scttfrdmn/pctl/releases/latest/download/pctl-linux-amd64

# Make it executable
chmod +x pctl-linux-amd64

# Move to your PATH
sudo mv pctl-linux-amd64 /usr/local/bin/pctl

# Verify installation
pctl version
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/scttfrdmn/pctl.git
cd pctl

# Build and install
make build
sudo make install

# Verify installation
pctl version
```

## Configure AWS Credentials

If you haven't already, configure AWS CLI credentials:

```bash
aws configure
```

You'll be prompted for:
- AWS Access Key ID
- AWS Secret Access Key
- Default region (e.g., `us-east-1`)
- Output format (recommended: `json`)

## Your First Cluster

Let's create a simple HPC cluster using pctl.

### Step 1: Choose a Template

pctl includes several example templates. Let's start with the minimal example:

```bash
# View the minimal template
cat templates/examples/minimal.yaml
```

This shows a basic cluster configuration:
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

### Step 2: Validate the Template

Before creating a cluster, validate the template:

```bash
pctl validate -t templates/examples/minimal.yaml
```

You should see: `✅ Template is valid!`

### Step 3: Review What Will Be Created

Use dry-run mode to see what will be created without actually creating resources:

```bash
pctl create -t templates/examples/minimal.yaml --dry-run
```

This shows:
- Cluster configuration
- Compute queue settings
- Software packages (if any)
- Users (if any)
- Data mounts (if any)

### Step 4: Create the Cluster

**Note:** Actual cluster creation is not yet implemented (coming in v0.2.0). For now, pctl validates templates and shows you what would be created.

```bash
# This will validate and show the plan
pctl create -t templates/examples/minimal.yaml
```

When implemented (v0.2.0), this command will:
1. Validate the template
2. Generate ParallelCluster configuration
3. Create VPC and networking (if needed)
4. Launch the head node
5. Configure compute queues
6. Install software (if specified)
7. Create users (if specified)
8. Mount data sources (if specified)

### Step 5: Check Cluster Status

Once cluster creation is implemented:

```bash
pctl status minimal-cluster
```

### Step 6: List All Clusters

```bash
pctl list
```

### Step 7: Connect to the Cluster

Once the cluster is running, you'll receive SSH connection details:

```bash
ssh -i ~/.ssh/your-key.pem ec2-user@<head-node-ip>
```

On the head node, you can:
- Submit jobs with SLURM: `sbatch job.sh`
- Check queue status: `squeue`
- Load software modules: `module load gcc openmpi`
- Monitor compute nodes: `sinfo`

### Step 8: Delete the Cluster

When you're done:

```bash
# With confirmation prompt
pctl delete minimal-cluster

# Skip confirmation
pctl delete minimal-cluster --force
```

## Working with Templates

### Using Example Templates

pctl includes several pre-built templates:

**Minimal** - Simplest possible cluster:
```bash
pctl validate -t templates/examples/minimal.yaml
```

**Starter** - Basic cluster with software and users:
```bash
pctl validate -t templates/examples/starter.yaml
```

**Bioinformatics** - Genomics and bioinformatics tools:
```bash
pctl validate -t templates/library/bioinformatics.yaml
```

**Machine Learning** - GPU instances with PyTorch/TensorFlow:
```bash
pctl validate -t templates/library/machine-learning.yaml
```

**Computational Chemistry** - MD and quantum chemistry:
```bash
pctl validate -t templates/library/computational-chemistry.yaml
```

### Creating Your Own Template

Create a new file `my-cluster.yaml`:

```yaml
cluster:
  name: my-research-cluster
  region: us-east-1

compute:
  head_node: t3.large

  queues:
    - name: compute
      instance_types:
        - c5.2xlarge
        - c5.4xlarge
      min_count: 0
      max_count: 20

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10

users:
  - name: researcher1
    uid: 5001
    gid: 5001

data:
  s3_mounts:
    - bucket: my-research-data
      mount_point: /shared/data
```

Validate your template:

```bash
pctl validate -t my-cluster.yaml
```

If validation fails, pctl will show detailed error messages:

```
❌ Template validation failed:

3 validation errors:
  - cluster.name must start with a letter and contain only alphanumeric characters and hyphens
  - compute.queues[0].max_count must be >= min_count
  - users[0].uid 500 is in system range (< 1000), recommend using 1000 or higher
```

Fix the errors and validate again.

## Template Basics

A pctl template has five main sections:

### 1. Cluster Configuration

```yaml
cluster:
  name: my-cluster      # Must start with letter, alphanumeric and hyphens only
  region: us-east-1     # Valid AWS region
```

### 2. Compute Configuration

```yaml
compute:
  head_node: t3.xlarge  # Instance type for head node

  queues:
    - name: compute     # Queue name (lowercase, starts with letter)
      instance_types:   # List of instance types for this queue
        - c5.2xlarge
        - c5.4xlarge
      min_count: 0      # Minimum nodes (0 for auto-scaling)
      max_count: 20     # Maximum nodes
```

### 3. Software Configuration (Optional)

```yaml
software:
  spack_packages:       # List of Spack packages to install
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10
```

### 4. Users Configuration (Optional)

```yaml
users:
  - name: user1         # Username (lowercase, starts with letter/underscore)
    uid: 5001           # User ID (recommend 1000-60000)
    gid: 5001           # Group ID (recommend 1000-60000)
```

### 5. Data Configuration (Optional)

```yaml
data:
  s3_mounts:
    - bucket: my-bucket         # S3 bucket name
      mount_point: /shared/data # Absolute path on cluster
```

## Advanced Usage

### Custom Cluster Name

Override the cluster name from the template:

```bash
pctl create -t my-template.yaml --name production-cluster
```

### Verbose Output

Get detailed output for debugging:

```bash
pctl validate -t my-template.yaml --verbose
pctl create -t my-template.yaml --verbose
```

### Configuration File

pctl looks for configuration at `~/.pctl/config.yaml`:

```yaml
defaults:
  region: us-east-1
  key_name: my-key-pair

preferences:
  validate_before_create: true
  confirm_destructive: true
```

## Common Workflows

### Workflow 1: Development Cluster

Quick cluster for testing code:

```bash
# Use minimal template
pctl create -t templates/examples/minimal.yaml --name dev-cluster

# When done
pctl delete dev-cluster --force
```

### Workflow 2: Production Research Cluster

Cluster with specific software and users:

```bash
# Create custom template
vim production-cluster.yaml

# Validate
pctl validate -t production-cluster.yaml

# Review with dry-run
pctl create -t production-cluster.yaml --dry-run

# Create
pctl create -t production-cluster.yaml
```

### Workflow 3: Multiple Environments

Manage dev, staging, and production:

```bash
# Development
pctl create -t my-template.yaml --name dev-cluster

# Staging
pctl create -t my-template.yaml --name staging-cluster

# Production
pctl create -t my-template.yaml --name prod-cluster

# List all
pctl list
```

## Troubleshooting

### Validation Errors

If validation fails, read the error messages carefully:

```bash
pctl validate -t my-template.yaml --verbose
```

Common issues:
- Invalid cluster name format
- Invalid AWS region
- Invalid instance type format
- Duplicate queue names
- Duplicate user names or UIDs
- Invalid S3 bucket names
- Relative paths for mount points

### Template Syntax Errors

If YAML parsing fails:
```
Error: failed to parse template: yaml: line 10: mapping values are not allowed in this context
```

Check your YAML syntax:
- Proper indentation (2 spaces)
- Correct key-value format
- No tabs (use spaces)

Use a YAML validator or linter to check syntax.

### AWS Credential Errors

If you see AWS credential errors:

```bash
# Check AWS configuration
aws configure list

# Test AWS access
aws ec2 describe-regions

# Check current identity
aws sts get-caller-identity
```

## Next Steps

- Read the [Template Specification](TEMPLATE_SPEC.md) for complete template options
- Explore the [Architecture Documentation](ARCHITECTURE.md) to understand how pctl works
- Check the [GitHub Issues](https://github.com/scttfrdmn/pctl/issues) for planned features
- Contribute your own templates to the community

## Getting Help

- **Issues**: https://github.com/scttfrdmn/pctl/issues
- **Discussions**: https://github.com/scttfrdmn/pctl/discussions
- **Documentation**: https://github.com/scttfrdmn/pctl/tree/main/docs

## What's Next (Roadmap)

**v0.1.0 - Foundation** (Current)
- ✅ Template system and validation
- ✅ CLI commands (validate, create, list, status, delete)
- ✅ Example templates

**v0.2.0 - AWS Integration**
- AWS ParallelCluster integration
- Actual cluster creation and management
- State management

**v0.3.0 - Software Management**
- Spack installation
- Lmod module system
- Automatic software provisioning

**v0.4.0 - Registry & Capture**
- GitHub-based template registry
- Configuration capture from existing clusters
- Community template sharing

**v1.0.0 - Production Ready**
- All features complete
- Comprehensive testing
- Production documentation
