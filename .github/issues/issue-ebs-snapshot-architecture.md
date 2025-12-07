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

## References

- [AWS ParallelCluster SharedStorage](https://docs.aws.amazon.com/parallelcluster/latest/ug/SharedStorage-v3.html)
- [EBS Snapshots](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSSnapshots.html)
- [ParallelCluster Best Practices](https://docs.aws.amazon.com/parallelcluster/latest/ug/best-practices-v3.html)

## Priority Justification

**High Priority** because:
1. Resolves critical architecture issue
2. Aligns with AWS best practices
3. Maintains petal's speed advantage
4. Cleaner, more maintainable solution
5. Blocks production use until resolved
