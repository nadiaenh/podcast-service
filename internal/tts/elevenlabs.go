package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	if e.APIKey == "" {
		return nil, errors.New("ELEVENLABS_API_KEY is not set")
	}
	if voiceID == "" {
		return nil, errors.New("ELEVENLABS_VOICE_ID is not set")
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

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("xi-api-key", e.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("ELEVENLABS_API_KEY is invalid or expired (elevenlabs api: %d %s)", resp.StatusCode, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("elevenlabs api: %d %s", resp.StatusCode, string(body))
	}
	return body, nil
}
