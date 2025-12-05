# pctl AWS Testing Plan

## Objective
Test templates against real AWS infrastructure to validate functionality, identify bugs, and re-prioritize gap issues based on real-world needs.

## Testing Phases

### Phase 1: Minimal Cluster (No Software)
**Goal**: Validate core cluster provisioning without software complexity

**Template**: `seeds/examples/minimal.yaml` (17 lines)
- No software packages
- No users
- No data mounts
- Just head node + compute queue

**Test Cases**:
```bash
# Test 1: Basic creation with auto VPC
pctl validate -t seeds/examples/minimal.yaml
pctl create -t seeds/examples/minimal.yaml --name test-minimal --key-name <key>

# Test 2: With existing subnet
pctl create -t seeds/examples/minimal.yaml --name test-minimal-subnet \
  --key-name <key> --subnet-id <subnet>

# Test 3: Status and lifecycle
pctl status test-minimal
pctl list
pctl delete test-minimal
```

**What to Measure**:
- Creation time (should be fast, no software)
- Cluster becomes CREATE_COMPLETE
- Can SSH to head node
- SLURM is running (`sinfo`, `squeue`)
- Compute nodes can scale up/down
- Delete works cleanly

**Expected Issues**:
- ParallelCluster configuration generation
- VPC auto-creation
- Networking/security groups
- SSH key handling

---

### Phase 2: Starter Cluster (Basic Software)
**Goal**: Test software installation workflow

**Template**: `seeds/examples/starter.yaml` (40 lines)
- 5 basic packages: gcc, openmpi, python, cmake, git
- 1 user
- 1 S3 mount

**Test Cases**:
```bash
# Create starter cluster
pctl create -t seeds/examples/starter.yaml --name test-starter --key-name <key>

# Verify software installation
# SSH and check:
# - module avail (should show gcc, openmpi, python)
# - module load gcc openmpi
# - which mpirun
# - /shared/spack/bin/spack find

# Verify user
# - id user1 (should be UID 5001)

# Verify S3 mount (if bucket exists)
# - ls /shared/data
```

**What to Measure**:
- Total creation time with software
- Spack package installation success
- Lmod module availability
- User creation
- S3 mount functionality

**Expected Issues**:
- Spack buildcache access
- Software installation failures
- Module generation
- User UID/GID conflicts
- S3 IAM permissions

---

### Phase 3: Bioinformatics Template (Real Workload)
**Goal**: Test simplest library template with domain-specific software

**Template**: `seeds/library/bioinformatics.yaml` (83 lines)
- ~10 bioinformatics packages
- Specific instance types
- Production-like configuration

**Test Cases**:
```bash
# Validate first
pctl validate -t seeds/library/bioinformatics.yaml

# Create cluster
pctl create -t seeds/library/bioinformatics.yaml \
  --name test-bio --key-name <key>

# Verify software
# SSH and test:
# - module avail (samtools, bwa, etc.)
# - Run simple bioinformatics command
```

**What to Measure**:
- Creation time (expect 30-90 min without AMI)
- Number of packages that fail
- Which packages take longest
- Total build time
- Usability once running

**Expected Issues**:
- Long installation times (motivation for AMI building)
- Package dependency conflicts
- Specific package versions unavailable
- Spack build failures
- Instance type availability in region

---

### Phase 4: AMI Build Testing
**Goal**: Test AMI building to speed up deployment

**Test Cases**:
```bash
# Build AMI from bioinformatics template
pctl ami build -t seeds/library/bioinformatics.yaml \
  --name bio-test-v1 \
  --subnet-id <subnet> \
  --detach

# Monitor build
pctl ami status <build-id> --watch

# Once complete, use AMI
pctl ami list
pctl create -t seeds/library/bioinformatics.yaml \
  --name test-bio-ami \
  --custom-ami <ami-id> \
  --key-name <key>
```

**What to Measure**:
- AMI build time
- AMI build success rate
- Cluster creation time with AMI (should be 2-3 min)
- Software availability on AMI-based cluster
- AMI size

