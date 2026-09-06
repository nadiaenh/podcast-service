package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"podcast-service/internal/httpx"
	"strings"
)

type ElevenLabs struct {
	APIKey string
}

type voiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

type ttsRequest struct {
	Text          string        `json:"text"`
	ModelID       string        `json:"model_id"`
	VoiceSettings voiceSettings `json:"voice_settings"`
}

func (e *ElevenLabs) Synthesize(ctx context.Context, script, voiceID string) ([]byte, error) {
	if err := Validate("elevenlabs", e.APIKey, "", voiceID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(script) == "" {
		return nil, errors.New("TTS script is empty")
	}

	reqBody := ttsRequest{
		Text:    script,
		ModelID: "eleven_turbo_v2_5",
		VoiceSettings: voiceSettings{
			Stability:       0.5,
			SimilarityBoost: 0.75,
		},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s?output_format=mp3_44100_128", voiceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("xi-api-key", e.APIKey)

	body, err := httpx.Do(providerClient, req, 48<<20, true)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs API: %w", err)
	}

	if err := validateAudio(body); err != nil {
		return nil, err
	}
	return body, nil
}
