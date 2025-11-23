#!/bin/bash
set -e

# Automated LKE (Linode Kubernetes Engine) Cluster Creation and Configuration
# Usage: ./scripts/create-lke-cluster.sh [OPTIONS]

# Default values
CLUSTER_NAME="load-test-cluster"
REGION="us-east"
NODE_TYPE="g6-standard-2"
NODE_COUNT=3
K8S_VERSION="1.28"
LINODE_TOKEN="${LINODE_TOKEN:-}"
INSTALL_MONITORING="true"
INSTALL_LOADTEST="true"
KUBECONFIG_DIR="$HOME/kubeconfigs"
SKIP_CLUSTER_CREATE="false"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

show_help() {
  cat << EOF
${GREEN}LKE Cluster Automation Script${NC}

Creates a Linode Kubernetes Engine (LKE) cluster and configures it for load testing.

${YELLOW}Usage:${NC}
  $0 [OPTIONS]

${YELLOW}Required:${NC}
  --token TOKEN             Linode API token (or set LINODE_TOKEN env var)

${YELLOW}Cluster Options:${NC}
  --name NAME               Cluster name (default: load-test-cluster)
  --region REGION           Linode region (default: us-east)
  --node-type TYPE          Linode instance type (default: g6-standard-2)
  --node-count COUNT        Number of worker nodes (default: 3)
  --k8s-version VERSION     Kubernetes version (default: 1.28)

${YELLOW}Installation Options:${NC}
  --skip-monitoring         Don't install Prometheus/Grafana
  --skip-loadtest           Don't install load testing harness
  --skip-cluster-create     Skip cluster creation (use existing)

${YELLOW}Other Options:${NC}
  --kubeconfig-dir DIR      Directory for kubeconfig (default: ~/kubeconfigs)
  --help                    Show this help message

${YELLOW}Available Regions:${NC}
  us-east, us-central, us-west, us-southeast
  eu-west, eu-central, ap-south, ap-northeast, ap-southeast

${YELLOW}Common Node Types:${NC}
  g6-standard-1   (1 CPU, 2GB RAM)    - \$12/month
  g6-standard-2   (2 CPU, 4GB RAM)    - \$24/month (recommended)
  g6-standard-4   (4 CPU, 8GB RAM)    - \$48/month
  g6-dedicated-2  (2 CPU, 4GB RAM)    - \$36/month (dedicated)
  g6-dedicated-4  (4 CPU, 8GB RAM)    - \$72/month (dedicated)

${YELLOW}Examples:${NC}
  # Create cluster with defaults
  $0 --token \$LINODE_TOKEN

  # Create production cluster
  $0 --token \$LINODE_TOKEN \\
     --name production-loadtest \\
     --region us-west \\
     --node-type g6-standard-4 \\
     --node-count 5

  # Configure existing cluster only
  $0 --token \$LINODE_TOKEN \\
     --skip-cluster-create \\
     --name existing-cluster

${YELLOW}Environment Variables:${NC}
  LINODE_TOKEN              Linode API token
  DOCKER_USERNAME           Docker Hub username
  DOCKER_PASSWORD           Docker Hub password
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --token)
      LINODE_TOKEN="$2"
      shift 2
      ;;
    --name)
      CLUSTER_NAME="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    --node-type)
      NODE_TYPE="$2"
      shift 2
      ;;
    --node-count)
      NODE_COUNT="$2"
      shift 2
      ;;
    --k8s-version)
      K8S_VERSION="$2"
      shift 2
      ;;
    --skip-monitoring)
      INSTALL_MONITORING="false"
      shift
      ;;
    --skip-loadtest)
      INSTALL_LOADTEST="false"
      shift
      ;;
    --skip-cluster-create)
      SKIP_CLUSTER_CREATE="true"
      shift
      ;;
    --kubeconfig-dir)
      KUBECONFIG_DIR="$2"
      shift 2
      ;;
    --help)
      show_help
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [[ -z "${LINODE_TOKEN}" ]]; then
  echo -e "${RED}ERROR: Linode API token is required${NC}"
  echo "Provide via --token flag or LINODE_TOKEN environment variable"
  exit 1
fi

echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  LKE Cluster Automation - Load Testing Infrastructure         ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "  Cluster Name:    ${CLUSTER_NAME}"
echo "  Region:          ${REGION}"
echo "  Node Type:       ${NODE_TYPE}"
echo "  Node Count:      ${NODE_COUNT}"
echo "  K8s Version:     ${K8S_VERSION}"
echo "  Install Monitor: ${INSTALL_MONITORING}"
echo "  Install LoadTest:${INSTALL_LOADTEST}"
echo "  Kubeconfig Dir:  ${KUBECONFIG_DIR}"
echo ""

