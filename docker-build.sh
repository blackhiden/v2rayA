#!/usr/bin/env bash
#
# build.sh - Linux/macOS Build Script
#
# This script automates the process of building and running the Docker container
# with version information dynamically injected at build time.

set -euo pipefail

if [[ "${1:-}" != "" ]]; then
  echo "Error: unknown option '${1}'."
  echo "Usage: ./docker-build.sh"
  exit 1
fi

# --- Step 1: Choose Environment ---
echo "Please select an option:"
echo "1) Run using Pre-built Image (Recommended)"
echo "2) Build from Source and Run (For Developers)"
echo "3) Run (For Developers)"
read -r -p "Enter choice [1-2]: " choice

# --- Step 2: Execute based on choice ---
COMMIT_VERSION="$(git rev-parse --short HEAD)"
IMAGE_NAME="v2raya:${COMMIT_VERSION}"
case "$choice" in
  1)
    echo "--- Running with Pre-built Image ---"
    docker compose up -d --remove-orphans --no-build
    echo "Services are starting from remote image."
    echo "Run 'docker compose logs -f' to see the logs."
    ;;
  2)
    echo "--- Building from Source and Running ---"

    # Get Version Information
    V2RAYA_VERSION="$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")"
    V2RAYA_COMMIT="$(git rev-parse --short HEAD)"
    XCORE_VERSION="$(git -C core/xray describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")"
    XCORE_COMMIT="$(git -C core/xray rev-parse --short HEAD)"
    BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    echo "Building with the following info:"
    echo "  Version: ${V2RAYA_VERSION}"
    echo "  Commit: ${V2RAYA_COMMIT}"
    echo "  Build Date: ${BUILD_DATE}"
    echo "  Image Name: ${IMAGE_NAME}"
    echo "----------------------------------------"

    # Build and start the services with a local-only image tag

    echo "Building the Docker image..."
    docker image build \
      -t ${IMAGE_NAME} \
      --no-cache \
      --build-arg V2RAYA_VERSION="${V2RAYA_VERSION}" \
      --build-arg V2RAYA_VERSION="${V2RAYA_VERSION}" \
      --build-arg XCORE_COMMIT="${XCORE_COMMIT}" \
      --build-arg XCORE_COMMIT="${XCORE_COMMIT}" \
      --build-arg BUILD_DATE="${BUILD_DATE}" .

    echo "Starting the services..."
    echo "Build complete"
    echo "Run 'docker compose logs -f' to see the logs."
    ;;
  3) docker run --rm \
    --name v2rayatest \
    -v $(pwd)/data:/etc/v2raya \
    --network host \
    $IMAGE_NAME
    ;;
  *)
    echo "Invalid choice. Please enter 1 or 2."
    exit 1
    ;;
esac
