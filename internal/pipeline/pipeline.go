package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"podcast-service/internal/extract"
	"podcast-service/internal/script"
	"podcast-service/internal/tts"
)

const MinSourceChars = 600

var scriptModels = []string{"claude-opus-5", "claude-opus-5", "claude-sonnet-5"}

type Config struct {
	AnthropicAPIKey  string
	ElevenLabsAPIKey string
	ElevenLabsVoice  string
	VoxtralAPIKey    string
	VoxtralVoice     string
	TTSProvider      string
}

func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.AnthropicAPIKey) == "" {
		return errors.New("ANTHROPIC_API_KEY is not set")
	}
	return tts.Validate(cfg.TTSProvider, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey, cfg.voice())
}

func (cfg Config) voice() string {
	return cfg.voiceFor(cfg.TTSProvider)
}

func (cfg Config) voiceFor(provider string) string {
	if provider == "voxtral" {
		return cfg.VoxtralVoice
	}
	return cfg.ElevenLabsVoice
}

func (cfg Config) ttsPlan() []string {
	primary, fallback := "elevenlabs", "voxtral"
	if cfg.TTSProvider == "voxtral" {
		primary, fallback = "voxtral", "elevenlabs"
	}
	plan := []string{primary, primary}
	if tts.Validate(fallback, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey, cfg.voiceFor(fallback)) == nil {
		plan = append(plan, fallback)
	} else {
		warnf("tts: fallback provider %q is not configured; no cross-provider retry available", fallback)
	}
	return plan
}

func Tag(cfg Config, url string) string {
	h := sha256.Sum256([]byte(url + "\x00" + cfg.TTSProvider + "\x00" + cfg.voice()))
	return hex.EncodeToString(h[:])[:12]
}

// Result is a finished episode.
type Result struct {
	Title  string
	Source string
	Script string
	Audio  []byte
}

func Run(cfg Config, url string) (Result, error) {
	return RunContext(context.Background(), cfg, url)
}

func RunContext(ctx context.Context, cfg Config, url string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	res, err := Transcript(ctx, cfg, url)
	if err != nil {
		return Result{}, err
	}
	audio, err := Speak(ctx, cfg, res.Script)
	if err != nil {
		return Result{}, err
	}
	res.Audio = audio
	return res, nil
}

func Fetch(ctx context.Context, url string) (Result, error) {
	article, err := extract.ArticleContext(ctx, url)
	if err != nil {
		warnf("fetch: extraction failed for %s: %v", url, err)
		return Result{}, fmt.Errorf("fetch: %w", err)
	}
	text := strings.TrimSpace(article.Text)
	if n := utf8.RuneCountInString(text); n < MinSourceChars {
		warnf("fetch: %s yielded %d characters, below the %d minimum; treating the source as unavailable", url, n, MinSourceChars)
		return Result{}, fmt.Errorf("fetch: source has %d characters, need at least %d", n, MinSourceChars)
	}
	return Result{Title: article.Title, Source: text}, nil
}

func Script(ctx context.Context, cfg Config, url, source string) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	if strings.TrimSpace(source) == "" {
		return "", errors.New("script: source text is empty")
	}

	attempts := make([]attempt[string], len(scriptModels))
	for i, model := range scriptModels {
		model := model
		attempts[i] = attempt[string]{model, func() (string, error) {
			return script.GenerateScriptContext(ctx, cfg.AnthropicAPIKey, model, url, source)
		}}
	}
	scriptText, err := tryInOrder(ctx, "script", attempts)
	if err != nil {
		return "", fmt.Errorf("script: %w", err)
	}
	if strings.TrimSpace(scriptText) == "" {
		warnf("script: every model returned an empty script")
		return "", errors.New("script: provider returned an empty script")
	}
	return scriptText, nil
}

// Transcript fetches the article and writes the spoken-summary script for it.
func Transcript(ctx context.Context, cfg Config, url string) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	res, err := Fetch(ctx, url)
	if err != nil {
		return Result{}, err
	}
	scriptText, err := Script(ctx, cfg, url, res.Source)
	if err != nil {
		return Result{}, err
	}
	res.Script = scriptText
	return res, nil
}

func Speak(ctx context.Context, cfg Config, scriptText string) ([]byte, error) {
	if strings.TrimSpace(scriptText) == "" {
		return nil, errors.New("tts: transcript is empty")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	plan := cfg.ttsPlan()
	attempts := make([]attempt[[]byte], len(plan))
	for i, provider := range plan {
		provider := provider
		attempts[i] = attempt[[]byte]{provider, func() ([]byte, error) {
			return synthesize(ctx, cfg, provider, scriptText)
		}}
	}
	audio, err := tryInOrder(ctx, "tts", attempts)
	if err != nil {
		return nil, fmt.Errorf("tts: %w", err)
	}
	return audio, nil
}

func VerifyKeys(ctx context.Context, cfg Config) error {
	var problems []string

	anthropic := []attempt[struct{}]{
		{"anthropic", func() (struct{}, error) { return struct{}{}, script.VerifyKey(ctx, cfg.AnthropicAPIKey) }},
		{"anthropic", func() (struct{}, error) { return struct{}{}, script.VerifyKey(ctx, cfg.AnthropicAPIKey) }},
	}
	if _, err := tryInOrder(ctx, "verify anthropic", anthropic); err != nil {
		problems = append(problems, fmt.Sprintf("anthropic: %v", err))
	}

	primary, fallback := "elevenlabs", "voxtral"
	if cfg.TTSProvider == "voxtral" {
		primary, fallback = "voxtral", "elevenlabs"
	}
	ttsAttempts := []attempt[struct{}]{
		{primary, func() (struct{}, error) { return struct{}{}, verifyTTS(ctx, cfg, primary) }},
		{primary, func() (struct{}, error) { return struct{}{}, verifyTTS(ctx, cfg, primary) }},
		{fallback, func() (struct{}, error) { return struct{}{}, verifyTTS(ctx, cfg, fallback) }},
	}
	if _, err := tryInOrder(ctx, "verify tts", ttsAttempts); err != nil {
		problems = append(problems, fmt.Sprintf("tts: no usable provider (%v)", err))
	}

	if len(problems) > 0 {
		warnf("verify: key check failed: %s", strings.Join(problems, "; "))
		return fmt.Errorf("key verification failed: %s", strings.Join(problems, "; "))
	}
	return nil
}

func verifyTTS(ctx context.Context, cfg Config, provider string) error {
	switch provider {
	case "voxtral":
		return tts.VerifyVoxtral(ctx, cfg.VoxtralAPIKey)
	case "elevenlabs":
		return tts.VerifyElevenLabs(ctx, cfg.ElevenLabsAPIKey, cfg.ElevenLabsVoice)
	default:
		return fmt.Errorf("unknown TTS provider %q", provider)
	}
}

func synthesize(ctx context.Context, cfg Config, provider, text string) ([]byte, error) {
	p, err := tts.New(provider, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey)
	if err != nil {
		return nil, err
	}
	audio, err := p.Synthesize(ctx, text, cfg.voiceFor(provider))
	if err != nil {
		return nil, err
	}
	if err := tts.ValidateAudio(audio); err != nil {
		return nil, err
	}
	return audio, nil
}
