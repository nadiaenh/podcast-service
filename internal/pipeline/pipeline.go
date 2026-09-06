package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"podcast-service/internal/extract"
	"podcast-service/internal/script"
	"podcast-service/internal/tts"
)

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
	if cfg.TTSProvider == "voxtral" {
		return cfg.VoxtralVoice
	}
	return cfg.ElevenLabsVoice
}

// Tag is a short deterministic id for a url/provider/voice combination.
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

// Transcript fetches the article and writes the spoken-summary script for it.
func Transcript(ctx context.Context, cfg Config, url string) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}

	article, err := extract.ArticleContext(ctx, url)
	if err != nil {
		return Result{}, fmt.Errorf("fetch: %w", err)
	}
	if strings.TrimSpace(article.Text) == "" {
		return Result{}, errors.New("fetch: article contains no usable text")
	}

	scriptText, err := script.GenerateScriptContext(ctx, cfg.AnthropicAPIKey, url, article.Text)
	if err != nil {
		return Result{}, fmt.Errorf("script: %w", err)
	}
	if strings.TrimSpace(scriptText) == "" {
		return Result{}, errors.New("script: provider returned an empty script")
	}

	return Result{Title: article.Title, Source: article.Text, Script: scriptText}, nil
}

// Speak synthesizes narrated audio from a transcript.
func Speak(ctx context.Context, cfg Config, scriptText string) ([]byte, error) {
	if strings.TrimSpace(scriptText) == "" {
		return nil, errors.New("tts: transcript is empty")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	audio, err := synthesize(ctx, cfg, scriptText)
	if err != nil {
		return nil, fmt.Errorf("tts: %w", err)
	}
	return audio, nil
}

func synthesize(ctx context.Context, cfg Config, text string) ([]byte, error) {
	provider, err := tts.New(cfg.TTSProvider, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey)
	if err != nil {
		return nil, err
	}
	audio, err := provider.Synthesize(ctx, text, cfg.voice())
	if err != nil {
		return nil, err
	}
	if err := tts.ValidateAudio(audio); err != nil {
		return nil, err
	}
	return audio, nil
}
