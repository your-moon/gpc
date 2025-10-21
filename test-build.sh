#!/bin/bash

# Test build script for CI/CD troubleshooting

echo "=== Build Test Script ==="
echo "Current directory: $(pwd)"
echo "Directory contents:"
ls -la

echo ""
echo "cmd directory:"
ls -la cmd/

echo ""
echo "cmd/gpc directory:"
ls -la cmd/gpc/

echo ""
echo "Building binary..."
go build -o gpc ./cmd/gpc/

echo ""
echo "Verifying binary:"
ls -la gpc
file gpc

echo ""
echo "Testing binary:"
echo "Test 1 - Correct file (should pass silently):"
./gpc ./testdata/correct.go
echo "Exit code: $?"

echo ""
echo "Test 2 - File with errors (should show errors):"
./gpc ./testdata/testdata.go
echo "Exit code: $?"

echo ""
echo "=== Build Test Complete ==="
