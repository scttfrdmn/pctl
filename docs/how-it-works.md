# How petal Works: Behind the Scenes

This guide explains what petal does behind the scenes when building AMIs and creating clusters.

## Table of Contents
- [AMI Build Process](#ami-build-process)
- [Spack Installation & Configuration](#spack-installation--configuration)
- [Package Installation Strategy](#package-installation-strategy)
- [Lmod Integration](#lmod-integration)
- [Architecture Considerations](#architecture-considerations)
- [Bootstrap Script Generation](#bootstrap-script-generation)

## AMI Build Process

When you run `petal ami build`, here's what happens:

### 1. Seed Validation
- Loads and validates your YAML seed
- Checks cluster name, region, instance types, package specs, etc.
- Validates Spack package syntax (e.g., `gcc@11.2.0`, `openmpi@4.1.4`)

### 2. Build Instance Launch
- Launches a temporary EC2 instance (default: `c6a.4xlarge`)
- Uses AWS ParallelCluster base AMI: `ami-088fb472c1fa0c425`
- Waits for instance to be ready (status checks pass)

### 3. Software Installation
Executes a generated bootstrap script on the instance that:
1. Installs Spack and its prerequisites
2. Configures Spack buildcache for faster installations
3. Installs your specified packages
4. Installs and configures Lmod for environment modules
5. Integrates Spack with Lmod

### 4. Cleanup & Optimization
- Removes temporary files and build artifacts
- Clears package manager caches
- Removes SSH host keys (regenerated on first boot)
- Optimizes AMI size

### 5. AMI Creation
- Stops the instance
- Creates an AMI snapshot
- Tags the AMI with metadata
- Terminates the temporary instance
- Returns the new AMI ID

## Spack Installation & Configuration

### Base Installation
```bash
# Install system prerequisites
yum groupinstall -y "Development Tools"
yum install -y git python3 gcc gcc-c++ gcc-gfortran \
  make patch patchelf bzip2 texinfo texinfo-tex

# Clone Spack v0.23.0
git clone -c feature.manyFiles=true \
  https://github.com/spack/spack.git /opt/spack
cd /opt/spack
git checkout v0.23.0

# Source Spack environment
. /opt/spack/share/spack/setup-env.sh
```

### Buildcache Configuration
petal configures Spack to use AWS's public buildcache for faster installations:

```bash
# Add AWS public buildcache mirror
spack mirror add --scope site aws-binaries \
  https://binaries.spack.io/releases/v0.23

# Install and trust GPG keys
spack buildcache keys --install --trust

# Configure padded installation paths (for relocation)
spack config add "config:install_tree:padded_length:128"

# Configure default target architecture
spack config add "packages:all:target:[x86_64]"
```

**Why x86_64 target?**
- AWS EC2 instances use various CPU architectures (Intel, AMD)
- Spack auto-detects specific architectures (e.g., `zen2`, `skylake`)
- Buildcache contains binaries for generic `x86_64` target
- Forcing `x86_64` ensures buildcache hits and avoids source builds

### Environment Setup
Creates `/etc/profile.d/z00_spack.sh` so Spack is available to all users:

```bash
export SPACK_ROOT=/opt/spack
if [ -f "$SPACK_ROOT/share/spack/setup-env.sh" ]; then
  . $SPACK_ROOT/share/spack/setup-env.sh
fi
```

## Package Installation Strategy

### Installation Order
1. **Compilers first**: Packages like `gcc@11.2.0` are installed before others
2. **Register compilers**: After compiler installation, runs `spack compiler find`
3. **Regular packages**: Installs remaining packages in parallel groups

### Buildcache Strategy
```bash
# Try buildcache first, fall back to source if needed
spack install --fail-fast --use-buildcache=auto <package>
```

**Buildcache behavior:**
- `--use-buildcache=auto`: Tries buildcache first, builds from source if no binary
- Dramatically faster when binaries are available (seconds vs. hours)
- Binary availability depends on exact spec match (version, variants, dependencies)

### Progress Reporting
During installation, petal reports progress:
- Base: Instance launch (0-10%), Spack install (10-20%)
- Packages: 20-80% distributed across package count
- Finalization: Module generation, cleanup (80-100%)

### Error Handling
petal uses strict error handling to fail fast:

```bash
if ! spack install --fail-fast <package>; then
  echo "ERROR: Failed to install <package>"
  exit 1
fi
```

This ensures build failures are caught immediately rather than creating incomplete AMIs.

## Lmod Integration

### Lmod Installation
```bash
# Install Lua dependencies
yum install -y lua lua-devel lua-filesystem lua-posix \
  lua-json tcl tcl-devel

# Download and build Lmod 8.7.37
wget https://github.com/TACC/Lmod/archive/8.7.37.tar.gz
tar xzf 8.7.37.tar.gz
cd Lmod-8.7.37
./configure --prefix=/opt/apps
make install
```

### Module Path Configuration
petal configures `MODULEPATH` to include both custom and Spack-generated modules:

```bash
export MODULEPATH=/opt/modules/Core:\
/opt/spack/share/spack/lmod/linux-amzn2-x86_64/Core
```

**Why this path structure?**
- `/opt/modules/Core`: Custom modules you create
- `/opt/spack/share/spack/lmod/...`: Auto-generated Spack modules
- Spack's Lmod integration generates hierarchical modules automatically

### Spack Module Generation
After package installation, petal generates Lmod modules:

```bash
# Configure Spack to use Lmod
cat > ~/.spack/modules.yaml << 'EOF'
modules:
  default:
    enable:
      - lmod
    lmod:
      core_compilers:
        - 'gcc@7.5.0'  # System compiler
      hierarchy:
        - mpi
      projections:
        all: '{name}/{version}-{hash:7}'
EOF

# Generate modules for all installed packages
spack module lmod refresh --delete-tree -y
```

**Module hierarchy:**
- Core: Packages built with system compiler
- Compiler: Compilers themselves (e.g., `gcc/11.2.0`)
- MPI: MPI-dependent packages appear when MPI is loaded

### Environment Setup
Creates `/etc/profile.d/z00_lmod.sh`:

```bash
export LMOD_CMD=/opt/apps/lmod/lmod/libexec/lmod
export MODULEPATH_ROOT=/opt/modules
export MODULEPATH=/opt/modules/Core:\
/opt/spack/share/spack/lmod/linux-amzn2-x86_64/Core

# Initialize Lmod
if [ -n "${BASH_VERSION:-}" ]; then
  . /opt/apps/lmod/lmod/init/bash
fi
```

**File naming (`z00_*`):**
- Alphabetically loads after other profile scripts
- Ensures dependencies (Spack) are loaded first

## Architecture Considerations

### CPU Architecture Detection
Spack automatically detects CPU architecture:

```bash
$ spack arch
linux-amzn2-zen2  # AMD EPYC 7R13 (Graviton)
```

### Buildcache Architecture Matching
**Problem:** Buildcache binaries are built for specific architectures:
- Buildcache has: `x86_64` (generic), `x86_64_v3`, `skylake`, etc.
- AMD EPYC detects as: `zen2`, `zen3`
- No exact match → builds from source → potential failures

**Solution:** Force generic `x86_64` target:
```bash
spack config add "packages:all:target:[x86_64]"
```

This ensures:
- All packages build/install for generic x86_64
- Maximum buildcache hit rate
- Binaries work across different EC2 instance types
- Slight performance trade-off for reliability

### Instance Type Considerations
petal uses `c6a.4xlarge` for AMI builds:
- **CPU**: AMD EPYC (16 vCPUs)
- **Memory**: 32 GB
- **Network**: 12.5 Gbps
- **Why?**: Good balance of compile performance and cost

## Bootstrap Script Generation

### Script Structure
petal generates a single comprehensive bootstrap script:

```bash
#!/bin/bash
set -e  # Exit on any error

# 1. User creation
groupadd -g 5001 user1
useradd -u 5001 -g 5001 -m -s /bin/bash user1

# 2. S3 mounts (if configured)
# Uses s3fs-fuse to mount S3 buckets

# 3. Spack installation
# [Detailed installation commands...]

# 4. Package installation
# [Per-package installation with progress reporting...]

# 5. Lmod installation
# [Lmod build and configuration...]

# 6. Spack-Lmod integration
# [Module generation...]

# 7. Cleanup
# [Remove temporary files...]
```

### Script Execution
The bootstrap script runs via cloud-init:
1. Uploaded to instance via SSH
2. Executed as root
3. Output logged to `/var/log/cloud-init-output.log`
4. Progress reported back to petal via stdout parsing

### Progress Monitoring
petal monitors progress by:
1. Reading cloud-init logs via EC2 console output
2. Parsing `PCTL_PROGRESS:` markers in output
3. Updating progress bar in terminal
4. Detecting errors via exit codes and error keywords

## Troubleshooting

### Common Issues

**1. Package not in buildcache**
```
Warning: No binary found, building from source
```
- **Solution**: Package will build from source (slower but works)
- **Prevention**: Use versions known to be in buildcache (e.g., gcc@11.2.0 vs 11.3.0)

**2. Architecture mismatch**
```
==> Error: No such variant for spec
```
- **Cause**: Variant doesn't exist for package
- **Solution**: Check package variants with `spack info <package>`

**3. Compilation failure**
```
ERROR: Failed to install <package>
```
- **Cause**: Missing dependencies, incompatible versions, or compiler issues
- **Solution**: Check package dependencies and ensure compatible versions

**4. Module not found**
```
module: command not found
```
- **Cause**: Lmod not properly configured
- **Solution**: Source `/etc/profile.d/z00_lmod.sh` or re-login

### Debug Mode
To see exactly what petal is doing:

```bash
# Check bootstrap script
petal ami build --template seed.yaml --dry-run

# Monitor build process
petal ami build --template seed.yaml --verbose

# SSH into build instance (if it fails)
# Instance ID is shown in output
ssh -i ~/.ssh/key.pem ec2-user@<instance-ip>
sudo tail -f /var/log/cloud-init-output.log
```

## Performance Tips

### Faster Builds
1. **Use buildcache-friendly versions**: Check available versions with `spack buildcache list`
2. **Minimize packages**: Only install what you need
3. **Use larger instance type**: More vCPUs = faster parallel builds
4. **Enable cleanup**: Reduces final AMI size significantly

### Cost Optimization
1. **Build AMI once, use many times**: Don't rebuild for every cluster
2. **Clean up failed builds**: petal auto-terminates failed instances
3. **Use spot instances**: (Future feature - not yet implemented)

## Best Practices

### Seed Design
```yaml
software:
  spack_packages:
    # Use specific versions available in buildcache
    - gcc@11.2.0        # ✓ In buildcache
    # - gcc@11.3.0      # ✗ Not in buildcache

    # Use version ranges for flexibility
    - cmake@3.24        # Matches 3.24.x
    - python@3.10       # Matches 3.10.x

    # Specify variants when needed
    - openmpi@4.1.4+legacylaunchers
```

### Module Usage
```bash
# List available modules
module avail

# Load compiler
module load gcc/11.2.0

# Check what's loaded
module list

# Unload all
module purge
```

### Package Management
```bash
# On cluster head node, check installed packages
spack find

# See package dependencies
spack find -dl <package>

# Get package info
spack info <package>
```

## Additional Resources

- [Spack Documentation](https://spack.readthedocs.io/)
- [Lmod Documentation](https://lmod.readthedocs.io/)
- [AWS ParallelCluster Documentation](https://docs.aws.amazon.com/parallelcluster/)
- [Spack Buildcache](https://spack.readthedocs.io/en/latest/binary_caches.html)
