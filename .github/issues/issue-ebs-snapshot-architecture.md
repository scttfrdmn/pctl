# Feature: EBS Snapshot Architecture for Software Storage

## Status: Planned - High Priority

## Priority: High - Solves Critical Architecture Issue

## Overview

Implement EBS volume snapshots for software storage instead of custom AMIs. This aligns with AWS ParallelCluster best practices while maintaining petal's speed advantage.

**Key Insight**: Template → EBS Volume Build → EBS Snapshot → Fast Deploy

## Problem This Solves

From Issue: CRITICAL Shared Storage Architecture

Current approach:
- ❌ Install software to `/opt/spack` on AMI root volume
- ❌ Deviates from AWS best practices
- ❌ Software may not be shared properly to compute nodes
- ❌ Custom AMI maintenance burden

New approach:
- ✅ Install software to `/shared/spack` on EBS volume
- ✅ Follows AWS ParallelCluster architecture
- ✅ Head node NFS-exports `/shared` to compute nodes (standard)
- ✅ Use official ParallelCluster AMI (always up-to-date)
- ✅ Same speed via EBS snapshots

## Architecture

### Build Phase (One-time, ~45-90 min)

```
┌──────────────────────────────────────┐
│ 1. Parse Template                    │
│    bioinformatics.yaml               │
│    - gcc@11.3.0                      │
│    - samtools@1.17                   │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 2. Launch Build Instance             │
│    - Official ParallelCluster AMI    │
│    - c6a.4xlarge (fast builds)       │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 3. Create & Attach EBS Volume        │
│    - 500GB gp3                       │
│    - Attach to /dev/sdf              │
│    - Format ext4                     │
│    - Mount to /shared                │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 4. Install Software                  │
│    cd /shared                        │
│    git clone spack                   │
│    spack install gcc@11.3.0          │
│    spack install samtools@1.17       │
│    Generate Lmod modules             │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 5. Create EBS Snapshot               │
│    aws ec2 create-snapshot           │
│    → snap-0abc123def456               │
│    Tag: petal:fingerprint=sha256...  │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 6. Cleanup                           │
│    Detach volume                     │
│    Terminate build instance          │
│    Delete temporary volume           │
└──────────────────────────────────────┘
```

### Deploy Phase (Fast, ~2-3 min)

```
┌──────────────────────────────────────┐
│ 1. Lookup Snapshot                   │
│    Compute fingerprint from template │
│    Query AWS: tags.petal:fingerprint │
│    → snap-0abc123def456               │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 2. Generate ParallelCluster Config   │
│    SharedStorage:                    │
│      - Name: software                │
│        StorageType: Ebs              │
│        MountDir: /shared             │
│        EbsSettings:                  │
│          SnapshotId: snap-0abc...    │
│          VolumeType: gp3             │
│          Size: 500                   │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 3. ParallelCluster Create            │
│    - Use official AMI                │
│    - Create EBS from snapshot        │
│    - Attach to head node at /shared  │
│    - NFS-export /shared (automatic)  │
└────────────┬─────────────────────────┘
             ↓
┌──────────────────────────────────────┐
│ 4. Compute Nodes Boot                │
│    - Official AMI                    │
│    - NFS-mount /shared from head     │
│    - Software immediately available  │
│    - module load gcc (works!)        │
└──────────────────────────────────────┘
```

## Implementation Details

### 1. Update AMI Builder → EBS Builder

**Old**: `pkg/ami/builder.go`
**New**: `pkg/ebs/builder.go`

