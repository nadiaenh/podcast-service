package tts

import (
	"context"
	"fmt"
)

type Provider interface {
	Synthesize(ctx context.Context, script, voiceID string) ([]byte, error)
}

func New(providerName, elevenlabsKey, voxtralKey string) (Provider, error) {
	switch providerName {
	case "elevenlabs":
		return &ElevenLabs{APIKey: elevenlabsKey}, nil
	case "voxtral":
		return &Voxtral{APIKey: voxtralKey}, nil
	default:
		return nil, fmt.Errorf("unknown tts provider: %s", providerName)
	}
}
