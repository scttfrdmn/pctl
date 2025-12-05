# User Personas & Walkthroughs

This document defines key user personas for pctl, their needs, workflows, and how features are prioritized. Use these personas to guide feature development and UX decisions.

---

## Persona 1: The Bioinformatics Researcher

**Name:** Dr. Sarah Chen
**Role:** Computational Biologist
**Institution:** Mid-size research university
**Background:** Expert in genomics, basic Linux/HPC knowledge, limited AWS experience

### Context
Sarah's lab has been using an aging on-premises HPC cluster for genomics pipelines. The cluster is slow, maintenance is expensive, and scaling for large projects is impossible. Leadership has approved moving to AWS.

### Goals
- Migrate existing genomics pipelines (GATK, BWA, samtools) to AWS
- Launch clusters on-demand for large sequencing projects
- Keep costs low by only running when needed
- Share data easily between lab members
- Maintain familiar SLURM workflow

### Pain Points
- Overwhelmed by AWS complexity (VPCs, subnets, security groups)
- Don't know how to install bioinformatics software in cloud
- Need consistent environments across compute nodes
- Worried about data transfer and S3 access
- Limited time to learn new tools

### pctl Solution

#### Initial Setup (One Time)
```bash
# Install pctl
curl -LO https://github.com/scttfrdmn/pctl/releases/latest/download/pctl
chmod +x pctl && sudo mv pctl /usr/local/bin/

# Find a bioinformatics template
pctl registry search bioinformatics
pctl registry get bioinformatics/gatk-pipeline > gatk-cluster.yaml
```

#### Build Custom AMI (30-90 minutes, one time)
```bash
# Build AMI with all genomics software pre-installed
pctl ami build -t gatk-cluster.yaml --name genomics-v1 --detach

# Check progress from laptop later
pctl ami status <build-id> --watch
```

#### Daily Workflow (2-3 minutes per cluster)
```bash
# Launch cluster for sequencing run (uses pre-built AMI)
pctl create -t gatk-cluster.yaml --custom-ami ami-genomics-v1

# Upload data to S3 (auto-mounted in cluster)
aws s3 sync ./fastq-files s3://lab-data/run-2024-03/

# Submit SLURM jobs (software already installed via modules)
ssh head-node
module load gatk bwa samtools
sbatch pipeline.sh

# Delete cluster when done
pctl delete my-cluster
```

### Key Features (Priority Order)
1. **Custom AMIs** (v0.5.0) - Pre-install software, fast clusters
2. **Auto VPC/networking** (v0.2.0) - No AWS networking knowledge needed
3. **S3 mounts** - Easy data access
4. **Template registry** (v0.4.0) - Find working bioinformatics configs
5. **Detached builds** (v0.5.1) - Build AMIs without waiting
6. **Progress monitoring** (v0.5.1) - See build progress

### Success Metrics
- Cluster ready in <5 minutes vs 4+ hours
- Zero AWS networking configuration
- All software working via `module load`
- Data accessible at `/shared/data`
- Same SLURM commands as before

---

## Persona 2: The ML Engineer

**Name:** Alex Rodriguez
**Role:** Machine Learning Engineer
**Company:** AI startup
**Background:** Strong Python/PyTorch, familiar with cloud, needs GPUs

### Context
Alex's team trains large language models and needs elastic GPU capacity. They currently use EC2 instances manually, but want something more repeatable and cost-effective. Need to spin up training clusters frequently for experiments.

### Goals
- Rapid experimentation with different GPU configurations
- Install specific PyTorch/CUDA versions consistently
- Share training data from S3 efficiently
- Track GPU utilization and costs
- Minimize idle GPU time

### Pain Points
- Manual EC2 setup is slow and error-prone
- Installing CUDA/PyTorch from scratch every time (hours)
- Inconsistent environments between experiments
- Hard to reproduce successful training runs
- Wasted money on idle GPUs

### pctl Solution

#### Create ML Training Template
```yaml
cluster:
  name: ml-training
  region: us-west-2

compute:
  head_node: t3.large
  queues:
    - name: gpu-training
      instance_types: [p3.8xlarge, p3.16xlarge]
      min_count: 0
      max_count: 4

software:
  spack_packages:
    - cuda@11.8.0
    - cudnn@8.7.0
    - python@3.10
    - py-torch@2.0.1+cuda
    - py-transformers

data:
  s3_mounts:
    - bucket: ml-training-data
      mount_point: /shared/data
    - bucket: ml-checkpoints
      mount_point: /shared/checkpoints
```

#### Build GPU AMI (90 minutes, one time)
```bash
# Build AMI with CUDA, PyTorch, libraries
pctl ami build -t ml-training.yaml --name ml-gpu-v1 --detach

# Check progress via API/CI
pctl ami status <build-id> --watch
```