```go
package ebs

type Builder struct {
    EC2Client  *ec2.Client
    Region     string
    InstanceID string
}

type SoftwareVolume struct {
    SnapshotID  string
    Size        int
    VolumeType  string
    Fingerprint string
}

func (b *Builder) BuildSoftwareVolume(tmpl *template.Template) (*SoftwareVolume, error) {
    // 1. Launch build instance with official ParallelCluster AMI
    instance := b.launchBuildInstance()

    // 2. Create EBS volume
    volume := b.createVolume(&ec2.CreateVolumeInput{
        Size:       aws.Int32(500),
        VolumeType: "gp3",
        Iops:       aws.Int32(3000),
        Throughput: aws.Int32(125),
    })

    // 3. Attach and mount
    b.attachVolume(instance.InstanceID, volume.VolumeID, "/dev/sdf")
    b.runSSHCommand(instance, "sudo mkfs.ext4 /dev/sdf")
    b.runSSHCommand(instance, "sudo mkdir -p /shared")
    b.runSSHCommand(instance, "sudo mount /dev/sdf /shared")
    b.runSSHCommand(instance, "sudo chmod 755 /shared")

    // 4. Generate and run bootstrap script
    script := b.generateBootstrapScript(tmpl)
    b.runBootstrapScript(instance, script)

    // 5. Create snapshot
    snapshot := b.createSnapshot(volume.VolumeID, tmpl.Cluster.Name)

    // 6. Tag with fingerprint
    fingerprint := tmpl.ComputeFingerprint()
    b.tagSnapshot(snapshot.SnapshotID, fingerprint)

    // 7. Cleanup
    b.detachVolume(volume.VolumeID)
    b.terminateInstance(instance.InstanceID)
    b.deleteVolume(volume.VolumeID)

    return &SoftwareVolume{
        SnapshotID:  *snapshot.SnapshotID,
        Size:        500,
        VolumeType:  "gp3",
        Fingerprint: fingerprint.Hash,
    }, nil
}
```

### 2. Bootstrap Script Updates

**Old**: Install to `/opt/spack`
**New**: Install to `/shared/spack`

```bash
#!/bin/bash
set -e

# Install system dependencies
sudo yum groupinstall -y "Development Tools"
sudo yum install -y git python3 gcc gcc-c++ gcc-gfortran

# Clone Spack to /shared
cd /shared
sudo git clone -c feature.manyFiles=true https://github.com/spack/spack.git
cd spack
sudo git checkout v0.23.0

# Set permissions
sudo chown -R ec2-user:ec2-user /shared/spack

# Source Spack
. share/spack/setup-env.sh

# Configure buildcache
spack mirror add aws-binaries https://binaries.spack.io/releases/v0.23
spack buildcache keys --install --trust

# Install packages
spack install gcc@11.3.0
spack install openmpi@4.1.4
spack install samtools@1.17

# Install Lmod
spack install lmod

# Generate modules
spack module lmod refresh --delete-tree -y

# Create environment setup script
sudo tee /etc/profile.d/z00_spack.sh > /dev/null << 'EOF'
export SPACK_ROOT=/shared/spack
if [ -f "$SPACK_ROOT/share/spack/setup-env.sh" ]; then
    . $SPACK_ROOT/share/spack/setup-env.sh
fi
EOF
```

### 3. Update Config Generator

**pkg/config/generator.go**

```go
func (g *Generator) GenerateParallelClusterConfig(tmpl *template.Template, snapshotID string) (string, error) {
    config := map[string]interface{}{
        "Region": tmpl.Cluster.Region,
        "Image": map[string]interface{}{
            "Os": "al2023",  // Official ParallelCluster AMI
        },
        "HeadNode": g.generateHeadNode(tmpl),
        "Scheduling": g.generateScheduling(tmpl),
        "SharedStorage": []map[string]interface{}{
            {
                "Name":        "software",
                "StorageType": "Ebs",
                "MountDir":    "/shared",
                "EbsSettings": map[string]interface{}{
                    "SnapshotId": snapshotID,  // From EBS build
                    "VolumeType": "gp3",
                    "Size":       500,
                    "Iops":       3000,
                    "Throughput": 125,
                    "Encrypted":  true,
                },
            },
        },
    }

    return yaml.Marshal(config)
}
```

### 4. Update State Management

**pkg/state/state.go**

```go
type ClusterState struct {
    Name              string
    Region            string
    Status            string
    SnapshotID        string    // NEW: EBS snapshot ID
    HeadNodeIP        string
    SeedFile          string
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

### 5. Update CLI Commands

**cmd/petal/ami.go** → **cmd/petal/build.go**

```go
var buildCmd = &cobra.Command{
    Use:     "build",
    Aliases: []string{"bake", "prepare"},
    Short:   "Build software volume from seed",
    Long: `Build a software volume (EBS snapshot) with pre-installed packages.

This is a one-time operation that takes 45-90 minutes but enables
2-3 minute cluster deployments forever.`,
    RunE: runBuild,
}

func init() {
    buildCmd.Flags().StringVar(&buildSeed, "seed", "", "seed file (required)")
    buildCmd.Flags().StringVar(&buildName, "name", "", "build name for tracking")
    buildCmd.Flags().BoolVar(&buildDetach, "detach", false, "run in background")
    buildCmd.MarkFlagRequired("seed")
}

