package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Voxtral struct {
	APIKey string
}

type voxtralRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	Voice string `json:"voice"`
}

type voxtralResponse struct {
	AudioData string `json:"audio_data"`
}

func (v *Voxtral) Synthesize(ctx context.Context, script, voiceID string) ([]byte, error) {
	if v.APIKey == "" {
		return nil, errors.New("VOXTRAL_API_KEY is not set")
	}
	if voiceID == "" {
		return nil, errors.New("VOXTRAL_VOICE_ID is not set")
	}

	reqBody := voxtralRequest{
		Model: "voxtral-mini-tts-2603",
		Input: script,
		Voice: voiceID,
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
		return nil, fmt.Errorf("VOXTRAL_API_KEY is invalid or expired (voxtral api: %d %s)", resp.StatusCode, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("voxtral api: %d %s", resp.StatusCode, string(body))
	}

	var data voxtralResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("voxtral api: decode response: %w", err)
	}
	audio, err := base64.StdEncoding.DecodeString(data.AudioData)
	if err != nil {
		return nil, fmt.Errorf("voxtral api: decode audio: %w", err)
	}
	return audio, nil
}
