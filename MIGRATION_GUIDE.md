# Migration Guide: pctl → petal

This guide helps you migrate from pctl to petal. The rename brings a gardening metaphor and fun command aliases while maintaining all functionality.

## What Changed?

### Binary & Module Name
- **Old**: `pctl`
- **New**: `petal`

### Terminology
- **Old**: "templates"
- **New**: "seeds" (primary term, though "template" still works)

### Configuration Directory
- **Old**: `~/.pctl/`
- **New**: `~/.petal/` (auto-migrates on first run)

### Directory Structure
- **Old**: `templates/` directory
- **New**: `seeds/` directory

### Module Path (for developers)
- **Old**: `github.com/scttfrdmn/pctl`
- **New**: `github.com/scttfrdmn/petal`

## Migration Steps

### 1. Install petal Binary

Choose your installation method:

**Homebrew (macOS)**
```bash
# Uninstall old version
brew uninstall pctl

# Install petal
brew install scttfrdmn/tap/petal
```

**Direct Download**
```bash
# Download new binary
curl -LO https://github.com/scttfrdmn/petal/releases/latest/download/petal_linux_x86_64.tar.gz
tar xzf petal_linux_x86_64.tar.gz

# Replace old binary
sudo rm /usr/local/bin/pctl
sudo mv petal /usr/local/bin/

# Verify
petal version
```

**Build from Source**
```bash
# Clone repo
git clone https://github.com/scttfrdmn/petal.git
cd petal

# Build and install
make build
sudo make install

# Verify
petal version
```

### 2. Configuration Migration (Automatic)

Your configuration and state are automatically migrated on first run:

```bash
# First time you run petal, you'll see:
$ petal list

🌸 Migrating pctl config to petal...
   Moving ~/.pctl → ~/.petal
✅ Migration complete!

# All your clusters, AMI builds, and configuration preserved!
```

**Manual Migration** (if needed):
```bash
# If automatic migration fails, do it manually:
mv ~/.pctl ~/.petal
```

### 3. Update Your Scripts

Replace `pctl` commands with `petal`:

**Before:**
```bash
#!/bin/bash
pctl create --template my-cluster.yaml --name production
pctl status production
pctl ssh production
pctl delete production
```

**After:**
```bash
#!/bin/bash
petal create --seed my-cluster.yaml --name production
petal status production
petal ssh production
petal delete production
```

**Quick Find & Replace:**
```bash
# Update all scripts in a directory
find . -name "*.sh" -type f -exec sed -i 's/pctl /petal /g' {} \;

# Update specific script
sed -i 's/pctl /petal /g' my-script.sh
```

### 4. Update YAML Files (Optional)

Your existing YAML files work without changes! But you can update comments/paths for clarity:

**Before:**
```yaml
# my-cluster.yaml
# This is a pctl template for bioinformatics
cluster:
  name: bio-cluster
  region: us-east-1

compute:
  head_node: t3.xlarge
  queues:
    - name: compute
      instance_types: [c5.4xlarge]
      min_count: 0
      max_count: 10

software:
  spack_packages:
    - gcc@11.3.0
    - samtools@1.17
```

**After:**
```yaml
# my-cluster.yaml
# This is a petal seed for bioinformatics
cluster:
  name: bio-cluster
  region: us-east-1

compute:
  head_node: t3.xlarge
  queues:
    - name: compute
      instance_types: [c5.4xlarge]
      min_count: 0
      max_count: 10

software:
  spack_packages:
    - gcc@11.3.0
    - samtools@1.17
```

**Note:** This is cosmetic only. The YAML structure hasn't changed.

### 5. Update CI/CD Pipelines

**GitHub Actions Example:**

**Before:**
```yaml
# .github/workflows/deploy.yml
name: Deploy Cluster
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Download pctl
        run: |
          curl -LO https://github.com/scttfrdmn/pctl/releases/latest/download/pctl
          chmod +x pctl
          sudo mv pctl /usr/local/bin/

      - name: Create cluster
        run: pctl create --template cluster.yaml --name ci-cluster
```

**After:**
```yaml
# .github/workflows/deploy.yml
name: Deploy Cluster
on: [push]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Download petal
        run: |
          curl -LO https://github.com/scttfrdmn/petal/releases/latest/download/petal_linux_x86_64.tar.gz
          tar xzf petal_linux_x86_64.tar.gz
          chmod +x petal
          sudo mv petal /usr/local/bin/

      - name: Create cluster
        run: petal create --seed cluster.yaml --name ci-cluster
```

### 6. Update Documentation

Update your project documentation:

```bash
# Find all markdown files referencing pctl
grep -r "pctl" docs/

# Update references
find docs/ -name "*.md" -type f -exec sed -i 's/pctl /petal /g' {} \;
find docs/ -name "*.md" -type f -exec sed -i 's/template/seed/g' {} \;
```

## Backwards Compatibility

petal maintains backwards compatibility in several ways:

### 1. Flag Compatibility

The `--template` flag still works (with deprecation warning):

