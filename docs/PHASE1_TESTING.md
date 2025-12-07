# Phase 1 Workload Testing

## Critical Architecture Validation

Phase 1 testing answers the **fundamental architectural question** for petal:

> **Can compute nodes access software installed at `/opt/spack` on the AMI?**

This determines whether petal's current fast deployment approach (custom AMI with pre-installed software) is viable or if we must immediately pivot to the EBS snapshot architecture.

## Why This Is Critical

From `.github/issues/issue-critical-shared-storage-architecture.md`:

AWS ParallelCluster documentation recommends:
- Software on **shared storage** (`/shared` via EBS/EFS)
- Head node NFS-exports `/shared` to compute nodes
- Use official ParallelCluster AMI (not custom AMI)

But petal currently:
- Installs software to **AMI root volume** (`/opt/spack`)
- Uses custom AMI with pre-installed software
- Achieves 97% faster cluster creation

**The question:** Does our unconventional approach actually work?

## What Phase 1 Tests

### 1. Software Installation (Head Node)
```bash
module avail
module load gcc/11.3.0
gcc --version
```

**Validates:**
- Bootstrap script installed software correctly
- Lmod module system configured properly
- Software accessible from head node

### 2. SLURM Functionality
```bash
sinfo
squeue
sbatch test.sh
```

**Validates:**
- ParallelCluster SLURM configured correctly
- Compute nodes can be reached
- Job scheduler works

### 3. Compute Node Software Access (CRITICAL)
```bash
# Inside SLURM job on compute node:
ls -la /opt/spack
module load gcc/11.3.0
gcc --version
```

**Validates:**
- Compute nodes boot from custom AMI with `/opt/spack` intact
- Software binaries executable from compute nodes
- Environment modules work on compute nodes

## Possible Outcomes

### Outcome A: Success ✅

**If compute nodes can access `/opt/spack`:**
- Current AMI approach is viable
- "97% faster" value proposition valid
- Continue with Phases 2-4 testing
- Document as "unconventional but working" approach

**Why this might work:**
- AWS ParallelCluster boots ALL nodes from the specified custom AMI
- Each compute node gets a full copy of the AMI root volume
- `/opt/spack` exists locally on every compute node
- No NFS sharing needed for software

### Outcome B: Failure ❌

**If compute nodes CANNOT access `/opt/spack`:**
- Must implement EBS snapshot architecture immediately
- Cannot use root volume for software storage
- Must pivot to shared storage approach
- Deployment time will be slower (10-20 min instead of 2-3 min)

**Why this might fail:**
- ParallelCluster may reset compute node root volumes
- Compute nodes may boot from base AMI (not custom)
- `/opt/spack` not accessible from compute nodes
- Need to use `/shared` (EBS volume) instead

## Test Procedure

### Prerequisites
- SSH key uploaded: `petal-testing-key`
- AWS credentials configured with profile: `aws`
- Seed file created: `seeds/testing/workload-basic.yaml`

### Execution
```bash
# Run automated test script
./scripts/phase1-test.sh

# Or manual steps:
petal create --seed seeds/testing/workload-basic.yaml \
  --name workload-test-basic \
  --key-name petal-testing-key

# Wait for CREATE_COMPLETE
petal status workload-test-basic

# SSH and test
petal ssh workload-test-basic

# Test software on head node
module avail
module load gcc/11.3.0
gcc --version

# CRITICAL: Test on compute node via SLURM
cat > test.sh <<EOF
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1
#SBATCH -t 00:02:00

echo "=== Hostname ==="
hostname

echo "=== /opt/spack exists? ==="
ls -la /opt/spack

echo "=== Module system ==="
module avail

echo "=== GCC version ==="
module load gcc/11.3.0
gcc --version
EOF

sbatch test.sh
watch squeue  # Wait for completion

# Check results
cat slurm-*.out

# Cleanup
exit
petal delete workload-test-basic
```

### Success Criteria

**Must have:**
- [x] Cluster creates successfully
- [x] Head node can load modules
- [x] Head node software works (gcc, openmpi, python)
- [x] SLURM job submits and runs
- [x] **Compute node can access `/opt/spack`**
- [x] **Compute node can load modules**
- [x] **Compute node can run GCC**

**Should have:**
- [x] User account created with correct UID/GID
- [x] Total cost < $3
- [x] Creation time < 30 minutes

## Next Steps

### If Phase 1 Succeeds
1. Update documentation with findings
2. Proceed to Phase 2 (MPI testing)
3. Continue with current architecture
4. Add note about "unconventional but validated" approach

### If Phase 1 Fails
1. **STOP** - Do not proceed to Phase 2
2. Implement EBS snapshot architecture immediately
3. Update all seed files to use `/shared/spack`
4. Test new architecture with Phase 1
5. Adjust value proposition claims (slower deployment)

## Architecture Decision Tree

```
Phase 1 Test
    |
    ├─ SUCCESS: Compute nodes CAN access /opt/spack
    |   ├─ Keep current AMI approach
    |   ├─ "97% faster" claim valid
    |   ├─ Proceed to Phase 2-4
    |   └─ Document as validated approach
    |
    └─ FAILURE: Compute nodes CANNOT access /opt/spack
        ├─ Implement EBS snapshot architecture
        ├─ Software moves to /shared (EBS volume)
        ├─ Use official ParallelCluster AMI
        ├─ Deployment time increases to 10-20 min
        └─ Re-test with new architecture
```

## Cost Estimate

| Resource | Duration | Cost |
|----------|----------|------|
| t3.medium head node | 1.5 hours | $0.06 |
| t3.small compute (on-demand) | ~15 minutes | $0.15 |
| EBS volumes (50GB) | 2 hours | $0.02 |
| Data transfer | minimal | $0.01 |
| **Total** | **1.5 hours** | **~$2-3** |

## Timeline

| Step | Duration |
|------|----------|
| Cluster creation | 25-30 minutes |
| SSH + manual testing | 15 minutes |
| SLURM job execution | 5 minutes |
| Result analysis | 10 minutes |
| Cleanup | 5 minutes |
| **Total** | **~60 minutes** |

## Risk Assessment

**Low Risk:**
- Small instance types (cheap)
- Short test duration
- Well-defined pass/fail criteria
- Easy cleanup

**High Value:**
- Validates fundamental architecture
- Answers critical design question
- Informs all future development
- Required before production use

## Documentation Updates

After Phase 1, update:
- [ ] `AWS_TEST_RESULTS.md` - Add Phase 1 results
- [ ] `.github/issues/issue-critical-shared-storage-architecture.md` - Close or escalate
- [ ] `README.md` - Update architecture section if needed
- [ ] `CHANGELOG.md` - Note architecture validation

## Related Issues

- `.github/issues/issue-critical-shared-storage-architecture.md` - Root cause
- `.github/issues/issue-ebs-snapshot-architecture.md` - Alternative if Phase 1 fails

## References

- `WORKLOAD_TESTING_PLAN.md` - Overall testing strategy
- AWS ParallelCluster Custom AMI docs: https://docs.aws.amazon.com/parallelcluster/latest/ug/building-custom-ami-v3.html
- AWS ParallelCluster SharedStorage docs: https://docs.aws.amazon.com/parallelcluster/latest/ug/SharedStorage-v3.html
