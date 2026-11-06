# mordye — Command Runner
# Run `just` or `just --list` to see available commands.

default:
    @just --list

# ══════════════════════════════════════════════════════════════════════════════
# Setup
# ══════════════════════════════════════════════════════════════════════════════

# Set up a fresh clone: turn on the git hooks and check the tools (idempotent - safe to run multiple times)
[group('Setup')]
bootstrap:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🚀 Bootstrapping development environment..."
    echo ""

    # Configure git hooks (idempotent - git config overwrites existing value)
    echo "🔧 Configuring git hooks..."
    git config core.hooksPath .githooks
    echo "✅ Git hooks configured (using .githooks/)"
    echo ""

    # Check required tools
    echo "📋 Checking required tools..."
    MISSING=0
    command -v go >/dev/null 2>&1 || { echo "❌ go not found (https://go.dev/dl, 1.27+)"; MISSING=1; }
    command -v kingfisher >/dev/null 2>&1 || { echo "❌ kingfisher not found (https://github.com/mongodb/kingfisher) - the pre-commit hook needs it"; MISSING=1; }
    [ $MISSING -eq 0 ] && echo "✅ All required tools found ($(go env GOVERSION))"
    echo ""

    # Check optional tools
    echo "📦 Checking optional tools..."
    MISSING_OPTIONAL=0
    command -v gh >/dev/null 2>&1 || { echo "⚪ gh not found (https://cli.github.com) - needed for issues and PRs"; MISSING_OPTIONAL=1; }
    [ $MISSING_OPTIONAL -eq 0 ] && echo "✅ All optional tools found"
    echo ""

    [ $MISSING -eq 0 ] || exit 1
    echo "🎉 Bootstrap complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Run: just check"

# ══════════════════════════════════════════════════════════════════════════════
# Develop
# ══════════════════════════════════════════════════════════════════════════════

# Build the mordye binary into the repo root
[group('Develop')]
build:
    go build ./cmd/mordye

# Run every test
[group('Develop')]
test:
    go test ./...

# Format every Go file in place
[group('Develop')]
fmt:
    gofmt -w .

# Check formatting, vet, and tests: what a change must pass before it lands
[group('Develop')]
check:
    test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }
    go vet ./...
    go test ./...
