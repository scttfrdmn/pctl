# Add Progress Reporting for AMI Snapshot Creation Phase

## Problem
During AMI builds, step 6 "Waiting for AMI to be available..." provides no progress updates, leaving users uncertain whether the build is stuck or progressing normally. This phase can take 2-5 minutes with no feedback.

**Current behavior:**
```
6️⃣  Waiting for AMI to be available...
[no updates for several minutes]
```

## Proposed Solution
Add real-time progress reporting for the AMI snapshot creation phase using AWS EBS snapshot progress API.

### Implementation Options

**Option 1: Poll EBS Snapshot Progress** (Recommended)
- Query `describe-snapshots` API for the Progress field (returns percentage)
- Update display with snapshot creation progress: "Creating snapshot... 45% complete"
- Provide visual feedback similar to software installation phase

**Option 2: Elapsed Time Display**
- Show elapsed time: "Waiting for AMI to be available... (3m 15s elapsed)"
- Set expectations: "Typically takes 2-5 minutes"
- Simpler but less informative

**Option 3: Periodic Heartbeats**
- Print periodic status messages so users know it's still working
- "Still waiting for AMI... (checking every 15s)"
- Minimal change, least informative

### Technical Details

AMI creation involves EBS snapshot creation. The snapshot ID can be retrieved from the AMI's `BlockDeviceMappings`:

```go
// Get snapshot ID from AMI
amiResp, _ := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
    ImageIds: []string{amiID},
})
snapshotID := amiResp.Images[0].BlockDeviceMappings[0].Ebs.SnapshotId

// Poll snapshot progress
snapshotResp, _ := ec2Client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
    SnapshotIds: []string{*snapshotID},
})
progress := *snapshotResp.Snapshots[0].Progress // e.g., "75%"
```

### User Experience Goals
- Consistent progress reporting across all AMI build phases
- Clear indication that the build is progressing (not stuck)
- Accurate time estimates when possible
- Visual feedback (progress bar) matching the software installation phase

### Files to Modify
- `pkg/ami/builder.go` - Add snapshot progress polling in `waitForAMI()` or similar method
- Integrate with existing progress bar library (`github.com/schollz/progressbar/v3`)

### Related Work
- #101 - Tag-based progress monitoring (completed for software installation)
- This issue focuses specifically on the AMI snapshot creation phase

## Acceptance Criteria
- [ ] AMI snapshot creation shows real-time progress updates
- [ ] Users receive feedback at least every 15 seconds during snapshot creation
- [ ] Progress display is consistent with software installation phase UX
- [ ] No silent periods > 30 seconds during AMI builds
