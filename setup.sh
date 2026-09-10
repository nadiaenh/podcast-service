#!/usr/bin/env bash
cd "$(dirname "$0")"
source scripts/common.sh

step "1. Install tools"
source scripts/homebrew.sh

step "2. Provider credentials"
source scripts/env.sh

step "3. GitHub secrets"
source scripts/github.sh

step "Setup complete"
echo "  run: gh workflow run podcast.yml -f url=<article-url>"
