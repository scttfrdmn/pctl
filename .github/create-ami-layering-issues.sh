#!/bin/bash
# Script to create GitHub issues from AMI layering documentation
# Requires: gh CLI (https://cli.github.com/)

set -e

echo "Creating GitHub issues for AMI Layering milestone..."

# Create milestone first
echo "Creating milestone..."
gh api repos/:owner/:repo/milestones -f title="v1.1.0 - AMI Layering" \
  -f description="Enable stackable AMIs with template inheritance" \
  -f state="open" || echo "Milestone may already exist"

# Get milestone number
MILESTONE=$(gh api repos/:owner/:repo/milestones --jq '.[] | select(.title=="v1.1.0 - AMI Layering") | .number')

if [ -z "$MILESTONE" ]; then
  echo "Warning: Could not find milestone number, creating issues without milestone"
fi

echo ""
echo "Creating Phase 1 issue..."
gh issue create \
  --title "Phase 1: Manual Base AMI Support" \
  --body-file .github/issues/phase1-manual-base-ami.md \
  --label "enhancement,ami,phase-1" \
  --milestone "$MILESTONE" || true

echo ""
echo "Creating Phase 2 issue..."
gh issue create \
  --title "Phase 2: Template Inheritance" \
  --body-file .github/issues/phase2-template-inheritance.md \
  --label "enhancement,ami,templates,phase-2" \
  --milestone "$MILESTONE" || true

echo ""
echo "Creating Phase 3 issue..."
gh issue create \
  --title "Phase 3: Auto-chaining & Caching" \
  --body-file .github/issues/phase3-auto-chaining-caching.md \
  --label "enhancement,ami,caching,phase-3" \
  --milestone "$MILESTONE" || true

echo ""
echo "Creating Phase 4 issue..."
gh issue create \
  --title "Phase 4: Advanced Features (AMI Sharing, Multi-region)" \
  --body-file .github/issues/phase4-advanced-features.md \
  --label "enhancement,ami,sharing,phase-4" \
  --milestone "$MILESTONE" || true

echo ""
echo "✅ Issues created successfully!"
echo ""
echo "View milestone: gh issue list --milestone 'v1.1.0 - AMI Layering'"
