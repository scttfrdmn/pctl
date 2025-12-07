# AWS ParallelCluster on Amazon EKS (PCS) Support

## What is PCS?

**AWS ParallelCluster on Amazon EKS** (announced re:Invent 2024) is a managed HPC service that runs on Kubernetes instead of traditional EC2 with SLURM.

### Traditional ParallelCluster vs PCS

| Feature | Traditional ParallelCluster | PCS (ParallelCluster on EKS) |
|---------|----------------------------|------------------------------|
| **Scheduler** | SLURM | Kubernetes + SLURM emulation |
| **Infrastructure** | EC2, CloudFormation | EKS, Kubernetes operators |
| **Management** | Self-managed | AWS-managed control plane |
| **Scaling** | Custom SLURM autoscaling | Kubernetes autoscaling + Karpenter |
| **Software** | Manual install or custom AMIs | Containers (required) |
| **Multi-tenancy** | Limited | Kubernetes namespaces |
| **Networking** | Traditional VPC | EKS networking + CNI |
| **Storage** | EBS, EFS, Lustre | Same + Kubernetes PVCs |
| **Cost** | Pay for EC2 + storage | Pay for EKS + EC2 + storage |

### Key Differences for pctl

**Traditional ParallelCluster** (current petal target):
- Software via Spack on shared NFS
- Lmod modules on all nodes
- SLURM native
- EC2 instances with traditional Linux

**PCS**:
- **Software via containers** (required)
- Pods instead of nodes
- Kubernetes-native workloads
- Container orchestration

## Should petal Support PCS?

**YES - but it requires a different approach.** Here's why and how:

### Pros of Supporting PCS

1. **AWS-Managed Control Plane**
   - No head node to manage
   - HA by default
   - Easier operations

2. **Better Multi-Tenancy**
   - Kubernetes namespaces
   - RBAC built-in
   - Resource quotas

3. **Modern Architecture**
   - Container-based (portable)
   - Cloud-native patterns
   - Better CI/CD integration

4. **Unified Platform**
   - HPC + ML workflows
   - Same infrastructure
   - Share nodes across workloads

5. **Future Direction**
   - AWS is pushing EKS-based HPC
   - Industry trend toward containers
   - Better for hybrid cloud

### Cons of Supporting PCS

1. **Container Requirement**
   - All software must be containerized
   - Different from traditional HPC
   - Learning curve for users

2. **Cost**
   - EKS control plane: ~$73/month
   - Additional complexity

3. **Different Paradigm**
   - Kubernetes concepts vs traditional HPC
   - Different debugging approach
   - New operational model

4. **Maturity**
   - Newer than traditional PC
   - Fewer examples
   - Evolving best practices

## Recommendation: Support Both

petal should support **both traditional ParallelCluster and PCS**, chosen at seed creation time:

```yaml
cluster:
  name: my-cluster
  region: us-east-1
  platform: pcs  # or "parallelcluster" (default)
```

### Use Cases

**Traditional ParallelCluster:**
- Legacy HPC applications
- SLURM-native workflows
- Users familiar with traditional HPC
- Module-based software management
- Maximum performance (no container overhead)

**PCS (EKS-based):**
- Modern HPC + ML workloads
- Container-based workflows
- Multi-tenant environments
- Organizations already using Kubernetes
- Need for HA control plane

## Implementation Strategy

### Phase 1: Traditional ParallelCluster (v0.2-0.4)
Focus on traditional PC first:
- Most mature
- Largest user base
- Proven patterns
- Foundation for PCS later

### Phase 2: Container Preparation (v0.5)
Build container capabilities:
- Container-based software option
- Singularity/Apptainer support
- Container registry integration
- This benefits both platforms

### Phase 3: PCS Support (v0.6-0.7)
Add PCS as alternative platform:
- New provisioner backend
- Kubernetes integration
- EKS cluster management
- SLURM-on-Kubernetes setup

## Seed Design for PCS

### Option A: Unified Seed (Recommended)

