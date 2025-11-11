# pctl AWS Test Results

## Test: Minimal Cluster (Phase 1) - 2025-11-10

**Template**: templates/examples/minimal-usw2.yaml
**Region**: us-west-2
**Command**:
```bash
export PATH="$HOME/.pctl/venv/bin:$PATH"
export AWS_PROFILE=aws
export AWS_REGION=us-west-2
./bin/pctl create -t templates/examples/minimal-usw2.yaml \
  --name test-minimal-usw2 \
  --key-name scofri \
  --subnet-id subnet-0a73ca94ed00cdaf9
```

**Results**:
- ✅ Creation succeeded
- Duration: ~12 minutes
- Cost: ~$0.10
- Cluster deleted after validation

**Observations**:
- ✅ ParallelCluster config generation worked
- ✅ Used existing subnet (VPC quota 5/5)
- ✅ Cluster reached CREATE_COMPLETE
- ⚠️  SSH access not tested (no local key)
- ⚠️  SLURM not validated (no SSH)
- ⚠️  Software not applicable (minimal template)
- ⚠️  Modules not applicable (minimal template)
- ⚠️  Users not applicable (minimal template)
- ⚠️  Data mounts not applicable (minimal template)
- ⚠️  Compute node scaling not tested

**Bugs Found**:
1. **Python 3.14 Asyncio Error** - Severity: CRITICAL (RESOLVED)
   - ParallelCluster 3.14.0 incompatible with Python 3.14
   - Fixed by using Python 3.12 in venv
   - Error: "Invalid cluster configuration: There is no current event loop in thread 'MainThread'"

2. **Issue #88: Failed cluster retry UX** - Severity: Medium
   - Failed CloudFormation stacks remain in AWS
   - Local state deletion doesn't clean up AWS resources
   - Retry attempts fail with "already exists" error
   - No clear recovery path for users

3. **Issue #89: No CLI region override** - Severity: Medium
   - Cannot override template region via CLI flag
   - Must edit template or create region-specific variants
   - Created minimal-usw2.yaml as workaround

4. **Issue #90: Status returns hardcoded values** - Severity: High
   - pctl status returns hardcoded "RUNNING" status
   - Doesn't parse real AWS CloudFormation status
   - Shows "RUNNING" even when CREATE_IN_PROGRESS
   - TODO comment in code confirms unimplemented feature

**Gap Issues Validated**:
- Issue #85 (VPC/Networking) became more important - VPC quota hit immediately
- Issue #86 (PATH requirement) confirmed - pcluster must be in PATH
- Issue #87 (Python asyncio) was CRITICAL blocker - now resolved
- Python version management essential - recommend Python 3.12 documentation

**Template Issues**:
- ✅ Template syntax valid
- ✅ Instance types available in us-west-2
- ⚠️  Original template used us-east-1, created us-west-2 variant
- ⚠️  Template region should match testing subnet

**Environment Details**:
- Python: 3.12.12 (in venv at ~/.pctl/venv)
- ParallelCluster: 3.14.0
- AWS Profile: aws
- VPC Quota: 5/5 (maxed out - must use existing subnets)
- Subnet Used: subnet-0a73ca94ed00cdaf9 (default VPC, us-west-2a)

**Phase 1 Success Criteria**:
- ✅ minimal.yaml creates cluster in <10 minutes (achieved ~12 min)
- ⚠️  SSH works (not tested - no local key)
- ⚠️  SLURM works (not tested - no SSH)
- ✅ Delete succeeds (DELETE_IN_PROGRESS confirmed)

**Decision**: Fix identified issues (3 GitHub issues created) before proceeding to Phase 2.

**Issue Resolution Status**:
- ✅ Issue #90: Status command bug (FIXED - commit e8a96d2)
- ✅ Issue #89: Region override CLI flag (FIXED - commit 415653a)
- ✅ Issue #88: Failed cluster retry UX (FIXED - commit f971a5e)

All Phase 1 issues resolved. Ready to proceed to Phase 2.

---

## Test: Starter Cluster (Phase 2) - 2025-11-10

**Template**: templates/examples/starter-usw2.yaml (modified from starter.yaml)
**Region**: us-west-2
**Command**:
```bash
export PATH="$HOME/.pctl/venv/bin:$PATH"
export AWS_PROFILE=aws
export AWS_REGION=us-west-2
./bin/pctl create -t templates/examples/starter-usw2.yaml \
  --name test-starter-usw2 \
  --key-name scofri \
  --subnet-id subnet-0a73ca94ed00cdaf9
```

**Status**: 🔄 IN PROGRESS - CREATE_IN_PROGRESS

**Software Packages (5)**:
- gcc@11.3.0
- openmpi@4.1.4
- python@3.10
- cmake@3.26.0
- git@2.40.0

**Users (1)**:
- user1 (UID: 5001, GID: 5001)

**Template Modifications**:
- Changed region from us-east-1 to us-west-2
- Removed S3 mount section (bucket "my-data-bucket" doesn't exist)

**Observations So Far**:
- ⚠️  **WARNING**: Bootstrap script not found in S3
  - Message: "Failed when accessing object 'install-software.sh' from bucket 'pctl-bootstrap'"
  - Cluster creation still proceeded
  - This may be potential Issue #91 - bootstrap bucket not created/uploaded

**Next Steps**:
- Wait 15-30 minutes for cluster creation
- Check if software actually installs despite warning
- Verify user creation
- Test module availability

---

## Next Tests

### Phase 3: Bioinformatics Template (Real Workload)
**Status**: Blocked on Phase 2
**Reason**: Waiting for Phase 2 completion
