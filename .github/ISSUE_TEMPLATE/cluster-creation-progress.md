# Add Progress Reporting for ParallelCluster Creation

## Problem
During cluster creation, users see only a static message: "⏳ This may take 10-15 minutes..." with no real-time progress updates. The cluster creation can take 10-20 minutes, leaving users uncertain whether the process is stuck or progressing normally.

**Current behavior:**
```
🚀 Creating cluster: my-cluster
📝 Generating ParallelCluster configuration...
🔧 Provisioning cluster infrastructure...
⏳ This may take 10-15 minutes...

[no updates for 10-15 minutes]
```

## Proposed Solution
Implement hybrid progress monitoring that combines CloudFormation stack events with cluster status polling for comprehensive real-time feedback.

### Implementation: Option 3 - Hybrid Approach (RECOMMENDED)

**Phase 1: Infrastructure Creation (CloudFormation Events)**
Monitor CloudFormation stack events to show detailed resource creation progress:

```
🚀 Creating cluster: my-cluster

📦 Infrastructure Provisioning:
  ├─ VPC                      CREATE_COMPLETE ✅
  ├─ InternetGateway          CREATE_COMPLETE ✅
  ├─ PublicSubnet             CREATE_COMPLETE ✅
  ├─ PrivateSubnet            CREATE_COMPLETE ✅
  ├─ SecurityGroup            CREATE_IN_PROGRESS
  ├─ IAM Role                 CREATE_IN_PROGRESS
  ├─ HeadNode                 PENDING
  └─ ...

Resources: 15/42 created (36%)
Elapsed: 3m 42s | Estimated: ~6m remaining
```

**Phase 2: Cluster Configuration**
Once infrastructure is up, show cluster-level status:

```
🎯 Cluster Configuration:
  ├─ Head Node Initialization  IN_PROGRESS
  ├─ Slurm Setup               PENDING
  └─ Cluster Ready             PENDING

Status: CREATE_IN_PROGRESS
```

### Technical Implementation

#### 1. Add CloudFormation SDK Dependency
```bash
go get github.com/aws/aws-sdk-go-v2/service/cloudformation
```

#### 2. Create Progress Monitor Module
**File:** `pkg/provisioner/progress.go`

Key functions:
- `monitorClusterCreation(ctx, stackName, region)` - Main orchestrator
- `pollCloudFormationEvents(ctx, stackName, region)` - Track resource creation
- `pollClusterStatus(ctx, clusterName, region)` - Track cluster configuration
- `displayProgress(events, elapsed, estimated)` - Render progress UI

#### 3. AWS APIs to Use
- `cloudformation.DescribeStackEvents` - Get resource creation events
- `cloudformation.DescribeStacks` - Get overall stack status and progress
- `ec2.DescribeInstances` - Monitor head node state
- `pcluster describe-cluster` - Get cluster-level status

#### 4. Progress Calculation
```go
// Resource-based progress (0-70%)
infrastructureProgress := (completedResources / totalResources) * 70

// Cluster status progress (70-100%)
clusterProgress := 70 + calculateClusterConfigProgress(clusterStatus) * 30

// Total progress
totalProgress := infrastructureProgress + clusterProgress
```

#### 5. Update Locations

**pkg/provisioner/parallelcluster.go:303**
- Modify `runPClusterCreate()` to return stack name immediately
- Don't wait for completion synchronously

**pkg/provisioner/parallelcluster.go:55**
- Modify `CreateCluster()` to launch monitoring after initiating creation
- Call `monitorClusterCreation()` after `runPClusterCreate()`

**cmd/pctl/create.go:285**
- Replace static message with progress monitor call

### User Experience Goals
- Consistent progress reporting across infrastructure and configuration phases
- Clear indication of what's being created at each step
- Real-time updates every 10-15 seconds
- No silent periods > 30 seconds during cluster creation
- Accurate time estimates based on completed/remaining resources
- Visual progress bar showing percentage completion

