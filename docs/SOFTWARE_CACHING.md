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

pctl will support multiple caching strategies, from fastest to most flexible:

### Tier 1: Pre-built AMIs (Fastest)

**Concept**: Bake software into custom AMIs, nodes boot ready-to-use.

**How it works:**
```yaml
cluster:
  name: my-cluster
  region: us-east-1

compute:
  head_node: t3.xlarge
  custom_ami: ami-0123456789abcdef  # Pre-built with software

software:
  # Software already in AMI
  use_custom_ami: true
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - samtools@1.17
```

**pctl workflow:**
1. First time: `pctl build-ami -t template.yaml`
   - Launches temporary instance
   - Installs all software
   - Creates AMI
   - Saves AMI ID to template or config
2. Subsequent clusters: Use pre-built AMI
   - Nodes boot in 2-3 minutes with software ready
   - No compilation needed

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

**pctl workflow:**
1. First cluster: Builds from source, uploads to S3
2. Subsequent clusters: Downloads binaries from S3
   - 10-15 minute install time (download + extract)
   - No compilation

**Pros:**
- Shared across clusters
- Automatic: pctl manages cache
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

**pctl workflow:**
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
- `pctl build-ami` command
- Automated AMI creation
- AMI lifecycle management

### v0.6.0 - Container Support
- Singularity/Apptainer integration
- Container-based modules
- ECR integration

## pctl Commands

### Build Custom AMI
```bash
# Build AMI from template
pctl build-ami -t production.yaml --name bio-v1

# Use in template
pctl create -t production.yaml --ami ami-0123456789
```

### Manage Binary Cache
```bash
# Initialize cache
pctl cache init --bucket my-spack-cache

# Check cache status
pctl cache status

# Pre-populate cache
pctl cache build -t template.yaml

# Clear cache
pctl cache clear --older-than 90d
```

### Monitor Build Progress
```bash
# Watch build in real-time
pctl create -t template.yaml --watch

# Show build logs
pctl logs my-cluster --build
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

**Build-from-source:**
- Head node: t3.xlarge ($0.166/hour)
- 6 hour build: $1.00
- Per cluster: $1.00

**Binary cache:**
- S3 storage: $0.023/GB/month
- Typical cache: 50GB = $1.15/month
- Transfer: $0.09/GB (cross-region)
- Shared across unlimited clusters

**Custom AMI:**
- Storage: $0.05/GB/month
- Typical AMI: 50GB = $2.50/month
- No transfer costs
- Shared across unlimited clusters

**Recommendation:** AMI for production (cheapest per cluster)

## Technical Details

### ParallelCluster AMI Integration

ParallelCluster supports custom AMIs:
```yaml
Image:
  CustomAmi: ami-0123456789abcdef
```

pctl will:
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
# pctl build-ami workflow
1. Launch temporary EC2 instance (ParallelCluster AMI base)
2. Install Spack
3. Build all packages from template
4. Generate Lmod modules
5. Run validation tests
6. Create AMI
7. Tag AMI with template hash
8. Terminate temporary instance
9. Output AMI ID
```

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
