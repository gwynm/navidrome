#!/bin/bash
set -e

echo "=== Navidrome Validation ==="
echo ""

# Unset env vars that interfere with config tests
unset ND_MUSICFOLDER
unset ND_DATAFOLDER

# Run the same checks as pre-push hook
make pre-push

echo ""
echo "=== All validation passed! ==="
