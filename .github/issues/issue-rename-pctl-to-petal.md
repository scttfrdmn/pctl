# Project Rename: pctl → petal

## Status: Planned

## Priority: Medium

## Overview
Rename the project from `pctl` to `petal` with a cohesive metaphor system where templates become "seeds" and clusters "bloom" from them. Keep professional commands as primary with fun aliases for discovery and brand personality.

## Rationale

**Why petal?**
- 🌸 **Phonetically identical** to "pctl" (no confusion when spoken)
- 🌱 **Perfect metaphor**: Templates (seeds) grow into clusters (blooms)
- 💡 **Memorable**: Easier to remember than an acronym
- ⚡ **Distinctive**: Stands out in the HPC/cloud tools space
- 😊 **Approachable**: Less intimidating for new users
- 🎨 **Brand identity**: Unique personality in a technical space

**Why now?**
- Project is pre-1.0, smaller user base
- Better time to rename than after wider adoption
- Builds stronger brand identity early

## Design Philosophy

**Professional by Default, Playful by Discovery**
- Primary commands remain clear and technical (`create`, `delete`, `list`)
- Fun aliases available for those who want them (`bloom`, `harvest`, `garden`)
- Seeds replace templates as the primary term (shorter, distinctive)
- Documentation shows professional commands first

## Proposed Changes

### 1. Repository & Binary Name

```bash
# Repository
github.com/scttfrdmn/pctl → github.com/scttfrdmn/petal

# Binary name
pctl → petal

# Go module
module github.com/scttfrdmn/pctl → module github.com/scttfrdmn/petal
```

### 2. Primary Terminology Changes

| Old Term | New Term | Reason |
|----------|----------|--------|
| template | seed | Shorter, metaphor-consistent, distinctive |
| template file | seed file | Natural terminology |
| templates/ | seeds/ | Directory naming |

### 3. Command Structure

**Primary Commands (Professional)**
```bash
petal create --seed bio.yaml --name cluster1    # Create cluster
petal delete cluster1                           # Delete cluster
petal list                                      # List clusters
petal status cluster1                           # Cluster status
petal ssh cluster1                              # SSH to cluster
petal ami build --seed bio.yaml                 # Build AMI
petal ami list                                  # List AMIs
petal registry update                           # Update seed registry
```

**Fun Aliases (Optional)**
```bash
petal bloom --seed bio.yaml --name cluster1     # = create
petal harvest cluster1                          # = delete
petal garden                                    # = list
petal inspect cluster1                          # = status
petal stem cluster1                             # = ssh
petal greenhouse build --seed bio.yaml          # = ami build
petal greenhouse list                           # = ami list
```

### 4. Flag Changes

```bash
# Old flags
--template, -t

# New flags (with backwards compatibility)
--seed, -s          # Primary flag
--template, -t      # Deprecated alias (warn users)
```

### 5. Directory Structure

```bash
# Before
pctl/
├── templates/
│   ├── library/
│   └── examples/
└── ~/.pctl/

# After
petal/
├── seeds/
│   ├── library/
│   └── examples/
└── ~/.petal/
```

### 6. Configuration Files

```bash
# Old
~/.pctl/config.yaml
~/.pctl/ami-cache.json
~/.pctl/state/

# New
~/.petal/config.yaml
~/.petal/ami-cache.json
~/.petal/state/

# Migration: Auto-migrate on first run if ~/.pctl exists
```

## Implementation Plan

### Phase 1: Code Changes (Non-Breaking)

1. **Update Go module path**
   ```bash
   # Update go.mod
   module github.com/scttfrdmn/petal

   # Update all imports
   find . -type f -name "*.go" -exec sed -i '' \
     's|github.com/scttfrdmn/pctl|github.com/scttfrdmn/petal|g' {} +
   ```

2. **Add command aliases**
   ```go
   // cmd/petal/create.go
   var createCmd = &cobra.Command{
       Use:   "create",
       Aliases: []string{"bloom", "grow"},
       Short: "Create a new HPC cluster from a seed",
       // ...
   }

   // cmd/petal/delete.go
   var deleteCmd = &cobra.Command{
       Use:   "delete",
       Aliases: []string{"harvest", "wilt"},
       // ...
   }

   // cmd/petal/list.go
   var listCmd = &cobra.Command{
       Use:   "list",
       Aliases: []string{"garden"},
       // ...
   }

   // cmd/petal/ssh.go
   var sshCmd = &cobra.Command{
       Use:   "ssh",
       Aliases: []string{"stem", "connect"},
       // ...
   }

   // cmd/petal/status.go
   var statusCmd = &cobra.Command{
       Use:   "status",
       Aliases: []string{"inspect"},
       // ...
   }

   // cmd/petal/ami/build.go
   var amiBuildCmd = &cobra.Command{
       Use:   "build",
       Aliases: []string{"greenhouse"},
       // ...
   }
   ```

