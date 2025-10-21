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
echo "cmd/preloadcheck directory:"
ls -la cmd/preloadcheck/

echo ""
echo "Building binary..."
go build -o preloadcheck ./cmd/preloadcheck/

echo ""
echo "Verifying binary:"
ls -la preloadcheck
file preloadcheck

echo ""
echo "Testing binary:"
echo "Test 1 - Correct file (should pass silently):"
./preloadcheck ./testdata/correct.go
echo "Exit code: $?"

echo ""
echo "Test 2 - File with errors (should show errors):"
./preloadcheck ./testdata/testdata.go
echo "Exit code: $?"

echo ""
echo "=== Build Test Complete ==="
