#!/bin/bash
set -e

# Deploy load test to remote Kubernetes cluster
# Usage: ./scripts/deploy-remote.sh [OPTIONS]

# Default values
NAMESPACE="load-testing"
CONFIG_NAME="smoke"
IMAGE="otherjamesbrown/ai-aas-loadtest:latest"
KUBECONFIG_PATH="${KUBECONFIG:-$HOME/.kube/config}"
CONTEXT=""
API_ROUTER_URL=""
USER_ORG_URL=""
PUSHGATEWAY_URL="http://prometheus-pushgateway.monitoring.svc.cluster.local:9091"
DRY_RUN="false"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --config)
      CONFIG_NAME="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --image)
      IMAGE="$2"
      shift 2
      ;;
    --context)
      CONTEXT="$2"
      shift 2
      ;;
    --kubeconfig)
      KUBECONFIG_PATH="$2"
      shift 2
      ;;
    --api-router-url)
      API_ROUTER_URL="$2"
      shift 2
      ;;
    --user-org-url)
      USER_ORG_URL="$2"
      shift 2
      ;;
    --pushgateway-url)
      PUSHGATEWAY_URL="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN="true"
      shift
      ;;
    --help)
      cat << EOF
Usage: $0 [OPTIONS]

Deploy AI-AAS Load Testing Harness to a remote Kubernetes cluster.