3. **Update flag names**
   ```go
   // Support both --seed and --template (deprecated)
   createCmd.Flags().StringP("seed", "s", "", "Seed file (template)")
   createCmd.Flags().StringP("template", "t", "", "DEPRECATED: Use --seed instead")

   // In validation, check both and warn if using deprecated
   if templateFlag != "" {
       fmt.Printf("⚠️  Warning: --template is deprecated, use --seed instead\n")
       seedFlag = templateFlag
   }
   ```

4. **Update internal types**
   ```go
   // pkg/template/template.go → pkg/seed/seed.go
   package seed

   // Template struct → Seed struct (internal name change)
   type Seed struct {
       Cluster  ClusterConfig  `yaml:"cluster"`
       Compute  ComputeConfig  `yaml:"compute"`
       Software SoftwareConfig `yaml:"software"`
       // ...
   }
   ```

5. **Rename directories**
   ```bash
   git mv templates/ seeds/
   git mv cmd/pctl/ cmd/petal/
   git mv pkg/template/ pkg/seed/
   ```

### Phase 2: Documentation Updates

1. **README.md**
   - Update all examples to use `petal` and `--seed`
   - Add "Fun Commands" callout mentioning aliases
   - Update installation instructions

2. **CHANGELOG.md**
   - Add breaking change notice for rename
   - Document migration path
   - List all new aliases

