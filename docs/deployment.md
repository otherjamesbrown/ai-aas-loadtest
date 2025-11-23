## Deployment Guide

This guide covers deploying the AI-AAS Load Testing Harness to production Kubernetes environments.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured with cluster access
- Prometheus Pushgateway deployed (for metrics)
- Access to AI-AAS platform endpoints
- Docker image pushed to registry

## Architecture

```
┌─────────────────────────┐         HTTP/HTTPS         ┌─────────────────────────┐
│   Load Test Cluster     │ ───────────────────────▶  │   Platform Cluster      │
│                         │         (External)         │                         │
│  ┌───────────────────┐  │                            │  - API Router           │
│  │ Load Test Jobs    │  │                            │  - User/Org Service     │
│  │ (Workers)         │  │                            │  - vLLM Backends        │
│  └───────────────────┘  │                            └─────────────────────────┘
│           │             │
│           │ Push        │
│           ▼             │
│  ┌───────────────────┐  │
│  │ Prometheus        │  │
│  │ Pushgateway       │  │
│  └───────────────────┘  │
│           │             │
│           │ Scrape      │
│           ▼             │
│  ┌───────────────────┐  │
│  │ Prometheus        │  │
│  └───────────────────┘  │
│           │             │
│           ▼             │
│  ┌───────────────────┐  │
│  │ Grafana           │  │
│  └───────────────────┘  │
└─────────────────────────┘
```

## Quick Start

### 1. Setup Namespace and RBAC

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/rbac.yaml
```

### 2. Create Test Configuration

Create a ConfigMap with your test configuration:

```bash
kubectl apply -f deploy/k8s/configmap-smoke-test.yaml
```

Or create a custom one:

```bash
kubectl create configmap load-test-config-custom \
  -n load-testing \
  --from-file=load-test-config.yaml=configs/examples/single-org-50-users.yaml
```

### 3. Deploy Load Test Job

Using the deployment script:

```bash
./scripts/deploy.sh --config smoke
```

Or manually:

```bash
kubectl apply -f deploy/k8s/job.yaml
```

### 4. Monitor Progress

```bash
# View logs
kubectl logs -n load-testing -f job/load-test-smoke-<timestamp>

# Check status
kubectl get job -n load-testing

# View pods
kubectl get pods -n load-testing
```

## Configuration

### Environment Variables

Set these in the Job manifest or via ConfigMap:

| Variable | Description | Default |
|----------|-------------|---------|
| `CONFIG_PATH` | Path to configuration file | Required |
| `PUSHGATEWAY_URL` | Prometheus Pushgateway URL | `http://localhost:9091` |

### Command-Line Flags

- `--config`: Path to configuration YAML file
- `--pushgateway`: Pushgateway URL for metrics
- `--log-level`: Logging level (debug, info, warn, error)

### Test Configuration

Update `targets` section to point to your platform:

```yaml
spec:
  targets:
    apiRouterUrl: "https://api.your-platform.com"
    userOrgUrl: "https://admin.your-platform.com"
    adminApiKey: ""  # Optional, for bootstrap
    tlsVerify: true
    timeout: 60
```

## Scaling

### Horizontal Scaling

Run multiple jobs concurrently for higher load:

```bash
# Deploy 3 parallel jobs
for i in {1..3}; do
  ./scripts/deploy.sh --config smoke
done
```

### Vertical Scaling

Adjust resource limits in job manifest:

```yaml
resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 2000m
    memory: 4Gi
```

### User Count Scaling

Modify configuration to increase simulated users:

```yaml
spec:
  organizations:
    count: 3  # More organizations

  users:
    perOrg:
      min: 100  # More users per org
      max: 200
```

## Monitoring

### Prometheus Metrics

Load test metrics are pushed to Pushgateway and scraped by Prometheus.

**Key Metrics:**

- `loadtest_requests_total` - Total requests sent
- `loadtest_request_latency_seconds` - Request latency
- `loadtest_llm_time_to_first_token_milliseconds` - TTFT
- `loadtest_llm_tokens_per_second` - TPS
- `loadtest_errors_total` - Error count

**Query Examples:**

```promql
# Request rate
rate(loadtest_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(loadtest_request_latency_seconds_bucket[5m]))

# Error rate
rate(loadtest_errors_total[5m]) / rate(loadtest_requests_total[5m])

# Average TTFT
histogram_quantile(0.5, rate(loadtest_llm_time_to_first_token_milliseconds_bucket[5m]))

# Average TPS
histogram_quantile(0.5, rate(loadtest_llm_tokens_per_second_bucket[5m]))
```

