#!/bin/sh
set -eu

compose_file=docker-compose.external-integration.yml
cleanup() {
  docker compose -f "$compose_file" down --volumes --remove-orphans
}
trap cleanup EXIT INT TERM

docker compose --progress quiet -f "$compose_file" up --build --wait
APEXVOID_EXTERNAL_TEST_URL=http://localhost:18386 \
APEXVOID_EXTERNAL_BACKEND_URL=http://localhost:16868 \
  go run ./tests/docker_external_module -phase setup
docker compose -f "$compose_file" restart backend
docker compose --progress quiet -f "$compose_file" up --wait
APEXVOID_EXTERNAL_TEST_URL=http://localhost:18386 \
APEXVOID_EXTERNAL_BACKEND_URL=http://localhost:16868 \
  go run ./tests/docker_external_module -phase verify
docker compose -f "$compose_file" stop fixture
APEXVOID_EXTERNAL_TEST_URL=http://localhost:18386 \
APEXVOID_EXTERNAL_BACKEND_URL=http://localhost:16868 \
  go run ./tests/docker_external_module -phase outage
docker compose -f "$compose_file" start fixture
docker compose --progress quiet -f "$compose_file" up --wait fixture
APEXVOID_EXTERNAL_TEST_URL=http://localhost:18386 \
APEXVOID_EXTERNAL_BACKEND_URL=http://localhost:16868 \
  go run ./tests/docker_external_module -phase finalize