Options:
  --config NAME             Test configuration name (default: smoke)
  --namespace NAMESPACE     Kubernetes namespace (default: load-testing)
  --image IMAGE             Docker image to deploy (default: otherjamesbrown/ai-aas-loadtest:latest)
  --context CONTEXT         Kubernetes context to use
  --kubeconfig PATH         Path to kubeconfig file (default: \$KUBECONFIG or ~/.kube/config)
  --api-router-url URL      Override API Router URL in config
  --user-org-url URL        Override User/Org Service URL in config
  --pushgateway-url URL     Prometheus Pushgateway URL (default: http://prometheus-pushgateway.monitoring.svc.cluster.local:9091)
  --dry-run                 Show what would be deployed without actually deploying
  --help                    Show this help message

Examples:
  # Deploy smoke test to default context
  $0 --config smoke

  # Deploy to specific cluster context
  $0 --config smoke --context production-cluster

  # Deploy with custom URLs
  $0 --config smoke \\
    --api-router-url https://api.example.com \\
    --user-org-url https://admin.example.com

  # Deploy latest version after build
  $0 --config single-org-50-users --image otherjamesbrown/ai-aas-loadtest:v0.1.0

  # Dry run to preview changes
  $0 --config smoke --dry-run

Environment Variables:
  KUBECONFIG               Path to kubeconfig file
  KUBECTL_CONTEXT          Kubernetes context to use
EOF
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Use environment variable for context if not specified
if [[ -z "${CONTEXT}" ]] && [[ -n "${KUBECTL_CONTEXT}" ]]; then
  CONTEXT="${KUBECTL_CONTEXT}"
fi

echo -e "${GREEN}==> Deploying Load Test to Remote Kubernetes Cluster${NC}"
echo "    Namespace: ${NAMESPACE}"
echo "    Config: ${CONFIG_NAME}"
echo "    Image: ${IMAGE}"
echo "    Kubeconfig: ${KUBECONFIG_PATH}"
[[ -n "${CONTEXT}" ]] && echo "    Context: ${CONTEXT}"
[[ "${DRY_RUN}" == "true" ]] && echo -e "${YELLOW}    DRY RUN MODE${NC}"
echo ""

# Set kubeconfig
export KUBECONFIG="${KUBECONFIG_PATH}"

# Build kubectl command with optional context
KUBECTL="kubectl"
[[ -n "${CONTEXT}" ]] && KUBECTL="kubectl --context=${CONTEXT}"

# Verify cluster access
echo -e "${YELLOW}==> Verifying cluster access${NC}"
if ! ${KUBECTL} cluster-info > /dev/null 2>&1; then
  echo -e "${RED}ERROR: Cannot access Kubernetes cluster${NC}"
  echo "Check your kubeconfig and context settings"
  exit 1
fi

CURRENT_CONTEXT=$(${KUBECTL} config current-context)
echo -e "${GREEN}✓ Connected to cluster: ${CURRENT_CONTEXT}${NC}"
echo ""

# Dry run flag for kubectl
DRY_RUN_FLAG=""
[[ "${DRY_RUN}" == "true" ]] && DRY_RUN_FLAG="--dry-run=client"

# Create namespace if it doesn't exist
echo -e "${YELLOW}==> Creating namespace${NC}"
${KUBECTL} apply -f deploy/k8s/namespace.yaml ${DRY_RUN_FLAG}

# Apply RBAC
echo -e "${YELLOW}==> Applying RBAC${NC}"
${KUBECTL} apply -f deploy/k8s/rbac.yaml ${DRY_RUN_FLAG}

# Check if local config file exists
LOCAL_CONFIG="configs/examples/${CONFIG_NAME}-test.yaml"
K8S_CONFIGMAP="deploy/k8s/configmap-${CONFIG_NAME}-test.yaml"

if [[ -f "${K8S_CONFIGMAP}" ]]; then
  echo -e "${YELLOW}==> Applying ConfigMap from: ${K8S_CONFIGMAP}${NC}"

  # If URLs are provided, we need to modify the ConfigMap
  if [[ -n "${API_ROUTER_URL}" ]] || [[ -n "${USER_ORG_URL}" ]]; then
    echo -e "${YELLOW}==> Customizing configuration with provided URLs${NC}"

    # Create temporary modified ConfigMap
    TMP_CONFIGMAP="/tmp/load-test-configmap-${CONFIG_NAME}.yaml"
    cp "${K8S_CONFIGMAP}" "${TMP_CONFIGMAP}"

    if [[ -n "${API_ROUTER_URL}" ]]; then
      echo "    API Router URL: ${API_ROUTER_URL}"
      sed -i "s|apiRouterUrl:.*|apiRouterUrl: \"${API_ROUTER_URL}\"|g" "${TMP_CONFIGMAP}"
    fi

    if [[ -n "${USER_ORG_URL}" ]]; then
      echo "    User/Org URL: ${USER_ORG_URL}"
      sed -i "s|userOrgUrl:.*|userOrgUrl: \"${USER_ORG_URL}\"|g" "${TMP_CONFIGMAP}"
    fi

    ${KUBECTL} apply -f "${TMP_CONFIGMAP}" ${DRY_RUN_FLAG}
    rm -f "${TMP_CONFIGMAP}"
  else
    ${KUBECTL} apply -f "${K8S_CONFIGMAP}" ${DRY_RUN_FLAG}
  fi
elif [[ -f "${LOCAL_CONFIG}" ]]; then
  echo -e "${YELLOW}==> Creating ConfigMap from local file: ${LOCAL_CONFIG}${NC}"
  ${KUBECTL} create configmap "load-test-config-${CONFIG_NAME}" \
    -n "${NAMESPACE}" \
    --from-file=load-test-config.yaml="${LOCAL_CONFIG}" \
    --dry-run=client -o yaml | ${KUBECTL} apply -f - ${DRY_RUN_FLAG}
else
  echo -e "${RED}ERROR: No configuration found for: ${CONFIG_NAME}${NC}"
  echo "Looked for:"
  echo "  - ${K8S_CONFIGMAP}"
  echo "  - ${LOCAL_CONFIG}"
  exit 1
fi

# Generate unique job name with timestamp
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
JOB_NAME="load-test-${CONFIG_NAME}-${TIMESTAMP}"

echo ""
echo -e "${YELLOW}==> Creating Job: ${JOB_NAME}${NC}"

# Create temporary job manifest
TMP_JOB="/tmp/load-test-job-${TIMESTAMP}.yaml"
cat > "${TMP_JOB}" <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
  labels:
    app: load-test-worker
    test-type: ${CONFIG_NAME}
    deployed-by: deploy-remote-script
    deployed-at: "${TIMESTAMP}"
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 3600

  template:
    metadata:
      labels:
        app: load-test-worker
        test-type: ${CONFIG_NAME}
    spec:
      serviceAccountName: load-test-worker
      restartPolicy: Never

      containers:
      - name: load-test-worker
        image: ${IMAGE}
        imagePullPolicy: Always

        command:
        - /app/load-test-worker
        - --config=/config/load-test-config.yaml
        - --log-level=info

        env:
        - name: PUSHGATEWAY_URL
          value: "${PUSHGATEWAY_URL}"

        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true

        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi

      volumes:
      - name: config
        configMap:
          name: load-test-config-${CONFIG_NAME}
EOF

# Apply the job
${KUBECTL} apply -f "${TMP_JOB}" ${DRY_RUN_FLAG}

if [[ "${DRY_RUN}" == "true" ]]; then
  echo ""
  echo -e "${BLUE}==> DRY RUN: Job manifest${NC}"
  cat "${TMP_JOB}"
  rm -f "${TMP_JOB}"
  exit 0
fi

rm -f "${TMP_JOB}"

echo ""
echo -e "${GREEN}==> Deployment complete!${NC}"
echo ""
echo -e "${BLUE}Job Name:${NC} ${JOB_NAME}"
echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo ""
echo "# View logs (follow)"
echo "  ${KUBECTL} logs -n ${NAMESPACE} -f job/${JOB_NAME}"
echo ""
echo "# View job status"
echo "  ${KUBECTL} get job -n ${NAMESPACE} ${JOB_NAME}"
echo ""
echo "# View pods"
echo "  ${KUBECTL} get pods -n ${NAMESPACE} -l job-name=${JOB_NAME}"
echo ""
echo "# Describe job (for debugging)"
echo "  ${KUBECTL} describe job -n ${NAMESPACE} ${JOB_NAME}"
echo ""
echo "# Delete the job"
echo "  ${KUBECTL} delete job -n ${NAMESPACE} ${JOB_NAME}"
echo ""
echo "# View all load test jobs"
echo "  ${KUBECTL} get jobs -n ${NAMESPACE} -l app=load-test-worker"
echo ""

# Optionally follow logs
read -p "Follow logs now? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo -e "${YELLOW}==> Following logs (Ctrl+C to exit)${NC}"
  sleep 2  # Give job a moment to start
  ${KUBECTL} logs -n ${NAMESPACE} -f job/${JOB_NAME}
fi
