#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://localhost:8080/api/echo/hello}"
COUNT="${2:-20}"
DELAY="${3:-0.1}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

for ((i=1; i<=COUNT; i++)); do
  echo "==> Request $i"
  curl -sS -w "\nstatus=%{http_code}\n" "$URL"
  if (( i < COUNT )); then
    sleep "$DELAY"
  fi
done