func runBuild(cmd *cobra.Command, args []string) error {
    // Load template
    tmpl := template.LoadFromFile(buildSeed)

    // Create EBS builder
    builder := ebs.NewBuilder(cfg.Defaults.Region)

    if buildDetach {
        // Launch async build
        buildID := builder.BuildAsync(tmpl, buildName)
        fmt.Printf("🌸 Software volume build started\n")
        fmt.Printf("   Build ID: %s\n", buildID)
        fmt.Printf("   Track with: petal build status %s\n", buildID)
    } else {
        // Synchronous build with progress
        volume, err := builder.Build(tmpl, buildName)
        if err != nil {
            return err
        }
        fmt.Printf("✅ Software volume built: %s\n", volume.SnapshotID)
    }
}
```

**cmd/petal/create.go**

```go
func runCreate(cmd *cobra.Command, args []string) error {
    // Load template
    tmpl := template.LoadFromFile(seedFile)

    // Compute fingerprint
    fingerprint := tmpl.ComputeFingerprint()

    // Lookup existing snapshot
    snapshotID, err := findSnapshotByFingerprint(fingerprint.Hash)
    if err != nil {
        return fmt.Errorf("no software volume found for this seed\n" +
            "Build one first with: petal build --seed %s", seedFile)
    }

    fmt.Printf("📦 Using software volume: %s\n", snapshotID)

    // Generate config with snapshot
    config := generator.GenerateParallelClusterConfig(tmpl, snapshotID)

    // Create cluster
    provisioner.CreateCluster(clusterName, config)
}
```

## User Workflow

### Build Software Volume (Once)

```bash
# Build software volume from seed
petal build --seed bioinformatics.yaml --name bio-v1

# Or run in background
petal build --seed bioinformatics.yaml --name bio-v1 --detach

# Monitor progress
petal build status <build-id> --watch

# List all software volumes
petal build list
```

Output:
```
🌸 Building software volume from bioinformatics.yaml...

Build Phase:
✅ Launched build instance (i-0abc123)
✅ Created EBS volume (vol-0def456) - 500GB gp3
✅ Attached and mounted to /shared
⏳ Installing software (this takes 45-90 minutes)...
   [████████████████░░░░] 80% - Installing samtools@1.17

✅ Software installation complete
✅ Created snapshot (snap-0ghi789)
✅ Tagged with fingerprint: sha256:abc123...
✅ Cleaned up build resources

Software Volume Ready!
  Snapshot ID: snap-0ghi789
  Fingerprint: sha256:abc123def456...
  Packages: gcc@11.3.0, openmpi@4.1.4, samtools@1.17

Deploy clusters with:
  petal create --seed bioinformatics.yaml --name my-cluster
```

### Deploy Cluster (Fast)

```bash
# Create cluster using pre-built software volume
petal create --seed bioinformatics.yaml --name research-cluster
```

Output:
```
🌸 Creating cluster: research-cluster

📦 Found software volume: snap-0ghi789
✅ Generated ParallelCluster config
⏳ Creating cluster infrastructure...
   [████████████████████] 100% - Cluster ready!

✅ Cluster created in 2 minutes 34 seconds

SSH: petal ssh research-cluster
Status: petal status research-cluster

Software available via:
  module avail
  module load gcc openmpi samtools
```

### Automatic Snapshot Discovery

```bash
# petal automatically finds the right snapshot
petal create --seed bioinformatics.yaml --name cluster-1
# Uses: snap-0abc123 (bio packages)

petal create --seed machine-learning.yaml --name cluster-2
# Uses: snap-0def456 (ML packages)

