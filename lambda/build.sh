#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

GOOS=linux GOARCH=arm64 go build -o bootstrap .
zip -q -j bootstrap.zip bootstrap
echo "built lambda/bootstrap.zip"
