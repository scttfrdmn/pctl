# Phase 1 Testing - Current Status

**Date:** December 6, 2025
**Status:** IN PROGRESS - Cluster Creating

## Summary

Phase 1 workload testing is underway. The test cluster is currently being created with infrastructure provisioning nearly complete. The bootstrap script will install software packages (gcc, openmpi, python) on the head node, which will take 30-90 minutes.

## Cluster Information

- **Name:** workload-test-basic2
- **Region:** us-east-2
- **Head Node:** t3.medium
- **Compute:** t3.small (min: 0, max: 2)
- **Software:** gcc@11.3.0, openmpi@4.1.4, python@3.10
- **Bootstrap Method:** --force-bootstrap (installing during cluster creation, not using custom AMI)

## Current Progress

### Infrastructure Provisioning: 66% Complete
- **Resources:** 38/40 created
- **Elapsed:** ~3 minutes
- **Remaining:** ~2-3 minutes
- **CloudFormation Stack:** CREATE_IN_PROGRESS

### Next Phase: Software Bootstrap (30-90 minutes)
Once infrastructure is complete, the bootstrap script will:
1. Install Spack to `/opt/spack`
2. Install gcc@11.3.0
3. Install openmpi@4.1.4
4. Install python@3.10
5. Configure Lmod modules
6. Create testuser account

## Issues Resolved

### 1. OS Validation Error (CRITICAL FIX)
**Problem:** ParallelCluster rejected cluster creation with:
```
"message": "[('Image', {'Os': ['Must be one of: alinux2, alinux2023, ubuntu2204, ubuntu2404, rhel8, rocky8, rhel9, rocky9.']})]"
```

**Root Cause:** pkg/config/generator.go line 60 set OS to "al2023" (incorrect)

**Fix:** Changed to "alinux2023" (correct ParallelCluster 3.14.0 value)

**Commit:** `2fbb440` - "fix: correct OS value for ParallelCluster and update branding"

### 2. Version String Branding
**Problem:** `petal version` still displayed "pctl v0.0.0-dev"

**Fix:** Updated internal/version/version.go String() method to use "petal"

**Test Fix:** Updated version_test.go to expect "petal" instead of "pctl"

**Commits:**
- `2fbb440` - version.go fix
- `d1a642d` - version_test.go fix

### 3. VPC Limit in us-west-2
**Problem:** Hit AWS VPC limit (5/5 VPCs used) in us-west-2

**Solution:** Switched to us-east-2 (only 2/5 VPCs used)

**Updated:** seeds/testing/workload-basic.yaml region to us-east-2

## Monitor Progress

Check cluster creation progress:
```bash
# Check background process output
tail -f phase1-creation-fixed.log

# Or check CloudFormation stack directly
AWS_PROFILE=aws aws cloudformation describe-stacks \
  --stack-name workload-test-basic2 \
  --region us-east-2 \
  --query 'Stacks[0].StackStatus'

# Or use petal status (once infrastructure complete)
AWS_PROFILE=aws ./petal status workload-test-basic2
```

## When Cluster is Ready

### Step 1: Verify Head Node Software
```bash
# SSH to cluster
AWS_PROFILE=aws ./petal ssh workload-test-basic2

# Check modules
module avail
# Should show: gcc/11.3.0, openmpi/4.1.4, python/3.10

# Test GCC
module load gcc/11.3.0
gcc --version
# Should show: gcc (Spack GCC) 11.3.0

# Test OpenMPI
module load openmpi/4.1.4
mpirun --version
# Should show: mpirun (Open MPI) 4.1.4

# Test Python
module load python/3.10
python3 --version
# Should show: Python 3.10.x
```

### Step 2: CRITICAL TEST - Compute Node Software Access
This is the **critical architectural validation**. Does the compute node have access to software installed at `/opt/spack` on the head node?

```bash
# Create test job script
cat > test_compute.sh <<'EOF'
#!/bin/bash
#SBATCH -J compute-test
#SBATCH -p compute
#SBATCH -n 1
#SBATCH -t 00:05:00

echo "==================================="
echo "CRITICAL: Testing /opt/spack access"
echo "==================================="
echo ""

echo "Hostname:"
hostname
echo ""

echo "Testing /opt/spack directory:"
if [ -d /opt/spack ]; then
    echo "✅ SUCCESS: /opt/spack exists"
    ls -la /opt/spack
else
    echo "❌ FAILURE: /opt/spack not found"
    exit 1
fi
echo ""

echo "Testing module system:"
module avail
echo ""

echo "Testing GCC:"
module load gcc/11.3.0
if command -v gcc &> /dev/null; then
    echo "✅ SUCCESS: gcc found"
    gcc --version
else
    echo "❌ FAILURE: gcc not found"
    exit 1
fi
echo ""

echo "Testing OpenMPI:"
module load openmpi/4.1.4
if command -v mpirun &> /dev/null; then
    echo "✅ SUCCESS: mpirun found"
    mpirun --version
else
    echo "❌ FAILURE: mpirun not found"
    exit 1
fi
echo ""

echo "==================================="
echo "ALL TESTS PASSED!"
echo "==================================="
EOF

# Submit job
sbatch test_compute.sh

# Monitor job
watch squeue

# Once complete, check results
cat slurm-*.out
```

