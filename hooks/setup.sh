#!/bin/bash

# Setup script to install git hooks

HOOKS_DIR="$(cd "$(dirname "$0")" && pwd)"
GIT_HOOKS_DIR="$(git rev-parse --git-dir)/hooks"

echo "Installing git hooks..."

# Install pre-push hook
if [ -f "$HOOKS_DIR/pre-push" ]; then
    cp "$HOOKS_DIR/pre-push" "$GIT_HOOKS_DIR/pre-push"
    chmod +x "$GIT_HOOKS_DIR/pre-push"
    echo "✓ Installed pre-push hook"
else
    echo "✗ pre-push hook not found"
    exit 1
fi

echo "✓ Git hooks installed successfully!"
echo
echo "The pre-push hook will run the following checks before each push:"
echo "  - Code formatting (gofmt)"
echo "  - Go vet"
echo "  - Tests"
echo
echo "This ensures local changes match CI requirements before pushing."
