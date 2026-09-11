#!/bin/sh
# Starts as root so a mounted volume (root-owned by default on a fresh PVC)
# can be handed to the app user, then drops privileges for the server itself.
set -eu

DATA_DIR="${DUCKDB_DATA_DIR:-/data/duckdb}"

if [ "$(id -u)" = "0" ]; then
  mkdir -p "$DATA_DIR"
  chown appuser:appuser "$DATA_DIR"
  exec setpriv --reuid=appuser --regid=appuser --init-groups "$@"
fi

exec "$@"
