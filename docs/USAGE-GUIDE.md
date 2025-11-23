# AI-AAS Load Testing Harness - Complete Usage Guide

This comprehensive guide covers all aspects of using the AI-AAS Load Testing Harness, from quick starts to advanced scenarios.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Running Tests](#running-tests)
5. [Monitoring Results](#monitoring-results)
6. [Common Scenarios](#common-scenarios)
7. [Troubleshooting](#troubleshooting)
8. [Advanced Usage](#advanced-usage)

---

## Quick Start

### 5-Minute Quickstart (Using Existing Cluster)

If you already have a Kubernetes cluster:

```bash
# 1. Clone the repository
git clone https://github.com/otherjamesbrown/ai-aas-loadtest.git
cd ai-aas-loadtest

# 2. Deploy infrastructure
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/rbac.yaml
kubectl apply -f deploy/k8s/configmap-smoke-test.yaml

# 3. Run smoke test
./scripts/deploy.sh --config smoke

# 4. View logs
kubectl logs -n load-testing -f job/load-test-smoke-$(date +%Y%m%d)
```

### Complete Setup (New Linode Cluster)

Start from scratch with automated cluster creation:

```bash
# 1. Clone the repository
git clone https://github.com/otherjamesbrown/ai-aas-loadtest.git
cd ai-aas-loadtest

# 2. Set your Linode API token
export LINODE_TOKEN="your-linode-api-token"

# 3. Create and configure cluster (10-15 minutes)
./scripts/create-lke-cluster.sh \
  --name my-loadtest-cluster \
  --region us-east \
  --node-type g6-standard-2 \
  --node-count 3

# 4. The script will install everything and show you next steps!
```

---

## Installation

### Prerequisites

- **Kubernetes Cluster**: 1.24+ (LKE, EKS, GKE, or any standard K8s)
- **kubectl**: Configured with cluster access
- **Docker**: For building custom images (optional)
- **Helm**: For installing monitoring (optional)
- **Go 1.22+**: For local development (optional)

### Option 1: Automated LKE Cluster

Perfect for creating a dedicated load testing cluster:

```bash
./scripts/create-lke-cluster.sh --token $LINODE_TOKEN
```

This script:
- ✅ Creates LKE cluster
- ✅ Installs Prometheus + Pushgateway
- ✅ Configures namespace and RBAC
- ✅ Downloads kubeconfig
- ✅ Validates everything works

**Cost**: ~$72/month for 3x g6-standard-2 nodes (2 CPU, 4GB RAM each)

### Option 2: Use Existing Cluster

If you already have a Kubernetes cluster:

```bash
# Set your kubeconfig
export KUBECONFIG=/path/to/your/kubeconfig.yaml

# Deploy infrastructure
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/rbac.yaml
```

### Option 3: Local Development

For testing and development:

```bash
# Install dependencies
go mod download

# Build binary
make build

# Run locally (requires config file)
./bin/load-test-worker \
  --config configs/examples/smoke-test.yaml \
  --pushgateway http://localhost:9091
```

---

## Configuration

### Configuration Structure

Tests are defined using YAML configuration files. The structure follows Kubernetes-like conventions:

```yaml
apiVersion: loadtest.ai-aas.dev/v1
kind: LoadTestScenario
metadata:
  name: my-test
  namespace: load-testing
spec:
  organizations: # How many orgs to create
  users:         # How many users per org
  userBehavior:  # How users behave
  targets:       # What to test
  limits:        # Safety limits
```

### Key Configuration Sections

#### Organizations

```yaml
spec:
  organizations:
    count: 3                    # Create 3 organizations
    namePrefix: "test-org"      # Names: test-org-1, test-org-2, test-org-3
    budget:
      limitUSD: 100.0           # $100 total budget per org
      dailyUSD: 20.0            # $20 daily budget
```

#### Users

```yaml
spec:
  users:
    perOrg:
      min: 10                   # Minimum 10 users per org
      max: 50                   # Maximum 50 users per org
    namePrefix: "test-user"
    apiKeys:
      perUser: 1                # 1 API key per user
      expiryDays: 7             # Keys expire in 7 days
```

#### User Behavior (Think Time)

```yaml
spec:
  userBehavior:
    thinkTimeSeconds:
      base: 10                  # Base think time: 10 seconds
      variance: 5               # Variance: ±5 seconds
      distribution: "gaussian"  # uniform, gaussian, or exponential
      min: 2                    # Minimum 2 seconds
      max: 30                   # Maximum 30 seconds

    questionsPerSession:
      min: 5                    # Each user asks 5-15 questions
      max: 15

    conversationStyle:
      multiTurnProbability: 0.3 # 30% chance of follow-up questions
      turnsPerConversation:
        min: 2
        max: 5
```

#### Targets (Platform Endpoints)

```yaml
spec:
  targets:
    apiRouterUrl: "https://api.your-platform.com"
    userOrgUrl: "https://admin.your-platform.com"
    tlsVerify: true
    timeout: 60                 # Request timeout in seconds
```

#### Safety Limits

```yaml
spec:
  limits:
    maxCostUSD: 500.0           # Stop if cost exceeds $500
    maxDurationSec: 3600        # Stop after 1 hour
    maxErrorRate: 0.1           # Stop if >10% error rate
```

### Example Configurations

The repository includes several pre-built configurations:

**configs/examples/smoke-test.yaml**
- 1 organization, 5 users
- 3-5 questions per user
- 5 minute duration
- Perfect for quick validation

**configs/examples/single-org-50-users.yaml**
- 1 organization, 50 users
- 5-15 questions per user
- Realistic think times
- Production-like load

---

## Running Tests

### Method 1: Quick Deploy (Local or Remote)

```bash
# Using default smoke test
./scripts/deploy.sh --config smoke

# Using custom config
./scripts/deploy.sh --config single-org-50-users
```

### Method 2: Remote Cluster Deploy

```bash
# Deploy to specific cluster context
./scripts/deploy-remote.sh \
  --context production-cluster \
  --config load-test \
  --api-router-url https://api.prod.example.com \
  --user-org-url https://admin.prod.example.com
```

### Method 3: Full Build and Deploy

Build Docker image and deploy in one command:

```bash
# Build latest and deploy
./scripts/full-deploy.sh --config smoke

# Build specific version and deploy to production
./scripts/full-deploy.sh \
  --version v1.0.0 \
  --context production \
  --config production-load-test
```

### Method 4: Manual Kubernetes

For maximum control:

```bash
# 1. Create your ConfigMap
kubectl create configmap load-test-config-custom \
  -n load-testing \
  --from-file=load-test-config.yaml=my-config.yaml

# 2. Edit and apply job manifest
kubectl apply -f deploy/k8s/job.yaml
```

---

## Monitoring Results

### View Logs

```bash
# Follow logs in real-time
kubectl logs -n load-testing -f job/load-test-smoke-20251123-120000

# View specific pod logs
kubectl logs -n load-testing <pod-name>

# Get last 100 lines
kubectl logs -n load-testing job/load-test-smoke-20251123-120000 --tail=100
```

### Check Job Status

```bash
# View all jobs
kubectl get jobs -n load-testing

# Describe specific job
kubectl describe job -n load-testing load-test-smoke-20251123-120000

# View pods
kubectl get pods -n load-testing -l app=load-test-worker
```

### Prometheus Metrics

If you installed monitoring:

```bash
# Port-forward Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Open browser to http://localhost:9090
```

**Key Metrics to Query:**

```promql
# Request rate
rate(loadtest_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(loadtest_request_latency_seconds_bucket[5m]))

# Error rate
rate(loadtest_errors_total[5m]) / rate(loadtest_requests_total[5m])

# Average TTFT (Time to First Token)
histogram_quantile(0.5, rate(loadtest_llm_time_to_first_token_milliseconds_bucket[5m]))

# Average TPS (Tokens per Second)
histogram_quantile(0.5, rate(loadtest_llm_tokens_per_second_bucket[5m]))
```

### Grafana Dashboards

```bash
# Port-forward Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Open browser to http://localhost:3000
# Login: admin / prom-operator
```

---

## Common Scenarios

### Scenario 1: Smoke Test (Quick Validation)

**Use Case**: Validate platform is working before larger tests

```bash
./scripts/deploy.sh --config smoke
```

**What it does:**
- Creates 1 org with 5 users
- Each user asks 3-5 questions
- Runs for ~5 minutes
- Stops at $5 cost limit

### Scenario 2: Load Test (Realistic Production Load)

**Use Case**: Simulate realistic user traffic

```yaml
# Create config: configs/examples/production-load.yaml
spec:
  organizations:
    count: 5
  users:
    perOrg:
      min: 50
      max: 100
  userBehavior:
    thinkTimeSeconds:
      base: 15
      variance: 10
      distribution: "gaussian"
  limits:
    maxDurationSec: 3600  # 1 hour
    maxCostUSD: 200.0
```

```bash
./scripts/deploy-remote.sh --config production-load
```

### Scenario 3: Stress Test (Find Limits)

**Use Case**: Determine maximum capacity

```yaml
# High concurrency, minimal think time
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
      distribution: "uniform"
```

### Scenario 4: Soak Test (Long-Running Stability)

**Use Case**: Test for memory leaks and long-term stability

```yaml
spec:
  organizations:
    count: 3
  users:
    perOrg:
      min: 50
      max: 50
  limits:
    maxDurationSec: 14400  # 4 hours
    maxCostUSD: 500.0
```

### Scenario 5: Multi-Region Test

**Use Case**: Test from different geographic locations

```bash
# Deploy to US-East cluster
./scripts/deploy-remote.sh \
  --context us-east-cluster \
  --config load-test

# Deploy to EU cluster
./scripts/deploy-remote.sh \
  --context eu-west-cluster \
  --config load-test

# Deploy to Asia cluster
./scripts/deploy-remote.sh \
  --context ap-southeast-cluster \
  --config load-test
```

---

## Troubleshooting

### Job Fails Immediately

**Symptoms**: Pod status shows `Error` or `CrashLoopBackOff`

**Debug**:
```bash
# View pod logs
kubectl logs -n load-testing <pod-name>

# Describe pod for events
kubectl describe pod -n load-testing <pod-name>
```

**Common Causes**:
- Invalid configuration YAML
- Cannot reach platform APIs
- Missing RBAC permissions
- Invalid API endpoints

### High Error Rate

**Symptoms**: Logs show many failed requests

**Debug**:
```bash
# Check Prometheus
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Query error rate
rate(loadtest_errors_total[5m])
```

**Common Causes**:
- Platform capacity limits reached
- Network connectivity issues
- Invalid API keys
- Rate limiting

### Metrics Not Appearing

**Symptoms**: No data in Prometheus

**Debug**:
```bash
# Check Pushgateway is running
kubectl get svc -n monitoring prometheus-pushgateway

# Check worker logs for push errors
kubectl logs -n load-testing <pod-name> | grep -i push

# Verify Prometheus is scraping Pushgateway
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Check Targets page
```

### Cluster Out of Resources

**Symptoms**: Pods pending, not enough CPU/memory

**Debug**:
```bash
# Check node resources
kubectl top nodes

# Check pod resource requests
kubectl describe pod -n load-testing <pod-name>
```

**Solutions**:
- Scale up cluster nodes
- Reduce resource requests in Job manifest
- Use fewer concurrent users

---

## Advanced Usage

### Custom Question Strategies

Create custom test scenarios by mixing question types:

```yaml
spec:
  testTypes:
    - name: "technical_heavy"
      weight: 70
      questionStrategy: "technical"
      modelTargeting:
        mediumModels: ["gpt-4o"]

    - name: "simple_queries"
      weight: 30
      questionStrategy: "mathematical"
      modelTargeting:
        slmModels: ["gpt-3.5-turbo"]
```

### Document Analysis Testing

Test long-form document processing:

```yaml
spec:
  testTypes:
    - name: "document_analysis"
      weight: 100
      questionStrategy: "technical"
      documentLibrary:
        enabled: true
        bucketName: "test-documents"
        documentPaths: ["technical/", "manuals/"]
        sampleSize: 10
```

### Cache Testing

Test cache behavior and isolation:

```yaml
spec:
  testTypes:
    - name: "cache_test"
      weight: 100
      questionStrategy: "historical"
      cacheBehavior: "prefer"      # Reuse same questions
      cacheSalt: "test-salt-123"   # Test cache isolation
```

### CI/CD Integration

Run tests as part of deployment pipeline:

```yaml
# .github/workflows/load-test.yml
name: Load Test
on:
  workflow_dispatch:
    inputs:
      config:
        description: 'Test configuration'
        required: true
        default: 'smoke'

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Deploy load test
        env:
          KUBECONFIG_DATA: ${{ secrets.KUBECONFIG }}
        run: |
          echo "$KUBECONFIG_DATA" | base64 -d > /tmp/kubeconfig
          export KUBECONFIG=/tmp/kubeconfig

          ./scripts/deploy-remote.sh --config ${{ github.event.inputs.config }}

      - name: Wait for completion
        run: |
          kubectl wait --for=condition=complete --timeout=1h \
            job/load-test-${{ github.event.inputs.config }}-* \
            -n load-testing
```

### Multi-Cluster Load Distribution

Distribute load across multiple clusters:

```bash
# Function to deploy to cluster
deploy_to_cluster() {
  local context=$1
  local users=$2

  kubectl --context=$context create configmap load-test-config-distributed \
    -n load-testing \
    --from-literal=users_per_org=$users \
    --dry-run=client -o yaml | kubectl --context=$context apply -f -

  ./scripts/deploy-remote.sh \
    --context $context \
    --config distributed
}

# Deploy 100 users across 3 clusters (33/33/34 split)
deploy_to_cluster us-east-1 33
deploy_to_cluster us-west-1 33
deploy_to_cluster eu-west-1 34
```

---

## Best Practices

### 1. Start Small, Scale Up

Always begin with a smoke test before large-scale tests:

```bash
# Start with 5 users
./scripts/deploy.sh --config smoke

# Then 50 users
./scripts/deploy.sh --config single-org-50-users

# Then full scale
./scripts/deploy.sh --config production-load
```

### 2. Set Cost Limits

Always set `maxCostUSD` to prevent unexpected charges:

```yaml
spec:
  limits:
    maxCostUSD: 100.0  # Hard stop at $100
```

### 3. Monitor During Tests

Watch metrics in real-time:

```bash
# Terminal 1: Logs
kubectl logs -n load-testing -f job/...

# Terminal 2: Metrics
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

# Terminal 3: Resource usage
watch kubectl top nodes
```

### 4. Clean Up After Tests

Remove completed jobs to save space:

```bash
# Delete all completed jobs
kubectl delete jobs -n load-testing --field-selector status.successful=1

# Delete all load test jobs
kubectl delete jobs -n load-testing -l app=load-test-worker
```

### 5. Version Your Configurations

Keep test configurations in version control:

```bash
# configs/environments/
configs/
  production/
    load-test.yaml
    stress-test.yaml
  staging/
    smoke-test.yaml
  development/
    quick-test.yaml
```

---

## Getting Help

### Documentation

- **README.md**: Project overview and quick start
- **docs/development.md**: Local development guide
- **docs/deployment.md**: Production deployment guide
- **docs/SPECIFICATION.md**: Complete technical specification

### Support

- **Issues**: https://github.com/otherjamesbrown/ai-aas-loadtest/issues
- **Platform Issues**: https://github.com/otherjamesbrown/ai-aas

### Examples

See `configs/examples/` for ready-to-use test configurations:
- `smoke-test.yaml` - Quick validation
- `single-org-50-users.yaml` - Production-like load

---

## Next Steps

1. ✅ **Run your first smoke test**: `./scripts/deploy.sh --config smoke`
2. 📊 **Set up monitoring**: Install Prometheus and Grafana
3. 🔧 **Customize configuration**: Create your own test scenarios
4. 📈 **Scale up gradually**: Start small, increase load incrementally
5. 🚀 **Automate**: Integrate with CI/CD pipelines

Happy Load Testing! 🎯
