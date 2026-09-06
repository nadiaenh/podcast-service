package pipeline

import (
	"context"
	"testing"
)

func TestTTSChainOrdersPrimaryThenConfiguredFallback(t *testing.T) {
	both := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v", ElevenLabsAPIKey: "k", ElevenLabsVoice: "v"}
	if got := both.ttsChain(); len(got) != 2 || got[0] != "voxtral" || got[1] != "elevenlabs" {
		t.Fatalf("chain = %v", got)
	}
	noFallback := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v"}
	if got := noFallback.ttsChain(); len(got) != 1 || got[0] != "voxtral" {
		t.Fatalf("chain without fallback = %v", got)
	}
}

func testConfig() Config {
	return Config{AnthropicAPIKey: "test", ElevenLabsAPIKey: "test", ElevenLabsVoice: "voice", TTSProvider: "elevenlabs"}
}

func TestValidate(t *testing.T) {
	if err := testConfig().validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	missingKey := testConfig()
	missingKey.AnthropicAPIKey = ""
	if err := missingKey.validate(); err == nil {
		t.Fatal("missing ANTHROPIC_API_KEY accepted")
	}

	missingVoice := testConfig()
	missingVoice.ElevenLabsVoice = ""
	if err := missingVoice.validate(); err == nil {
		t.Fatal("missing voice accepted")
	}

	voxtralNoKey := testConfig()
	voxtralNoKey.TTSProvider = "voxtral"
	voxtralNoKey.VoxtralVoice = "v"
	if err := voxtralNoKey.validate(); err == nil {
		t.Fatal("voxtral without key accepted")
	}
}

func TestScriptValidatesBeforeNetwork(t *testing.T) {
	bad := testConfig()
	bad.AnthropicAPIKey = ""
	if _, err := Script(context.Background(), bad, "https://example.com", "source text"); err == nil {
		t.Fatal("invalid config reached the network")
	}
}