# Different package versions = different snapshots
petal create --seed bio-updated.yaml --name cluster-3
# Builds new snapshot if packages changed
```

## Benefits

### Speed
- **Build once**: 45-90 minutes (background, one-time)
- **Deploy forever**: 2-3 minutes per cluster
- **Public snapshots**: ZERO build time (use pre-built official snapshots)
- **Same as custom AMI approach** but cleaner architecture

### AWS Alignment
- ✅ Uses official ParallelCluster AMI (always current)
- ✅ Software on `/shared` (AWS best practice)
- ✅ Head node NFS-exports (standard ParallelCluster)
- ✅ No custom AMI maintenance

### Flexibility
- Size EBS per cluster (override snapshot size)
- Attach multiple volumes if needed
- Easy to manage and update
- Clean separation: infrastructure vs software
- **Snapshot sharing**: Share across accounts or publicly

### Cost
- Only pay for EBS when cluster running
- Delete cluster = delete EBS (automatic)
- Snapshot storage is cheap (~$0.05/GB/month)
- No duplicate AMI storage

## Migration from Current Approach

### Phase 1: Parallel Support (v1.x)
```go
// Support both approaches
if customAMI != "" {
    // Old way: custom AMI
    useCustomAMI(customAMI)
} else if snapshotID != "" {
    // New way: EBS snapshot
    useEBSSnapshot(snapshotID)
}
```

### Phase 2: Deprecate Custom AMI (v2.0)
```go
// Warn about custom AMI
if customAMI != "" {
    fmt.Printf("⚠️  Warning: Custom AMI is deprecated\n")
    fmt.Printf("   Use 'petal build' instead\n")
}
```

### Phase 3: Remove Custom AMI (v3.0)
- Only support EBS snapshot approach
- Cleaner codebase
- Simpler documentation

## Acceptance Criteria

- [ ] EBS builder creates and snapshots software volumes
- [ ] Bootstrap script installs to /shared/spack
- [ ] Fingerprint tagging works for snapshots
- [ ] Config generator includes SharedStorage with SnapshotId
- [ ] Cluster creation uses snapshot (not custom AMI)
- [ ] Software accessible from compute nodes via NFS
- [ ] `petal build` command implemented
- [ ] `petal build list` shows all software volumes
- [ ] `petal build status` monitors async builds
- [ ] Documentation updated
- [ ] Tests validate end-to-end workflow
- [ ] Migration guide from custom AMI approach

## Testing Plan

### 1. Build Software Volume
```bash
petal build --seed seeds/testing/workload-basic.yaml --name test-build-1

# Verify:
# - EBS volume created (500GB gp3)
# - Software installed to /shared/spack
# - Snapshot created and tagged
# - Build resources cleaned up
```

### 2. Deploy from Snapshot
```bash
petal create --seed seeds/testing/workload-basic.yaml --name test-cluster

# Verify:
# - Snapshot found via fingerprint
# - ParallelCluster config has SharedStorage
# - EBS created from snapshot
# - Mounted at /shared on head node
```

### 3. Validate Software Access
```bash
petal ssh test-cluster

# On head node:
ls /shared/spack
module avail
module load gcc
gcc --version

# On compute node (via SLURM):
sbatch <<EOF
#!/bin/bash
#SBATCH -J test
#SBATCH -p compute
#SBATCH -n 1

ls /shared/spack
module avail
module load gcc
gcc --version
EOF
```

### 4. Performance Testing
```bash
# Time cluster creation
time petal create --seed bioinformatics.yaml --name perf-test

# Should be 2-3 minutes with pre-built snapshot
```

## Documentation Updates

### README.md
```markdown
## How petal Works

petal uses EBS snapshots for lightning-fast cluster deployment:

1. **Build Once** (45-90 minutes, background)
   ```bash
   petal build --seed bioinformatics.yaml --name bio-v1 --detach
   ```
   - Installs all software to `/shared` on EBS volume
   - Creates snapshot for instant reuse

2. **Deploy Forever** (2-3 minutes per cluster)
   ```bash
   petal create --seed bioinformatics.yaml --name my-cluster
   ```
   - Creates EBS from snapshot (instant!)
   - Head node NFS-exports `/shared`
   - Software immediately available on all nodes

**Result**: 97% faster than installing software at cluster create time!
```

### docs/ARCHITECTURE.md
Add detailed explanation of EBS snapshot architecture vs custom AMI approach.

## Estimated Effort

- EBS builder implementation: 8 hours
- Config generator updates: 2 hours
- CLI command changes: 3 hours
- State management updates: 2 hours
- Testing: 4 hours
- Documentation: 3 hours
- **Total: 22-24 hours**

## Related Issues

- Issue: CRITICAL Shared Storage Architecture (RESOLVES)
- Issue: AOCC and Intel Compiler Support (benefits from this)
- Workload Testing Plan (validates this approach)

## Public Snapshot Registry (Future Feature)

### Overview

EBS snapshots can be shared publicly or with specific AWS accounts, enabling a **petal official software registry** for instant cluster deployment.

According to [AWS documentation](https://docs.aws.amazon.com/ebs/latest/userguide/ebs-modifying-snapshot-permissions.html), snapshots can be:
- ✅ Shared publicly (all AWS accounts)
- ✅ Shared privately (specific accounts)
- ✅ Copied to other regions

### Architecture Tiers

#### Tier 1: Private Snapshots (Default, Phase 1)
```bash
# User builds their own snapshots (encrypted, private)
petal build --seed bioinformatics.yaml --name bio-v1
# → Creates snapshot in user's account
# → Only that user can access
# → Encrypted by default
```

#### Tier 2: Organization Sharing (Phase 2)
```bash
# Share with specific AWS accounts
petal build --seed bioinformatics.yaml \
  --share-with 123456789012,987654321098