#### Training Workflow (2-3 minutes)
```bash
# Launch GPU cluster (software pre-installed)
pctl create -t ml-training.yaml --custom-ami ami-ml-gpu-v1

# Run training (immediate, no waiting)
ssh head-node
module load cuda cudnn python py-torch
sbatch train-llm.sh

# Auto-scales to 4x p3.16xlarge when needed
# Auto-scales down to 0 when idle

# Delete when experiment complete
pctl delete ml-training
```

### Key Features (Priority Order)
1. **Custom AMIs** - Pre-install 2+ hours of CUDA/PyTorch
2. **Fast cluster creation** - Experiment quickly
3. **Auto-scaling** - Only pay for what you use
4. **S3 mounts** - Efficient data access
5. **Detached builds** - Build AMIs in CI/CD
6. **Status monitoring** - Track build progress in automation

### Success Metrics
- Cluster ready in <3 minutes vs 2-3 hours
- Identical environments for reproducibility
- Auto-scale to 0 GPUs when idle (cost savings)
- Training starts immediately after cluster ready

---

## Persona 3: The DevOps Engineer

**Name:** Jamie Kim
**Role:** Platform/DevOps Engineer
**Company:** Biotech company
**Background:** AWS expert, Terraform/CloudFormation, manages infrastructure for science teams

### Context
Jamie supports 5 research teams using AWS. Each team has different software requirements. Teams constantly ask for help setting up clusters, installing software, and debugging issues. Jamie needs to provide self-service infrastructure.

### Goals
- Enable researchers to create their own clusters
- Standardize cluster configurations across teams
- Reduce support burden (currently 10+ hours/week)
- Track cluster usage and costs
- Ensure security and compliance
- Automate everything in CI/CD

### Pain Points
- ParallelCluster configs are complex (100+ lines)
- Researchers can't self-service, always need Jamie's help
- Each team has custom software needs
- Hard to maintain consistent configurations
- Manual cluster creation is error-prone
- No good way to share working configs between teams

### pctl Solution

#### Create Template Library
```bash
# Create team-specific templates
seeds/
  chemistry/
    gromacs.yaml
    quantum-espresso.yaml
  bioinformatics/
    rnaseq.yaml
    genomics.yaml
  ml/
    pytorch-gpu.yaml
```

#### Build AMIs in CI/CD
```yaml
# .github/workflows/build-amis.yml
name: Build AMIs
on:
  push:
    paths: ['seeds/**']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build Chemistry AMI
        run: |
          pctl ami build -t seeds/chemistry/gromacs.yaml \
            --name gromacs-$(date +%Y%m%d) \
            --detach

          # Track build
          BUILD_ID=$(pctl ami list-builds | tail -1 | awk '{print $1}')
          pctl ami status $BUILD_ID --watch
```

#### Self-Service for Researchers
```bash
# Researchers can now self-service
pctl registry search chemistry
pctl create -t chemistry/gromacs --name my-experiment

# Or customize their own template
cp seeds/chemistry/gromacs.yaml my-cluster.yaml
# Edit my-cluster.yaml
pctl create -t my-cluster.yaml
```

### Key Features (Priority Order)
1. **Template registry** (v0.4.0) - Share configs between teams
2. **Custom AMIs** (v0.5.0) - Pre-build software, consistent environments
3. **Detached builds** (v0.5.1) - CI/CD integration
4. **Status monitoring** (v0.5.1) - Programmatic tracking
5. **Auto VPC** (v0.2.0) - Simplify for researchers
6. **Configuration capture** (v0.4.0) - Document existing clusters

### Success Metrics
- 80% reduction in support time (10hr → 2hr/week)
- Researchers can create clusters without help
- Consistent environments across all teams
- All configs version-controlled
- AMIs built automatically in CI/CD

---

## Persona 4: The Research Lab Manager

**Name:** Dr. Michael Thompson
**Role:** Principal Investigator / Lab Manager
**Institution:** Large research university
**Background:** Senior researcher, manages 15 graduate students, budget-conscious

### Context
Michael's lab has a $50K annual AWS budget. Need to support multiple concurrent projects (genomics, proteomics, metabolomics). Students have varying technical skills. Must maximize research output while controlling costs.

### Goals
- Support 3-5 concurrent research projects
- Keep costs predictable and within budget
- Enable students to work independently
- Share expensive software licenses across projects
- Maintain reproducible computational environments

### Pain Points
- Students waste AWS credits on idle resources
- Hard to track which projects use what resources
- Software installation is a time sink for students
- Can't easily share working configurations
- Difficult to reproduce published results later

