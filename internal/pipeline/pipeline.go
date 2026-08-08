package pipeline

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"podcast-service/internal/extract"
	"podcast-service/internal/script"
	"podcast-service/internal/store"
	"podcast-service/internal/tts"
)

type Config struct {
	AnthropicAPIKey  string
	ElevenLabsAPIKey string
	ElevenLabsVoice  string
	VoxtralAPIKey    string
	VoxtralVoice     string
	TTSProvider      string
	DataDir          string
}

func idFor(url string) string {
	h := sha1.Sum([]byte(url))
	return hex.EncodeToString(h[:])[:12]
}

func Run(st *store.Store, cfg Config, url string) (store.Episode, error) {
	id := idFor(url)
	ep, ok := st.Get(id)
	if !ok {
		ep = store.Episode{ID: id, URL: url, Title: url, Status: "fetching", CreatedAt: store.Now()}
	}
	ep.Error = ""
	ep.UpdatedAt = store.Now()
	st.Upsert(ep)

	if ep.Status == "fetching" || ep.Title == url {
		res, err := extract.Article(url)
		if err != nil {
			return fail(st, ep, "fetch", err)
		}
		ep.Title = res.Title
		ep.Text = res.Text
		ep.Status = "writing"
		ep.UpdatedAt = store.Now()
		st.Upsert(ep)
	}

	if ep.Script == "" {
		scriptText, err := script.GenerateScript(cfg.AnthropicAPIKey, url, ep.Text)
		if err != nil {
			return fail(st, ep, "script", err)
		}
		ep.Script = scriptText
		ep.Status = "synthesizing"
		ep.UpdatedAt = store.Now()
		st.Upsert(ep)
	}

	mp3Path := filepath.Join(cfg.DataDir, ep.ID+".mp3")
	if _, err := os.Stat(mp3Path); err != nil {
		provider, err := tts.New(cfg.TTSProvider, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey)
		if err != nil {
			return fail(st, ep, "tts", err)
		}
		voiceID := cfg.ElevenLabsVoice
		if cfg.TTSProvider == "voxtral" {
			voiceID = cfg.VoxtralVoice
		}
		audio, err := provider.Synthesize(context.Background(), ep.Script, voiceID)
		if err != nil {
			return fail(st, ep, "tts", err)
		}
		if err := os.WriteFile(mp3Path, audio, 0o644); err != nil {
			return fail(st, ep, "save", err)
		}
	}

	ep.MP3Path = mp3Path
	ep.Status = "done"
	ep.UpdatedAt = store.Now()
	st.Upsert(ep)
	return ep, nil
}

func fail(st *store.Store, ep store.Episode, step string, err error) (store.Episode, error) {
	ep.Status = "error"
	ep.Error = fmt.Sprintf("%s: %v", step, err)
	ep.UpdatedAt = store.Now()
	st.Upsert(ep)
	return ep, fmt.Errorf("%s", ep.Error)
}

func RunStateless(cfg Config, url string) (title, scriptText string, audio []byte, err error) {
	res, err := extract.Article(url)
	if err != nil {
		return "", "", nil, fmt.Errorf("fetch: %w", err)
	}

	scriptText, err = script.GenerateScript(cfg.AnthropicAPIKey, url, res.Text)
	if err != nil {
		return res.Title, "", nil, fmt.Errorf("script: %w", err)
	}

	provider, err := tts.New(cfg.TTSProvider, cfg.ElevenLabsAPIKey, cfg.VoxtralAPIKey)
	if err != nil {
		return res.Title, scriptText, nil, fmt.Errorf("tts: %w", err)
	}
	voiceID := cfg.ElevenLabsVoice
	if cfg.TTSProvider == "voxtral" {
		voiceID = cfg.VoxtralVoice
	}
	audio, err = provider.Synthesize(context.Background(), scriptText, voiceID)
	if err != nil {
		return res.Title, scriptText, nil, fmt.Errorf("tts: %w", err)
	}
	return res.Title, scriptText, audio, nil
}