# → Creates snapshot
# → Shares with specified accounts only
# → Encrypted with customer-managed key (CMK)
```

#### Tier 3: Public Official Registry (Phase 3)
```bash
# petal maintains official public snapshots
petal registry list --official

Official Software Volumes:
  bioinformatics-v1     snap-0abc123  us-east-1  ⭐⭐⭐⭐⭐
  machine-learning-v1   snap-0def456  us-west-2  ⭐⭐⭐⭐⭐
  chemistry-v1          snap-0ghi789  eu-west-1  ⭐⭐⭐⭐⭐

# Use official snapshot (ZERO build time!)
petal create --seed bioinformatics.yaml --name my-cluster
# → petal automatically uses public snapshot
# → Instant deployment, no 45-90 min build needed
```

### User Workflow with Public Registry

```bash
# First time user - no build needed!
petal create --seed bioinformatics.yaml --name research-1 --region us-east-1

# Behind the scenes:
# 1. Compute fingerprint from seed → sha256:abc123...
# 2. Check local snapshots in AWS → not found
# 3. Fetch GitHub registry index
# 4. Find matching fingerprint → bioinformatics-v1
# 5. Filter snapshots: region=us-east-1, os=al2023 (from seed)
# 6. Select best match → snap-0abc123
# 7. Use AWS snapshot → instant cluster!

# Output:
📦 Using official software volume: bioinformatics-v1
   Snapshot: snap-0abc123 (us-east-1, AL2023, 500GB)
   Source: petal-official (AWS account 999888777666)
✅ Cluster created in 2 minutes 18 seconds

# User can override OS
petal create --seed bioinformatics.yaml --name research-2 --os ubuntu2404

# Output:
📦 Using official software volume: bioinformatics-v1
   Snapshot: snap-0def456 (us-east-1, Ubuntu 24.04, 500GB)
   Source: petal-official (AWS account 999888777666)
✅ Cluster created in 2 minutes 18 seconds
```

**Flow:**
```
Seed → Fingerprint → GitHub Registry → Filter by Region/OS → Snapshot ID → AWS EBS
bioinformatics.yaml → sha256:abc123... → bioinformatics-v1.json →
  filter(region=us-east-1, os=al2023) → snap-0abc123 → EBS in AWS
```

**Multiple Snapshots Support:**
```
Same seed + Same region + Different OS = Different snapshots
bioinformatics.yaml (us-east-1):
  - AL2023:      snap-0abc123
  - Ubuntu 24.04: snap-0def456
  - RHEL 9:      snap-0ghi789

Same seed + Same region + Different size = Different snapshots
bioinformatics.yaml (us-east-1, AL2023):
  - Full (500GB):   snap-0abc123
  - Minimal (250GB): snap-0xyz789
```

### Public Registry Implementation

**petal Official Account:**
```yaml
# petal-owned AWS account maintains public snapshots
account: 999888777666  # petal-official
snapshots:
  bioinformatics-v1:
    snapshot_id: snap-0abc123
    regions: [us-east-1, us-west-2, eu-west-1]
    packages:
      - gcc@11.3.0
      - openmpi@4.1.4
      - samtools@1.17
      - bwa@0.7.17
      - gatk@4.3.0
    size: 500GB
    public: true
    verified: true
    updated: 2025-01-15
```

**GitHub Registry Repository (Metadata Only):**

The GitHub repo stores **metadata** about snapshots, NOT the snapshots themselves. The actual EBS snapshots live in AWS.

```
https://github.com/scttfrdmn/petal-snapshots
├── README.md
├── snapshots/
│   ├── index.json              # Master index (snapshot directory)
│   ├── bioinformatics-v1.json  # Snapshot metadata (regions, IDs)
│   ├── machine-learning-v1.json
│   └── chemistry-v1.json
└── seeds/                      # Associated seed templates
    ├── bioinformatics-v1.yaml
    ├── machine-learning-v1.yaml
    └── chemistry-v1.yaml
