#!/usr/bin/env bash
set -euo pipefail
umask 077
cd "$(dirname "$0")"

if [ ! -f .env ]; then
  cp .env.example .env
  chmod 600 .env
  echo "Created .env from .env.example — fill in your API keys, then re-run this script."
  exit 0
fi

command -v gh >/dev/null || { echo "Install the GitHub CLI (https://cli.github.com), run 'gh auth login', then re-run this script." >&2; exit 1; }
gh auth status >/dev/null 2>&1 || { echo "Run 'gh auth login', then re-run this script." >&2; exit 1; }

echo "Pushing .env values to GitHub as repo secrets for the Action..."
while IFS='=' read -r key value; do
  case "$key" in
    ''|\#*) continue ;;
  esac
  value=${value%\"}; value=${value#\"}
  [ -n "$value" ] || continue
  printf '%s' "$value" | gh secret set "$key"
  echo "  set secret $key"
done < .env

echo
echo "Done. Trigger a run:  gh workflow run podcast.yml -f url=<article-url>"