### pctl Solution

#### One-Time Lab Setup
```bash
# Build lab's standard AMI with common software
pctl ami build -t lab-standard.yaml --name lab-2024 --detach

# Template includes:
# - All common bioinformatics tools
# - Shared Python/R environments
# - Data analysis pipelines
# - All student accounts (consistent UIDs)
```

#### Project-Specific Templates
```bash
# seeds/projects/
project-a-genomics.yaml    # Uses lab-2024 AMI + specific tools
project-b-proteomics.yaml  # Uses lab-2024 AMI + mass spec software
project-c-metabolomics.yaml # Uses lab-2024 AMI + metabolomics stack
```

#### Student Workflow (Self-Service)
```bash
# Student 1: Genomics project
pctl create -t project-a-genomics --custom-ami ami-lab-2024
# Ready in 2 minutes, all software installed

# Student 2: Proteomics project
pctl create -t project-b-proteomics --custom-ami ami-lab-2024
# Ready in 2 minutes, different software

# Both students have their accounts
# Both can access shared lab data in S3
# Both use same module system

# Delete clusters when done (save money)
pctl delete project-a-cluster
```

#### Cost Control
```bash
# List all running clusters
pctl list

# Check what's been running
pctl ami list-builds  # Track AMI usage
```

### Key Features (Priority Order)
1. **Custom AMIs** - One lab standard, many projects
2. **Fast deployment** - Students work independently
3. **User management** - Consistent accounts across projects
4. **S3 mounts** - Shared lab data
5. **Template library** - Project-specific configs
6. **Simple CLI** - Non-expert students can use it

### Success Metrics
- Students create clusters without PI help
- 70% cost reduction (only run when needed)
- Identical software environments across projects
- New students productive in <1 day
- Easy to recreate environments for published papers

---

## Persona 5: The On-Prem Migration Specialist

**Name:** Carlos Mendoza
**Role:** HPC System Administrator
**Organization:** National research lab
**Background:** 20 years managing on-prem clusters, first cloud migration

### Context
Carlos manages a 10,000-core on-premises cluster that's reaching end-of-life. Management wants to migrate to AWS to reduce capital expenses and improve flexibility. Carlos needs to migrate 50+ research groups with minimal disruption.

### Goals
- Capture existing cluster configurations
- Migrate users and workloads smoothly
- Maintain familiar user experience (modules, SLURM)
- Test workloads in AWS before full migration
- Train users on cloud-based workflows

### Pain Points
- Don't know how existing software is configured
- 50+ research groups have different needs
- Users are comfortable with current environment
- Fear of breaking existing workflows
- Need to prove AWS works before committing

### pctl Solution

#### Phase 1: Discovery & Capture
```bash
# SSH to existing on-prem cluster, capture config
pctl capture ssh cluster.example.org \
  --user admin \
  --output onprem-config.yaml

# Captured automatically:
# - Operating system and kernel
# - All Lmod modules and versions
# - User accounts and UIDs/GIDs
# - Storage mounts and paths
# - Batch system configuration
# - Installed compilers and libraries
```

#### Phase 2: Create AWS Template
```bash
# Map on-prem modules to Spack packages
pctl capture analyze onprem-config.yaml > aws-template.yaml

# Review and customize
vim aws-template.yaml

# Result: pctl template matching on-prem environment
```

#### Phase 3: Build Test Cluster
```bash
# Build AMI with all software
pctl ami build -t aws-template.yaml --name migration-test --detach

# Monitor progress
pctl ami status <build-id> --watch

# Create test cluster
pctl create -t aws-template.yaml --custom-ami ami-migration-test

# Test with pilot users
```

#### Phase 4: Gradual Migration
```bash
# Create group-specific templates from base
cp aws-template.yaml chemistry-group.yaml
cp aws-template.yaml biology-group.yaml

# Each group gets customized cluster
# All start from same captured base
# Users see familiar module names and versions
```

### Key Features (Priority Order)
1. **Configuration capture** (v0.4.0) - Discover on-prem setup
2. **Module mapping** (v0.4.0) - Match existing software
3. **Custom AMIs** (v0.5.0) - Recreate on-prem environment
4. **User management** - Preserve UIDs/GIDs
5. **Template library** - Share configs across groups
6. **Detached builds** - Build test AMIs without waiting

### Success Metrics
- Capture on-prem config in <1 hour
- Create equivalent AWS environment
- Users see same `module avail` output
- Same SLURM commands work
- Pilot users can't tell the difference
- Full migration in 6 months vs 18+ months

---

## Feature Prioritization Matrix

Based on persona needs, here's how features are prioritized:

