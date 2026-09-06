package pipeline

import "testing"

func testConfig() Config {
	return Config{AnthropicAPIKey: "test", ElevenLabsAPIKey: "test", ElevenLabsVoice: "voice", TTSProvider: "elevenlabs"}
}

func TestValidate(t *testing.T) {
	if err := testConfig().Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	missingKey := testConfig()
	missingKey.AnthropicAPIKey = ""
	if err := missingKey.Validate(); err == nil {
		t.Fatal("missing ANTHROPIC_API_KEY accepted")
	}

	missingVoice := testConfig()
	missingVoice.ElevenLabsVoice = ""
	if err := missingVoice.Validate(); err == nil {
		t.Fatal("missing voice accepted")
	}

	voxtralNoKey := testConfig()
	voxtralNoKey.TTSProvider = "voxtral"
	voxtralNoKey.VoxtralVoice = "v"
	if err := voxtralNoKey.Validate(); err == nil {
		t.Fatal("voxtral without key accepted")
	}
}

func TestTagSeparatesURLAndVoice(t *testing.T) {
	cfg := testConfig()
	a := Tag(cfg, "https://example.com/one")
	if a == Tag(cfg, "https://example.com/two") {
		t.Fatal("different URLs share a tag")
	}
	cfg.ElevenLabsVoice = "other"
	if a == Tag(cfg, "https://example.com/one") {
		t.Fatal("different voices share a tag")
	}
	if Tag(testConfig(), "https://example.com/one") != a {
		t.Fatal("tag is not deterministic")
	}
}

func TestRunContextValidatesBeforeNetwork(t *testing.T) {
	bad := testConfig()
	bad.AnthropicAPIKey = ""
	if _, err := Run(bad, "https://example.com"); err == nil {
		t.Fatal("invalid config reached the network")
	}
}
