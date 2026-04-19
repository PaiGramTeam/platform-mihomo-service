#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
compose_file="$repo_root/docker-compose.integration.yml"

usage() {
  printf '%s\n' 'Usage: ./scripts/integration.sh [doctor|test|deps-up|deps-down]'
}

run_go() {
  GOWORK=off go "$@"
}

assert_docker_compose() {
  if ! command -v docker >/dev/null 2>&1; then
    printf '%s\n' 'Docker is required but was not found in PATH.' >&2
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    printf '%s\n' 'Docker Compose is required and docker compose is not available.' >&2
    exit 1
  fi
}

command_name="${1:-test}"

cd "$repo_root"

case "$command_name" in
  doctor)
    run_go run ./cmd/integration-doctor
    ;;
  test)
    run_go run ./cmd/integration-doctor
    run_go test -tags=integration ./integration/...
    ;;
  deps-up)
    assert_docker_compose
    docker compose -f "$compose_file" up -d
    ;;
  deps-down)
    assert_docker_compose
    docker compose -f "$compose_file" down --remove-orphans
    ;;
  *)
    usage
    exit 1
    ;;
esac