| Feature | Sarah | Alex | Jamie | Michael | Carlos | Priority |
|---------|-------|------|-------|---------|--------|----------|
| Custom AMIs (v0.5.0) | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | **CRITICAL** |
| Auto VPC (v0.2.0) | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐ | **CRITICAL** |
| Software Management (v0.3.0) | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | **CRITICAL** |
| Template Registry (v0.4.0) | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | **HIGH** |
| Detached Builds (v0.5.1) | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐ | ⭐⭐ | **HIGH** |
| Progress Monitoring (v0.5.1) | ⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐ | ⭐⭐ | **HIGH** |
| Config Capture (v0.4.0) | ⭐ | - | ⭐⭐ | - | ⭐⭐⭐ | **MEDIUM** |
| Progress Bars (v0.6.0) | ⭐ | ⭐ | ⭐⭐ | - | ⭐ | **MEDIUM** |
| Time Estimates (v0.6.0) | ⭐⭐ | ⭐⭐ | ⭐ | - | ⭐ | **MEDIUM** |
| Notifications (v0.6.0) | ⭐ | ⭐ | ⭐ | - | - | **LOW** |

⭐⭐⭐ = Critical for persona
⭐⭐ = Important for persona
⭐ = Nice to have for persona
\- = Not relevant for persona

---

## Development Guidelines

### When designing new features, ask:

1. **Which personas benefit most?**
   - Focus on features that help 3+ personas
   - Prioritize researcher-facing features (Sarah, Michael)
   - DevOps features (Jamie) enable scaling

2. **What's the use case?**
   - Write a persona walkthrough first
   - Show concrete commands and workflows
   - Measure impact on persona's goals

3. **Does it reduce complexity?**
   - pctl exists to simplify ParallelCluster
   - Features should reduce user effort, not add
   - When in doubt, prioritize simplicity

4. **Is it production-ready?**
   - Sarah and Michael need reliable tools
   - Alex and Jamie need automation
   - Test with realistic workloads

### Example: Evaluating "Phase 4" Features

**Progress Bars:**
- Sarah: ⭐ Nice while waiting for first build
- Alex: ⭐ Useful for debugging CI/CD
- Jamie: ⭐⭐ Important for monitoring automation
- Michael: - Students use --detach anyway
- Carlos: ⭐ Helpful during testing
- **Decision**: Implement, but lower priority than core features

**Time Estimates:**
- Sarah: ⭐⭐ Helps plan experiment timing
- Alex: ⭐⭐ Helps plan training runs
- Jamie: ⭐ Nice for capacity planning
- Michael: - Not critical for lab operations
- Carlos: ⭐ Useful during migration testing
- **Decision**: Implement together with progress bars

**Desktop Notifications:**
- Sarah: ⭐ Useful if building from laptop
- Alex: - CI/CD handles notifications
- Jamie: - Monitoring tools handle alerts
- Michael: - Not relevant for lab workflow
- Carlos: - Testing is interactive anyway
- **Decision**: Low priority, implement later if requested

---

## Persona-Driven Roadmap

### ✅ v0.5.0 - COMPLETE
All 5 personas can now:
- Create clusters in 2-3 minutes with custom AMIs
- Pre-install software once, use many times
- Deploy without AWS networking knowledge
- Share templates and capture on-prem configs

### 🚧 v0.5.1 - IN PROGRESS (Issue #26)
Enables Alex (ML) and Jamie (DevOps) to:
- Build AMIs in detached mode (CI/CD integration)
- Monitor build progress programmatically
- Track multiple concurrent builds

**Status**: ✅ Phase 1-3 complete, Phase 4 pending

### 📋 v0.6.0 - NEXT (Optional UX)
Improves experience for Sarah and Carlos:
- Visual progress bars during builds
- Time estimates ("~45 minutes remaining")
- Desktop notifications when builds complete

**Impact**: Nice to have, but not critical

### 🔮 v1.0.0 - FUTURE (PCS Support)
Strategic for large-scale deployments:
- AWS ParallelCluster on EKS support
- Multi-cluster management
- Enterprise features

**Impact**: Scales pctl to very large organizations

---

## Conclusion

These personas represent real users who benefit from pctl. When designing features:

1. **Start with a persona** - Who is this for?
2. **Write a walkthrough** - Show the complete workflow
3. **Measure impact** - Does it solve their problem?
4. **Prioritize ruthlessly** - Focus on high-impact features

The matrix above shows current priorities: **Custom AMIs, auto-networking, and software management are critical for all personas**. Async builds enable DevOps/ML workflows. UX enhancements (Phase 4) are valuable but lower priority.

Use this document to guide feature development, UX decisions, and roadmap planning.