One seed format, platform-specific sections:

```yaml
cluster:
  name: my-cluster
  region: us-east-1
  platform: pcs  # or parallelcluster

compute:
  head_node: t3.xlarge  # Ignored for PCS (managed)
  queues:
    - name: compute
      instance_types: [c5.4xlarge]
      min_count: 0
      max_count: 20

# Software definition same for both
software:
  # Traditional PC: Install via Spack
  # PCS: Package in containers
  containers:  # Used for PCS
    - name: bioinformatics
      image: 123456789.dkr.ecr.us-east-1.amazonaws.com/bio:v1
      packages:
        - samtools
        - bwa
        - gatk
  spack_packages:  # Used for traditional PC
    - samtools@1.17
    - bwa@0.7.17
    - gatk@4.3.0

# Users work same way
users:
  - name: researcher1
    uid: 5001
    gid: 5001

# Data mounting similar
data:
  s3_mounts:
    - bucket: my-data
      mount_point: /shared/data
```

### Option B: Separate Seeds

Different seeds for different platforms:

```yaml
# traditional-cluster.yaml
cluster:
  name: my-cluster
  region: us-east-1
  type: parallelcluster

# ...traditional config...
```

```yaml
# pcs-cluster.yaml
cluster:
  name: my-cluster
  region: us-east-1
  type: pcs

kubernetes:
  version: "1.28"
  node_groups:
    - name: compute
      instance_types: [c5.4xlarge]

# ...PCS config...
```

**Recommendation: Option A (unified)** - Easier for users to switch platforms

## Container Strategy for PCS

### Building Containers for HPC

petal should help users containerize their software:

```bash
# Build container from seed
petal build-container -t bioinformatics.yaml

# This creates Dockerfile:
FROM amazonlinux:2023
RUN yum install -y spack
RUN spack install samtools@1.17 bwa@0.7.17 gatk@4.3.0
RUN spack module lmod refresh

# Push to ECR
petal push-container my-bio-stack --ecr my-registry
```

### Container-Native Seeds

For PCS, users can specify pre-built containers:

```yaml
cluster:
  name: pcs-cluster
  platform: pcs

software:
  containers:
    - name: bio-tools
      image: public.ecr.aws/bio/tools:v1.0
      entrypoint: /bin/bash
      modules:
        - samtools/1.17
        - bwa/0.7.17
```

## Hybrid Approach: Best of Both Worlds

Support running both simultaneously:

```yaml
cluster:
  name: hybrid-cluster
  region: us-east-1

platforms:
  # Traditional PC for legacy workflows
  parallelcluster:
    head_node: t3.xlarge
    queues:
      - name: legacy
        instance_types: [c5.4xlarge]

  # PCS for containerized workflows
  pcs:
    queues:
      - name: modern
        instance_types: [c5.4xlarge]
        containers:
          - bio-tools:v1.0
```

Users submit to appropriate queue based on workload.

## Roadmap for PCS Support

### v0.6.0 - Container Foundation
**Goal:** Enable container-based software for traditional PC
- Container building from seeds
- ECR integration
- Singularity/Apptainer support
- Container modules in Lmod

**Deliverables:**
- `petal build-container` command
- Container seeds
- ECR push/pull
- Container-based software option

### v0.7.0 - PCS Alpha
**Goal:** Basic PCS support for early adopters
- EKS cluster provisioning
- SLURM-on-Kubernetes setup
- Basic workload submission
- Single-tenant clusters

**Deliverables:**
- `platform: pcs` in seeds
- EKS provisioner
- PCS documentation
- Migration guide from traditional PC

### v0.8.0 - PCS GA
**Goal:** Production-ready PCS support
- Multi-tenancy (namespaces)
- Advanced networking
- Hybrid traditional+PCS
- Cost optimization

**Deliverables:**
- Multi-tenant seeds
- Hybrid cluster support
- PCS best practices guide
- Performance benchmarks

