# AI-AAS Load Testing Harness

**External load testing tool for the AI-as-a-Service platform**

## Overview

This is a **separate operational tool** that runs in an isolated Kubernetes cluster to perform load testing against the AI-AAS platform. It simulates realistic user behavior to validate platform performance, measure LLM metrics (TTFT, TPS), and identify bottlenecks.

### Key Distinction

⚠️ **This is NOT part of the AI-AAS platform codebase.** This is an external testing tool that:
- Runs in a **separate Kubernetes cluster** (load test cluster)
- Tests the platform from the **outside** (like a real user would)
- Has **no internal dependencies** on platform code
- Communicates **only via public APIs** (HTTP/HTTPS)

```
┌─────────────────────────┐         HTTP/HTTPS         ┌─────────────────────────┐
│   Load Test Cluster     │ ───────────────────────▶  │   Platform Cluster      │
│   (This Repository)     │         (External)         │   (ai-aas repository)   │
│                         │                            │                         │
│  - Load Test Workers    │                            │  - API Router           │
│  - Prometheus           │                            │  - User/Org Service     │
│  - Grafana              │                            │  - Budget Service       │
│                         │                            │  - vLLM Backends        │
└─────────────────────────┘                            └─────────────────────────┘
     (This Repo)                                             (ai-aas Repo)
```

## Features

✅ **Self-Bootstrapping** - Workers automatically create orgs, users, and API keys
✅ **Realistic Behavior** - Variable think times, unique questions, multi-turn conversations
✅ **Template-Driven** - Change test scenarios via ConfigMaps (no recompilation)
✅ **LLM Metrics** - Measure TTFT, TPS, cost efficiency
✅ **Scalable** - 10 to 1000+ concurrent simulated users
✅ **Kubernetes-Native** - Deploy as Jobs, scale horizontally

## Quick Start

### Prerequisites

- Kubernetes cluster for load testing (separate from platform cluster)
- Network access from load test cluster to platform API endpoints
- Prometheus Pushgateway for metrics collection

### Deploy

```bash
# 1. Create namespace
kubectl apply -f deploy/k8s/namespace.yaml

# 2. Apply RBAC
kubectl apply -f deploy/k8s/rbac.yaml

# 3. Create test configuration
kubectl apply -f configs/examples/single-org-50-users.yaml

# 4. Run load test
./scripts/deploy.sh --config single-org-50-users
```

### Monitor

```bash
# View logs
kubectl logs -n load-testing -f job/load-test-<run-id>

# Check metrics in Prometheus
# Query: loadtest_run_requests_per_second{test_run_id="test-<run-id>"}

# View Grafana dashboard
# Dashboard: Load Testing → Overview
```

## Architecture

### Components

```
Load Test Worker (Pod)
├── Config Loader          # Read ConfigMap
├── Bootstrap Manager      # Create orgs/users/keys via platform API
├── Question Generator     # Generate unique questions
├── User Simulator         # Simulate realistic user behavior
│   ├── HTTP Client        # Send requests to platform
│   ├── Think Time         # Wait between requests
│   └── Metrics Collector  # Track performance
└── Metrics Exporter       # Push to Prometheus
```

### Workflow

1. **Bootstrap Phase**
   - Worker reads configuration from ConfigMap
   - Creates organization via User/Org Service API
   - Creates N users and API keys
   - Stores credentials in memory

2. **Simulation Phase**
   - Spawns goroutine per user (concurrent simulation)
   - Each user:
     * Generates unique questions
     * Sends chat completion requests to API Router
     * Measures latency and token usage
     * Waits (think time) before next question
     * Repeats until session complete

3. **Metrics Phase**
   - Exports metrics to Prometheus Pushgateway every 10s
   - Final metrics export on completion

4. **Cleanup Phase** (optional)
   - Deletes test orgs, users, and API keys
   - Exits cleanly

## Configuration

Test scenarios are defined via Kubernetes ConfigMaps:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: load-test-config
  namespace: load-testing
data:
  load-test-config.yaml: |
    apiVersion: loadtest.ai-aas.dev/v1
    kind: LoadTestScenario
    spec:
      organizations:
        count: 1
      users:
        perOrg: { min: 50, max: 50 }
      userBehavior:
        thinkTimeSeconds: { base: 5, variance: 2 }
      targets:
        apiRouterUrl: "https://api.your-platform.com"
        userOrgUrl: "https://admin.your-platform.com"
```

See [Configuration Reference](docs/configuration.md) for full schema.

## Metrics

The harness exports comprehensive metrics to Prometheus:

### Worker Metrics
- `loadtest_worker_status` - Worker lifecycle state
- `loadtest_worker_users_active` - Active user simulations

### User Metrics
- `loadtest_user_requests_total` - Request count per user
- `loadtest_user_latency_seconds` - Request latency distribution
- `loadtest_user_tokens_total` - Token consumption
- `loadtest_user_cost_usd` - Cost per user
- `loadtest_user_errors_total` - Error count by type

### LLM Performance Metrics
- `loadtest_llm_time_to_first_token_milliseconds` - TTFT distribution
- `loadtest_llm_tokens_per_second` - TPS distribution
- `loadtest_llm_backend_saturation_ratio` - Backend load

See [Metrics Reference](docs/metrics.md) for full list.

## Development

### Build

```bash
# Build binary
make build

# Build Docker image
make docker-build

# Run tests
make test

# Run integration tests (requires local platform)
make test-integration
```

### Local Testing

```bash
# Start local platform services (see ai-aas repository)
cd /path/to/ai-aas
make up

# Run worker locally
cd /path/to/ai-aas-loadtest
go run ./cmd/load-test-worker --config configs/examples/smoke-test.yaml
```

## Repository Structure

```
ai-aas-loadtest/
├── cmd/
│   └── load-test-worker/        # Main binary
├── internal/
│   ├── config/                  # Configuration loading
│   ├── bootstrap/               # Org/user creation
│   ├── questions/               # Question generation
│   ├── simulator/               # User simulation
│   ├── metrics/                 # Prometheus metrics
│   └── worker/                  # Main orchestration
├── configs/
│   └── examples/                # Example test configs
├── deploy/
│   └── k8s/                     # Kubernetes manifests
├── test/
│   ├── integration/             # Integration tests
│   └── unit/                    # Unit tests
├── scripts/                     # Build/deploy scripts
├── docs/                        # Documentation
├── Dockerfile
├── Makefile
└── README.md
```

## Platform Repository

The platform being tested lives in a separate repository:
- **Repository**: `ai-aas` (separate repo)
- **Location**: https://github.com/otherjamesbrown/ai-aas
- **Relationship**: This tool tests that platform via external APIs

## Related Documentation

- [Specification](docs/SPECIFICATION.md) - Copied from ai-aas/specs/014-load-testing-harness/
- [Configuration Reference](docs/configuration.md)
- [Metrics Reference](docs/metrics.md)
- [Development Guide](docs/development.md)
- [Deployment Guide](docs/deployment.md)

## License

[Same as platform repository]

## Support

For issues or questions:
- Platform issues → [ai-aas repository](https://github.com/otherjamesbrown/ai-aas)
- Load tester issues → This repository (ai-aas-loadtest)