### Step 3: Interpret Results

**✅ SUCCESS SCENARIO:**
```
✅ SUCCESS: /opt/spack exists
✅ SUCCESS: gcc found
gcc (Spack GCC) 11.3.0
✅ SUCCESS: mpirun found
mpirun (Open MPI) 4.1.4
ALL TESTS PASSED!
```

**Interpretation:** Current AMI approach works! Compute nodes CAN access software at `/opt/spack`.
**Next Steps:**
- Proceed to Phase 2 (MPI testing)
- Continue with current architecture
- "97% faster" value proposition validated

**❌ FAILURE SCENARIO:**
```
❌ FAILURE: /opt/spack not found
❌ FAILURE: gcc not found
❌ FAILURE: mpirun not found
```

**Interpretation:** Current AMI approach DOES NOT work. Compute nodes CANNOT access software at `/opt/spack`.
**Next Steps:**
- **STOP Phase 2 testing**
- Implement EBS snapshot architecture immediately (see `.github/issues/issue-ebs-snapshot-architecture.md`)
- Move software to `/shared` (EBS volume)
- Use official ParallelCluster AMI + EBS snapshots
- Re-test Phase 1 with new architecture

### Step 4: Cleanup
```bash
# Exit SSH
exit

# Delete cluster
AWS_PROFILE=aws ./petal delete workload-test-basic2 --force

# Verify deletion
AWS_PROFILE=aws ./petal list
```

## Cost Tracking

**Estimated costs for this test:**
- t3.medium head node: ~$0.0416/hour × 1.5 hours = $0.06
- t3.small compute (spun up for testing): ~$0.0208/hour × 0.25 hours = $0.01
- EBS volumes (50GB): ~$0.10/GB-month ÷ 720 hours × 1.5 hours = $0.01
- Data transfer: ~$0.01
- **Total:** ~$0.10-0.15 (well under $3 budget)

## Files Modified

- `pkg/config/generator.go` - Fixed OS value from "al2023" to "alinux2023"
- `internal/version/version.go` - Changed "pctl" to "petal" in String() method
- `internal/version/version_test.go` - Updated test to expect "petal"
- `seeds/testing/workload-basic.yaml` - Changed region to us-east-2
- `.gitignore` - Added `/petal` binary

## Git Commits

- `34b6409` - feat: add Phase 1 workload testing infrastructure
- `2fbb440` - fix: correct OS value for ParallelCluster and update branding
- `d1a642d` - fix: update version test to expect 'petal' instead of 'pctl'

## Related Documentation

- `PHASE1_QUICKSTART.md` - Quick reference for running Phase 1
- `docs/PHASE1_TESTING.md` - Detailed Phase 1 documentation
- `WORKLOAD_TESTING_PLAN.md` - Overall testing strategy
- `.github/issues/issue-critical-shared-storage-architecture.md` - Architecture question
- `.github/issues/issue-ebs-snapshot-architecture.md` - Alternative architecture

## Timeline

| Time | Event |
|------|-------|
| 21:40 | Started cluster creation |
| 21:43 | Infrastructure provisioning at 48% |
| 21:45 | Infrastructure provisioning at 66% |
| 21:47 | Infrastructure complete (estimated) |
| 22:00-23:30 | Software bootstrap running |
| 23:30+ | Cluster ready for testing |

**Current time:** ~21:45 (infrastructure 66% complete)
**Infrastructure ETA:** ~21:47 (~2 minutes)
**Bootstrap ETA:** ~22:30-23:30 (45-90 minutes after infrastructure)
**Ready for SSH testing:** ~23:30 (estimated)

## Next Update

Check back in ~1 hour (22:45) to see if bootstrap has completed. The cluster will be in CREATE_IN_PROGRESS until the bootstrap script finishes installing all software packages.

Monitor with:
```bash
tail -f phase1-creation-fixed.log
# Look for "CREATE_COMPLETE" status
```
