# Geneos Testing Documentation

This document describes the comprehensive test suite for the `tools/geneos` package.

## Overview

The test suite covers all major components and functionality of the geneos tool, including:

- Main entry point and CLI routing logic
- Core geneos functionality (components, versions, errors)
- Instance management
- Component types and behaviors
- Command line interface handling

## Test Structure

### Test Files

1. **`main_test.go`** - Tests for the main entry point
   - Executable name parsing and `ctl` suffix detection
   - Component name extraction from executables
   - Command argument handling
   - CLI routing logic

2. **`internal/geneos/`** - Core functionality tests
   - `geneos_test.go` - Version comparison (existing)
   - `component_test.go` - Component registration and management
   - `errors_test.go` - Error constants and handling

3. **`internal/instance/instance_test.go`** - Instance management tests
   - Instance creation and validation
   - Configuration handling
   - Time-based operations

4. **`internal/component/gateway/gateway_test.go`** - Component implementation tests
   - Gateway component registration
   - Component aliases and parsing
   - Component configuration

5. **`cmd/root_test.go`** - CLI command tests
   - Root command initialization
   - Command annotations and flags
   - Context management

## Running Tests

### Basic Test Execution

```bash
# Run all tests
./test_runner.sh

# Run tests with coverage
./test_runner.sh --coverage

# Run tests with race detection
./test_runner.sh --race

# Run benchmarks
./test_runner.sh --bench

# Show help
./test_runner.sh --help
```

### Using Go Directly

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/geneos
go test ./cmd
go test ./internal/instance

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

## Test Coverage

The test suite aims to provide comprehensive coverage of:

### Main Package (`main_test.go`)
- ✅ Entry point logic
- ✅ Symlink detection (`ctl` suffix)
- ✅ Component name extraction
- ✅ Command argument parsing
- ✅ CLI routing between geneos and ctl commands

### Internal/Geneos Package
- ✅ Version comparison (`geneos_test.go`)
- ✅ Component registration (`component_test.go`)
- ✅ Component parsing and validation
- ✅ Error constants and handling (`errors_test.go`)
- ✅ Component aliases and lookups

### Instance Management (`internal/instance/`)
- ✅ Instance creation and initialization
- ✅ Configuration management
- ✅ Host operations
- ✅ Time-based functionality

### Component Types (`internal/component/`)
- ✅ Gateway component (example implementation)
- ✅ Component registration and discovery
- ✅ Component field validation
- ✅ Component methods and interfaces

### Command Interface (`cmd/`)
- ✅ Root command initialization
- ✅ Command annotations and flags
- ✅ Context management
- ✅ Command validation

## Test Patterns and Best Practices

### Table-Driven Tests
Most tests use table-driven patterns for comprehensive scenario coverage:

```go
tests := []struct {
    name     string
    input    string
    expected string
    wantErr  bool
}{
    // test cases
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

### Error Testing
Error conditions are thoroughly tested:

```go
func TestErrorConstants(t *testing.T) {
    if ErrNotExist.Error() != "does not exist" {
        t.Errorf("unexpected error message")
    }
}
```

### Context Testing
Command context management is validated:

```go
func TestCmddata(t *testing.T) {
    ctx := context.WithValue(context.Background(), CmdKey, cmdVal)
    cmd.SetContext(ctx)
    result := cmddata(cmd)
    // validation
}
```

## Test Dependencies

The test suite requires:

- Go 1.24+ (as specified in go.mod)
- Standard Go testing framework
- github.com/spf13/cobra (for CLI testing)
- github.com/itrs-group/cordial (internal dependency)

### Optional Dependencies for Enhanced Testing

- `gocovmerge` for combined coverage reports:
  ```bash
  go install github.com/wadey/gocovmerge@latest
  ```

## Continuous Integration

The test suite is designed to work in CI environments:

```bash
# Basic CI test run
go test -v ./...

# CI with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# CI with race detection
go test -race ./...
```

## Adding New Tests

When adding new functionality:

1. **Create test files** alongside source files with `_test.go` suffix
2. **Follow naming conventions**: `TestFunctionName` for functions
3. **Use table-driven tests** for multiple scenarios
4. **Test error conditions** thoroughly
5. **Update test runner** if adding new packages
6. **Document test purpose** in comments

### Example Test Structure

```go
package mypackage

import "testing"

func TestNewFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewFunction(tt.input)
            
            if tt.wantErr && err == nil {
                t.Error("expected error but got none")
            }
            
            if !tt.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

## Troubleshooting Tests

### Common Issues

1. **Import cycle errors**: Ensure test files don't create circular dependencies
2. **Missing dependencies**: Run `go mod tidy` to resolve dependencies
3. **Context-related tests**: Some tests may require proper setup of geneos LOCAL host
4. **Platform-specific tests**: Some functionality may behave differently on different OS

### Debugging Failed Tests

```bash
# Run specific failing test with verbose output
go test -v -run TestSpecificFunction ./package

# Run with additional debug information
go test -v -x ./package

# Check test logs
./test_runner.sh 2>&1 | tee test_output.log
```

## Future Enhancements

Potential areas for test expansion:

- Integration tests with real geneos instances
- Performance benchmarks for critical paths
- Mock testing for external dependencies
- End-to-end CLI testing
- Configuration file testing
- Network operation testing

---

For questions or issues with the test suite, please refer to the main project documentation or create an issue in the project repository.