```

**What's in the GitHub repo:**
- ✅ Snapshot metadata (which regions, snapshot IDs)
- ✅ Fingerprint/hash to match seed templates
- ✅ Seed template files
- ✅ Package lists, descriptions

**What's NOT in GitHub:**
- ❌ Actual EBS snapshots (those are in AWS)
- ❌ Software binaries (those are in the snapshots)

**snapshots/index.json:**
```json
{
  "version": "1.0",
  "registry": "https://github.com/scttfrdmn/petal-snapshots",
  "snapshots": [
    {
      "name": "bioinformatics-v1",
      "fingerprint": "sha256:abc123...",
      "metadata_url": "https://raw.githubusercontent.com/scttfrdmn/petal-snapshots/main/snapshots/bioinformatics-v1.json"
    },
    {
      "name": "machine-learning-v1",
      "fingerprint": "sha256:def456...",
      "metadata_url": "https://raw.githubusercontent.com/scttfrdmn/petal-snapshots/main/snapshots/machine-learning-v1.json"
    }
  ]
}
```

**snapshots/bioinformatics-v1.json (Metadata):**
```json
{
  "name": "bioinformatics-v1",
  "description": "Genomics and bioinformatics software stack",

  "fingerprint": "sha256:abc123def456...",
  "seed_url": "https://raw.githubusercontent.com/scttfrdmn/petal-snapshots/main/seeds/bioinformatics-v1.yaml",

  "snapshots": [
    {
      "region": "us-east-1",
      "snapshot_id": "snap-0abc123",
      "os": "al2023",
      "size_gb": 500,
      "description": "Full stack - Amazon Linux 2023",
      "architecture": "x86_64"
    },
    {
      "region": "us-east-1",
      "snapshot_id": "snap-0def456",
      "os": "ubuntu2404",
      "size_gb": 500,
      "description": "Full stack - Ubuntu 24.04",
      "architecture": "x86_64"
    },
    {
      "region": "us-east-1",
      "snapshot_id": "snap-0ghi789",
      "os": "al2023",
      "size_gb": 250,
      "description": "Minimal stack - Core tools only",
      "architecture": "x86_64"
    },
    {
      "region": "us-west-2",
      "snapshot_id": "snap-0jkl012",
      "os": "al2023",
      "size_gb": 500,
      "description": "Full stack - Amazon Linux 2023",
      "architecture": "x86_64"
    },
    {
      "region": "us-west-2",
      "snapshot_id": "snap-0mno345",
      "os": "ubuntu2404",
      "size_gb": 500,
      "description": "Full stack - Ubuntu 24.04",
      "architecture": "x86_64"
    },
    {
      "region": "eu-west-1",
      "snapshot_id": "snap-0pqr678",
      "os": "al2023",
      "size_gb": 500,
      "description": "Full stack - Amazon Linux 2023",
      "architecture": "x86_64"
    }
  ],

  "packages": [
    "gcc@11.3.0",
    "openmpi@4.1.4",
    "samtools@1.17",
    "bwa@0.7.17",
    "gatk@4.3.0"
  ],

  "aws_account": "999888777666",
  "public": true,
  "verified": true,
  "created": "2025-01-15",
  "updated": "2025-01-20"
}
```

**Multiple Snapshots Per Region:**

This supports multiple snapshot variants per region:
- **Different OS**: AL2023, Ubuntu 24.04, RHEL 9
- **Different sizes**: Full stack (500GB), Minimal (250GB)
- **Different architectures**: x86_64, arm64
- **Different configurations**: Debug builds, optimized builds

**Selection Logic:**
```go
// User preferences
region := "us-east-1"
os := "al2023"        // Default or from seed
size := 500           // Default or user override

// Find matching snapshot
for _, snap := range metadata.Snapshots {
    if snap.Region == region &&
       snap.OS == os &&
       snap.SizeGB >= size {
        return snap.SnapshotID
    }
}
```

The actual EBS snapshots are stored in AWS, not GitHub. GitHub just maps configurations to snapshot IDs.

**Benefits of GitHub Registry:**
- ✅ Version controlled
- ✅ Easy to update (git push)
- ✅ Community contributions via PR
- ✅ Free hosting
- ✅ Built-in review process (PR reviews)
- ✅ Issue tracking for snapshots
- ✅ Same pattern as seed registry

**User Config:**
```yaml
# ~/.petal/config.yaml
registry:
  use_public_snapshots: true  # Default: false for security
  trusted_accounts:
    - 999888777666  # petal-official
    - 123456789012  # my-organization
