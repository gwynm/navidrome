#!/bin/bash
set -e

echo "=== Navidrome Validation ==="
echo ""

# Unset env vars that interfere with config tests
unset ND_MUSICFOLDER
unset ND_DATAFOLDER

echo "▶ Running Go linter..."
make lint
echo "✓ Go lint passed"
echo ""

echo "▶ Running Go tests..."
make test
echo "✓ Go tests passed"
echo ""

echo "▶ Running JavaScript tests..."
cd ui && npm test
cd ..
echo "✓ JavaScript tests passed"
echo ""

echo "=== All validation passed! ==="

