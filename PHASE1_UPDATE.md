# Phase 1 Testing - Update (December 6, 2025)

## Critical Bootstrap Fix Applied

### Issue Discovered
First cluster creation attempt **failed and rolled back** after 15 minutes with error:
```
'fetch_and_run - Failed to execute OnNodeConfigured script 1, return code: 1'
```

### Root Cause
Bootstrap script used `ec2-metadata` command to get instance metadata:
```bash
INSTANCE_ID=$(ec2-metadata --instance-id | cut -d ' ' -f 2)
REGION=$(ec2-metadata --availability-zone | cut -d ' ' -f 2 | sed 's/[a-z]$//')
```

**Problem:** `ec2-metadata` command **does not exist** on Amazon Linux 2023.

### Solution Applied
Replaced with IMDSv2-compatible metadata retrieval:
```bash
TOKEN=$(curl -s -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")
INSTANCE_ID=$(curl -s -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/instance-id)
REGION=$(curl -s -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/region)
```

### Retry Status
**Cluster:** workload-test-basic3 (us-east-2)
**Status:** CREATE_IN_PROGRESS
**Progress:** Infrastructure provisioning at 43%
**ETA:** ~30-90 minutes for full bootstrap

## Issues Fixed Summary

1. **OS Validation Error** (commit: 2fbb440)
   - Changed Image.Os from "al2023" → "alinux2023"
   - ParallelCluster 3.14.0 requires exact OS values

2. **Version Branding** (commits: 2fbb440, d1a642d)
   - Updated "pctl" → "petal" in version output and tests

3. **Bootstrap Script Failure** (commit: ce74688)
   - Replaced ec2-metadata with IMDSv2 curl commands
   - Critical fix for Amazon Linux 2023 compatibility

## Files Modified

- `pkg/config/generator.go` - OS value fix
- `internal/version/version.go` - Branding update
- `internal/version/version_test.go` - Test fix
- `pkg/software/manager.go` - **IMDSv2 metadata retrieval**
- `seeds/testing/workload-basic.yaml` - Region change to us-east-2

## Git Commits

- `34b6409` - feat: add Phase 1 workload testing infrastructure
- `2fbb440` - fix: correct OS value for ParallelCluster and update branding
- `d1a642d` - fix: update version test to expect 'petal' instead of 'pctl'
- `f9ba5ef` - docs: add Phase 1 testing status document
- `ce74688` - **fix: replace ec2-metadata with IMDSv2 for Amazon Linux 2023 compatibility**

## Monitoring Current Attempt

Check progress:
```bash
tail -f phase1-creation-retry.log

# Or check CloudFormation
AWS_PROFILE=aws aws cloudformation describe-stacks \
  --stack-name workload-test-basic3 \
  --region us-east-2 \
  --query 'Stacks[0].StackStatus'
```

## Expected Timeline

| Time | Event |
|------|-------|
| ~23:43 | Started cluster creation (workload-test-basic3) |
| ~23:50 | Infrastructure complete (estimated) |
| ~00:30-01:30 | Software bootstrap running (30-90 min) |
| ~01:30 | Cluster ready for testing (estimated) |

## Next Steps When Ready

1. SSH to cluster: `AWS_PROFILE=aws ./petal ssh workload-test-basic3`
2. Verify head node software: `module avail`, `gcc --version`
3. **CRITICAL TEST:** Submit SLURM job to compute node to test `/opt/spack` access
4. Interpret results: Success = continue to Phase 2, Failure = implement EBS snapshots

## Learning: Amazon Linux 2023 Changes

Amazon Linux 2023 made several breaking changes from AL2:
- No `ec2-metadata` command (use IMDSv2 with curl)
- IMDSv2 required by default (need token for metadata access)
- Different package names and system tools

These changes require careful testing of bootstrap scripts to ensure AL2023 compatibility.
