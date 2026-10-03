# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Building and Testing

```bash
# Build the plugin binary
make build

# Run tests
make test

# Run linting (requires golangci-lint)
make lint

# Install to local plugin directory
make install
```

## Project Architecture

This is a gRPC plugin that implements the CostSource service from `finfocus-spec`. Key components:

- **gRPC Server**: Listens on port 50051, implements the CostSource service methods
- **Kubecost Client**: HTTP client that queries the Kubecost API for allocation data
- **Configuration**: Supports both environment variables and YAML config files

## Key Implementation Details

### Resource ID Mapping
The plugin maps resource IDs to Kubecost filters:
- `namespace/<name>` → filter by namespace
- `pod/<namespace>/<name>` → filter by namespace and pod
- `controller/<namespace>/<name>` → filter by namespace and controller
- `node/<name>` → filter by node

### Cost Projections
`GetProjectedCost` calculates a 30-day average from historical data and extrapolates monthly costs. This is a simple MVP approach that can be enhanced with more sophisticated forecasting.

### Cost Prediction API
The plugin supports IBM Kubecost Cost Prediction API for proactive cost forecasting:
- **Endpoint**: `POST /model/prediction/speccost`
- **Purpose**: Predict cost impact for Kubernetes workloads before deployment
- **Supported Formats**: YAML and JSON workload specifications
- **Parameters**: cluster ID, namespace, prediction window, usage data options

### Error Handling
- HTTP client includes timeout support via context
- TLS certificate verification can be disabled for development
- All errors are propagated with context

## Dependencies

The plugin depends on:
- `github.com/rshade/finfocus-spec/sdk/go/proto` - Protocol buffer definitions
- Standard gRPC and protobuf libraries
- `gopkg.in/yaml.v3` for configuration parsing

## Testing Approach

- Unit tests for individual components (client, config, server methods)
- Integration tests can use the testdata/ JSON files
- For live testing, set `KUBECOST_BASE_URL` environment variable

## Common Development Tasks

### Adding New Resource Types
1. Update the `Supports()` method in `kubecost_server.go`
2. Add mapping logic in `GetActualCost()` for the new resource ID format
3. Update `plugin.manifest.json` with the new resource type

### Using Cost Prediction API
The prediction API allows forecasting costs before deployment:

```go
// Configure prediction settings
cfg := kubecost.Config{
    BaseURL:          "https://kubecost.example.com",
    APIToken:         "your-api-token",
    ClusterID:        "production-cluster",
    DefaultNamespace: "default",
    PredictionWindow: "7d",
}

// Create prediction request
req := kubecost.PredictionRequest{
    ClusterID:        "production-cluster",
    DefaultNamespace: "web-services",
    Window:           "2d",
    WorkloadSpec:     yamlDeploymentSpec,
    NoUsage:          false, // Include historical usage data
}

// Get cost prediction
resp, err := client.PredictSpecCost(ctx, req)
// resp.CostBefore: "$42.50/month"
// resp.CostAfter:  "$67.80/month"
// resp.CostChange: "+$25.30/month"
```

### Environment Variables for Prediction
```bash
export KUBECOST_CLUSTER_ID="production-cluster"
export KUBECOST_DEFAULT_NAMESPACE="default"
export KUBECOST_PREDICTION_WINDOW="7d"
```

### Debugging the gRPC Server
The server includes reflection support, so you can use tools like `grpcurl`:
```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe CostSource
```

### Modifying Kubecost API Calls
The HTTP client in `client.go` handles the Kubecost API interaction. To add new endpoints:
1. Add new methods to the Client struct
2. Define request/response types
3. Handle the API call with proper error handling and timeout

## Go-Specific Development Patterns

### Struct Field Consistency
- **Critical**: Field names must be consistent across related structs
- Example issue: `PVCost` vs `PVCCost` caused compilation failures
- Always verify field names when copying/adapting struct definitions

### URL Parameter Handling
- Map iteration order is random in Go, affecting URL parameter order
- Use flexible testing patterns that accept multiple valid orders:
  ```go
  if !contains(url, "order1") && !contains(url, "order2") {
      t.Errorf("Expected URL to contain parameters in either order")
  }
  ```

### Error Handling Best Practices
- Use `errors.Is(err, os.ErrNotExist)` for file existence checks
- Graceful degradation: missing config files should not fail hard
- Always check for unused variables in strict builds (`_ = variable`)

### Protobuf and Timestamp Handling
- Use `timestamppb.New(time)` instead of manual timestamp construction
- Avoid duplicate helper functions when standard library provides them

## Project-Specific Architecture Insights

### Allocation API Methods
The project has two allocation methods with different capabilities:
- **Basic `Allocation`**: Simple method, was incomplete (fixed in recent update)
- **Enhanced `EnhancedAllocation`**: Full-featured method using `GetDetailedAllocation` + `ConvertToSimpleResponse`
- **Recommendation**: Use `EnhancedAllocation` for new implementations

### Testing Architecture
- Uses mock HTTP servers (`httptest.NewServer`) for integration testing
- Mock protobuf types defined locally due to missing `finfocus-spec` dependency
- Integration tests validate end-to-end functionality including URL building and response parsing

### Configuration Handling
- Supports both environment variables and YAML files
- Environment variables take precedence
- Config loading gracefully handles missing files with `os.ErrNotExist`

## Common Development Issues & Solutions

### Dependency Management
- Issue: Missing `go.sum` entries cause build failures
- Solution: Run `go mod tidy` when adding new imports or after git operations
- Symptom: "missing go.sum entry for module" errors

### Linting Configuration
- Issue: `embeddedstructfieldcheck` linter requires golangci-lint v2.3+
- Solution: Disable incompatible linters in `.golangci.yml` for older versions
- Check version compatibility before enabling new linters

### URL Building and Testing
- Issue: Go's map iteration randomness affects URL parameter order
- Solution: Test for multiple valid parameter orders or use URL parsing
- Debug tip: Create temporary debug files to inspect generated URLs

### Parallel Process Conflicts
- Issue: "parallel golangci-lint is running" errors
- Solution: Wait between runs or use direct `golangci-lint run` command
- Alternative: Use `--timeout` flag to prevent hanging processes

## Testing Strategies

### Integration Testing with Mock Servers
```go
// Effective pattern for testing HTTP API clients
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Validate request parameters
    // Return mock response
}))
defer mockServer.Close()

// Point client to mock server
client := NewClient(Config{BaseURL: mockServer.URL})
```

### URL Parameter Testing
- Test both parameter presence and proper encoding
- Account for random map iteration order in Go
- Use helper functions to reduce test code duplication

### Error Scenario Testing
- Test missing configuration files
- Test network timeouts and failures  
- Test malformed API responses
- Test invalid parameter combinations

## Tool Usage and Workflow Optimizations

### Go Testing Commands
```bash
# Test specific package with verbose output
go test ./internal/allocation -v

# Test specific function
go test ./internal/allocation -v -run TestFunctionName

# Test with race detection
go test -race ./...
```

### Debugging Techniques
- Create temporary debug files for URL inspection
- Use `fmt.Printf` debugging in tests (remove before commit)
- Check HTTP request/response details in mock server handlers

### Build and Validation Workflow
```bash
# Complete validation sequence
go mod tidy
make build
make test
make lint
```

### golangci-lint Best Practices
- Check version compatibility before updating config
- Use `--timeout=60s` flag for slow systems
- Disable strict linters during development, enable for production