#!/usr/bin/env bash
set -euo pipefail
umask 077
cd "$(dirname "$0")"

command -v go >/dev/null || { echo "go is required (https://go.dev/dl/)" >&2; exit 1; }

if [ ! -f .env ]; then
  cp .env.example .env
  chmod 600 .env
  echo "Created .env from .env.example — fill in your API keys."
fi

echo
echo "Local use:  go run . <article-url>"
echo

# Optionally push the .env values to GitHub as repo secrets for the Action.
if command -v gh >/dev/null && gh auth status >/dev/null 2>&1; then
  read -rp "Push .env values to GitHub repo secrets for the Action? [y/N] " ans
  if [ "$ans" = y ] || [ "$ans" = Y ]; then
    while IFS='=' read -r key value; do
      case "$key" in
        ''|\#*) continue ;;
      esac
      value=${value%\"}; value=${value#\"}
      [ -n "$value" ] || continue
      printf '%s' "$value" | gh secret set "$key"
      echo "  set secret $key"
    done < .env
    echo "GHA use:  ./scripts/make.sh <article-url>"
  fi
else
  echo "For the GitHub Action: install/auth 'gh', add repo secrets"
  echo "(ANTHROPIC_API_KEY, ELEVENLABS_API_KEY, ELEVENLABS_VOICE_ID, ...),"
  echo "then run ./scripts/make.sh <article-url>"
fi
