package pipeline

import (
	"context"
	"testing"
)

func TestTTSChainSkipsUnconfiguredProviders(t *testing.T) {
	both := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v", ElevenLabsAPIKey: "k", ElevenLabsVoice: "v"}
	if got := both.ttsChain(); len(got) != 2 || got[0] != "voxtral" || got[1] != "elevenlabs" {
		t.Fatalf("chain = %v, want [voxtral elevenlabs]", got)
	}

	primaryOnly := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v"}
	if got := primaryOnly.ttsChain(); len(got) != 1 || got[0] != "voxtral" {
		t.Fatalf("chain = %v, want [voxtral]", got)
	}

	// Primary selected but only the fallback has credentials: the fallback must still run.
	fallbackOnly := Config{TTSProvider: "voxtral", ElevenLabsAPIKey: "k", ElevenLabsVoice: "v"}
	if got := fallbackOnly.ttsChain(); len(got) != 1 || got[0] != "elevenlabs" {
		t.Fatalf("chain = %v, want [elevenlabs]", got)
	}

	none := Config{TTSProvider: "voxtral"}
	if got := none.ttsChain(); len(got) != 0 {
		t.Fatalf("chain = %v, want empty", got)
	}
}

func TestValidateAnthropic(t *testing.T) {
	if err := (Config{AnthropicAPIKey: "test"}).validateAnthropic(); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	if err := (Config{}).validateAnthropic(); err == nil {
		t.Fatal("missing ANTHROPIC_API_KEY accepted")
	}
}

func TestScriptChecksAnthropicKeyBeforeNetwork(t *testing.T) {
	if _, err := Script(context.Background(), Config{}, "https://example.com", "source text"); err == nil {
		t.Fatal("missing key reached the network")
	}
}

func TestSpeakRejectsWhenNoProviderConfigured(t *testing.T) {
	if _, err := Speak(context.Background(), Config{}, "some transcript"); err == nil {
		t.Fatal("speak accepted a config with no TTS provider")
	}
}
