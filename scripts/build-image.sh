#!/usr/bin/env bash

set -euo pipefail

PREFER_PODMAN="${PREFER_PODMAN:-0}"
PUSH="${PUSH:-0}"
TAG="${TAG:-0.0.1}"
REGISTRY_ORG="${REGISTRY_ORG:-openshift-observability-ui}"

if [[ -x "$(command -v podman)" && $PREFER_PODMAN == 1 ]]; then
    OCI_BIN="podman"
else
    OCI_BIN="docker"
fi

BASE_IMAGE="quay.io/${REGISTRY_ORG}/console-dashboards-plugin"
IMAGE=${BASE_IMAGE}:${TAG}

echo "Building image '${IMAGE}' with ${OCI_BIN}"
$OCI_BIN build -t $IMAGE -f Dockerfile.dev .

if [[ $PUSH == 1 ]]; then
    $OCI_BIN push $IMAGE
fi
