# CRITICAL: ParallelCluster Shared Storage Architecture for Software

## Status: Critical - Requires Investigation

## Priority: CRITICAL - Blocks Production Use

## Problem Statement

**petal currently installs software on `/opt/spack` in custom AMIs, but AWS ParallelCluster best practices recommend installing software on shared storage (EBS/EFS/FSx) mounted at `/shared`, not on the AMI root volume.**

This is a fundamental architectural misalignment that may cause:
1. Software not accessible on compute nodes
2. Inefficient storage usage (software duplicated on every node's root volume)
3. Deviation from AWS best practices
4. Potential issues with node scaling and software availability

## AWS ParallelCluster Documentation

According to [AWS ParallelCluster SharedStorage documentation](https://docs.aws.amazon.com/parallelcluster/latest/ug/SharedStorage-v3.html) and [best practices](https://docs.aws.amazon.com/parallelcluster/latest/ug/best-practices-v3.html):

> **Shared Storage Approach (Recommended):**
> - The /home directory of the head node is shared by default as an NFS share across all compute nodes
> - It is recommended to have an additional shared storage volume attached to the head node during cluster creation
> - Amazon EBS volumes are attached to the head node and shared with compute nodes through NFS
> - This allows software to be available to all compute nodes without being in the AMI

> **Custom AMI Approach (Not Recommended):**
> - "Building a custom AMI is not the recommended approach for customizing AWS ParallelCluster"
> - "Once you build your own AMI, you will no longer receive updates or bug fixes with future releases"
> - "You will need to repeat the steps used to create your custom AMI with each new release"

## Current petal Architecture

### What We Do Now (Incorrect?)

```yaml
# petal installs Spack to /opt/spack in custom AMI
# Bootstrap script runs during AMI build:
- Install Spack to /opt/spack
- Install packages with Spack
- Create Lmod modules
- Bake everything into AMI

# Then deploy cluster with custom AMI
# Problem: Software is on root volume, not shared storage
```

### What ParallelCluster Expects (Correct?)

```yaml
# ParallelCluster expects:
1. Use official ParallelCluster AMI (3.14.0)
2. Define shared storage in cluster config
3. Install software to /shared during bootstrap
4. Shared storage is NFS-mounted to all compute nodes
5. Software automatically available everywhere
```

## Evidence from Our Code

Looking at `pkg/software/spack.go` and `pkg/software/lmod.go`:

```go
// We currently use /opt/spack
InstallPath: "/opt/spack"  // ❌ On root volume (AMI)

// Should probably be:
InstallPath: "/shared/spack"  // ✅ On shared storage (EBS/EFS)
```

Looking at `pkg/config/generator.go:161`:

```go
"MountDir": "/shared",  // We reference /shared
```

But we install Spack to `/opt/spack`, not `/shared/spack`!

## Questions Requiring Investigation

### 1. **Does our current approach actually work?**
   - Are compute nodes seeing `/opt/spack`?
   - Is Spack accessible from compute nodes?
   - Do modules load correctly?
   - **Need to test this in Phase 1 workload testing**

### 2. **Is there a hybrid approach?**
   - Install Spack infrastructure to AMI (`/opt/spack` for binary)
   - Install actual packages to shared storage (`/shared/spack/opt`)
   - Spack supports this with `install_tree` config

### 3. **What's the performance impact?**
   - AMI approach: Fast (everything local)
   - Shared storage approach: Network latency for NFS
   - Which is better for HPC workloads?

### 4. **What's AWS's actual recommendation?**
   - Bootstrap actions on shared storage (official approach)
   - vs. Custom AMI (discouraged but possible)
   - vs. Hybrid (Spack binary in AMI, packages on shared)

## Potential Solutions

### Option A: Full Shared Storage (AWS Recommended)

**Use official ParallelCluster 3.14 AMI + shared storage:**

```yaml
# petal seed
cluster:
  name: my-cluster
  region: us-west-2

compute:
  head_node: t3.xlarge
  queues:
    - name: compute
      instance_types: [c5.4xlarge]
      min_count: 0
      max_count: 10

# Add shared storage configuration
shared_storage:
  - name: spack-storage
    storage_type: Ebs
    mount_dir: /shared
    ebs_settings:
      volume_type: gp3
      size: 500  # GB for software
      encrypted: true
      iops: 3000
      throughput: 125

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
```

**Implementation:**
```go
// pkg/software/spack.go
InstallPath: "/shared/spack"  // On EBS volume

// Bootstrap script installs to /shared
spack_install_script = `
#!/bin/bash
# Install Spack to shared storage
cd /shared
git clone https://github.com/spack/spack.git
cd spack
git checkout v0.23.0

# Install packages (only runs once on head node)
. share/spack/setup-env.sh
spack install gcc@11.3.0
spack install openmpi@4.1.4

# Setup Lmod on shared storage
# Modules available to all compute nodes via NFS
`
```

**Pros:**
- Follows AWS best practices
- Uses official ParallelCluster AMIs (get updates)
- Software automatically shared to compute nodes
- No custom AMI maintenance

**Cons:**
- Slower cluster creation (software installs at cluster create time, not AMI build time)
- Loses petal's "97% faster" value proposition
- Network latency for NFS access
- Bootstrap script runs every cluster creation

### Option B: Hybrid Approach (Best of Both?)

**Spack binary in AMI, packages on shared storage:**

```yaml
# Custom AMI with:
- Spack binary at /opt/spack (lightweight)
- Spack cache/buildcache configured
- Lmod installed
- System dependencies installed

# Shared storage with:
- /shared/spack/opt (actual packages)
- /shared/spack/var (package metadata)
- /shared/lmod/modules (module files)

# Bootstrap script:
- Point Spack to use /shared for install_tree
- Install packages to /shared (but use AMI's Spack binary)
- Generate modules to /shared
```

**Configuration:**
```yaml
# ~/.spack/config.yaml (in custom AMI)
config:
  install_tree:
    root: /shared/spack/opt  # Packages on shared storage
    projections:
      all: '{architecture}/{compiler.name}-{compiler.version}/{name}-{version}-{hash}'
```

**Pros:**
- Faster than full shared storage (Spack binary already built)
- Still follows AWS model (packages on shared storage)
- Compute nodes access via NFS (as expected)
- Can use custom AMI for system dependencies

**Cons:**
- More complex architecture
- Still slower than pure AMI approach
- Need to validate this works with ParallelCluster

### Option C: Pure AMI Approach (Current petal)

**Everything in AMI (what we do now):**

```yaml
# Custom AMI with:
- Spack at /opt/spack
- All packages installed
- All modules configured

# No shared storage needed for software
# Just use ParallelCluster's default /home NFS share
```

**But verify compute nodes can access /opt/spack:**
- Is /opt/spack accessible from compute nodes?
- Or does each compute node get a copy from AMI?
- Does this actually work in practice?

**Pros:**
- Fastest cluster creation (everything pre-installed)
- petal's key value proposition (97% faster)
- No network latency
- Simpler architecture

**Cons:**
- Goes against AWS recommendations
- Software duplicated on every node's root volume
- No automatic updates from AWS
- Need to rebuild AMI for every ParallelCluster release

## Investigation Plan

### Step 1: Test Current Architecture (Phase 1 Workload Testing)

Create cluster with current approach and verify:

```bash
# On head node
module avail
module load gcc
which gcc
ls -la /opt/spack

# On compute node (via sbatch)
sbatch <<EOF
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1

# Can compute node see /opt/spack?
ls -la /opt/spack
module avail
module load gcc
which gcc
gcc --version
EOF
```

**If this works**, our current approach is viable (even if unconventional).

**If this fails**, we MUST move to shared storage.

### Step 2: Research ParallelCluster AMI Behavior

Questions:
- When compute nodes boot from custom AMI, do they have full root volume?
- Or do they mount shared storage for software?
- What's the actual ParallelCluster architecture?

### Step 3: Benchmark Performance

If both approaches work, compare:
- Cluster creation time
- Software access latency
- Job execution performance
- Storage costs

### Step 4: Decision Matrix

| Criterion | AMI Approach | Shared Storage | Hybrid |
|-----------|--------------|----------------|--------|
| Cluster creation speed | ⭐⭐⭐ (2-3 min) | ⭐ (30-90 min) | ⭐⭐ (10-20 min) |
| AWS best practices | ❌ | ✅ | ⚠️ |
| Software access speed | ⭐⭐⭐ (local) | ⭐⭐ (NFS) | ⭐⭐ (NFS) |
| Update maintenance | ❌ (manual) | ✅ (automatic) | ⚠️ (semi-manual) |
| Storage efficiency | ❌ (duplicated) | ✅ (shared) | ✅ (shared) |
| Complexity | ⭐⭐⭐ (simple) | ⭐⭐ (moderate) | ⭐ (complex) |

## Recommended Next Steps

1. **DO NOT PANIC** - Current approach may work fine
2. **Execute Phase 1 workload testing** - Verify compute nodes can access software
3. **Document findings** - Does our approach actually work?
4. **If it works**: Consider it valid even if unconventional
5. **If it fails**: Implement Option A or B immediately

## Impact Assessment

**If our current approach works:**
- ✅ petal's value proposition is intact (97% faster)
- ✅ No architectural changes needed
- ⚠️ Document deviation from AWS recommendations
- ⚠️ Note in documentation that we use custom AMI approach

**If our current approach fails:**
- ❌ Major architectural redesign required
- ❌ "97% faster" claim may be overstated
- ❌ Need to implement shared storage
- ❌ Delays production release

## References

- [AWS ParallelCluster SharedStorage Documentation](https://docs.aws.amazon.com/parallelcluster/latest/ug/SharedStorage-v3.html)
- [AWS ParallelCluster Best Practices](https://docs.aws.amazon.com/parallelcluster/latest/ug/best-practices-v3.html)
- [Custom AMIs with ParallelCluster 3 (AWS Blog)](https://aws.amazon.com/blogs/hpc/custom-amis-with-parallelcluster-3/)
- [Stack Overflow: AWS Parallel Cluster software installation](https://stackoverflow.com/questions/71168100/aws-parallel-cluster-software-installation)
- [Building Custom AMIs](https://docs.aws.amazon.com/parallelcluster/latest/ug/building-custom-ami-v3.html)

## Action Required

**IMMEDIATE**: Execute Phase 1 workload testing to validate current architecture.

**This is a blocking issue for production use until resolved.**
