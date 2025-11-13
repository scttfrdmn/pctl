# Milestone: AMI Layering & Template Inheritance

**Goal**: Enable stackable AMIs with template inheritance to dramatically reduce build times and improve reusability.

**Business Value**:
- 43%+ time savings when building multiple applications sharing a base
- Improved developer experience with faster iteration cycles
- Storage-efficient (AWS handles delta snapshots automatically)

**Timeline**: Post v1.0.0 (after basic functionality is stable)

## Problem Statement

Currently, every AMI build starts from scratch:
- Building Gromacs: 70 minutes (foundation + gromacs)
- Building NAMD: 70 minutes (foundation + namd)
- Total: 140 minutes, with 60 minutes of redundant work

**With layering**:
- Foundation: 60 minutes (once)
- Gromacs: 10 minutes (foundation + gromacs delta)
- NAMD: 10 minutes (foundation + namd delta)
- Total: 80 minutes (43% savings)

## Architecture Overview

```
Templates (YAML):                    AMIs (AWS):
┌─────────────────┐                 ┌──────────────────┐
│ foundation.yaml │────builds───────│ foundation-ami   │
│ - gcc           │                 │ ami-123          │
│ - openmpi       │                 │ 10 GB            │
│ - python        │                 └────────┬─────────┘
│ - cmake, git    │                          │
└─────────────────┘                          │ reused as base
                                              │
┌─────────────────┐                 ┌────────▼─────────┐
│ gromacs.yaml    │────builds───────│ gromacs-ami      │
│ extends:        │                 │ ami-456          │
│   foundation    │                 │ +2 GB delta      │
│ + gromacs       │                 └──────────────────┘
└─────────────────┘
                                    ┌──────────────────┐
┌─────────────────┐                 │ namd-ami         │
│ namd.yaml       │────builds───────│ ami-789          │
│ extends:        │                 │ +2 GB delta      │
│   foundation    │                 └──────────────────┘
│ + namd          │
└─────────────────┘
```

## Implementation Phases

### Phase 1: Manual Base AMI Support (v1.1.0)
**Milestone**: Enable users to manually specify base AMIs
- Issues: #X, #Y
- Estimated: 2-3 days
- Deliverable: `--base-ami` flag working

### Phase 2: Template Inheritance (v1.2.0)
**Milestone**: Add `extends` keyword to templates
- Issues: #A, #B, #C
- Estimated: 1 week
- Deliverable: Template merging and validation

### Phase 3: Auto-chaining & Caching (v1.3.0)
**Milestone**: Automatic base AMI detection and reuse
- Issues: #D, #E, #F
- Estimated: 1-2 weeks
- Deliverable: Fully automated layered builds

### Phase 4: Advanced Features (v1.4.0+)
**Milestone**: Multi-level inheritance, visualization, dependency tracking
- Issues: #G, #H, #I
- Estimated: 1-2 weeks
- Deliverable: Production-ready AMI management

## Success Metrics

- [ ] Build time reduction: 40%+ for apps sharing base
- [ ] Template reuse: 3+ templates using foundation base
- [ ] User satisfaction: Positive feedback on AMI layering
- [ ] Zero storage issues: Delta snapshots working as expected

## Dependencies

- ✅ v1.0.0 release complete (basic AMI building working)
- ⬜ Template system stable and well-tested
- ⬜ AMI metadata tracking in place

## Related Documentation

- [Issue Template: AMI Layering](.github/ISSUE_TEMPLATE/ami-layering.md)
- [Design Doc: Template Inheritance](docs/design/template-inheritance.md)
- [ADR: AMI Layering Strategy](docs/adr/003-ami-layering.md)