### Grafana Dashboards

Import the provided dashboard (TBD) or create custom dashboards using the metrics above.

## Production Best Practices

### 1. Resource Limits

Always set resource limits to prevent runaway jobs:

```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 4Gi
```

### 2. Cost Limits

Set cost limits in configuration to prevent excessive spending:

```yaml
spec:
  limits:
    maxCostUSD: 100.0
    maxDurationSec: 3600
    maxErrorRate: 0.1
```

### 3. Cleanup

Jobs are configured with `ttlSecondsAfterFinished` to auto-cleanup:

```yaml
spec:
  ttlSecondsAfterFinished: 3600  # Clean up after 1 hour
```

Manual cleanup:

```bash
# Delete specific job
kubectl delete job -n load-testing load-test-smoke-20250123-120000

# Delete all completed jobs
kubectl delete job -n load-testing --field-selector status.successful=1
```

### 4. Network Policies

Restrict network access if needed:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: load-test-egress
  namespace: load-testing
spec:
  podSelector:
    matchLabels:
      app: load-test-worker
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: monitoring  # Pushgateway
  - to:
    - podSelector: {}  # Platform APIs (adjust as needed)
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8081
```

### 5. Secrets Management

For sensitive configuration (API keys), use Kubernetes Secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: load-test-secrets
  namespace: load-testing
type: Opaque
stringData:
  admin-api-key: "your-admin-key"
---
# Reference in Job
env:
- name: ADMIN_API_KEY
  valueFrom:
    secretKeyRef:
      name: load-test-secrets
      key: admin-api-key
```

## Common Test Scenarios

### Smoke Test

Quick validation with minimal load:

```yaml
spec:
  organizations:
    count: 1
  users:
    perOrg:
      min: 5
      max: 5
  limits:
    maxDurationSec: 300
```

```bash
./scripts/deploy.sh --config smoke
```

### Load Test

Sustained load with realistic user behavior:

```yaml
spec:
  organizations:
    count: 3
  users:
    perOrg:
      min: 50
      max: 100
  userBehavior:
    thinkTimeSeconds:
      base: 10
      variance: 5
  limits:
    maxDurationSec: 3600
```

### Stress Test

High concurrent load to find limits:

```yaml
spec:
  organizations:
    count: 10
  users:
    perOrg:
      min: 200
      max: 200
  userBehavior:
    thinkTimeSeconds:
      base: 1
      variance: 0
```

### Soak Test

Long-running test for stability:

```yaml
spec:
  organizations:
    count: 5
  users:
    perOrg:
      min: 100
      max: 100
  limits:
    maxDurationSec: 14400  # 4 hours
```

## Troubleshooting

### Job Fails Immediately

Check pod logs:

```bash
kubectl logs -n load-testing <pod-name>
```

Common causes:
- Invalid configuration
- Cannot reach platform APIs
- Missing RBAC permissions

### High Error Rate

Check error metrics:

```promql
rate(loadtest_errors_total[5m])
```

Common causes:
- Platform capacity limits
- Network issues
- Invalid API keys

### Metrics Not Appearing

1. Verify Pushgateway is accessible:
   ```bash
   kubectl get svc -n monitoring prometheus-pushgateway
   ```

2. Check worker logs for push errors
3. Verify Prometheus is scraping Pushgateway

## Advanced Topics

### Multi-Cluster Load Testing

Deploy workers in multiple clusters to test from different regions:

```bash
# Cluster 1 (US-East)
kubectl --context=us-east apply -f deploy/k8s/

# Cluster 2 (US-West)
kubectl --context=us-west apply -f deploy/k8s/

# Cluster 3 (EU)
kubectl --context=eu apply -f deploy/k8s/
```

### Scheduled Load Tests

Use CronJobs for recurring tests:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: load-test-daily
  namespace: load-testing
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      # Same as Job spec
```

### Custom Metrics Export

Extend metrics collection:

```go
// internal/metrics/metrics.go
func (c *Collector) RecordCustomMetric(value float64) {
    c.customMetric.Set(value)
}
```

## Support

For issues or questions:
- Load tester issues: [ai-aas-loadtest repository](https://github.com/otherjamesbrown/ai-aas-loadtest)
- Platform issues: [ai-aas repository](https://github.com/otherjamesbrown/ai-aas)
