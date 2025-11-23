#!/bin/bash
set -e

# Build Docker image and push to registry
# Usage: ./scripts/build-and-push.sh [OPTIONS]

# Default values
IMAGE_NAME="${IMAGE_NAME:-otherjamesbrown/ai-aas-loadtest}"
VERSION="${VERSION:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-true}"
BUILD_TYPE="${BUILD_TYPE:-multiarch}"  # 'multiarch' or 'single'

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --image)
      IMAGE_NAME="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --platforms)
      PLATFORMS="$2"
      shift 2
      ;;
    --no-push)
      PUSH="false"
      shift
      ;;
    --single-arch)
      BUILD_TYPE="single"
      shift
      ;;
    --help)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --image NAME         Docker image name (default: otherjamesbrown/ai-aas-loadtest)"
      echo "  --version VERSION    Image version/tag (default: latest)"
      echo "  --platforms PLATFORMS Comma-separated platforms (default: linux/amd64,linux/arm64)"
      echo "  --no-push            Build only, don't push to registry"
      echo "  --single-arch        Build for current architecture only (faster)"
      echo "  --help               Show this help message"
      echo ""
      echo "Environment variables:"
      echo "  IMAGE_NAME           Same as --image"
      echo "  VERSION              Same as --version"
      echo "  DOCKER_USERNAME      Docker Hub username for login"
      echo "  DOCKER_PASSWORD      Docker Hub password for login"
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

echo -e "${GREEN}==> Building Load Test Worker Docker Image${NC}"
echo "    Image: ${FULL_IMAGE}"
echo "    Platforms: ${PLATFORMS}"
echo "    Build Type: ${BUILD_TYPE}"
echo "    Push: ${PUSH}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
  echo -e "${RED}ERROR: Docker is not running${NC}"
  exit 1
fi

# Login to Docker Hub if credentials provided
if [[ -n "${DOCKER_USERNAME}" ]] && [[ -n "${DOCKER_PASSWORD}" ]]; then
  echo -e "${YELLOW}==> Logging into Docker Hub${NC}"
  echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin
fi

# Run tests before building
echo -e "${YELLOW}==> Running tests${NC}"
go test ./... || {
  echo -e "${RED}ERROR: Tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Tests passed${NC}"
echo ""

# Build based on type
if [[ "${BUILD_TYPE}" == "multiarch" ]]; then
  # Multi-architecture build (requires buildx)
  echo -e "${YELLOW}==> Setting up Docker buildx${NC}"

  # Create buildx builder if it doesn't exist
  if ! docker buildx inspect loadtest-builder > /dev/null 2>&1; then
    docker buildx create --name loadtest-builder --use
  else
    docker buildx use loadtest-builder
  fi

  # Bootstrap the builder
  docker buildx inspect --bootstrap

  echo -e "${YELLOW}==> Building multi-architecture image${NC}"

  if [[ "${PUSH}" == "true" ]]; then
    docker buildx build \
      --platform "${PLATFORMS}" \
      --tag "${FULL_IMAGE}" \
      --push \
      .
  else
    docker buildx build \
      --platform "${PLATFORMS}" \
      --tag "${FULL_IMAGE}" \
      --load \
      .
  fi

else
  # Single architecture build (faster for local dev)
  echo -e "${YELLOW}==> Building single-architecture image${NC}"

  docker build \
    --tag "${FULL_IMAGE}" \
    .

  if [[ "${PUSH}" == "true" ]]; then
    echo -e "${YELLOW}==> Pushing image to registry${NC}"
    docker push "${FULL_IMAGE}"
  fi
fi

echo ""
echo -e "${GREEN}==> Build complete!${NC}"
echo ""
echo "Image: ${FULL_IMAGE}"
echo ""

if [[ "${PUSH}" == "true" ]]; then
  echo "To deploy to Kubernetes:"
  echo "  ./scripts/deploy.sh --config smoke --image ${FULL_IMAGE}"
else
  echo "Image built but not pushed (--no-push flag used)"
  echo "To push manually:"
  echo "  docker push ${FULL_IMAGE}"
fi
echo ""
