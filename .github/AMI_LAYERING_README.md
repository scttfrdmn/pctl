# AMI Layering & Template Inheritance

**Status**: Planned for post-v1.0.0
**Milestone**: v1.1.0 - v1.4.0
**Design Philosophy**: **"Just Works" by default, with power-user controls**

## Vision

Enable users to build HPC cluster AMIs in minutes instead of hours through intelligent caching and automatic template chaining - without requiring them to understand the underlying complexity.

## User Experience

### The "Just Works" Approach (Default)

**What users do:**
```bash
$ pctl ami build -t gromacs.yaml --name my-gromacs
```

**What pctl does automatically:**
1. ✅ Detects that gromacs.yaml extends foundation.yaml
2. ✅ Checks if foundation AMI already exists
3. ✅ Builds foundation AMI if needed (60 min, once)
4. ✅ Builds gromacs on top (10 min)
5. ✅ Caches both for future reuse

**Result**: 70-minute build becomes 10 minutes for all future builds sharing the same foundation.

### Example: Real-world Scenario

**Day 1**: Build Gromacs
```bash
$ pctl ami build -t gromacs.yaml
📋 Detected template chain: foundation → gromacs
🔍 No foundation AMI found, building automatically...
   ... (60 min)
   ✅ Foundation AMI: ami-123
🚀 Building gromacs...
   ... (10 min)
   ✅ Gromacs AMI: ami-456
```

**Day 2**: Build NAMD (reuses foundation)
```bash
$ pctl ami build -t namd.yaml
📋 Detected template chain: foundation → namd
✅ Found foundation AMI: ami-123 (reusing)
🚀 Building namd...
   ... (10 min)
   ✅ NAMD AMI: ami-789
```

**Time saved**: 60 minutes

### Power User Controls (Optional)

For advanced users who need fine-grained control:

```bash
# Force rebuild everything
$ pctl ami build -t gromacs.yaml --rebuild-base

# Use specific base AMI
$ pctl ami build -t gromacs.yaml --base-ami ami-custom-123

# Build in multiple regions
$ pctl ami build -t gromacs.yaml --regions us-west-2,us-east-1

# Share publicly
$ pctl ami share ami-456 --public
```

## Architecture

```
                ┌─────────────────────────────────────────┐
                │  User runs: pctl ami build -t gromacs   │
                └─────────────┬───────────────────────────┘
                              │
                ┌─────────────▼───────────────────────────┐
                │  Parse gromacs.yaml                     │
                │  - Detect: extends foundation.yaml      │
                └─────────────┬───────────────────────────┘
                              │
                ┌─────────────▼───────────────────────────┐
                │  Compute template hash                  │
                │  - Hash includes: content + all parents │
                │  - foundation hash: abc123              │
                │  - gromacs hash: def456                 │
                └─────────────┬───────────────────────────┘
                              │
                ┌─────────────▼───────────────────────────┐
                │  Query AWS for foundation AMI           │
                │  - Search tags: TemplateHash=abc123     │
                └─────────────┬───────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  AMI found?       │
                    └┬────────────────┬─┘
                     │ NO             │ YES
                     │                │
       ┌─────────────▼─┐    ┌────────▼──────────┐
       │ Build         │    │ Reuse ami-123     │
       │ foundation    │    │ (cache hit)       │
       │ → ami-123     │    └────────┬──────────┘
       └─────────────┬─┘             │
                     │                │
                     └────────┬───────┘
                              │
                ┌─────────────▼───────────────────────────┐
                │  Build gromacs on ami-123               │
                │  → ami-456                              │
                └─────────────┬───────────────────────────┘
                              │
                ┌─────────────▼───────────────────────────┐
                │  Tag ami-456 with:                      │
                │  - TemplateHash: def456                 │
                │  - BaseAMI: ami-123                     │
                │  - BaseTemplate: foundation             │
                └─────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Manual Base AMI (v1.1.0) - 2-3 days
**Goal**: Enable manual AMI layering
**Status**: Ready to implement after v1.0.0

**Deliverable**:
```bash
$ pctl ami build -t foundation.yaml --name foundation-v1
$ pctl ami build -t gromacs.yaml --base-ami ami-123 --name gromacs-v1
```

**Files**: [phase1-manual-base-ami.md](issues/phase1-manual-base-ami.md)

---

### Phase 2: Template Inheritance (v1.2.0) - 1 week
**Goal**: Add `extends` keyword to templates
**Status**: Requires Phase 1

**Deliverable**:
```yaml
# gromacs.yaml
extends: foundation.yaml
software:
  spack_packages:
    - gromacs@2023.1
