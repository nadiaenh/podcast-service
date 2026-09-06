#!/usr/bin/env bash
# Trigger the podcast GitHub Action for an article URL and wait for it.
# Prints the download link for the resulting MP3 (a GitHub Release asset).
set -euo pipefail
cd "$(dirname "$0")/.."

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: ./scripts/make.sh <article-url> [elevenlabs|voxtral]" >&2
  exit 1
fi
url=$1
provider=${2:-elevenlabs}
tag="ep-$(printf '%s' "$url" | shasum -a 256 | cut -c1-12)"

gh workflow run podcast.yml -f url="$url" -f provider="$provider"

# Wait for the queued run to appear, then follow it.
run_id=""
for _ in $(seq 1 15); do
  sleep 3
  run_id=$(gh run list --workflow=podcast.yml --event=workflow_dispatch --limit=1 --json databaseId --jq '.[0].databaseId' 2>/dev/null || true)
  [ -n "$run_id" ] && break
done
[ -n "$run_id" ] || { echo "could not find the triggered run" >&2; exit 1; }

gh run watch "$run_id" --exit-status
gh release view "$tag" --json assets --jq '.assets[0].url'
