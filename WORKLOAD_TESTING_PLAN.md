# Workload Testing Plan

## Objective

Validate that petal creates **functional** clusters where users can actually run real workloads, not just infrastructure.

## Critical Gaps from Previous Testing

From `AWS_TEST_RESULTS.md`:
- ✅ Infrastructure creation works
- ✅ Bootstrap script uploads to S3
- ❌ **No SSH access to clusters** - cannot verify software installation
- ❌ **No SLURM job execution** - cannot verify scheduler works
- ❌ **No module system validation** - cannot verify `module load` works
- ❌ **No multi-node jobs** - cannot verify MPI/distributed computing
- ❌ **No S3 data access** - cannot verify data mounts work

## Test Strategy

### Phase 1: Basic Cluster Functionality (Est. $2-3, 1-2 hours)

**Goal**: Verify software installs and basic SLURM works

**Seed**: Create `seeds/testing/workload-basic.yaml`
```yaml
cluster:
  name: workload-test-basic
  region: us-west-2

compute:
  head_node: t3.medium  # Small/cheap head node
  queues:
    - name: compute
      instance_types: [t3.small]  # Smallest practical instance
      min_count: 0
      max_count: 2

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10

users:
  - name: testuser
    uid: 5001
    gid: 5001
```

**Tests**:
```bash
# 1. Create cluster
petal create --seed seeds/testing/workload-basic.yaml \
  --name workload-test-basic \
  --key-name YOUR_KEY

# 2. SSH to head node
petal ssh workload-test-basic

# 3. Verify software installation
module avail              # Should show gcc, openmpi, python
module load gcc
gcc --version             # Should be 11.3.0

module load openmpi
mpirun --version          # Should show OpenMPI 4.1.4

module load python
python3 --version         # Should be 3.10.x

# 4. Verify SLURM
sinfo                     # Should show compute queue
squeue                    # Should be empty

# 5. Run basic SLURM job
cat > test.sh <<EOF
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1
#SBATCH -t 00:02:00

echo "Hello from SLURM"
hostname
module load gcc
gcc --version
EOF

sbatch test.sh
squeue                    # Should show job
# Wait for job to complete
cat slurm-*.out           # Should have output

# 6. Verify user creation
id testuser               # Should show UID 5001, GID 5001

# 7. Delete cluster
petal delete workload-test-basic
```

**Success Criteria**:
- [ ] All modules load without errors
- [ ] Software versions match specifications
- [ ] SLURM job completes successfully
- [ ] User exists with correct UID/GID
- [ ] Total cost < $3

**Estimated Time**: 30 min create + 30 min testing + 15 min cleanup = ~1.5 hours

---

### Phase 2: Multi-Node MPI Job (Est. $3-5, 1-2 hours)

**Goal**: Verify MPI jobs across multiple compute nodes

**Seed**: `seeds/testing/workload-mpi.yaml`
```yaml
cluster:
  name: workload-test-mpi
  region: us-west-2

compute:
  head_node: t3.medium
  queues:
    - name: compute
      instance_types: [c5.xlarge]  # Better CPU for MPI
      min_count: 0
      max_count: 3  # Test 3-node MPI job

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
```

**Tests**:
```bash
# 1. Create cluster
petal create --seed seeds/testing/workload-mpi.yaml \
  --name workload-test-mpi \
  --key-name YOUR_KEY

# 2. SSH and setup MPI test
petal ssh workload-test-mpi

# 3. Create MPI hello world program
cat > mpi_hello.c <<EOF
#include <mpi.h>
#include <stdio.h>

int main(int argc, char** argv) {
    MPI_Init(&argc, &argv);
    int world_size, world_rank;
    MPI_Comm_size(MPI_COMM_WORLD, &world_size);
    MPI_Comm_rank(MPI_COMM_WORLD, &world_rank);
    char processor_name[MPI_MAX_PROCESSOR_NAME];
    int name_len;
    MPI_Get_processor_name(processor_name, &name_len);
    printf("Hello from processor %s, rank %d out of %d processors\n",
           processor_name, world_rank, world_size);
    MPI_Finalize();
    return 0;
}
EOF

# 4. Compile
module load gcc openmpi
mpicc -o mpi_hello mpi_hello.c

# 5. Create SLURM batch script
cat > mpi_test.sh <<EOF
#!/bin/bash
#SBATCH -J mpi-test
#SBATCH -p compute
#SBATCH -n 8         # 8 MPI processes
#SBATCH -N 2         # Across 2 nodes
#SBATCH -t 00:05:00

module load gcc openmpi
mpirun ./mpi_hello
EOF

# 6. Submit job
sbatch mpi_test.sh

# 7. Monitor
watch squeue  # Wait for job to run and complete

# 8. Check results
cat slurm-*.out
# Should show messages from 8 processes across 2 different nodes

# 9. Cleanup
petal delete workload-test-mpi
```

