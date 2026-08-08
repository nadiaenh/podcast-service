#!/usr/bin/env bash

env_file=".env"
touch "$env_file"

get_existing() {
  grep -E "^$1=" "$env_file" 2>/dev/null | head -1 | cut -d= -f2-
}

set_env_line() {
  local key="$1" value="$2"
  if grep -qE "^$key=" "$env_file"; then
    sed -i '' "s|^$key=.*|$key=$value|" "$env_file"
  else
    echo "$key=$value" >> "$env_file"
  fi
}

prompt_var() {
  local key="$1" hint="$2" pulumi_key="$3" plain="${4:-}"
  local existing
  existing=$(get_existing "$key")
  if [ -n "$existing" ]; then
    ok "$key already set in .env"
  else
    read -rp "→ $key ($hint): " value
    [ -z "$value" ] && { echo "  skipped (leave blank in .env, set later)"; return 0; }
    set_env_line "$key" "$value"
    existing="$value"
  fi
  if [ -n "$pulumi_key" ]; then
    if [ "$plain" = "plain" ]; then
      quiet pulumi config set "$pulumi_key" "$existing" || fail "failed to set $pulumi_key" "see error output below"
    else
      quiet pulumi config set --secret "$pulumi_key" "$existing" || fail "failed to set $pulumi_key" "see error output below"
    fi
  fi
}

echo
echo "→ app api keys (stored in .env, gitignored, and synced into pulumi config):"
prompt_var ANTHROPIC_API_KEY "console.anthropic.com -> API Keys" anthropicApiKey
prompt_var TTS_PROVIDER "elevenlabs or voxtral" ttsProvider plain
prompt_var ELEVENLABS_API_KEY "elevenlabs.io -> Profile -> API Keys" elevenlabsApiKey
prompt_var ELEVENLABS_VOICE_ID "elevenlabs.io -> Voices -> pick one -> copy Voice ID" elevenlabsVoiceId plain
prompt_var VOXTRAL_API_KEY "console.mistral.ai -> API Keys" voxtralApiKey
prompt_var VOXTRAL_VOICE_ID "curl https://api.mistral.ai/v1/audio/voices -H 'authorization: Bearer \$VOXTRAL_API_KEY'" voxtralVoiceId plain
ok "app secrets synced into pulumi config (encrypted, safe to commit)"