```

### Snapshot Sharing Commands

```bash
# Share with specific accounts
petal build share snap-0abc123 \
  --accounts 123456789012,987654321098

# Make public (for petal team only)
petal build share snap-0abc123 --public

# Copy to other regions
petal build copy snap-0abc123 \
  --from us-east-1 \
  --to us-west-2,eu-west-1

# List shared snapshots
petal build list --shared

# Revoke sharing
petal build unshare snap-0abc123 --accounts 123456789012
```

### Security Considerations

#### For petal Official Snapshots:
- ✅ **Unencrypted** (required for public sharing)
- ✅ **Only open-source software** (no proprietary packages)
- ✅ **Security scanned** before publishing
- ✅ **CVE monitored** and updated regularly
- ✅ **No sensitive data** (verified clean)
- ✅ **Clearly documented** contents

#### For User Snapshots:
- ✅ **Private by default** (encrypted)
- ⚠️ **Be cautious sharing** (review contents)
- ❌ **Never share with sensitive data**
- ✅ **Use Block Public Access** in AWS settings

#### AWS Security Features:
```bash
# Enable Block Public Access for EBS snapshots (recommended)
aws ec2 enable-snapshot-block-public-access --region us-east-1

# Prevents accidental public sharing
```

### Community Contributions

Future feature: Allow community to contribute snapshots

```bash
# Submit community snapshot
petal build submit snap-0abc123 \
  --name "genomics-pipeline-v2" \
  --description "Advanced genomics pipeline with latest tools" \
  --tags bioinformatics,genomics,ngs

# petal team reviews and verifies
# If approved: added to community registry
# Users can opt-in to community snapshots
```

### Implementation Phases

#### Phase 1: Private Snapshots (Immediate)
- Users build their own snapshots
- Encrypted by default
- No sharing

#### Phase 2: Team/Organization Sharing (6 months)
```go
// pkg/ebs/sharing.go
func (s *Snapshot) ShareWithAccounts(accountIDs []string) error {
    // Modify snapshot permissions
    // Share with specific accounts only
}
```

#### Phase 3: Public Official Registry (12 months)
- petal team maintains official snapshots
- Public registry infrastructure
- Multi-region replication
- Community contribution process

### Benefits of Public Registry

1. **Zero Build Time**: Users deploy instantly with pre-built snapshots
2. **Curated Stacks**: Tested, verified software combinations
3. **Community**: Share successful configurations
4. **Cost Savings**: No need to rebuild common stacks
5. **Best Practices**: Official snapshots follow HPC best practices

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Security vulnerabilities | Regular CVE scanning and updates |
| Unintended data exposure | Strict review process, automated checks |
| Account compromise | Separate official account, limited access |
| Snapshot costs | Lifecycle policies, deprecate old versions |
| Regional availability | Multi-region replication |

### Configuration Options

```yaml
# User preferences for snapshot sources
snapshot_sources:
  # Check order (stops at first match)
  - type: local        # User's own snapshots
    priority: 1

  - type: organization # Organization-shared
    account: 123456789012
    priority: 2

  - type: official     # petal-official
    account: 999888777666
    priority: 3
    enabled: true      # Opt-in for security

  - type: community    # Community snapshots
    priority: 4
    enabled: false     # Opt-in required
```

### Example User Experience

**Without Public Registry:**
```bash
# User must build (45-90 minutes)
$ petal build --seed bioinformatics.yaml --name bio-v1
⏳ Building software volume... (this takes 45-90 minutes)
   [████████░░░░░░░░] 50% - Installing samtools...
```

**With Public Registry:**
```bash
# User deploys instantly
$ petal create --seed bioinformatics.yaml --name my-cluster

📦 Using official software volume: bioinformatics-v1
   Source: petal-official (snap-0abc123)
   Packages: gcc, openmpi, samtools, bwa, gatk
   Last updated: 2025-01-15

✅ Cluster created in 2 minutes 18 seconds
```

### CLI Commands for Registry

```bash
# List official snapshots
petal registry snapshots list

# Search for snapshots
petal registry snapshots search bioinformatics

# Show snapshot details
petal registry snapshots info bioinformatics-v1

# List available regions
petal registry snapshots regions bioinformatics-v1

# Check if snapshot exists for seed
petal registry snapshots check --seed bioinformatics.yaml

# Update registry cache (fetch from GitHub)
petal registry snapshots update
```

### Registry Implementation (GitHub-based)

```go
// pkg/registry/snapshots.go
type SnapshotRegistry struct {
    Owner  string  // "scttfrdmn"
    Repo   string  // "petal-snapshots"
    Branch string  // "main"
    client *http.Client
}

