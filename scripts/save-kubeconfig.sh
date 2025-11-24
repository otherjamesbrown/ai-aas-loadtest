#!/bin/bash
set -e

# Helper script to save kubeconfig to both secrets/ and local location
# Usage: ./scripts/save-kubeconfig.sh <path-to-downloaded-kubeconfig>

if [ $# -eq 0 ]; then
    echo "Usage: $0 <path-to-kubeconfig>"
    echo ""
    echo "Example:"
    echo "  $0 ~/Downloads/kubeconfig-load-test-paris.yaml"
    echo ""
    echo "This script will:"
    echo "  1. Copy kubeconfig to secrets/kubeconfigs/ (encrypted)"
    echo "  2. Copy kubeconfig to ~/.kube/ (local use)"
    echo "  3. Verify the kubeconfig works"
    exit 1
fi

KUBECONFIG_SOURCE="$1"
CLUSTER_NAME="${2:-load-test-paris}"

# Validate source file exists
if [ ! -f "$KUBECONFIG_SOURCE" ]; then
    echo "Error: File not found: $KUBECONFIG_SOURCE"
    exit 1
fi

echo "==> Saving kubeconfig for cluster: $CLUSTER_NAME"
echo ""

# Create directories if needed
mkdir -p secrets/kubeconfigs
mkdir -p ~/.kube

# Define destination paths
ENCRYPTED_PATH="secrets/kubeconfigs/kubeconfig-${CLUSTER_NAME}.yaml"
LOCAL_PATH="$HOME/.kube/kubeconfig-${CLUSTER_NAME}.yaml"

# Copy to encrypted secrets directory
echo "1. Copying to encrypted location: $ENCRYPTED_PATH"
cp "$KUBECONFIG_SOURCE" "$ENCRYPTED_PATH"
chmod 600 "$ENCRYPTED_PATH"

# Copy to local .kube directory
echo "2. Copying to local kubectl directory: $LOCAL_PATH"
cp "$KUBECONFIG_SOURCE" "$LOCAL_PATH"
chmod 600 "$LOCAL_PATH"

# Verify the kubeconfig works
echo ""
echo "3. Verifying kubeconfig..."
export KUBECONFIG="$LOCAL_PATH"

if kubectl cluster-info &> /dev/null; then
    echo "✓ Kubeconfig is valid and cluster is reachable"
    echo ""
    echo "Cluster Info:"
    kubectl cluster-info
    echo ""
    echo "Nodes:"
    kubectl get nodes
else
    echo "⚠ Warning: Could not connect to cluster (this is normal if cluster is still provisioning)"
fi

echo ""
echo "==> Success!"
echo ""
echo "Kubeconfig saved to:"
echo "  Encrypted: $ENCRYPTED_PATH (will be encrypted on git commit)"
echo "  Local:     $LOCAL_PATH"
echo ""
echo "To use this cluster:"
echo "  export KUBECONFIG=$LOCAL_PATH"
echo "  kubectl get nodes"
echo ""
echo "Or specify in commands:"
echo "  kubectl --kubeconfig=$LOCAL_PATH get nodes"
echo ""
echo "Next steps:"
echo "  1. Commit the encrypted kubeconfig:"
echo "     git add secrets/"
echo "     git commit -m 'Add kubeconfig for $CLUSTER_NAME'"
echo ""
echo "  2. Deploy monitoring and load testing:"
echo "     export KUBECONFIG=$LOCAL_PATH"
echo "     ./scripts/deploy-monitoring.sh"
echo ""
