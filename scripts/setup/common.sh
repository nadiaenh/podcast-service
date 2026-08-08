#!/usr/bin/env bash

# temp file that captures each command's output for error reporting.
log=$(mktemp)
trap 'rm -f "$log"' EXIT
export AWS_PAGER=""

ok()   { echo "✓ $1"; }

fail() {
  echo "✗ $1"
  echo "  → $2"
  if [ -s "$log" ]; then
    echo "  ── error output ──"
    sed 's/^/  /' "$log"
  fi
  exit 1
}

quiet() { "$@" >"$log" 2>&1 && : >"$log"; }