**Success Criteria**:
- [ ] MPI program compiles
- [ ] Job spans multiple nodes (check hostnames in output)
- [ ] All 8 MPI processes execute
- [ ] No MPI communication errors
- [ ] Total cost < $5

**Estimated Time**: ~2 hours

---

### Phase 3: Real Scientific Workload (Est. $5-10, 2-3 hours)

**Goal**: Run actual bioinformatics analysis

**Seed**: `seeds/testing/workload-bio.yaml`
```yaml
cluster:
  name: workload-test-bio
  region: us-west-2

compute:
  head_node: t3.large
  queues:
    - name: compute
      instance_types: [c5.2xlarge]
      min_count: 0
      max_count: 2

software:
  spack_packages:
    - gcc@11.3.0
    - samtools@1.17
    - bwa@0.7.17
    - python@3.10

data:
  s3_mounts:
    - bucket: petal-test-data-{YOUR_ACCOUNT_ID}  # Create this bucket
      mount_point: /shared/data
```

**Setup**:
```bash
# Create test S3 bucket and upload sample data
aws s3 mb s3://petal-test-data-942542972736
aws s3 cp test_sample.fastq s3://petal-test-data-942542972736/input/
```

**Tests**:
```bash
# 1. Create cluster
petal create --seed seeds/testing/workload-bio.yaml \
  --name workload-test-bio \
  --key-name YOUR_KEY

# 2. SSH and verify data access
petal ssh workload-test-bio
ls /shared/data/input/     # Should show test_sample.fastq

# 3. Run bioinformatics workflow
cat > bio_workflow.sh <<EOF
#!/bin/bash
#SBATCH -J bio-test
#SBATCH -p compute
#SBATCH -n 4
#SBATCH -t 00:30:00

module load samtools bwa python

# Test samtools
samtools --version

# Test BWA
bwa

# Process sample data
cd /shared/data
samtools view input/test_sample.bam | head -n 10 > output/test_results.txt
EOF

sbatch bio_workflow.sh
watch squeue

# 4. Verify output
ls /shared/data/output/
cat /shared/data/output/test_results.txt

# 5. Cleanup
petal delete workload-test-bio
aws s3 rb s3://petal-test-data-942542972736 --force
```

**Success Criteria**:
- [ ] S3 bucket mounts correctly at /shared/data
- [ ] Bioinformatics tools run without errors
- [ ] Output files created successfully
- [ ] Can read/write to S3-backed storage
- [ ] Total cost < $10

**Estimated Time**: ~3 hours

---

### Phase 4: Custom AMI Workflow (Est. $10-15, 3-4 hours)

**Goal**: Validate AMI build + fast cluster creation

**Tests**:
```bash
# 1. Build custom AMI with software pre-installed
petal ami build \
  --seed seeds/testing/workload-bio.yaml \
  --name test-bio-ami-v1 \
  --detach

# 2. Monitor build progress
BUILD_ID=$(petal ami list-builds | tail -1 | awk '{print $1}')
petal ami status $BUILD_ID --watch

# 3. Wait for completion (~45-90 minutes)
# Cost: ~$5-8 for build instance time

# 4. Deploy cluster with custom AMI (should be 2-3 minutes)
petal create \
  --seed seeds/testing/workload-bio.yaml \
  --name test-bio-fast \
  --custom-ami ami-XXXXXXXXX \
  --key-name YOUR_KEY

# Time the creation - should be MUCH faster

# 5. SSH and verify software is pre-installed
petal ssh test-bio-fast
module avail  # Should immediately show all software
module load samtools
samtools --version  # Should work instantly (no Spack build time)

# 6. Compare timing
# - Without AMI: 30-90 minutes (Spack builds)
# - With AMI: 2-3 minutes (everything pre-installed)

# 7. Cleanup
petal delete test-bio-fast
aws ec2 deregister-image --image-id ami-XXXXXXXXX
```

**Success Criteria**:
- [ ] AMI build completes successfully
- [ ] Custom AMI contains all software packages
- [ ] Cluster creation with AMI < 5 minutes
- [ ] Software immediately available (no compilation)
- [ ] Total cost < $15