**Expected Issues**:
- Build instance type insufficient (see Issue #70)
- Build failures mid-process
- AMI not bootable
- Software not preserved correctly
- Lmod modules missing

---

### Phase 5: Advanced Features Testing
**Goal**: Test edge cases and advanced configurations

**Test Cases**:
```bash
# Test machine-learning template (94 lines)
pctl create -t seeds/library/machine-learning.yaml --name test-ml

# Test with multiple queues
# Test with GPU instances (if needed)
# Test with large package counts
# Test different regions
```

---

## Tracking Template

For each test, record:

```markdown
### Test: [Template Name] - [Date]

**Template**: seeds/[path]
**Region**: us-east-1 / us-west-2 / etc.
**Command**:
\`\`\`bash
[exact command used]
\`\`\`

**Results**:
- ✅ or ❌ Creation succeeded
- Duration: [X minutes]
- Cost: ~$X.XX

**Observations**:
- [ ] ParallelCluster config generation worked
- [ ] VPC creation (if auto)
- [ ] Cluster reached CREATE_COMPLETE
- [ ] SSH access successful
- [ ] SLURM operational
- [ ] Software installed correctly
- [ ] Modules available
- [ ] Users created
- [ ] Data mounts working

**Bugs Found**:
1. [Description] - [Severity: Critical/High/Medium/Low]
2. ...

**Gap Issues Validated**:
- Issue #XX became more/less important because...
- Missing feature discovered: ...

**Template Issues**:
- Instance types unavailable in region
- Package version doesn't exist
- Conflicting dependencies
- Template syntax errors
```

---

## Priority Order for Testing

1. **minimal.yaml** - Fastest, validates core
2. **starter.yaml** - Adds software complexity
3. **bioinformatics.yaml** - Simplest real-world template
4. **machine-learning.yaml** - Different domain
5. **computational-chemistry.yaml** - More complex software

**Stop conditions**:
- If minimal.yaml fails, fix core issues first
- If starter.yaml fails on software, investigate Spack integration
- If bioinformatics.yaml works, AMI build testing becomes high priority

---

## Re-prioritization Triggers

Based on testing, re-prioritize gap issues:

**If we discover**:
- Software installation is unreliable → Issue #73 (cost estimation) becomes critical
- AMI builds fail often → Issue #74 (dry-run) becomes urgent
- Hard to debug failures → Issue #76 (logs) becomes high priority
- Need to iterate on templates → Issue #77 (scaffolding) becomes important
- Clusters get stuck → Issue #76 (debugging) critical

**If we learn**:
- Creation time is acceptable → AMI building less urgent
- Software works well → Focus on operational features
- VPC auto-creation problematic → Network management priority increases

---

## Success Criteria

**Phase 1 Success**:
- minimal.yaml creates cluster in <10 minutes
- SSH works, SLURM works
- Delete succeeds

**Phase 2 Success**:
- starter.yaml creates with all 5 packages
- Modules loadable
- User exists with correct UID

**Phase 3 Success**:
- bioinformatics.yaml creates (even if slow)
- All packages install
- Usable for real work

**Phase 4 Success**:
- AMI build completes
- AMI-based cluster deploys in 2-3 minutes
- Software works identically

---

## Test Environment

**AWS Setup**:
- Region: us-west-2 (primary)
- AWS Profile: `aws`
- Key pair: `scofri`
- Subnet: `subnet-0a73ca94ed00cdaf9` (default VPC, us-west-2a, 4086 available IPs)
- VPC Situation: 5/5 quota (must use existing subnets)
- S3 bucket: [TBD] (for data mount tests)

**Local Setup**:
- pctl version: current development (v0.6.0+)
- AWS CLI: 2.31.32
- ParallelCluster CLI: 3.14.0 (in venv at ~/.pctl/venv)
- Credentials: Configured with profile 'aws'

---

## Next Steps

1. Start with minimal.yaml (Phase 1)
2. Document all observations
3. Create bug issues for blocking problems
4. Continue to Phase 2 if Phase 1 succeeds
5. After Phase 3, reassess gap issue priorities
6. Focus on fixing critical bugs before new features
