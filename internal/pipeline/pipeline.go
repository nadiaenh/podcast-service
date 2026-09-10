package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"sockpuppet/internal/extract"
	"sockpuppet/internal/script"
	"sockpuppet/internal/tts"
)

const minSourceChars = 600

var scriptModels = []string{"claude-opus-5", "claude-sonnet-5"}

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

func (cfg Config) validateAnthropic() error {
	if strings.TrimSpace(cfg.AnthropicAPIKey) == "" {
		return errors.New("ANTHROPIC_API_KEY is not set")
	}
	return nil
}

func (cfg Config) ttsProviders() (primary, fallback string) {
	if cfg.TTSProvider == "voxtral" {
		return "voxtral", "elevenlabs"
	}
	return "elevenlabs", "voxtral"
}

// ttsChain lists the providers to try, preferred first, skipping any whose
// credentials are absent so a misconfigured primary never blocks a working fallback.
func (cfg Config) ttsChain() []string {
	primary, fallback := cfg.ttsProviders()
	var chain []string
	for _, name := range []string{primary, fallback} {
		if tts.Validate(name, cfg.credentials()) == nil {
			chain = append(chain, name)
		}
	}
	return chain
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
	if err := cfg.validateAnthropic(); err != nil {
		return "", err
	}
	if strings.TrimSpace(source) == "" {
		return "", errors.New("script: source text is empty")
	}

	var lastErr error
	for _, model := range scriptModels {
		text, err := script.Generate(ctx, cfg.AnthropicAPIKey, model, url, source)
		if err != nil {
			warnf("script: model %q failed: %v", model, err)
			lastErr = err
			continue
		}
		if strings.TrimSpace(text) == "" {
			return "", errors.New("script: provider returned an empty script")
		}
		return text, nil
	}
	return "", fmt.Errorf("script: %w", lastErr)
}

func Speak(ctx context.Context, cfg Config, transcript string) ([]byte, error) {
	if strings.TrimSpace(transcript) == "" {
		return nil, errors.New("tts: transcript is empty")
	}
	chain := cfg.ttsChain()
	if len(chain) == 0 {
		primary, fallback := cfg.ttsProviders()
		return nil, fmt.Errorf("tts: no usable provider; configure %s or %s credentials", primary, fallback)
	}
	if len(chain) == 1 {
		warnf("tts: only %q is configured; no cross-provider fallback", chain[0])
	}

	creds := cfg.credentials()
	var lastErr error
	for _, name := range chain {
		audio, err := tts.Synthesize(ctx, name, creds, transcript)
		if err != nil {
			warnf("tts: provider %q failed: %v", name, err)
			lastErr = err
			continue
		}
		return audio, nil
	}
	return nil, fmt.Errorf("tts: %w", lastErr)
}

func VerifyKeys(ctx context.Context, cfg Config) error {
	creds := cfg.credentials()
	primary, fallback := cfg.ttsProviders()

	var problems []string
	if err := script.VerifyKey(ctx, cfg.AnthropicAPIKey); err != nil {
		problems = append(problems, fmt.Sprintf("anthropic: %v", err))
	}
	if err := tts.Verify(ctx, primary, creds); err != nil {
		if fbErr := tts.Verify(ctx, fallback, creds); fbErr != nil {
			problems = append(problems, fmt.Sprintf("tts: no usable provider (%v; %v)", err, fbErr))
		}
	}

	if len(problems) > 0 {
		warnf("verify: key check failed: %s", strings.Join(problems, "; "))
		return fmt.Errorf("key verification failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