func (r *SnapshotRegistry) List() ([]*SnapshotMetadata, error) {
    // Fetch index.json from GitHub
    indexURL := fmt.Sprintf(
        "https://raw.githubusercontent.com/%s/%s/%s/snapshots/index.json",
        r.Owner, r.Repo, r.Branch)

    resp, err := r.client.Get(indexURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var index SnapshotIndex
    json.NewDecoder(resp.Body).Decode(&index)

    return index.Snapshots, nil
}

func (r *SnapshotRegistry) Get(name string) (*SnapshotMetadata, error) {
    // Fetch individual snapshot metadata
    metadataURL := fmt.Sprintf(
        "https://raw.githubusercontent.com/%s/%s/%s/snapshots/%s.json",
        r.Owner, r.Repo, r.Branch, name)

    resp, err := r.client.Get(metadataURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var metadata SnapshotMetadata
    json.NewDecoder(resp.Body).Decode(&metadata)

    return &metadata, nil
}

func (r *SnapshotRegistry) FindByFingerprint(
    fingerprint string,
    region string,
    os string,
    minSize int) (string, error) {

    snapshots, err := r.List()
    if err != nil {
        return "", err
    }

    for _, s := range snapshots {
        if s.Fingerprint == fingerprint {
            // Fetch full metadata
            metadata, err := r.Get(s.Name)
            if err != nil {
                continue
            }

            // Find matching snapshot with filters
            for _, snap := range metadata.Snapshots {
                if snap.Region == region &&
                   (os == "" || snap.OS == os) &&
                   snap.SizeGB >= minSize {
                    return snap.SnapshotID, nil
                }
            }
        }
    }

    return "", fmt.Errorf("no snapshot found for region=%s, os=%s", region, os)
}

// List all available variants for a fingerprint
func (r *SnapshotRegistry) ListVariants(fingerprint string, region string) ([]*SnapshotVariant, error) {
    metadata, err := r.GetByFingerprint(fingerprint)
    if err != nil {
        return nil, err
    }

    var variants []*SnapshotVariant
    for _, snap := range metadata.Snapshots {
        if snap.Region == region {
            variants = append(variants, &SnapshotVariant{
                SnapshotID:  snap.SnapshotID,
                OS:          snap.OS,
                SizeGB:      snap.SizeGB,
                Description: snap.Description,
            })
        }
    }

    return variants, nil
}
```

### Community Contributions via GitHub

Users contribute snapshots via pull requests:

```bash
# 1. Fork petal-snapshots repo
# 2. Add snapshot metadata
cat > snapshots/my-custom-stack.json <<EOF
{
  "name": "my-custom-stack",
  "description": "Custom genomics pipeline",
  "fingerprint": "sha256:xyz789...",
  "snapshots": {
    "us-east-1": "snap-0xyz789"
  },
  "packages": ["gcc@11.3.0", "custom-tool@1.0"],
  "size_gb": 300,
  "public": true,
  "verified": false,
  "created": "2025-01-25"
}
EOF

# 3. Add seed file
cp my-seed.yaml seeds/my-custom-stack.yaml

# 4. Update index.json
# 5. Create PR
git add .
git commit -m "Add my-custom-stack snapshot"
git push origin my-custom-stack
# Open PR on GitHub

# petal team reviews and merges
```

## References

- [AWS ParallelCluster SharedStorage](https://docs.aws.amazon.com/parallelcluster/latest/ug/SharedStorage-v3.html)
- [EBS Snapshots](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSSnapshots.html)
- [ParallelCluster Best Practices](https://docs.aws.amazon.com/parallelcluster/latest/ug/best-practices-v3.html)
- [Share EBS Snapshots](https://docs.aws.amazon.com/ebs/latest/userguide/ebs-modifying-snapshot-permissions.html)
- [Block Public Sharing of EBS Snapshots](https://aws.amazon.com/blogs/aws/new-block-public-sharing-of-amazon-ebs-snapshots/)
- [EBS Snapshot Security](https://www.trendmicro.com/cloudoneconformity/knowledge-base/aws/EBS/public-snapshots.html)

## Priority Justification

**High Priority** because:
1. Resolves critical architecture issue
2. Aligns with AWS best practices
3. Maintains petal's speed advantage
4. Cleaner, more maintainable solution
5. Blocks production use until resolved
