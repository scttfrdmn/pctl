# Software Build Caching Strategy

## The Problem

Spack builds from source can take hours or days for complex software stacks:
- GCC: 30-60 minutes
- OpenMPI: 10-20 minutes
- GROMACS: 45-90 minutes
- Full bioinformatics stack: 4-8 hours
- Full chemistry stack with Intel compilers: 12-24 hours

**This makes cluster creation impractical for production use.**

## Solution: Multi-Tier Caching

petal will support multiple caching strategies, from fastest to most flexible:

### Tier 1: Pre-built AMIs (Fastest)

**Concept**: Bake software into custom AMIs, nodes boot ready-to-use.

**How it works:**
```yaml
cluster:
  name: my-cluster
  region: us-east-1

compute:
  head_node: t3.xlarge  # Small instance for running cluster
  custom_ami: ami-0123456789abcdef  # Pre-built with software

software:
  # Software already in AMI
  use_custom_ami: true

  # Build configuration (used during AMI creation)
  build:
    instance_type: c5.18xlarge  # Large instance for fast builds
    spot: true  # Use spot instances to save ~70% on build costs
    timeout: 12h  # Maximum build time

  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
```

**petal workflow:**
1. First time: `petal build-ami -t seed.yaml`
   - Launches **build instance** (c5.18xlarge with 72 cores)
   - Installs all software in parallel (uses all cores)
   - Creates AMI
   - Terminates build instance
   - Saves AMI ID to seed or config
2. Subsequent clusters: Use pre-built AMI
   - Head node launches with small instance (t3.xlarge)
   - Nodes boot in 2-3 minutes with software ready
   - No compilation needed

**Build Instance Selection:**
The build instance is separate from the head node:
- **Build instance**: c5.18xlarge (72 cores) - fast parallel builds
- **Head node**: t3.xlarge (4 cores) - sufficient for scheduler
- **Cost**: Build instance only runs during AMI creation (~2-4 hours)

**Example build times:**
- c5.18xlarge (72 cores): 2 hours for full bio stack
- t3.xlarge (4 cores): 8 hours for same stack
- **Savings: 75% faster, only $7-14 build cost**

**Pros:**
- Fastest: 2-3 minute boot times
- Most reliable: Software tested once
- Lowest cluster creation cost

**Cons:**
- Maintenance: AMIs need updates
- Storage cost: ~$0.50/month per AMI
- Region-specific: Need AMI per region

**Best for:** Production clusters, repeated deployments

### Tier 2: Spack Binary Cache (Fast)

**Concept**: Build once, cache binaries in S3, reuse across clusters.

**How it works:**
```yaml
cluster:
  name: my-cluster
  region: us-east-1

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
  binary_cache:
    s3_bucket: my-spack-cache
    s3_prefix: /caches/bioinformatics
```

**petal workflow:**
1. First cluster: Builds from source, uploads to S3
2. Subsequent clusters: Downloads binaries from S3
   - 10-15 minute install time (download + extract)
   - No compilation

**Pros:**
- Shared across clusters
- Automatic: petal manages cache
- Works across regions (with cross-region S3)
- Updates easier than AMIs

**Cons:**
- Slower than AMI (10-15 min vs 2-3 min)
- S3 transfer costs
- Cache invalidation complexity

**Best for:** Multiple clusters with same software stack

### Tier 3: Container Images (Flexible)

**Concept**: Package software in containers, pull at runtime.

**How it works:**
```yaml
cluster:
  name: my-cluster
  region: us-east-1

software:
  containers:
    - name: bioinformatics
      image: myregistry.io/bio:v1.0
      modules:
        - samtools
        - bwa
        - gatk
```

**petal workflow:**
1. Pre-build container with software
2. Push to ECR/Docker Hub
3. Nodes pull container at boot
4. Singularity/Apptainer wraps for HPC use

**Pros:**
- Portable across cloud/on-prem
- Version controlled
- CI/CD friendly

