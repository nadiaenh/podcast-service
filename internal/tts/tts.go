package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"podcast-service/internal/httpx"
)

type Credentials struct {
	ElevenLabsKey, ElevenLabsVoice string
	VoxtralKey, VoxtralVoice       string
}

type provider struct {
	apiKey     func(Credentials) string
	voiceID    func(Credentials) string
	synthesize func(ctx context.Context, script, voiceID, apiKey string) ([]byte, error)
	verify     func(ctx context.Context, voiceID, apiKey string) error
}

var providers = map[string]provider{
	"elevenlabs": {
		apiKey:     func(c Credentials) string { return c.ElevenLabsKey },
		voiceID:    func(c Credentials) string { return c.ElevenLabsVoice },
		synthesize: synthesizeElevenLabs,
		verify:     verifyElevenLabs,
	},
	"voxtral": {
		apiKey:     func(c Credentials) string { return c.VoxtralKey },
		voiceID:    func(c Credentials) string { return c.VoxtralVoice },
		synthesize: synthesizeVoxtral,
		verify:     verifyVoxtral,
	},
}

var (
	httpClient   = httpx.NewClient(3*time.Minute, false)
	voicePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

func Validate(name string, c Credentials) error {
	p, ok := providers[name]
	if !ok {
		return fmt.Errorf("unknown TTS provider %q", name)
	}
	if strings.TrimSpace(p.apiKey(c)) == "" {
		return fmt.Errorf("%s API key is not set", name)
	}
	if !voicePattern.MatchString(p.voiceID(c)) {
		return fmt.Errorf("%s voice ID must be 1-128 characters of letters, digits, underscores or hyphens", name)
	}
	return nil
}

func Verify(ctx context.Context, name string, c Credentials) error {
	if err := Validate(name, c); err != nil {
		return err
	}
	p := providers[name]
	if err := p.verify(ctx, p.voiceID(c), p.apiKey(c)); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func Synthesize(ctx context.Context, name string, c Credentials, script string) ([]byte, error) {
	if err := Validate(name, c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(script) == "" {
		return nil, errors.New("script is empty")
	}
	p := providers[name]
	audio, err := p.synthesize(ctx, script, p.voiceID(c), p.apiKey(c))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if err := checkMP3(audio); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return audio, nil
}

func checkMP3(audio []byte) error {
	if len(audio) < 128 {
		return errors.New("audio is empty or truncated")
	}
	if string(audio[:3]) != "ID3" && !(audio[0] == 0xff && audio[1]&0xe0 == 0xe0) {
		return errors.New("response is not MP3 audio")
	}
	return nil
}

func synthesizeElevenLabs(ctx context.Context, script, voiceID, apiKey string) ([]byte, error) {
	body, err := json.Marshal(map[string]any{
		"text":     script,
		"model_id": "eleven_turbo_v2_5",
		"voice_settings": map[string]float64{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	})
	if err != nil {
		return nil, err
	}
	url := "https://api.elevenlabs.io/v1/text-to-speech/" + voiceID + "?output_format=mp3_44100_128"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("xi-api-key", apiKey)
	return httpx.Do(httpClient, req, 48<<20, true)
}

func verifyElevenLabs(ctx context.Context, voiceID, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.elevenlabs.io/v1/voices/"+voiceID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("xi-api-key", apiKey)
	_, err = httpx.Do(httpClient, req, 1<<20, false)
	return err
}

func synthesizeVoxtral(ctx context.Context, script, voiceID, apiKey string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{
		"model":           "voxtral-mini-tts-2603",
		"input":           script,
		"voice_id":        voiceID,
		"response_format": "mp3",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mistral.ai/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+apiKey)

	raw, err := httpx.Do(httpClient, req, 48<<20, true)
	if err != nil {
		return nil, err
	}
	var out struct {
		AudioData string `json:"audio_data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("invalid response")
	}
	audio, err := base64.StdEncoding.DecodeString(out.AudioData)
	if err != nil {
		return nil, errors.New("invalid audio encoding")
	}
	return audio, nil
}

func verifyVoxtral(ctx context.Context, _, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.mistral.ai/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "Bearer "+apiKey)
	_, err = httpx.Do(httpClient, req, 1<<20, false)
	return err
}
