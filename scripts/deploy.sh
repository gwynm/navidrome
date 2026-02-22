#!/usr/bin/env bash
set -euo pipefail

DOCKER_TAG="${DOCKER_TAG:-gwynm/navidrome:develop}"
IMAGE_PLATFORMS="${IMAGE_PLATFORMS:-linux/amd64}"
NUC_HOST="${NUC_HOST:-nuc@nuc.fritz.box}"
NUC_COMPOSE_DIR="${NUC_COMPOSE_DIR:-/home/nuc/Documents/docker}"

echo "==> Building Docker image ${DOCKER_TAG} for ${IMAGE_PLATFORMS}..."
DOCKER_TAG="$DOCKER_TAG" IMAGE_PLATFORMS="$IMAGE_PLATFORMS" make docker-image

echo "==> Transferring image to ${NUC_HOST}..."
docker save "$DOCKER_TAG" | ssh "$NUC_HOST" "sudo docker load"

echo "==> Restarting navidrome on ${NUC_HOST}..."
ssh "$NUC_HOST" "cd ${NUC_COMPOSE_DIR} && sudo docker compose up -d --force-recreate --pull never navidrome"

echo "==> Verifying..."
LOCAL_SHA=$(docker inspect --format='{{.Id}}' "$DOCKER_TAG")
REMOTE_SHA=$(ssh "$NUC_HOST" "sudo docker ps --filter 'name=navidrome' -q | xargs -I{} sudo docker inspect --format='{{.Image}}' {}")

if [ "$LOCAL_SHA" = "$REMOTE_SHA" ]; then
    echo "Deploy successful. Running image: ${LOCAL_SHA}"
else
    echo "WARNING: SHA mismatch!"
    echo "  Local:  ${LOCAL_SHA}"
    echo "  Remote: ${REMOTE_SHA}"
    exit 1
fi