**Cons:**
- Different paradigm than modules
- Container overhead
- Learning curve for users

**Best for:** Cloud-native workflows, portable workloads

### Tier 4: Build from Source (Most Flexible)

**Concept**: Traditional approach, build everything at cluster creation.

**How it works:**
```yaml
cluster:
  name: my-cluster
  region: us-east-1

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - custom-app@main
  build_parallel: true  # Use all head node cores
```

**Pros:**
- Most flexible: Any package, any version
- Latest code (git branches)
- Custom patches

**Cons:**
- Slowest: Hours to days
- Expensive: Head node running during builds
- Can fail mid-build

**Best for:** Development, custom software, one-off clusters

## Recommended Approach: Hybrid Strategy

For production deployments, use a combination:

```yaml
cluster:
  name: production-bio
  region: us-east-1

compute:
  head_node: t3.xlarge
  # Use custom AMI for base software
  custom_ami: ami-base-bio-v2

software:
  # Base stack in AMI (slow-building packages)
  in_ami:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10
    - r@4.2.0

  # Domain software from binary cache
  spack_packages:
    - samtools@1.17
    - bwa@0.7.17
    - gatk@4.3.0
  binary_cache:
    s3_bucket: lab-spack-cache

  # Custom tools build from source
  build_from_source:
    - lab-pipeline@develop
```

**Result:**
- Base toolchain: Pre-installed (0 minutes)
- Standard tools: Binary cache (10 minutes)
- Custom code: Fresh build (15 minutes)
- **Total: ~25 minutes** vs 4-8 hours

## Implementation Roadmap

### v0.3.0 - Software Management (Spack + Lmod)
- Build from source (traditional)
- Basic Spack installation
- Lmod module generation

### v0.4.0 - Binary Caching
- Spack binary cache to S3
- Automatic cache population
- Cache hit/miss metrics

### v0.5.0 - AMI Builder
- `petal build-ami` command
- Automated AMI creation
- AMI lifecycle management

### v0.6.0 - Container Support
- Singularity/Apptainer integration
- Container-based modules
- ECR integration

## petal Commands

### Build Custom AMI
```bash
# Build AMI from seed (uses build instance from seed)
petal build-ami -t production.yaml --name bio-v1

# Override build instance type
petal build-ami -t production.yaml --build-instance c5.24xlarge

# Use spot instances (default) or on-demand
petal build-ami -t production.yaml --build-instance c5.18xlarge --spot
petal build-ami -t production.yaml --build-instance c5.18xlarge --on-demand

# Watch build progress
petal build-ami -t production.yaml --watch

# Save AMI ID to seed
petal build-ami -t production.yaml --save-to-template

# Use AMI in cluster creation
petal create -t production.yaml  # Uses AMI from seed
petal create -t production.yaml --ami ami-0123456789  # Override
```

### Manage Binary Cache
```bash
# Initialize cache
petal cache init --bucket my-spack-cache

# Check cache status
petal cache status

# Pre-populate cache
petal cache build -t seed.yaml

# Clear cache
petal cache clear --older-than 90d
```

### Monitor Build Progress
```bash
# Watch build in real-time
petal create -t seed.yaml --watch

# Show build logs
petal logs my-cluster --build
```

## Best Practices

### For Development Clusters
- Build from source
- Iterate quickly
- Accept longer creation times

### For Testing/Staging
- Binary cache
- Faster than source
- Validate before production

### For Production
- Custom AMIs
- Fastest boot times
- Tested and stable

### For Shared Infrastructure
- Base AMI + binary cache
- Balance speed and flexibility
- Shared cache across teams

## Cost Considerations

**Build-from-source (head node):**
- Head node: t3.xlarge ($0.166/hour)
- 6 hour build: $1.00
- Per cluster: $1.00
- Slow, ties up head node

**Build-from-source (dedicated build instance):**
- Build instance: c5.18xlarge spot ($0.612/hour spot, ~$2.04/hour on-demand)
- 2 hour build: $1.22 (spot) or $4.08 (on-demand)
- One-time cost per AMI
- 75% faster than head node build

