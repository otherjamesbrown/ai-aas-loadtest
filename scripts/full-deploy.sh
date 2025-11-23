#!/bin/bash
set -e

# Complete build and deploy workflow for remote clusters
# This script builds the Docker image and deploys to Kubernetes in one step
# Usage: ./scripts/full-deploy.sh [OPTIONS]

# Default values
IMAGE_NAME="otherjamesbrown/ai-aas-loadtest"
VERSION="latest"
CONFIG_NAME="smoke"
NAMESPACE="load-testing"
CONTEXT=""
BUILD="true"
SKIP_TESTS="false"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

show_help() {
  cat << EOF
Usage: $0 [OPTIONS]

Complete build and deploy workflow for AI-AAS Load Testing Harness.
This script:
  1. Runs tests
  2. Builds Docker image
  3. Pushes to registry
  4. Deploys to Kubernetes cluster

Options:
  --config NAME             Test configuration name (default: smoke)
  --version VERSION         Docker image version (default: latest)
  --image NAME              Docker image name (default: otherjamesbrown/ai-aas-loadtest)
  --namespace NAMESPACE     Kubernetes namespace (default: load-testing)
  --context CONTEXT         Kubernetes context to use
  --skip-build              Skip Docker build step (use existing image)
  --skip-tests              Skip running tests before build
  --api-router-url URL      Override API Router URL
  --user-org-url URL        Override User/Org Service URL
  --help                    Show this help message

Examples:
  # Build and deploy smoke test
  $0

  # Build specific version and deploy to production
  $0 --version v0.1.0 --context production --config single-org-50-users

  # Quick deploy without rebuilding (uses existing image)
  $0 --skip-build --config smoke

  # Deploy with custom platform URLs
  $0 --api-router-url https://api.prod.example.com \\
     --user-org-url https://admin.prod.example.com \\
     --config load-test

Environment Variables:
  DOCKER_USERNAME           Docker Hub username for login
  DOCKER_PASSWORD           Docker Hub password for login
  KUBECTL_CONTEXT           Kubernetes context to use
EOF
}

# Parse arguments
EXTRA_DEPLOY_ARGS=()
while [[ $# -gt 0 ]]; do
  case $1 in
    --config)
      CONFIG_NAME="$2"
      EXTRA_DEPLOY_ARGS+=("$1" "$2")
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --image)
      IMAGE_NAME="$2"
      shift 2
      ;;
    --namespace)
      NAMESPACE="$2"
      EXTRA_DEPLOY_ARGS+=("$1" "$2")
      shift 2
      ;;
    --context)
      CONTEXT="$2"
      EXTRA_DEPLOY_ARGS+=("$1" "$2")
      shift 2
      ;;
    --skip-build)
      BUILD="false"
      shift
      ;;
    --skip-tests)
      SKIP_TESTS="true"
      shift
      ;;
    --api-router-url|--user-org-url|--pushgateway-url)
      EXTRA_DEPLOY_ARGS+=("$1" "$2")
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

FULL_IMAGE="${IMAGE_NAME}:${VERSION}"

echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  AI-AAS Load Testing Harness - Full Deployment Workflow       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "  Image:       ${FULL_IMAGE}"
echo "  Test Config: ${CONFIG_NAME}"
echo "  Namespace:   ${NAMESPACE}"
[[ -n "${CONTEXT}" ]] && echo "  Context:     ${CONTEXT}"
echo "  Build:       ${BUILD}"
echo "  Skip Tests:  ${SKIP_TESTS}"
echo ""

# Step 1: Run Tests (unless skipped)
if [[ "${BUILD}" == "true" ]] && [[ "${SKIP_TESTS}" == "false" ]]; then
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 1: Running Tests${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  if ! go test ./...; then
    echo -e "${RED}✗ Tests failed${NC}"
    exit 1
  fi

  echo ""
  echo -e "${GREEN}✓ All tests passed${NC}"
  echo ""
fi

# Step 2: Build and Push Docker Image (unless skipped)
if [[ "${BUILD}" == "true" ]]; then
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 2: Building and Pushing Docker Image${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""

  ./scripts/build-and-push.sh \
    --image "${IMAGE_NAME}" \
    --version "${VERSION}" \
    --single-arch

  echo ""
  echo -e "${GREEN}✓ Image built and pushed${NC}"
  echo ""
else
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo -e "${YELLOW}STEP 2: Skipping Build (using existing image)${NC}"
  echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
  echo ""
fi

# Step 3: Deploy to Kubernetes
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}STEP 3: Deploying to Kubernetes${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo ""

./scripts/deploy-remote.sh \
  --image "${FULL_IMAGE}" \
  "${EXTRA_DEPLOY_ARGS[@]}"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Deployment Complete!                                          ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
