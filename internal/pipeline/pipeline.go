package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"podcast-service/internal/extract"
	"podcast-service/internal/script"
	"podcast-service/internal/tts"
)

const minSourceChars = 600

var scriptModels = []string{"claude-opus-5", "claude-opus-5", "claude-sonnet-5"}

type Config struct {
	AnthropicAPIKey  string
	ElevenLabsAPIKey string
	ElevenLabsVoice  string
	VoxtralAPIKey    string
	VoxtralVoice     string
	TTSProvider      string
}

type Source struct {
	Title string
	Text  string
}

func (cfg Config) credentials() tts.Credentials {
	return tts.Credentials{
		ElevenLabsKey:   cfg.ElevenLabsAPIKey,
		ElevenLabsVoice: cfg.ElevenLabsVoice,
		VoxtralKey:      cfg.VoxtralAPIKey,
		VoxtralVoice:    cfg.VoxtralVoice,
	}
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.AnthropicAPIKey) == "" {
		return errors.New("ANTHROPIC_API_KEY is not set")
	}
	return tts.Validate(cfg.TTSProvider, cfg.credentials())
}

func (cfg Config) ttsProviders() (primary, fallback string) {
	if cfg.TTSProvider == "voxtral" {
		return "voxtral", "elevenlabs"
	}
	return "elevenlabs", "voxtral"
}

func (cfg Config) ttsPlan() []string {
	primary, fallback := cfg.ttsProviders()
	plan := []string{primary, primary}
	if tts.Validate(fallback, cfg.credentials()) == nil {
		plan = append(plan, fallback)
	} else {
		warnf("tts: fallback provider %q is not configured; no cross-provider retry available", fallback)
	}
	return plan
}

func Fetch(ctx context.Context, url string) (Source, error) {
	article, err := extract.Article(ctx, url)
	if err != nil {
		warnf("fetch: extraction failed for %s: %v", url, err)
		return Source{}, fmt.Errorf("fetch: %w", err)
	}
	text := strings.TrimSpace(article.Text)
	if n := utf8.RuneCountInString(text); n < minSourceChars {
		warnf("fetch: %s yielded %d characters, below the %d-character minimum", url, n, minSourceChars)
		return Source{}, fmt.Errorf("fetch: source has %d characters, need at least %d", n, minSourceChars)
	}
	return Source{Title: article.Title, Text: text}, nil
}

func Script(ctx context.Context, cfg Config, url, source string) (string, error) {
	if err := cfg.validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(source) == "" {
		return "", errors.New("script: source text is empty")
	}

	attempts := make([]attempt[string], len(scriptModels))
	for i, model := range scriptModels {
		model := model
		attempts[i] = attempt[string]{model, func() (string, error) {
			return script.Generate(ctx, cfg.AnthropicAPIKey, model, url, source)
		}}
	}
	text, err := tryInOrder(ctx, "script", attempts)
	if err != nil {
		return "", fmt.Errorf("script: %w", err)
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("script: provider returned an empty script")
	}
	return text, nil
}

func Speak(ctx context.Context, cfg Config, transcript string) ([]byte, error) {
	if strings.TrimSpace(transcript) == "" {
		return nil, errors.New("tts: transcript is empty")
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	creds := cfg.credentials()
	plan := cfg.ttsPlan()
	attempts := make([]attempt[[]byte], len(plan))
	for i, name := range plan {
		name := name
		attempts[i] = attempt[[]byte]{name, func() ([]byte, error) {
			return tts.Synthesize(ctx, name, creds, transcript)
		}}
	}
	audio, err := tryInOrder(ctx, "tts", attempts)
	if err != nil {
		return nil, fmt.Errorf("tts: %w", err)
	}
	return audio, nil
}

func VerifyKeys(ctx context.Context, cfg Config) error {
	creds := cfg.credentials()
	primary, fallback := cfg.ttsProviders()

	anthropic := func() (struct{}, error) { return struct{}{}, script.VerifyKey(ctx, cfg.AnthropicAPIKey) }
	verifyTTS := func(name string) func() (struct{}, error) {
		return func() (struct{}, error) { return struct{}{}, tts.Verify(ctx, name, creds) }
	}

	var problems []string
	if _, err := tryInOrder(ctx, "verify anthropic", []attempt[struct{}]{
		{"anthropic", anthropic}, {"anthropic", anthropic},
	}); err != nil {
		problems = append(problems, fmt.Sprintf("anthropic: %v", err))
	}
	if _, err := tryInOrder(ctx, "verify tts", []attempt[struct{}]{
		{primary, verifyTTS(primary)}, {primary, verifyTTS(primary)}, {fallback, verifyTTS(fallback)},
	}); err != nil {
		problems = append(problems, fmt.Sprintf("tts: no usable provider (%v)", err))
	}

	if len(problems) > 0 {
		warnf("verify: key check failed: %s", strings.Join(problems, "; "))
		return fmt.Errorf("key verification failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
