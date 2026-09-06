package tts

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"podcast-service/internal/httpx"
)

func VerifyVoxtral(ctx context.Context, apiKey string) error {
	if apiKey == "" {
		return errors.New("VOXTRAL_API_KEY is not set")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.mistral.ai/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "Bearer "+apiKey)
	if _, err := httpx.Do(providerClient, req, 1<<20, false); err != nil {
		return fmt.Errorf("voxtral/mistral API: %w", err)
	}
	return nil
}

func VerifyElevenLabs(ctx context.Context, apiKey, voiceID string) error {
	if apiKey == "" {
		return errors.New("ELEVENLABS_API_KEY is not set")
	}
	if !voicePattern.MatchString(voiceID) {
		return errors.New("ELEVENLABS_VOICE_ID is missing or malformed")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.elevenlabs.io/v1/voices/"+voiceID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("xi-api-key", apiKey)
	if _, err := httpx.Do(providerClient, req, 1<<20, false); err != nil {
		return fmt.Errorf("elevenlabs API: %w", err)
	}
	return nil
}
