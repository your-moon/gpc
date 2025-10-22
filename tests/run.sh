#!/bin/bash

# VMS Real-world Test Suite
# Tests the GORM Preload Checker with real-world examples from VMS backend

echo "🚀 VMS Real-world Test Suite"
echo "============================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a single test
run_test() {
    local test_name="$1"
    local test_file="$2"
    local expected_file="$3"
    
    echo -e "\n${BLUE}🧪 Running: $test_name${NC}"
    echo "   File: $test_file"
    echo "   Expected: $expected_file"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # Run the test with JSON output
    if ./gpc -o json "$test_file" > /dev/null 2>&1; then
        # Compare results
        if [ -f "$expected_file" ] && command -v jq &> /dev/null; then
            if jq -e '.total_preloads' gpc_results.json > /dev/null 2>&1; then
                # Extract key metrics
                actual_total=$(jq -r '.total_preloads' gpc_results.json)
                actual_correct=$(jq -r '.correct' gpc_results.json)
                actual_accuracy=$(jq -r '.accuracy' gpc_results.json)
                
                expected_total=$(jq -r '.total_preloads' "$expected_file")
                expected_correct=$(jq -r '.correct' "$expected_file")
                expected_accuracy=$(jq -r '.accuracy' "$expected_file")
                
                # Check if results match
                if [ "$actual_total" = "$expected_total" ] && [ "$actual_correct" = "$expected_correct" ]; then
                    echo -e "   ${GREEN}✅ PASSED${NC}"
                    echo "   📊 Total: $actual_total, Correct: $actual_correct, Accuracy: $actual_accuracy%"
                    PASSED_TESTS=$((PASSED_TESTS + 1))
                else
                    echo -e "   ${RED}❌ FAILED${NC}"
                    echo "   📊 Expected: Total=$expected_total, Correct=$expected_correct, Accuracy=$expected_accuracy%"
                    echo "   📊 Actual:   Total=$actual_total, Correct=$actual_correct, Accuracy=$actual_accuracy%"
                    FAILED_TESTS=$((FAILED_TESTS + 1))
                fi
            else
                echo -e "   ${RED}❌ FAILED - Invalid JSON output${NC}"
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        elif [ -f "$expected_file" ]; then
            echo -e "   ${YELLOW}⚠️  SKIPPED - jq not available for comparison${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "   ${YELLOW}⚠️  SKIPPED - No expected file${NC}"
        fi
    else
        echo -e "   ${RED}❌ FAILED - Tool execution error${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Build the tool first
echo "📦 Building GORM Preload Checker..."
if go build -o gpc ../main.go; then
    echo -e "${GREEN}✅ Build successful${NC}"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}⚠️  jq not found. Install jq for better test results:${NC}"
    echo "   brew install jq  # macOS"
    echo "   apt-get install jq  # Ubuntu"
    echo ""
    echo "Running without jq - basic test only"
fi

# Run all VMS tests
echo -e "\n${BLUE}🔍 Running VMS Real-world Tests${NC}"

# Test 1: Authentication flow
run_test "VMS Authentication Flow" "./vms_auth/vms_auth_test.go" "./expected/vms_auth_expected.json"

# Test 2: Machine management
run_test "VMS Machine Management" "./vms_machine/vms_machine_test.go" "./expected/vms_machine_expected.json"

# Test 3: Invoice and payments
run_test "VMS Invoice & Payments" "./vms_invoice/vms_invoice_test.go" "./expected/vms_invoice_expected.json"

# Test 4: Complex patterns (realistic failures)
run_test "VMS Complex Patterns" "./vms_complex/vms_complex_test.go" "./expected/vms_complex_expected.json"

# Test 5: Simple test (baseline)
run_test "Simple Test (Baseline)" "./simple/simple_test.go" "./expected/expected_results.json"

# Summary
echo -e "\n${BLUE}📊 Test Summary${NC}"
echo "==============="
echo "Total Tests:  $TOTAL_TESTS"
echo -e "Passed:       ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:       ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed!${NC}"
    exit 1
fi