# Create kubeconfig directory
mkdir -p "${KUBECONFIG_DIR}"

# Check if linode-cli is installed
if ! command -v linode-cli &> /dev/null; then
  echo -e "${YELLOW}==> Installing linode-cli${NC}"
  pip3 install linode-cli --quiet || {
    echo -e "${RED}ERROR: Failed to install linode-cli${NC}"
    echo "Install manually: pip3 install linode-cli"
    exit 1
  }
fi

# Configure linode-cli with token
echo -e "${YELLOW}==> Configuring linode-cli${NC}"
export LINODE_CLI_TOKEN="${LINODE_TOKEN}"
linode-cli configure --token <<< "${LINODE_TOKEN}" > /dev/null 2>&1 || true

# Step 1: Create LKE Cluster (unless skipped)
if [[ "${SKIP_CLUSTER_CREATE}" == "false" ]]; then
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 1: Creating LKE Cluster${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  # Check if cluster already exists
  EXISTING_CLUSTER=$(linode-cli lke clusters-list --json 2>/dev/null | \
    jq -r ".[] | select(.label == \"${CLUSTER_NAME}\") | .id" || echo "")

  if [[ -n "${EXISTING_CLUSTER}" ]]; then
    echo -e "${YELLOW}Cluster '${CLUSTER_NAME}' already exists (ID: ${EXISTING_CLUSTER})${NC}"
    CLUSTER_ID="${EXISTING_CLUSTER}"
  else
    echo "Creating new LKE cluster..."
    echo "  This may take 5-10 minutes..."

    # Create cluster
    CLUSTER_JSON=$(linode-cli lke cluster-create \
      --label "${CLUSTER_NAME}" \
      --region "${REGION}" \
      --k8s_version "${K8S_VERSION}" \
      --node_pools.type "${NODE_TYPE}" \
      --node_pools.count "${NODE_COUNT}" \
      --json)

    CLUSTER_ID=$(echo "${CLUSTER_JSON}" | jq -r '.[0].id')

    echo -e "${GREEN}✓ Cluster created (ID: ${CLUSTER_ID})${NC}"
  fi

  # Wait for cluster to be ready
  echo ""
  echo "Waiting for cluster to be ready..."
  for i in {1..30}; do
    STATUS=$(linode-cli lke clusters-list --json | \
      jq -r ".[] | select(.id == ${CLUSTER_ID}) | .status" || echo "unknown")

    if [[ "${STATUS}" == "ready" ]]; then
      echo -e "${GREEN}✓ Cluster is ready${NC}"
      break
    fi

    echo "  Status: ${STATUS} (attempt $i/30)"
    sleep 10
  done

  if [[ "${STATUS}" != "ready" ]]; then
    echo -e "${RED}ERROR: Cluster did not become ready${NC}"
    exit 1
  fi

else
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 1: Skipping Cluster Creation${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  # Get cluster ID from name
  CLUSTER_ID=$(linode-cli lke clusters-list --json 2>/dev/null | \
    jq -r ".[] | select(.label == \"${CLUSTER_NAME}\") | .id" || echo "")

  if [[ -z "${CLUSTER_ID}" ]]; then
    echo -e "${RED}ERROR: Cluster '${CLUSTER_NAME}' not found${NC}"
    exit 1
  fi

  echo "Using existing cluster ID: ${CLUSTER_ID}"
fi

# Step 2: Download Kubeconfig
echo ""
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}STEP 2: Downloading Kubeconfig${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo ""

KUBECONFIG_PATH="${KUBECONFIG_DIR}/kubeconfig-${CLUSTER_NAME}.yaml"

linode-cli lke kubeconfig-view "${CLUSTER_ID}" --json | \
  jq -r '.[0].kubeconfig' | base64 -d > "${KUBECONFIG_PATH}"

export KUBECONFIG="${KUBECONFIG_PATH}"

echo -e "${GREEN}✓ Kubeconfig saved to: ${KUBECONFIG_PATH}${NC}"

# Verify cluster access
kubectl cluster-info > /dev/null 2>&1 || {
  echo -e "${RED}ERROR: Cannot access cluster${NC}"
  exit 1
}

echo -e "${GREEN}✓ Cluster access verified${NC}"

# Wait for nodes to be ready
echo ""
echo "Waiting for all nodes to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

NODES=$(kubectl get nodes --no-headers | wc -l)
echo -e "${GREEN}✓ All ${NODES} nodes are ready${NC}"

# Step 3: Install Monitoring Stack (optional)
if [[ "${INSTALL_MONITORING}" == "true" ]]; then
  echo ""
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 3: Installing Monitoring Stack (Prometheus + Pushgateway)${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  # Create monitoring namespace
  kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

  # Add Prometheus Helm repo
  echo "Adding Prometheus Helm repository..."
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
  helm repo update

  # Install Prometheus + Pushgateway
  echo "Installing Prometheus with Pushgateway..."
  helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
    --namespace monitoring \
    --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
    --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
    --wait --timeout=10m

  # Install Pushgateway separately for better control
  helm upgrade --install prometheus-pushgateway prometheus-community/prometheus-pushgateway \
    --namespace monitoring \
    --wait --timeout=5m

  echo -e "${GREEN}✓ Monitoring stack installed${NC}"

  # Wait for pods
  echo "Waiting for monitoring pods to be ready..."
  kubectl wait --for=condition=Ready pods --all -n monitoring --timeout=300s

  echo -e "${GREEN}✓ All monitoring pods are running${NC}"
fi

# Step 4: Install Load Testing Harness (optional)
if [[ "${INSTALL_LOADTEST}" == "true" ]]; then
  echo ""
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 4: Installing Load Testing Harness${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  # Apply namespace and RBAC
  kubectl apply -f deploy/k8s/namespace.yaml
  kubectl apply -f deploy/k8s/rbac.yaml

  # Apply smoke test ConfigMap
  kubectl apply -f deploy/k8s/configmap-smoke-test.yaml

  echo -e "${GREEN}✓ Load testing infrastructure installed${NC}"
fi

# Step 5: Display cluster information
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Cluster Setup Complete!                                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Cluster Information:${NC}"
echo "  Cluster Name: ${CLUSTER_NAME}"
echo "  Cluster ID:   ${CLUSTER_ID}"
echo "  Region:       ${REGION}"
echo "  Nodes:        ${NODES}"
echo "  Kubeconfig:   ${KUBECONFIG_PATH}"
echo ""

echo -e "${BLUE}Access Instructions:${NC}"
echo ""
echo "# Set kubeconfig for this cluster"
echo "  export KUBECONFIG=${KUBECONFIG_PATH}"
echo ""
echo "# View cluster info"
echo "  kubectl cluster-info"
echo ""
echo "# View nodes"
echo "  kubectl get nodes"
echo ""

if [[ "${INSTALL_MONITORING}" == "true" ]]; then
  echo -e "${BLUE}Monitoring:${NC}"
  echo ""
  echo "# Access Prometheus UI (port-forward)"
  echo "  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090"
  echo "  # Then open: http://localhost:9090"
  echo ""
  echo "# Access Grafana UI (port-forward)"
  echo "  kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80"
  echo "  # Then open: http://localhost:3000"
  echo "  # Default credentials: admin / prom-operator"
  echo ""
fi

if [[ "${INSTALL_LOADTEST}" == "true" ]]; then
  echo -e "${BLUE}Load Testing:${NC}"
  echo ""
  echo "# Deploy a smoke test"
  echo "  ./scripts/deploy.sh --config smoke"
  echo ""
  echo "# Or use the remote deploy script"
  echo "  ./scripts/deploy-remote.sh \\"
  echo "    --kubeconfig ${KUBECONFIG_PATH} \\"
  echo "    --config smoke"
  echo ""
fi

echo -e "${BLUE}Cleanup:${NC}"
echo ""
echo "# Delete the cluster when done"
echo "  linode-cli lke cluster-delete ${CLUSTER_ID}"
echo ""

# Save cluster info to file
INFO_FILE="${KUBECONFIG_DIR}/${CLUSTER_NAME}-info.txt"
cat > "${INFO_FILE}" <<EOF
Cluster Name: ${CLUSTER_NAME}
Cluster ID:   ${CLUSTER_ID}
Region:       ${REGION}
Node Type:    ${NODE_TYPE}
Node Count:   ${NODE_COUNT}
Created:      $(date)
Kubeconfig:   ${KUBECONFIG_PATH}

Access:
  export KUBECONFIG=${KUBECONFIG_PATH}
  kubectl cluster-info

Delete:
  linode-cli lke cluster-delete ${CLUSTER_ID}
EOF

echo -e "${GREEN}✓ Cluster information saved to: ${INFO_FILE}${NC}"
echo ""
