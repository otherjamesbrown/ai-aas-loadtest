#!/bin/bash
set -e

# Deploy load test to Kubernetes
# Usage: ./scripts/deploy.sh [--config CONFIG_NAME]

NAMESPACE="load-testing"
CONFIG_NAME="smoke"
IMAGE="otherjamesbrown/ai-aas-loadtest:latest"

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
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--config CONFIG_NAME] [--namespace NAMESPACE] [--image IMAGE]"
      exit 1
      ;;
  esac
done

echo "==> Deploying load test to Kubernetes"
echo "    Namespace: $NAMESPACE"
echo "    Config: $CONFIG_NAME"
echo "    Image: $IMAGE"

# Create namespace if it doesn't exist
echo "==> Creating namespace..."
kubectl apply -f deploy/k8s/namespace.yaml

# Apply RBAC
echo "==> Applying RBAC..."
kubectl apply -f deploy/k8s/rbac.yaml

# Apply ConfigMap
echo "==> Applying ConfigMap..."
kubectl apply -f deploy/k8s/configmap-${CONFIG_NAME}-test.yaml

# Generate unique job name with timestamp
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
JOB_NAME="load-test-${CONFIG_NAME}-${TIMESTAMP}"

echo "==> Creating Job: $JOB_NAME"

# Create temporary job manifest
cat > /tmp/load-test-job.yaml <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
  labels:
    app: load-test-worker
    test-type: ${CONFIG_NAME}
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
          value: "http://prometheus-pushgateway.monitoring.svc.cluster.local:9091"

        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true

        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi

      volumes:
      - name: config
        configMap:
          name: load-test-config-${CONFIG_NAME}
EOF

# Apply the job
kubectl apply -f /tmp/load-test-job.yaml

echo ""
echo "==> Job created successfully!"
echo ""
echo "To view logs:"
echo "  kubectl logs -n ${NAMESPACE} -f job/${JOB_NAME}"
echo ""
echo "To view job status:"
echo "  kubectl get job -n ${NAMESPACE} ${JOB_NAME}"
echo ""
echo "To view pods:"
echo "  kubectl get pods -n ${NAMESPACE} -l job-name=${JOB_NAME}"
echo ""
echo "To delete the job:"
echo "  kubectl delete job -n ${NAMESPACE} ${JOB_NAME}"
echo ""
