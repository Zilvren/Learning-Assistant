#!/usr/bin/env bash
# Deploy a checked-out GitHub Actions workspace without touching production data.
set -Eeuo pipefail

source_dir=${1:?Usage: deploy-from-runner.sh <checked-out-source-directory> <app-image>}
app_image=${2:?Usage: deploy-from-runner.sh <checked-out-source-directory> <app-image>}
deploy_root=/opt/Learning-Assistant
compose_dir="$deploy_root/deploy"

if [[ ! -f "$source_dir/deploy/docker-compose.yml" ]]; then
  echo "Deployment source is missing deploy/docker-compose.yml." >&2
  exit 1
fi

if [[ ! -f "$compose_dir/.env" ]]; then
  echo "Production configuration is missing: $compose_dir/.env" >&2
  exit 1
fi

if ! docker image inspect "$app_image" >/dev/null 2>&1; then
  echo "Production image is not loaded: $app_image" >&2
  exit 1
fi

# Keep server-only secrets and volumes intact. --delete makes source files match Git.
rsync -a --delete \
  --exclude='.git/' \
  --exclude='.tools/' \
  --exclude='dist/' \
  --exclude='frontend/node_modules/' \
  --exclude='deploy/.env' \
  --exclude='.env' \
  "$source_dir/" "$deploy_root/"

cd "$compose_dir"
TRACKER_APP_IMAGE="$app_image" docker compose up -d --no-build app

for attempt in $(seq 1 20); do
  if curl --fail --silent --show-error http://127.0.0.1/api/health; then
    echo
    docker compose ps
    exit 0
  fi
  sleep 3
done

echo "The application did not become healthy after deployment." >&2
docker compose ps >&2
docker compose logs --tail=100 app >&2
exit 1
