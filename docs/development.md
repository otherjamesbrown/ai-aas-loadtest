# Development Guide

This guide covers development workflows for the AI-AAS Load Testing Harness.

## Prerequisites

- Go 1.22 or later
- Docker (for building images)
- kubectl (for Kubernetes deployment)
- Access to a Kubernetes cluster
- Access to the AI-AAS platform API endpoints

## Project Structure

```
ai-aas-loadtest/
├── cmd/
│   └── load-test-worker/       # Main application entry point
├── internal/
│   ├── bootstrap/              # Organization/user/key creation
│   ├── config/                 # Configuration loading and types
│   ├── metrics/                # Prometheus metrics collection
│   ├── questions/              # Question generation strategies
│   ├── simulator/              # User behavior simulation
│   └── worker/                 # Main orchestration logic
├── configs/examples/           # Example test configurations
├── deploy/k8s/                 # Kubernetes manifests
├── scripts/                    # Deployment and utility scripts
└── docs/                       # Documentation
```

## Development Workflow

### 1. Local Development

#### Build the Binary

```bash
make build
```

This creates `bin/load-test-worker`.

#### Run Tests

```bash
# All tests
make test

# Specific package
go test -v ./internal/config/...

# With coverage
go test -cover ./...
```

#### Run Locally

```bash
# Using a local configuration file
./bin/load-test-worker \
  --config=configs/examples/smoke-test.yaml \
  --pushgateway=http://localhost:9091 \
  --log-level=debug
```

**Note**: Update the configuration file to point to your local or development API endpoints.

### 2. Code Quality

#### Format Code

```bash
make fmt
```

#### Lint Code

```bash
make lint
```

Requires [golangci-lint](https://golangci-lint.run/) to be installed.

#### Run All Checks

```bash
make check
```

Runs format, lint, and tests.

### 3. Docker Development

#### Build Docker Image

```bash
make docker-build
```

#### Build Multi-Architecture Image

```bash
make docker-build-multiarch
```

This builds for both `linux/amd64` and `linux/arm64` and pushes to Docker Hub.

#### Test Docker Image Locally

```bash
docker run --rm \
  -v $(pwd)/configs/examples:/config:ro \
  -e PUSHGATEWAY_URL=http://host.docker.internal:9091 \
  otherjamesbrown/ai-aas-loadtest:latest \
  --config=/config/smoke-test.yaml \
  --log-level=debug
```

## Adding New Features

### Adding a New Question Strategy

1. Open `internal/questions/generator.go`
2. Add your strategy constant to the `Strategy` type
3. Implement the generation function (e.g., `func (g *Generator) myNewStrategy() []string`)
4. Add case to the `Generate()` switch statement
5. Add tests in `internal/questions/generator_test.go`

Example:

```go
// In generator.go
const (
    // ... existing strategies
    StrategyScientific Strategy = "scientific"
)

func (g *Generator) scientificQuestions() []string {
    // Implementation
}

// Add to Generate() switch
case StrategyScientific:
    return g.scientificQuestions()
```

### Adding New Metrics

1. Open `internal/metrics/metrics.go`
2. Add metric definition in `NewCollector()`
3. Register the metric
4. Add collection method

Example:

```go
// Define
customMetric := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "loadtest_custom_metric",
        Help: "Description of custom metric",
    },
    []string{"label1", "label2"},
)

// Register
registry.MustRegister(customMetric)

// Add to Collector struct
type Collector struct {
    // ... existing fields
    customMetric *prometheus.CounterVec
}

// Add collection method
func (c *Collector) RecordCustomEvent(label1, label2 string) {
    c.customMetric.WithLabelValues(label1, label2).Inc()
}
```

### Adding Configuration Options

1. Add field to appropriate struct in `internal/config/types.go`
2. Add YAML tags
3. Add validation in `internal/config/config.go`
4. Update example configs in `configs/examples/`
5. Add tests in `internal/config/config_test.go`

## Testing

### Unit Tests

All packages should have comprehensive unit tests:

- `internal/config/config_test.go` - Configuration loading and validation
- `internal/bootstrap/bootstrap_test.go` - API client with mock server
- `internal/questions/generator_test.go` - Question generation and uniqueness

### Integration Tests

Integration tests require a running platform. Set up your environment:

```bash
# Start platform services (in ai-aas repository)
cd /path/to/ai-aas
make up

# Run integration tests
cd /path/to/ai-aas-loadtest
make test-integration
```

### Test Coverage

```bash
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

## Debugging

### Enable Debug Logging

```bash
./bin/load-test-worker \
  --config=configs/examples/smoke-test.yaml \
  --log-level=debug
```

### Common Issues

#### "No API key found for user"

- Check that bootstrap completed successfully
- Verify API key creation in logs
- Check `runtime.APIKeys` mapping

#### "Request failed with status 401"

- Verify API endpoints are accessible
- Check API key is valid
- Ensure Authorization header is set correctly

#### "Failed to push metrics"

- Verify Pushgateway is running and accessible
- Check Pushgateway URL configuration
- Review network connectivity

### Profiling

To profile CPU usage:

```bash
go run -cpuprofile=cpu.prof ./cmd/load-test-worker --config=...
go tool pprof cpu.prof
```

To profile memory:

```bash
go run -memprofile=mem.prof ./cmd/load-test-worker --config=...
go tool pprof mem.prof
```

## Kubernetes Development

### Deploy to Local Cluster

```bash
# Using kind or minikube
./scripts/deploy.sh --config smoke
```

### View Logs

```bash
kubectl logs -n load-testing -f job/load-test-smoke-<timestamp>
```

### Debug Pod

```bash
# Get pod name
kubectl get pods -n load-testing

# Exec into pod
kubectl exec -it -n load-testing <pod-name> -- /bin/sh

# View config
kubectl exec -it -n load-testing <pod-name> -- cat /config/load-test-config.yaml
```

## Release Process

1. Update version in relevant files
2. Run all tests: `make check`
3. Build and test Docker image
4. Tag release:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
5. Build and push multi-arch image:
   ```bash
   make docker-build-multiarch VERSION=0.1.0
   ```
6. Update documentation

## Contributing

1. Create feature branch from `main`
2. Make changes with tests
3. Run `make check` to ensure quality
4. Submit pull request
5. Ensure CI passes

## Resources

- [Specification](SPECIFICATION.md)
- [Configuration Reference](configuration.md)
- [Metrics Reference](metrics.md)
- [Deployment Guide](deployment.md)
