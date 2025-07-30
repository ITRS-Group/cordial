#!/bin/bash

# Test runner script for tools/geneos
# This script runs all tests for the geneos tool and provides comprehensive reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}        Geneos Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to run tests for a specific package
run_package_tests() {
    local package="$1"
    local description="$2"
    
    echo -e "${YELLOW}Testing: $description${NC}"
    echo -e "Package: $package"
    echo "----------------------------------------"
    
    if go test -v "$package" 2>&1 | tee "/tmp/test_${package//\//_}.log"; then
        echo -e "${GREEN}✓ PASSED: $description${NC}"
        return 0
    else
        echo -e "${RED}✗ FAILED: $description${NC}"
        return 1
    fi
    echo ""
}

# Function to run tests with coverage
run_with_coverage() {
    local package="$1"
    local description="$2"
    
    echo -e "${YELLOW}Testing with Coverage: $description${NC}"
    echo -e "Package: $package"
    echo "----------------------------------------"
    
    if go test -v -coverprofile="/tmp/coverage_${package//\//_}.out" "$package" 2>&1; then
        echo -e "${GREEN}✓ PASSED: $description${NC}"
        go tool cover -func="/tmp/coverage_${package//\//_}.out" | tail -1
        return 0
    else
        echo -e "${RED}✗ FAILED: $description${NC}"
        return 1
    fi
    echo ""
}

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

# Check if we're in the right directory
if [[ ! -f "main.go" ]]; then
    echo -e "${RED}Error: main.go not found. Please run this script from the tools/geneos directory${NC}"
    exit 1
fi

# Initialize counters
PASSED=0
FAILED=0
TOTAL=0

echo -e "${BLUE}Environment Information:${NC}"
echo "Go version: $(go version)"
echo "Working directory: $(pwd)"
echo "GOPATH: ${GOPATH:-not set}"
echo "GOROOT: ${GOROOT:-not set}"
echo ""

# Test packages in order of dependency
TEST_PACKAGES=(
    ".:Main package (entry point)"
    "./internal/geneos:Core geneos functionality"
    "./internal/instance:Instance management"
    "./internal/component/gateway:Gateway component"
    "./cmd:Command line interface"
)

echo -e "${BLUE}Running tests...${NC}"
echo ""

# Run tests for each package
for test_spec in "${TEST_PACKAGES[@]}"; do
    IFS=':' read -r package description <<< "$test_spec"
    
    ((TOTAL++))
    if run_package_tests "$package" "$description"; then
        ((PASSED++))
    else
        ((FAILED++))
    fi
done

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}        Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"

if [[ $FAILED -eq 0 ]]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    echo -e "Total: $TOTAL, Passed: $PASSED, Failed: $FAILED"
else
    echo -e "${RED}Some tests failed! ✗${NC}"
    echo -e "Total: $TOTAL, Passed: $PASSED, Failed: $FAILED"
fi

echo ""

# Run specific test categories if requested
if [[ "$1" == "--coverage" ]] || [[ "$1" == "-c" ]]; then
    echo -e "${BLUE}Running tests with coverage analysis...${NC}"
    echo ""
    
    # Run coverage for main packages
    COVERAGE_PACKAGES=(
        "./internal/geneos:Core geneos functionality"
        "./internal/instance:Instance management"
        "./cmd:Command line interface"
    )
    
    for test_spec in "${COVERAGE_PACKAGES[@]}"; do
        IFS=':' read -r package description <<< "$test_spec"
        run_with_coverage "$package" "$description"
    done
    
    # Generate combined coverage report
    echo -e "${BLUE}Generating combined coverage report...${NC}"
    echo "gocovmerge /tmp/coverage_*.out > /tmp/combined_coverage.out"
    if command -v gocovmerge &> /dev/null; then
        gocovmerge /tmp/coverage_*.out > /tmp/combined_coverage.out
        go tool cover -html="/tmp/combined_coverage.out" -o "/tmp/coverage_report.html"
        echo -e "${GREEN}Coverage report generated: /tmp/coverage_report.html${NC}"
    else
        echo -e "${YELLOW}gocovmerge not available. Install with: go install github.com/wadey/gocovmerge@latest${NC}"
    fi
fi

# Run race detection tests if requested
if [[ "$1" == "--race" ]] || [[ "$1" == "-r" ]]; then
    echo -e "${BLUE}Running tests with race detection...${NC}"
    echo ""
    
    for test_spec in "${TEST_PACKAGES[@]}"; do
        IFS=':' read -r package description <<< "$test_spec"
        echo -e "${YELLOW}Race testing: $description${NC}"
        if go test -race "$package"; then
            echo -e "${GREEN}✓ RACE CLEAN: $description${NC}"
        else
            echo -e "${RED}✗ RACE DETECTED: $description${NC}"
            ((FAILED++))
        fi
        echo ""
    done
fi

# Run benchmarks if requested
if [[ "$1" == "--bench" ]] || [[ "$1" == "-b" ]]; then
    echo -e "${BLUE}Running benchmarks...${NC}"
    echo ""
    
    for test_spec in "${TEST_PACKAGES[@]}"; do
        IFS=':' read -r package description <<< "$test_spec"
        echo -e "${YELLOW}Benchmarking: $description${NC}"
        go test -bench=. -benchmem "$package" || true
        echo ""
    done
fi

# Clean up temporary files
if [[ "$1" == "--clean" ]]; then
    echo -e "${BLUE}Cleaning up temporary test files...${NC}"
    rm -f /tmp/test_*.log
    rm -f /tmp/coverage_*.out
    rm -f /tmp/combined_coverage.out
    rm -f /tmp/coverage_report.html
    echo -e "${GREEN}Cleanup complete${NC}"
fi

# Show help if requested
if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
    echo ""
    echo -e "${BLUE}Usage: $0 [OPTIONS]${NC}"
    echo ""
    echo "Options:"
    echo "  -c, --coverage    Run tests with coverage analysis"
    echo "  -r, --race        Run tests with race detection"
    echo "  -b, --bench       Run benchmarks"
    echo "  --clean           Clean up temporary test files"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                Run all tests"
    echo "  $0 --coverage     Run tests with coverage"
    echo "  $0 --race         Run tests with race detection"
    echo "  $0 --bench        Run benchmarks"
    echo ""
fi

# Exit with failure code if any tests failed
exit $FAILED