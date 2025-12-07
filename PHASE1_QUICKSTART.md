# Phase 1 Testing - Quick Start

## What This Tests

**THE critical question:** Can compute nodes access software installed at `/opt/spack` on the custom AMI?

This determines if petal's current fast deployment approach is viable.

## Quick Start (5 minutes setup)

### Option 1: Automated Script

```bash
./scripts/phase1-test.sh
```

Then SSH in and run the tests shown by the script.

### Option 2: Manual Steps

```bash
# 1. Create cluster (~25-30 minutes)
petal create --seed seeds/testing/workload-basic.yaml \
  --name workload-test-basic \
  --key-name petal-testing-key

# 2. Check status
petal status workload-test-basic

# 3. SSH when ready
petal ssh workload-test-basic

# 4. Test software on HEAD NODE
module avail
module load gcc/11.3.0
gcc --version

# 5. CRITICAL TEST: Test on COMPUTE NODE
cat > test.sh <<'EOF'
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1
#SBATCH -t 00:02:00

echo "=== Testing /opt/spack access ==="
ls -la /opt/spack || echo "ERROR: /opt/spack not found!"

echo "=== Testing modules ==="
module avail

echo "=== Testing GCC ==="
module load gcc/11.3.0
gcc --version || echo "ERROR: gcc not found!"
EOF

sbatch test.sh
squeue  # Wait for completion
cat slurm-*.out  # Check results

# 6. Cleanup
exit
petal delete workload-test-basic
```

## What Success Looks Like

```
=== Testing /opt/spack access ===
drwxr-xr-x 10 root root 4096 Dec  6 12:34 /opt/spack

=== Testing modules ===
gcc/11.3.0
openmpi/4.1.4
python/3.10

=== Testing GCC ===
gcc (Spack GCC) 11.3.0
```

## What Failure Looks Like

```
=== Testing /opt/spack access ===
ERROR: /opt/spack not found!

=== Testing modules ===
No modules found

=== Testing GCC ===
ERROR: gcc not found!
```

## Next Steps

### ✅ If Success
- Current AMI approach works!
- Continue to Phase 2 (MPI testing)
- "97% faster" claim validated

### ❌ If Failure
- STOP - Implement EBS snapshot architecture
- Move software to `/shared` (EBS volume)
- Re-test Phase 1 with new architecture

## Cost & Time

- **Cost:** ~$2-3
- **Time:** ~60 minutes total
  - 25-30 min: cluster creation
  - 15 min: testing
  - 5 min: cleanup

## Files Created

- `seeds/testing/workload-basic.yaml` - Test seed configuration
- `scripts/phase1-test.sh` - Automated test script
- `docs/PHASE1_TESTING.md` - Detailed documentation

## Questions?

See `docs/PHASE1_TESTING.md` for complete details and architecture decision tree.