```

**Files**: [phase2-template-inheritance.md](issues/phase2-template-inheritance.md)

---

### Phase 3: Auto-chaining (v1.3.0) - 1-2 weeks
**Goal**: Automatic base AMI detection and caching
**Status**: Requires Phase 1 + Phase 2

**Deliverable**:
```bash
$ pctl ami build -t gromacs.yaml  # Automatically builds/reuses foundation
```

**Files**: [phase3-auto-chaining-caching.md](issues/phase3-auto-chaining-caching.md)

---

### Phase 4: Advanced Features (v1.4.0+) - 1-2 weeks
**Goal**: AMI sharing, dependency visualization, multi-region
**Status**: Requires Phase 1-3

**Deliverable**:
```bash
$ pctl ami share ami-456 --public
$ pctl ami tree ami-456
$ pctl ami copy ami-456 --regions us-east-1,eu-west-1
```

**Files**: [phase4-advanced-features.md](issues/phase4-advanced-features.md)

## Design Principles

### 1. Zero Configuration
**Users shouldn't need to think about layering.**

✅ **Good**:
```bash
pctl ami build -t gromacs.yaml
# Just works - automatically handles base AMI
```

❌ **Bad**:
```bash
pctl ami build -t foundation.yaml --name foundation
export FOUNDATION_AMI=$(pctl ami list | grep foundation | awk '{print $1}')
pctl ami build -t gromacs.yaml --base-ami $FOUNDATION_AMI --enable-layering
```

### 2. Progressive Disclosure
**Simple by default, powerful when needed.**

**Beginner**: Just build
```bash
pctl ami build -t gromacs.yaml
```

**Intermediate**: Control caching
```bash
pctl ami build -t gromacs.yaml --rebuild
```

**Advanced**: Full control
```bash
pctl ami build -t gromacs.yaml \
  --base-ami ami-123 \
  --rebuild-base \
  --regions us-west-2,us-east-1 \
  --share-public
```

### 3. Fail Safely
**Clear errors with actionable fixes.**

❌ **Bad error**:
```
Error: Template error
```

✅ **Good error**:
```
Error: Base template not found
  Template: gromacs.yaml
  Extends:  foundation.yaml
  Error:    No such file: ./foundation.yaml

Fix: Create foundation.yaml or remove 'extends' field
```

### 4. Transparent Operations
**Show users what's happening.**

```bash
$ pctl ami build -t gromacs.yaml

📋 Template chain detected:
   1. foundation.yaml (no AMI, will build)
   2. gromacs.yaml

🚀 Building foundation.yaml automatically...
   ... progress ...
   ✅ Foundation AMI: ami-123

🚀 Building gromacs.yaml...
   ... progress ...
   ✅ Gromacs AMI: ami-456

✅ Build complete! (70 minutes)
   Next time: Only 10 minutes (foundation cached)
```

## Security & Sharing

### Public AMI Sharing
AMIs can be shared publicly for community benefit:

```bash
# Share foundation AMI for others to build on
$ pctl ami share ami-123 --public

✅ AMI ami-123 is now public
   Anyone can use: --base-ami ami-123
   Or in templates:
     extends:
       ami: ami-123
       template: https://example.com/foundation.yaml
