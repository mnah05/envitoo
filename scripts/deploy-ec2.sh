#!/usr/bin/env bash
set -euo pipefail

image_tag="${IMAGE_TAG:?IMAGE_TAG is required}"
cd /home/ec2-user/envitoo

set -a
# This file contains only the registry login, never app runtime credentials.
. ./deploy-registry.env
set +a
printf '%s' "$DOCKERHUB_TOKEN" | docker login --username "$DOCKERHUB_USERNAME" --password-stdin
rm -f ./deploy-registry.env

IMAGE_TAG="$image_tag" docker compose -f docker-compose.prod.yml pull
IMAGE_TAG="$image_tag" docker compose -f docker-compose.prod.yml up -d --remove-orphans
docker image prune -f