**Estimated Time**: ~4 hours (mostly waiting for AMI build)

---

## Execution Strategy

### Option A: Sequential Testing (Recommended)
1. Start with Phase 1 (basic functionality)
2. If Phase 1 passes, proceed to Phase 2
3. If Phase 2 passes, proceed to Phase 3
4. Phase 4 validates the key value proposition

**Total Cost**: $20-30
**Total Time**: 8-10 hours (mostly waiting)

### Option B: Parallel Testing (Faster)
- Run Phase 1, 2, 3 in parallel in different regions
- Requires monitoring multiple clusters
- Higher cost if failures occur

**Total Cost**: $20-30
**Total Time**: 3-4 hours (parallelized)

### Option C: Minimal Validation (Cheapest)
- Only Phase 1 + Phase 4
- Validates core functionality and key value proposition
- Skips MPI and complex workloads

**Total Cost**: $15-20
**Total Time**: 5-6 hours

---

## Prerequisites

### 1. SSH Key Setup
```bash
# Create SSH key if needed
ssh-keygen -t rsa -b 2048 -f ~/.ssh/petal-test-key

# Import to AWS
aws ec2 import-key-pair \
  --key-name petal-test-key \
  --public-key-material fileb://~/.ssh/petal-test-key.pub \
  --region us-west-2
```

### 2. Test Data Preparation
```bash
# For Phase 3: Create test S3 bucket and sample data
aws s3 mb s3://petal-test-data-${AWS_ACCOUNT_ID}
# Download small FASTQ sample from public dataset
# Upload to test bucket
```

### 3. Budget Alert
```bash
# Set up AWS budget alert for testing
aws budgets create-budget \
  --account-id $AWS_ACCOUNT_ID \
  --budget file://budget.json
```

---

## Success Metrics

### Must Pass (Critical)
- [ ] Software packages install correctly
- [ ] Lmod modules load without errors
- [ ] SLURM jobs execute successfully
- [ ] User accounts created with correct UIDs/GIDs

### Should Pass (Important)
- [ ] MPI jobs span multiple nodes
- [ ] S3 data mounts work correctly
- [ ] Custom AMI reduces cluster creation time by >90%
- [ ] Compute nodes auto-scale on job submission

### Nice to Have (Future)
- [ ] GPU instances work (if tested)
- [ ] Spot instances handle interruptions gracefully
- [ ] Multiple queues work as expected

---

## Risk Mitigation

### Cost Control
1. Set AWS budget alerts
2. Use smallest instance types possible
3. Delete clusters immediately after testing
4. Clean up AMIs and snapshots
5. Test in off-peak hours (if applicable)

### Time Management
1. Use `--detach` for AMI builds
2. Monitor from CLI (no need to stay logged in)
3. Automate cleanup scripts
4. Document timing for future reference

### Failure Handling
1. Check CloudWatch logs if jobs fail
2. SSH to debug software issues
3. Keep detailed notes of errors
4. Create GitHub issues for bugs found

---

## Documentation

After testing, update:
1. `AWS_TEST_RESULTS.md` - Add workload test results
2. `README.md` - Add "Validated Workloads" section
3. `docs/GETTING_STARTED.md` - Add real-world examples
4. GitHub issues - Create issues for any bugs found

---

## Next Steps

1. **Review this plan** - Adjust based on budget/time constraints
2. **Prepare prerequisites** - SSH key, test data, budget alerts
3. **Execute Phase 1** - Start with basic functionality test
4. **Document results** - Record timings, costs, and issues
5. **Create issues** - For any bugs or gaps discovered

---

## Estimated Total Investment

| Phase | Time | Cost | Value |
|-------|------|------|-------|
| Phase 1: Basic | 1.5 hrs | $2-3 | Critical - validates core functionality |
| Phase 2: MPI | 2 hrs | $3-5 | Important - validates distributed computing |
| Phase 3: Bio | 3 hrs | $5-10 | Important - validates real workload |
| Phase 4: AMI | 4 hrs | $10-15 | Critical - validates key value prop |
| **Total** | **10-11 hrs** | **$20-33** | **Complete validation** |

---

## Recommendation

**Execute Option A (Sequential Testing)** starting with Phase 1 this week.

Phase 1 is low-cost, low-risk, and will immediately reveal if there are any critical issues with software installation or module system. If Phase 1 passes, we have high confidence the rest will work.

Would you like to proceed with Phase 1 testing?
