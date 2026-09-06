package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTryInOrderFallsBackThenSucceeds(t *testing.T) {
	old := retryWaits
	retryWaits = []time.Duration{0, 0, 0}
	defer func() { retryWaits = old }()

	calls := 0
	attempts := []attempt[int]{
		{"primary", func() (int, error) { calls++; return 0, errors.New("boom") }},
		{"primary", func() (int, error) { calls++; return 0, errors.New("boom") }},
		{"fallback", func() (int, error) { calls++; return 7, nil }},
	}
	got, err := tryInOrder(context.Background(), "test", attempts)
	if err != nil || got != 7 || calls != 3 {
		t.Fatalf("got=%d err=%v calls=%d", got, err, calls)
	}
}

func TestTryInOrderReturnsLastErrorAndHonorsContext(t *testing.T) {
	old := retryWaits
	retryWaits = []time.Duration{0, time.Hour}
	defer func() { retryWaits = old }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tryInOrder(ctx, "test", []attempt[int]{
		{"a", func() (int, error) { return 0, errors.New("first") }},
		{"b", func() (int, error) { return 0, errors.New("second") }},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestTTSPlanOrdersPrimaryThenConfiguredFallback(t *testing.T) {
	both := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v", ElevenLabsAPIKey: "k", ElevenLabsVoice: "v"}
	if got := both.ttsPlan(); len(got) != 3 || got[0] != "voxtral" || got[2] != "elevenlabs" {
		t.Fatalf("plan = %v", got)
	}
	noFallback := Config{TTSProvider: "voxtral", VoxtralAPIKey: "k", VoxtralVoice: "v"}
	if got := noFallback.ttsPlan(); len(got) != 2 {
		t.Fatalf("plan without fallback = %v", got)
	}
}

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
