# ServiceNow2 Integration Test Suite

This document describes the comprehensive test suite for the ServiceNow2 integration.

## Overview

The test suite provides comprehensive coverage for the ServiceNow2 integration, including:

- **Unit Tests**: Testing individual components and functions
- **Integration Tests**: Testing end-to-end workflows with mock ServiceNow server
- **Command Tests**: Testing CLI command structure and functionality

## Test Structure

### 1. Unit Tests

#### `internal/snow/options_test.go`
Tests for the options functionality:
- `TestLimit()`: Tests the Limit option function
- `TestFields()`: Tests the Fields option function
- `TestOffset()`: Tests the Offset option function
- `TestQuery()`: Tests the Query option function
- `TestDisplay()`: Tests the Display option function
- `TestSysID()`: Tests the SysID option function
- `TestEvalReqOptions()`: Tests option evaluation with multiple options
- `TestAssembleURL()`: Tests URL assembly with various option combinations

#### `internal/snow/records_test.go`
Tests for record operations and data structures:
- `TestResults_UnmarshalJSON()`: Tests JSON unmarshaling for Results type
- `TestTableConfig()`: Tests table configuration parsing and lookup
- `TestRecord_CreateRecord()`: Tests record creation structure
- `TestRecord_UpdateRecord()`: Tests record update structure
- `TestSnowError()`: Tests ServiceNow error response parsing
- `TestSnowResult()`: Tests ServiceNow success response parsing
- `TestTableQuery()`: Tests table query structure
- `TestTableStates()`: Tests table state configuration
- `TestTableResponses()`: Tests response configuration
- `TestTableData()`: Tests complete table data structure

#### `internal/snow/snow_test.go`
Tests for the core ServiceNow client functionality:
- `TestServiceNow_BasicAuth()`: Tests basic authentication configuration
- `TestServiceNow_OAuth()`: Tests OAuth authentication configuration
- `TestServiceNow_HTTPSConfig()`: Tests HTTPS and TLS configuration
- `TestServiceNow_HTTPConfig()`: Tests HTTP configuration
- `TestServiceNow_InvalidURL()`: Tests error handling for invalid URLs
- `TestServiceNow_DefaultPath()`: Tests default path handling
- `TestServiceNow_GlobalConnection()`: Tests global connection reuse
- `TestContext()`: Tests the custom Context wrapper
- `TestServiceNow_TLSConfiguration()`: Tests various TLS configurations
- `TestServiceNow_EmptyCredentials()`: Tests behavior with empty credentials
- `TestServiceNow_OAuthEmptyClientSecret()`: Tests OAuth fallback behavior

### 2. Command Tests

#### `cmd/root_test.go`
Tests for the root command functionality:
- `TestRootCommand()`: Tests root command initialization and structure
- `TestGlobalVariables()`: Tests global variable initialization
- `TestFlags()`: Tests flag parsing and handling
- `TestLoadConfigFile()`: Tests configuration file loading
- `TestConfigBasename()`: Tests configuration filename generation
- `TestCommandStructure()`: Tests command structure properties
- `TestExecutableName()`: Tests executable name detection
- `TestDebugFlag()`: Tests debug flag configuration
- `TestHelpFlag()`: Tests help flag configuration

#### `cmd/client/client_test.go`
Tests for the client command:
- `TestClientCommand()`: Tests client command initialization
- `TestClientFlags()`: Tests client-specific flags
- `TestClientCommandAddedToRoot()`: Tests command registration
- `TestGlobalFlags()`: Tests global flag variables
- `TestActionGroup()`: Tests ActionGroup structure and JSON handling
- `TestActionGroupEmpty()`: Tests empty ActionGroup handling
- `TestActionGroupJSONOmitEmpty()`: Tests JSON omitempty behavior
- `TestComplexActionGroup()`: Tests nested ActionGroup structures

#### `cmd/proxy/proxy_test.go`
Tests for the proxy command:
- `TestProxyCommand()`: Tests proxy command initialization
- `TestProxyFlags()`: Tests proxy-specific flags
- `TestProxyCommandAddedToRoot()`: Tests command registration
- `TestGlobalVariables()`: Tests global variable initialization
- `TestLogFileDefaultValue()`: Tests logfile flag default value
- `TestDaemonFlagDefaultValue()`: Tests daemon flag default value
- `TestCommandStructure()`: Tests command structure
- `TestCommandLongDescription()`: Tests command documentation
- `TestCommandHelpTemplate()`: Tests help generation

### 3. Integration Tests