```

**Use cases**:
- Open source software stacks
- Community-contributed HPC environments
- Public research infrastructure
- Educational environments

**Security considerations**:
- Public AMIs are visible to all AWS users
- Don't include sensitive data
- Review contents before sharing
- Recommended for FOSS/community tools only

### Account-specific Sharing
Share with trusted partners or within organization:

```bash
# Share with specific AWS accounts
$ pctl ami share ami-123 --accounts 123456789012,987654321098

# Share with AWS organization
$ pctl ami share ami-123 --organization o-abc123xyz
```

## Performance Benefits

### Time Savings

**Traditional approach** (no layering):
- Build foundation: 60 min
- Build gromacs: 70 min (foundation + gromacs)
- Build namd: 70 min (foundation + namd)
- **Total: 200 minutes**

**With AMI layering**:
- Build foundation: 60 min (once)
- Build gromacs: 10 min (delta only)
- Build namd: 10 min (delta only)
- **Total: 80 minutes (60% time savings)**

### Storage Efficiency

AWS EBS snapshots are delta-based:
- Foundation AMI: 10 GB stored
- Gromacs AMI: +2 GB delta (not 12 GB!)
- NAMD AMI: +2 GB delta
- **Total: 14 GB** (not 34 GB)

Storage cost: ~$0.05/GB-month = **$0.70/month total**

## Success Metrics

Post-implementation, we expect:

- **Build time reduction**: 40-60% for apps sharing bases
- **Template reuse**: 3+ apps using shared foundation
- **Storage efficiency**: <20% overhead vs monolithic AMIs
- **User satisfaction**: "It just works" feedback
- **Community adoption**: Public foundation templates available

## Documentation Roadmap

### For v1.1.0 (Phase 1)
- [ ] `docs/ami-building.md` - Add manual layering section
- [ ] `docs/tutorial.md` - Add layered build example

### For v1.2.0 (Phase 2)
- [ ] `docs/templates.md` - Add inheritance documentation
- [ ] `docs/examples/` - Add layered template examples

### For v1.3.0 (Phase 3)
- [ ] `docs/ami-caching.md` - Explain auto-chaining
- [ ] `docs/troubleshooting.md` - Common cache issues

### For v1.4.0 (Phase 4)
- [ ] `docs/ami-sharing.md` - AMI sharing guide
- [ ] `docs/multi-region.md` - Multi-region strategies
- [ ] `docs/best-practices.md` - Template design patterns

## Getting Started (Post-Implementation)

After Phase 3 is complete, users can immediately benefit:

### Create a foundation template
```yaml
# templates/foundation.yaml
cluster:
  name: foundation
  region: us-west-2

software:
  spack_packages:
    - gcc@11.3.0
    - openmpi@4.1.4
    - python@3.10
    - cmake@3.26.0
```

### Create derived templates
```yaml
# templates/gromacs.yaml
extends: foundation.yaml

cluster:
  name: gromacs-cluster

software:
  spack_packages:
    - gromacs@2023.1+mpi
```

### Build and profit!
```bash
$ pctl ami build -t gromacs.yaml --name my-gromacs
# Foundation built automatically (60 min first time)
# Gromacs builds on top (10 min)

$ pctl ami build -t namd.yaml --name my-namd
# Reuses foundation AMI (0 min)
# NAMD builds on top (10 min)
```

## Questions?

See individual phase documents for implementation details:
- [Phase 1: Manual Base AMI](.github/issues/phase1-manual-base-ami.md)
- [Phase 2: Template Inheritance](.github/issues/phase2-template-inheritance.md)
- [Phase 3: Auto-chaining](.github/issues/phase3-auto-chaining-caching.md)
- [Phase 4: Advanced Features](.github/issues/phase4-advanced-features.md)

Or review the [milestone document](MILESTONE_AMI_LAYERING.md) for high-level overview.
