#!/usr/bin/env bash

for pair in "openssl:openssl" "jq:jq" "zip:zip"; do
  bin="${pair%%:*}"; formula="${pair##*:}"
  if command -v "$bin" >/dev/null; then
    ok "$bin already installed"
  else
    quiet brew install "$formula" || fail "failed to install $bin" "see error output below, or install manually: brew install $formula"
    ok "installed $bin"
  fi
done

command -v go >/dev/null || fail "go not found" "install it (e.g. brew install go), then re-run"
ok "go found"