#### `integration_test.go`
End-to-end integration tests with mock ServiceNow server:
- `MockServiceNowServer()`: Creates a mock ServiceNow server for testing
- `TestServiceNowClientIntegration()`: Tests client creation and basic requests
- `TestServiceNowOAuthIntegration()`: Tests OAuth client creation
- `TestIncidentCreationFlow()`: Tests incident creation workflow
- `TestIncidentUpdateFlow()`: Tests incident update workflow
- `TestLookupRecordFlow()`: Tests record lookup with correlation ID
- `TestErrorHandling()`: Tests error handling with invalid URLs
- `TestTableConfiguration()`: Tests table configuration parsing
- `TestCommandLineIntegration()`: Tests command line structure
- `TestFullWorkflow()`: Tests complete lookup -> create/update workflow
- `TestJSONSerialization()`: Tests JSON serialization of data structures

#### `main_test.go`
Tests for the main entry point:
- `TestMainFunction()`: Tests main function structure and command registration
- `TestPackageImports()`: Tests that packages are properly imported
- `TestCommandExecution()`: Tests command execution structure
- `TestApplicationStructure()`: Tests overall application structure
- `TestSubcommandStructure()`: Tests subcommand structure
- `TestCommandLineCompatibility()`: Tests CLI convention compliance
- `TestApplicationMetadata()`: Tests application metadata

## Running Tests

### Run All Tests
```bash
go test ./...
```

### Run Tests with Verbose Output
```bash
go test -v ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Integration Tests Only
```bash
go test -v -run "Integration" ./...
```

### Run Unit Tests Only
```bash
go test -v ./internal/snow/...
```

### Run Command Tests Only
```bash
go test -v ./cmd/...
```

## Test Coverage

The test suite provides comprehensive coverage of:

### Core Functionality (internal/snow)
- ✅ ServiceNow client creation and configuration
- ✅ HTTP/HTTPS and OAuth authentication
- ✅ URL assembly and query parameter handling
- ✅ Record CRUD operations structure
- ✅ JSON serialization/deserialization
- ✅ Table configuration parsing
- ✅ Error response handling
- ✅ TLS configuration

### Command Line Interface (cmd)
- ✅ Root command structure and flags
- ✅ Client command structure and flags
- ✅ Proxy command structure and flags
- ✅ Command registration and hierarchy
- ✅ Flag parsing and validation
- ✅ Configuration file loading
- ✅ Help and version support

### Integration Workflows
- ✅ End-to-end incident creation
- ✅ End-to-end incident updates
- ✅ Record lookup with correlation IDs
- ✅ Complete workflow testing
- ✅ Error handling and edge cases
- ✅ Mock ServiceNow server interactions

## Mock ServiceNow Server

The integration tests include a comprehensive mock ServiceNow server that simulates:

- **GET /api/now/v2/table/incident**: Returns incidents based on query parameters
- **POST /api/now/v2/table/incident**: Creates new incidents
- **PUT /api/now/v2/table/incident/:sys_id**: Updates existing incidents
- **POST /oauth_token.do**: OAuth token generation

The mock server responds with realistic ServiceNow API responses and handles various query parameters and authentication methods.

## Best Practices

### When Adding New Tests

1. **Unit Tests**: Add tests for new functions in the same package with `_test.go` suffix
2. **Integration Tests**: Update `integration_test.go` for new end-to-end workflows
3. **Command Tests**: Add command tests when adding new CLI functionality
4. **Mock Updates**: Update the mock ServiceNow server for new API endpoints

### Test Data

- Use realistic ServiceNow field names and values
- Include edge cases and error conditions
- Test both success and failure scenarios
- Use table-driven tests for multiple input scenarios

### Naming Conventions

- Test functions: `TestFunctionName()`
- Helper functions: `testHelperFunction()` or `TestHelperFunction()` if exported
- Test data: Use descriptive variable names like `expectedResult`, `testConfig`

## Continuous Integration

The test suite is designed to run in CI environments:

- No external dependencies (uses mock servers)
- Deterministic test results
- Comprehensive error handling
- Proper cleanup of resources

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure all dependencies are available in go.mod
2. **Mock Server Issues**: Check that httptest server is properly started/stopped
3. **Configuration Issues**: Verify test configurations match expected structure
4. **Race Conditions**: Tests clean up global state to prevent interference

### Debug Mode

Run tests with additional debug information:
```bash
go test -v -debug ./...
```

### Test-Specific Debugging

Use `t.Logf()` for test-specific logging:
```go
t.Logf("Test data: %+v", testData)
```

## Future Improvements

Potential areas for test expansion:

- [ ] Performance benchmarks
- [ ] Load testing with mock server
- [ ] Configuration validation tests
- [ ] Network failure simulation
- [ ] Concurrent request handling
- [ ] Memory leak detection
- [ ] Security testing (authentication, TLS)