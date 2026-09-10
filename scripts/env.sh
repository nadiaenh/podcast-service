# Collect the provider credentials into .env.
prompt_secret ANTHROPIC_API_KEY "Anthropic API key (writes the transcript)" \
  "https://console.anthropic.com/settings/keys"

prompt_value TTS_PROVIDER "text-to-speech provider (voxtral or elevenlabs)" "voxtral"
provider=$(env_get TTS_PROVIDER)

case "$provider" in
  elevenlabs)
    prompt_secret ELEVENLABS_API_KEY "ElevenLabs API key" "https://elevenlabs.io/app/settings/api-keys"
    prompt_value  ELEVENLABS_VOICE_ID "ElevenLabs voice ID" ""
    ;;
  voxtral)
    prompt_secret VOXTRAL_API_KEY "Mistral API key (Voxtral TTS)" "https://console.mistral.ai/api-keys"
    prompt_value  VOXTRAL_VOICE_ID "Voxtral voice ID" "en_paul_cheerful"
    ;;
  *)
    fail "unknown TTS provider '$provider'" "choose voxtral or elevenlabs"
    ;;
esac

if confirm "also configure the other provider as a fallback?"; then
  case "$provider" in
    elevenlabs)
      prompt_secret VOXTRAL_API_KEY "Mistral API key (Voxtral TTS)" "https://console.mistral.ai/api-keys"
      prompt_value  VOXTRAL_VOICE_ID "Voxtral voice ID" "en_paul_cheerful" ;;
    voxtral)
      prompt_secret ELEVENLABS_API_KEY "ElevenLabs API key" "https://elevenlabs.io/app/settings/api-keys"
      prompt_value  ELEVENLABS_VOICE_ID "ElevenLabs voice ID" "" ;;
  esac
fi