**Binary cache:**
- S3 storage: $0.023/GB/month
- Typical cache: 50GB = $1.15/month
- Transfer: $0.09/GB (cross-region)
- Build once, share across unlimited clusters

**Custom AMI:**
- Initial build: $1.22-$4.08 (one-time, using build instance)
- Storage: $0.05/GB/month
- Typical AMI: 50GB = $2.50/month
- No transfer costs
- Share across unlimited clusters in same region

**Cost Comparison (10 clusters over 3 months):**

| Method | Initial | Monthly | 3 Month Total | Per Cluster |
|--------|---------|---------|---------------|-------------|
| Build on head node | $0 | $10/cluster | $300 | $30 |
| Build instance + AMI | $1.22 | $2.50 | $9.72 | $0.97 |
| Binary cache | $0 | $1.15 | $3.45 | $0.35 |

**Recommendation:**
- **Development**: Build from source on dedicated build instance
- **Production**: AMI for fastest boot, lowest per-cluster cost
- **Shared infrastructure**: Binary cache for maximum flexibility

## Technical Details

### ParallelCluster AMI Integration

ParallelCluster supports custom AMIs:
```yaml
Image:
  CustomAmi: ami-0123456789abcdef
```

petal will:
1. Generate ParallelCluster config with CustomAmi
2. Verify AMI exists and is compatible
3. Bootstrap remaining software on first boot

### Spack Binary Cache Setup

```bash
# On build node
spack mirror add pctl-cache s3://my-bucket/cache
spack buildcache push pctl-cache gcc openmpi samtools

# On compute nodes (automatic)
spack mirror add pctl-cache s3://my-bucket/cache
spack buildcache install gcc openmpi samtools
```

### AMI Creation Process

```bash
# petal build-ami workflow
1. Launch temporary build instance
   - Use instance type from seed (default: c5.18xlarge)
   - Prefer spot instances to save ~70% ($0.61/hr vs $2.04/hr)
   - Based on ParallelCluster AMI
2. Install Spack
3. Build all packages from seed in parallel
   - Use all CPU cores (72 cores on c5.18xlarge)
   - Parallel builds reduce time by 4-8x
4. Generate Lmod modules
5. Run validation tests
6. Create AMI from build instance
7. Tag AMI with seed hash and metadata
8. Terminate build instance
9. Output AMI ID and save to seed

Total time: 2-4 hours (vs 8-24 hours on small head node)
Total cost: $1.22-$4.08 (vs building on every cluster)
```

**Build Instance Types:**

| Instance Type | vCPUs | RAM | Cost/Hour (Spot) | Best For |
|---------------|-------|-----|------------------|----------|
| c5.4xlarge | 16 | 32 GB | $0.17 | Small stacks (< 10 packages) |
| c5.9xlarge | 36 | 72 GB | $0.38 | Medium stacks (10-30 packages) |
| c5.18xlarge | 72 | 144 GB | $0.61 | Large stacks (30+ packages) |
| c5.24xlarge | 96 | 192 GB | $0.82 | Massive stacks with memory needs |

**Recommendation**: c5.18xlarge spot for most builds (best balance of speed and cost)

## Future Enhancements

- **Incremental AMIs**: Layer software updates on base AMIs
- **Multi-region AMI copy**: Automatic AMI replication
- **AMI version tracking**: Git-like versioning for AMIs
- **Cache warming**: Pre-populate cache before peak times
- **Build farms**: Distributed build infrastructure
- **Smart caching**: ML-based prediction of needed packages

## References

- [Spack Binary Caching](https://spack.readthedocs.io/en/latest/binary_caches.html)
- [ParallelCluster Custom AMIs](https://docs.aws.amazon.com/parallelcluster/latest/ug/custom-bootstrap-actions-v3.html)
- [Singularity for HPC](https://sylabs.io/guides/latest/user-guide/)
