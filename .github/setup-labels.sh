#!/bin/bash
# Copyright 2025 Scott Friedman
# Setup GitHub labels for the project

set -e

# Type labels
gh label create "type: bug" --color "d73a4a" --description "Something isn't working" --force
gh label create "type: feature" --color "a2eeef" --description "New feature or request" --force
gh label create "type: enhancement" --color "a2eeef" --description "Improvement to existing feature" --force
gh label create "type: documentation" --color "0075ca" --description "Documentation improvements" --force
gh label create "type: refactor" --color "fbca04" --description "Code refactoring" --force
gh label create "type: test" --color "d4c5f9" --description "Testing related" --force

# Priority labels
gh label create "priority: critical" --color "b60205" --description "Critical priority" --force
gh label create "priority: high" --color "d93f0b" --description "High priority" --force
gh label create "priority: medium" --color "fbca04" --description "Medium priority" --force
gh label create "priority: low" --color "0e8a16" --description "Low priority" --force

# Status labels
gh label create "status: in-progress" --color "ededed" --description "Currently being worked on" --force
gh label create "status: blocked" --color "e99695" --description "Blocked by another issue" --force
gh label create "status: needs-review" --color "fbca04" --description "Needs code review" --force
gh label create "status: needs-testing" --color "bfd4f2" --description "Needs testing" --force

# Component labels
gh label create "component: cli" --color "c2e0c6" --description "CLI related" --force
gh label create "component: template" --color "c2e0c6" --description "Template system" --force
gh label create "component: provisioner" --color "c2e0c6" --description "Cluster provisioning" --force
gh label create "component: spack" --color "c2e0c6" --description "Spack integration" --force
gh label create "component: registry" --color "c2e0c6" --description "Template registry" --force
gh label create "component: capture" --color "c2e0c6" --description "Configuration capture" --force

# Other labels
gh label create "good first issue" --color "7057ff" --description "Good for newcomers" --force
gh label create "help wanted" --color "008672" --description "Extra attention is needed" --force
gh label create "question" --color "d876e3" --description "Further information is requested" --force
gh label create "wontfix" --color "ffffff" --description "This will not be worked on" --force
gh label create "duplicate" --color "cfd3d7" --description "This issue already exists" --force
gh label create "dependencies" --color "0366d6" --description "Dependency updates" --force

echo "Labels created successfully!"
