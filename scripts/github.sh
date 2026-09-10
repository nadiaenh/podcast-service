# Push the sensitive values to the repo as Actions secrets. TTS_PROVIDER stays a
# workflow input, not a secret.
have gh || fail "GitHub CLI not installed" "run: brew install gh"
quiet gh auth status || fail "GitHub CLI is not authenticated" "run: gh auth login"
ok "GitHub CLI authenticated"

while IFS='=' read -r key _; do
  case "$key" in ''|\#*|TTS_PROVIDER) continue ;; esac
  push_secret "$key"
done < "$ENV_FILE"