```bash
# Old flag (still works)
$ petal create --template my-cluster.yaml --name test

⚠️  Warning: --template is deprecated, use --seed instead

✅ Cluster created successfully!

# New flag (recommended)
$ petal create --seed my-cluster.yaml --name test
```

### 2. YAML Format

All existing YAML files work without modification. The syntax hasn't changed.

### 3. Configuration Format

Your `config.yaml` format is unchanged:

```yaml
# ~/.petal/config.yaml (same format as before)
defaults:
  region: us-east-1
  key_name: my-key

parallelcluster:
  version: 3.14.0
  install_method: pipx

preferences:
  auto_update_registry: true
  validate_before_create: true
  confirm_destructive: true
```

### 4. State Files

All cluster state files are preserved during migration. You can still manage clusters created with pctl:

```bash
# Clusters created with pctl are still visible
$ petal list

CLUSTER NAME       STATUS          CREATED
old-pctl-cluster   CREATE_COMPLETE 2024-12-01 10:30:00

# You can still manage them
$ petal status old-pctl-cluster
$ petal ssh old-pctl-cluster
$ petal delete old-pctl-cluster
```

## New Features: Fun Command Aliases! 🌸

petal introduces gardening-themed command aliases as a fun alternative to standard commands:

| Standard Command | Fun Alias | Description |
|-----------------|-----------|-------------|
| `petal create`  | `petal bloom` or `petal grow` | Plant a seed and watch your cluster bloom! |
| `petal delete`  | `petal harvest` or `petal wilt` | Harvest your cluster when done |
| `petal list`    | `petal garden` | View your garden of clusters |
| `petal ssh`     | `petal stem` or `petal connect` | Connect via the stem |
| `petal status`  | `petal inspect` | Inspect your cluster |

**Examples:**
```bash
# Professional commands (primary, always supported)
petal create --seed bioinformatics.yaml --name my-cluster
petal status my-cluster
petal ssh my-cluster
petal delete my-cluster

# Fun aliases (optional, same functionality)
petal bloom --seed bioinformatics.yaml --name my-cluster  🌸
petal inspect my-cluster
petal stem my-cluster
petal harvest my-cluster
```

Use whichever style you prefer! Both work identically.

## Troubleshooting

### Migration Fails

If automatic migration fails:

```bash
# Check if old config exists
ls -la ~/.pctl

# Check if new location exists
ls -la ~/.petal

# Manual migration
mv ~/.pctl ~/.petal

# Or start fresh (loses cluster state)
petal registry update
```

### Command Not Found

If you get "petal: command not found":

```bash
# Check installation
which petal

# Verify PATH
echo $PATH

# Reinstall to /usr/local/bin
sudo mv petal /usr/local/bin/
sudo chmod +x /usr/local/bin/petal
```

### Old pctl Binary Still Running

If `petal` commands actually run `pctl`:

```bash
# Find all copies
which -a pctl petal

# Remove old binary
sudo rm /usr/local/bin/pctl

# Verify petal is used
petal version
# Should show "petal version x.x.x"
```

### Scripts Still Reference pctl

If your scripts still use pctl:

```bash
# Find scripts using pctl
grep -r "pctl " .

# Update all at once
find . -type f \( -name "*.sh" -o -name "*.bash" \) -exec sed -i 's/pctl /petal /g' {} \;
```

### CI/CD Pipeline Failures

If CI/CD fails after upgrade:

1. **Update download URLs**: Change from `pctl` to `petal` releases
2. **Update binary names**: Change `pctl` to `petal` in commands
3. **Update flags**: Change `--template` to `--seed` (or leave as-is with deprecation warning)
4. **Cache invalidation**: Clear CI/CD caches that might have old binary

## Quick Migration Checklist

- [ ] Install petal binary (brew/download/build)
- [ ] Run `petal version` to trigger config migration
- [ ] Update shell scripts (`pctl` → `petal`)
- [ ] Update CI/CD pipelines
- [ ] Update documentation
- [ ] (Optional) Update YAML file comments
- [ ] (Optional) Switch `--template` to `--seed` flags
- [ ] Remove old pctl binary
- [ ] Test existing clusters still work

## Need Help?

- **Issues**: https://github.com/scttfrdmn/petal/issues
- **Discussions**: https://github.com/scttfrdmn/petal/discussions
- **Rename Details**: See `.github/issues/issue-rename-pctl-to-petal.md`

## Version Compatibility

- **petal v0.5.1+**: All features described in this guide
- **pctl ≤ v0.5.0**: Old version before rename

Both versions use the same underlying AWS ParallelCluster integration, so clusters are fully compatible.

## Summary

The migration from pctl to petal is straightforward:

1. ✅ **Install petal** - Download and install new binary
2. ✅ **Auto-migrate config** - Happens automatically on first run
3. ✅ **Update scripts** - Replace `pctl` with `petal`
4. ✅ **Optional improvements** - Use new `--seed` flag and fun aliases

Your clusters, configuration, and YAML files all continue to work. The change is primarily cosmetic with quality-of-life improvements.

Welcome to petal! Plant a seed, watch your cluster bloom! 🌸
