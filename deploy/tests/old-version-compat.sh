#!/usr/bin/env bash
set -euo pipefail
root=$(git rev-parse --show-toplevel)
baseline=2d5892593c94f9168ce7d60b932889c306bc8a87
scratch=$(mktemp -d)
container_id=''
cleanup() {
  if [[ -n "$container_id" ]]; then docker rm -f "$container_id" >/dev/null; fi
  rm -rf "$scratch"
}
trap cleanup EXIT
mkdir -p "$scratch/old" "$scratch/new"
git archive "$baseline" backend | tar -x -C "$scratch/old"
git archive HEAD backend | tar -x -C "$scratch/new"
for version in old new; do
  mkdir -p "$scratch/$version/backend/cmd/compatcheck"
  cp "$root/deploy/compatcheck/main.go" "$scratch/$version/backend/cmd/compatcheck/main.go"
  (cd "$scratch/$version/backend" && go build -o "$scratch/$version-check" ./cmd/compatcheck)
done
container_id=$(docker run -d --rm -e POSTGRES_PASSWORD=compat-local-only -p 127.0.0.1::5432 postgres:18-alpine)
ready=false
for attempt in $(seq 1 60); do
  # The entrypoint's temporary initialization server accepts only Unix sockets.
  if docker exec "$container_id" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  docker logs "$container_id" >&2
  exit 1
fi
port=$(docker port "$container_id" 5432/tcp | sed 's/.*://')
export COMPAT_DATABASE_URL="postgres://postgres:compat-local-only@127.0.0.1:$port/postgres?sslmode=disable"
"$scratch/old-check" baseline
"$scratch/new-check" upgrade
"$scratch/old-check" rollback
"$scratch/new-check" reupgrade