### v1.0.0 - Unified Platform
**Goal:** Seamless choice between platforms
- Platform abstraction
- Automatic platform selection
- Workload-aware routing
- Cost-aware placement

## Technical Considerations

### EKS Management

PCS requires EKS cluster:
```yaml
# petal manages EKS lifecycle
cluster:
  name: my-pcs-cluster
  platform: pcs

eks:
  version: "1.28"
  managed_node_groups: true
  karpenter: true  # For autoscaling

compute:
  # Translates to Karpenter NodePools
  queues:
    - name: compute
      instance_types: [c5.4xlarge, c5.9xlarge]
```

### SLURM Emulation

PCS provides SLURM compatibility:
- `sbatch` works as expected
- SLURM commands translate to Kubernetes
- Job scripts mostly unchanged
- Some limitations (host networking, etc.)

### Storage for PCS

- **EFS**: Works same as traditional PC
- **FSx Lustre**: Works with CSI driver
- **S3**: Mount via FluidFS or CSI driver
- **EBS**: Kubernetes PVCs

### Networking

- **VPC**: Same VPC concepts
- **Security Groups**: Managed by EKS
- **Node-to-node**: CNI plugin (VPC-CNI, Calico, etc.)
- **External access**: Load balancers

## Migration Path

### From Traditional PC to PCS

```bash
# 1. Containerize software
petal build-container -t traditional-cluster.yaml --output containers/

# 2. Update seed
petal convert --from parallelcluster --to pcs traditional-cluster.yaml

# 3. Test PCS cluster
petal create -t pcs-cluster.yaml --validate-only

# 4. Create PCS cluster
petal create -t pcs-cluster.yaml

# 5. Run parallel for testing
# Keep traditional PC running, test PCS alongside
```

### From PCS to Traditional PC

```bash
# Extract software from containers
petal extract-software --from containers/ --to spack-packages.yaml

# Create traditional PC seed
petal create -t traditional-cluster.yaml
```

## Cost Comparison

### Traditional ParallelCluster
- Head node: t3.xlarge = $0.166/hour = ~$120/month (24/7)
- Compute nodes: On-demand when needed
- Storage: EFS, Lustre as needed
- **Total (idle):** ~$120/month

### PCS
- EKS control plane: ~$73/month (flat fee)
- Managed node groups: Minimal cost when idle
- Compute nodes: Same as traditional
- Storage: Same as traditional
- **Total (idle):** ~$73/month

**Winner:** PCS is cheaper when idle (no head node), but has EKS overhead.

## User Experience

### For Traditional HPC Users

Traditional PC is more familiar:
```bash
ssh ec2-user@head-node
module load samtools
sbatch job.sh
```

### For Kubernetes-Savvy Users

PCS enables new patterns:
```bash
kubectl get nodes
kubectl get pods -n hpc-jobs
helm install job-monitor ./charts/job-monitor
```

## Recommendation Summary

**Yes, plan for PCS support**, but:

1. **v0.2-0.4**: Focus on traditional ParallelCluster (proven, stable)
2. **v0.5-0.6**: Build container capabilities (helps both)
3. **v0.7-0.8**: Add PCS support (future-proof)
4. **v1.0+**: Unified platform with automatic selection

**Priority:** Traditional PC first, PCS as strategic investment for future.

## References

- [AWS ParallelCluster on EKS Announcement](https://aws.amazon.com/blogs/hpc/)
- [EKS for HPC](https://aws.amazon.com/blogs/compute/running-hpc-workloads-on-eks/)
- [Karpenter for HPC](https://karpenter.sh/)
- [SLURM on Kubernetes](https://github.com/sylabs/slurm-operator)

## Issues to Create

For tracking PCS development:

- **v0.6.0**: Container building and ECR integration
- **v0.7.0**: Basic PCS cluster provisioning
- **v0.8.0**: Multi-tenancy and advanced PCS features
- **Research**: PCS vs traditional performance benchmarks
- **Research**: Cost modeling for PCS vs traditional PC
