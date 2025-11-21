#!/bin/bash
# Script to create GitHub issues for Progress Monitoring improvements
# Requires: gh CLI (https://cli.github.com/)

set -e

echo "Creating GitHub issues for Progress Monitoring milestone..."

# Create milestone first
echo "Creating milestone..."
gh api repos/:owner/:repo/milestones -f title="v1.1.0 - Progress Monitoring" \
  -f description="Improve AMI build progress visibility and observability" \
  -f state="open" || echo "Milestone may already exist"

# Get milestone number
MILESTONE=$(gh api repos/:owner/:repo/milestones --jq '.[] | select(.title=="v1.1.0 - Progress Monitoring") | .number')

if [ -z "$MILESTONE" ]; then
  echo "Warning: Could not find milestone number, creating issues without milestone"
fi

echo ""
echo "Creating Phase 1 issue..."
gh issue create \
  --title "Phase 1: Basic Observability (Disk Metrics + Package Progress)" \
  --body-file .github/issues/issue-progress-basic-observability.md \
  --label "enhancement,monitoring,progress,phase-1" \
  --milestone "$MILESTONE" || true

echo ""
echo "Creating Phase 2 issue..."
gh issue create \
  --title "Phase 2: Verbose Mode & Spack Integration" \
  --body-file .github/issues/issue-progress-verbose-mode.md \
  --label "enhancement,monitoring,spack,phase-2" \
  --milestone "$MILESTONE" || true

echo ""
echo "Creating Phase 3 issue..."
gh issue create \
  --title "Phase 3: Advanced Features (Time Estimates, Analytics, Error Detection)" \
  --body-file .github/issues/issue-progress-advanced-features.md \
  --label "enhancement,monitoring,analytics,phase-3" \
  --milestone "$MILESTONE" || true

echo ""
echo "✅ Issues created successfully!"
echo ""
echo "View milestone: gh issue list --milestone 'v1.1.0 - Progress Monitoring'"
