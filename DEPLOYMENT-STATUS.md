# Load Test Infrastructure - Deployment Status

**Cluster**: load-test-paris
**Region**: Paris, FR (fr-par-2)
**Created**: November 23, 2025
**Status**: ✅ **DEPLOYED AND READY**

---

## Cluster Information

- **Cluster ID**: lke536989
- **Kubernetes Version**: v1.34.0
- **Nodes**: 3x Linode 4GB (g6-standard-2)
  - 2 vCPUs, 4GB RAM per node
  - Total: 6 vCPUs, 12GB RAM
- **API Endpoint**: https://927c548f-8576-40ed-880e-c5a0fabe5a1d.fr-par-2-gw.linodelke.net:443

## Deployed Components

### ✅ Monitoring Stack (namespace: `monitoring`)

**Prometheus Operator Stack**
- ✅ Prometheus Server (StatefulSet)
- ✅ Grafana Dashboard
- ✅ Alertmanager
- ✅ Node Exporters (3x DaemonSet)
- ✅ Kube State Metrics
- ✅ Prometheus Operator

**Pushgateway**
- ✅ Prometheus Pushgateway (for load test metrics)

**Services**
- Prometheus: `prometheus-kube-prometheus-prometheus:9090`
- Grafana: `prometheus-grafana:80`
- Pushgateway: `prometheus-pushgateway:9091`

### ✅ Load Testing Infrastructure (namespace: `load-testing`)

- ✅ Namespace created
- ✅ ServiceAccount and RBAC configured
- ✅ Smoke test ConfigMap deployed

## Access Information

### Kubeconfig Locations

**Encrypted** (committed to git):
```
secrets/kubeconfigs/kubeconfig-load-test-paris.yaml
```

**Local** (for direct use):
```
~/.kube/kubeconfig-load-test-paris.yaml
```

### Using the Cluster

```bash
# Set kubeconfig
export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml

# Verify access
kubectl get nodes
kubectl get all -n monitoring
kubectl get all -n load-testing
```

### Accessing Monitoring UIs

#### Prometheus
```bash
export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
# Open http://localhost:9090
```

#### Grafana
```bash
export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Open http://localhost:3000
```

Get Grafana admin password:
```bash
kubectl get secret --namespace monitoring prometheus-grafana \
  -o jsonpath="{.data.admin-password}" | base64 -d ; echo
```

Default username: `admin`

#### Pushgateway
```bash
export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml
kubectl port-forward -n monitoring svc/prometheus-pushgateway 9091:9091
# Open http://localhost:9091
```

## Running Load Tests

### Deploy a Smoke Test

```bash
export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml
./scripts/deploy-remote.sh --config smoke
```

### Deploy Custom Tests

```bash
# Using local config
./scripts/deploy-remote.sh \
  --config single-org-50-users \
  --api-router-url https://api.your-platform.com \
  --user-org-url https://admin.your-platform.com
```

### Monitor Load Test

```bash
# List jobs
kubectl get jobs -n load-testing

# View logs
kubectl logs -n load-testing -f job/load-test-smoke-TIMESTAMP

# Check metrics in Prometheus
# Query: loadtest_run_requests_per_second{test_run_id="test-TIMESTAMP"}
```

## Node Status

```
NAME                            STATUS   ROLES    AGE   VERSION
lke536989-778640-226280350000   Ready    <none>   5m    v1.34.0
lke536989-778640-3fc9e1bd0000   Ready    <none>   5m    v1.34.0
lke536989-778640-64fe51a70000   Ready    <none>   5m    v1.34.0
```

## Monitoring Pods Status

All pods running in `monitoring` namespace:
- alertmanager-prometheus-kube-prometheus-alertmanager-0: 2/2 Running
- prometheus-grafana: 3/3 Running
- prometheus-kube-prometheus-operator: 1/1 Running
- prometheus-kube-state-metrics: 1/1 Running
- prometheus-prometheus-kube-prometheus-prometheus-0: 2/2 Running
- prometheus-prometheus-node-exporter (3x): 1/1 Running each
- prometheus-pushgateway: 1/1 Running

## Cost Estimate

- **3x Linode 4GB**: $24/month each = **$72/month total**
- **LKE Control Plane**: Free

**Note**: Remember to delete the cluster when done testing to avoid charges.

## Cleanup

When finished with testing:

```bash
# Delete via Linode CLI
linode-cli lke cluster-delete lke536989

# Or via Linode Cloud Manager
# https://cloud.linode.com/kubernetes/clusters
```

## Next Steps

1. **Build and push Docker image**:
   ```bash
   ./scripts/build-and-push.sh --version v0.1.0
   ```

2. **Deploy a load test**:
   ```bash
   export KUBECONFIG=~/.kube/kubeconfig-load-test-paris.yaml
   ./scripts/deploy-remote.sh --config smoke
   ```

3. **Monitor results** in Prometheus/Grafana

4. **Scale up** by deploying additional test configurations

## Troubleshooting

### Can't connect to cluster
```bash
# Verify kubeconfig is set
echo $KUBECONFIG

# Test connection
kubectl cluster-info
```

### Pods not starting
```bash
# Check pod status
kubectl get pods -n monitoring
kubectl describe pod -n monitoring <pod-name>

# Check events
kubectl get events -n monitoring --sort-by='.lastTimestamp'
```

### Need to update kubeconfig
```bash
# Download new kubeconfig from Linode and save it
./scripts/save-kubeconfig.sh ~/Downloads/new-kubeconfig.yaml
```

---

**Deployment completed**: November 23, 2025 17:49 UTC
**Deployment duration**: ~3 minutes
**Status**: All systems operational ✅