3. **docs/**
   - Update all references from pctl → petal
   - Update template spec → seed spec
   - Add alias reference guide

4. **Code comments**
   - Update package comments
   - Update function documentation

### Phase 3: GitHub & Infrastructure

1. **Rename GitHub repository**
   - Settings → Repository name: `pctl` → `petal`
   - GitHub automatically redirects old URLs

2. **Update GitHub metadata**
   - Repository description
   - Topics/tags
   - README badges

3. **Update CI/CD**
   - `.github/workflows/*.yml` - Update binary names
   - `.goreleaser.yml` - Update project name and binary names

4. **Package managers**
   - Homebrew tap: Update formula name and binary
   - Release assets: `pctl_*` → `petal_*`

### Phase 4: Migration Support

1. **Auto-migration script**
   ```go
   // internal/migration/pctl_to_petal.go
   func MigrateFromPctl() error {
       oldDir := filepath.Join(os.Getenv("HOME"), ".pctl")
       newDir := filepath.Join(os.Getenv("HOME"), ".petal")

       if _, err := os.Stat(oldDir); err == nil {
           if _, err := os.Stat(newDir); os.IsNotExist(err) {
               fmt.Printf("🌸 Migrating pctl config to petal...\n")
               if err := os.Rename(oldDir, newDir); err != nil {
                   return fmt.Errorf("migration failed: %w", err)
               }
               fmt.Printf("✅ Migration complete!\n")
           }
       }
       return nil
   }
   ```

2. **Migration message on first run**
   ```go
   // In root command initialization
   func init() {
       if err := migration.MigrateFromPctl(); err != nil {
           fmt.Printf("⚠️  Warning: %v\n", err)
       }
   }
   ```

### Phase 5: Communication

1. **Release notes**
   ```markdown
   ## 🌸 v2.0.0 - The Petal Release

   **Breaking Changes:**
   - Project renamed: `pctl` → `petal`
   - Templates now called "seeds"
   - Configuration directory: `~/.pctl` → `~/.petal` (auto-migrates)

   **What You Need to Do:**
   1. Reinstall: `brew upgrade petal` or download new binary
   2. Update scripts: `pctl` → `petal`
   3. Update flags: `--template` → `--seed` (old flag still works with warning)
   4. Rename `templates/` directories to `seeds/` (optional)

   **What's Automatic:**
   - Config migration from `~/.pctl` to `~/.petal`
   - Old `--template` flag still works (deprecated)
   - GitHub redirects old repository URLs

   **New Fun Features:**
   - Use `petal bloom` instead of `petal create`
   - Use `petal garden` instead of `petal list`
   - See all aliases with `petal --help`
   ```

2. **GitHub release with migration guide**

3. **Update social media / announcements**

## File Changes Checklist

### Go Files
- [ ] `go.mod` - Update module path
- [ ] All `*.go` files - Update imports
- [ ] `cmd/pctl/` → `cmd/petal/`
- [ ] Add command aliases to all commands
- [ ] Update flag names (seed/template)
- [ ] `pkg/template/` → `pkg/seed/`
- [ ] Update struct names internally

### Documentation
- [ ] `README.md` - All examples and instructions
- [ ] `CHANGELOG.md` - Add breaking change entry
- [ ] `docs/TEMPLATE_SPEC.md` → `docs/SEED_SPEC.md`
- [ ] All `*.md` files in `docs/`
- [ ] Code comments and package docs

### Configuration & Infrastructure
- [ ] `.goreleaser.yml` - Binary names and project name
- [ ] `.github/workflows/*.yml` - Binary references
- [ ] `Makefile` - Binary name and paths
- [ ] GitHub repository name
- [ ] GitHub repository description/topics

### Directories
- [ ] `templates/` → `seeds/`
- [ ] `cmd/pctl/` → `cmd/petal/`
- [ ] `pkg/template/` → `pkg/seed/`

### Tests
- [ ] All `*_test.go` files - Update references
- [ ] Test fixtures with template references
- [ ] Integration test scripts

## Backwards Compatibility

**What Remains Compatible:**
- `--template` flag (deprecated with warning)
- GitHub URL redirects (automatic)
- Old config directory (auto-migrates)

**What Breaks:**
- Binary name: `pctl` → `petal` (users must reinstall)
- Go imports (if anyone imports our packages)
- Hard-coded paths to `~/.pctl/` in user scripts

## Testing Plan

1. **Unit tests** - All tests pass after rename
2. **Integration tests** - Full workflow (build AMI, create cluster)
3. **Migration test** - Create `~/.pctl`, run petal, verify migration
4. **Backwards compat** - Test `--template` flag with deprecation warning
5. **Alias test** - Verify `petal bloom` = `petal create`
6. **Documentation** - Verify all examples work

## Rollout Strategy

### Option A: Big Bang (Recommended)
- Release v2.0.0 with full rename
- Clear migration guide
- Fast, clean break

### Option B: Gradual
- v1.9.0: Add aliases, keep pctl name
- v2.0.0: Rename to petal
- More complex, less clear

**Recommendation: Option A (Big Bang)**

## User Communication Template

```markdown
## 🌸 Introducing Petal!

We're excited to announce that `pctl` has been renamed to **petal**!

### Why the Change?
- **Memorable**: "Petal" is easier to remember than an acronym
- **Metaphor**: Templates are now "seeds" that bloom into clusters
- **Fun**: Optional playful aliases for commands
- **Brand**: Distinctive personality in the HPC space

### What You Need to Do

**1. Reinstall**
```bash
# Homebrew
brew uninstall pctl
brew install petal

# Direct download
curl -LO https://github.com/scttfrdmn/petal/releases/latest/download/petal_...
```

**2. Update Your Scripts**
```bash
# Old
pctl create --template bio.yaml

# New
petal create --seed bio.yaml

# Or use the fun alias!
petal bloom --seed bio.yaml
```

**3. That's It!**
- Config automatically migrates from `~/.pctl` to `~/.petal`
- Old `--template` flag still works (with deprecation warning)

### New Fun Commands

Try these optional aliases:
- `petal bloom` → create a cluster
- `petal garden` → list clusters
- `petal stem` → ssh to cluster
- `petal harvest` → delete cluster

All regular commands still work too! (`create`, `list`, `ssh`, `delete`)
```

## Success Criteria

- [ ] All tests pass with new names
- [ ] Binary renamed and builds successfully
- [ ] GitHub repository renamed
- [ ] Documentation fully updated
- [ ] Migration script tested and working
- [ ] Backwards compatibility verified
- [ ] Release notes written
- [ ] Homebrew formula updated

## Estimated Effort

- Code changes: 4-6 hours
- Documentation updates: 2-3 hours
- Testing: 2-3 hours
- GitHub/infrastructure: 1-2 hours
- Communication/release: 1-2 hours
- **Total: 10-16 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Users confused by rename | Medium | Clear migration guide, auto-migration script |
| Broken user scripts | High | Deprecation warnings, backwards compat period |
| SEO/discoverability loss | Low | GitHub redirects, update all links |
| Go import breaks | Low | Few external users, clear breaking change notice |

## Future Enhancements

Once renamed, we can:
- Create "seed library" with curated templates
- Add `petal seeds marketplace` command
- Build community around "growing clusters"
- Create visual brand (petal logo, etc.)
- Extend metaphor (fertilizer = optimizations, pruning = cleanup, etc.)

## Related Issues

- Rename is separate from OS override feature (can do independently)
- Should happen before 2.0.0 release
- Consider timing with other breaking changes

## References

- Original brainstorm session with user (2025-11-20)
- Design philosophy: Professional default, playful discovery
- Metaphor system: seeds → bloom → garden → stem