### Resource Categories to Track
**Critical Path Resources (show always):**
- VPC & Networking (InternetGateway, Subnets, RouteTables)
- Security Groups
- IAM Roles & Policies
- EC2 Instances (HeadNode, ComputeNodes)
- EBS Volumes
- CloudWatch LogGroups

**Secondary Resources (summarize):**
- Lambda Functions
- SNS Topics
- SQS Queues
- CloudFormation WaitConditions

### Time Estimation Strategy
```go
// Typical resource creation times (based on AWS averages)
estimatedTimes := map[string]time.Duration{
    "AWS::EC2::VPC":                30 * time.Second,
    "AWS::EC2::InternetGateway":    45 * time.Second,
    "AWS::EC2::Subnet":             60 * time.Second,
    "AWS::EC2::SecurityGroup":      45 * time.Second,
    "AWS::IAM::Role":               60 * time.Second,
    "AWS::EC2::Instance":           3 * time.Minute,
    "AWS::EC2::Volume":             2 * time.Minute,
    "AWS::CloudFormation::WaitCondition": 5 * time.Minute,
}

// Calculate remaining time
remainingTime := sum(estimatedTimes[resource] for resource in pendingResources)
```

### Progress Bar Design
Use existing `github.com/schollz/progressbar/v3` library for consistency with AMI builds:

```
Creating cluster: my-cluster
████████████░░░░░░░░ 65% | 6m 30s elapsed | ~3m 15s remaining
```

### Error Handling
- Show which resource failed if CloudFormation creation fails
- Display rollback progress if stack creation fails
- Provide actionable error messages with AWS console links

### Files to Create/Modify
1. **pkg/provisioner/progress.go** (NEW) - Progress monitoring logic
2. **pkg/provisioner/parallelcluster.go** - Integrate progress monitoring
3. **cmd/pctl/create.go** - Replace static message with monitor
4. **go.mod** - Add CloudFormation SDK dependency

### Related Work
- #101 - Tag-based progress monitoring (COMPLETE ✅)
- #102 - AMI snapshot progress reporting (COMPLETE ✅)
- This issue extends progress monitoring to cluster creation

## Acceptance Criteria
- [ ] CloudFormation resource creation shows real-time progress updates
- [ ] Users receive feedback every 10-15 seconds during cluster creation
- [ ] Progress display shows both infrastructure (0-70%) and configuration (70-100%) phases
- [ ] No silent periods > 30 seconds during cluster creation
- [ ] Time estimates are reasonably accurate (within 30% of actual time)
- [ ] Failed resource creation shows clear error messages
- [ ] Progress bar visually matches AMI build progress UX

## Implementation Phases

### Phase 1: Basic CloudFormation Event Monitoring (MVP)
- Add CloudFormation SDK
- Create basic event polling loop
- Display resource names and statuses
- Show elapsed time

### Phase 2: Progress Calculation & Time Estimates
- Calculate percentage based on completed/total resources
- Implement time estimation logic
- Add progress bar visualization

### Phase 3: Cluster Configuration Monitoring
- Poll cluster status after infrastructure complete
- Show head node initialization progress
- Display Slurm setup status

### Phase 4: Polish & Error Handling
- Add error messages for failed resources
- Show rollback progress on failures
- Optimize polling intervals
- Add comprehensive testing

## Benefits
- Reduces user anxiety during long cluster creation
- Clear visibility into what's being created
- Easy to identify if/where creation fails
- Consistent UX with AMI build progress
- Professional, production-ready tool experience

## Estimated Implementation Time
- Phase 1 (MVP): 2-3 hours
- Phase 2 (Progress calc): 1-2 hours
- Phase 3 (Cluster status): 1-2 hours
- Phase 4 (Polish): 1-2 hours
- Testing & verification: 2-3 hours
- **Total: ~8-12 hours**
