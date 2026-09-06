package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"podcast-service/internal/httpx"
	"strings"
)

type Voxtral struct {
	APIKey string
}

type voxtralRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice_id"`
	ResponseFormat string `json:"response_format"`
}

type voxtralResponse struct {
	AudioData string `json:"audio_data"`
}

func (v *Voxtral) Synthesize(ctx context.Context, script, voiceID string) ([]byte, error) {
	if err := Validate("voxtral", "", v.APIKey, voiceID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(script) == "" {
		return nil, errors.New("TTS script is empty")
	}

	reqBody := voxtralRequest{
		Model:          "voxtral-mini-tts-2603",
		Input:          script,
		Voice:          voiceID,
		ResponseFormat: "mp3",
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/audio/speech", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+v.APIKey)

	body, err := httpx.Do(providerClient, req, 48<<20, true)
	if err != nil {
		return nil, fmt.Errorf("voxtral API: %w", err)
	}

	var data voxtralResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.New("voxtral API: invalid response")
	}
	audio, err := base64.StdEncoding.DecodeString(data.AudioData)
	if err != nil {
		return nil, errors.New("voxtral API: invalid audio encoding")
	}
	if err := validateAudio(audio); err != nil {
		return nil, err
	}
	return audio, nil
}
