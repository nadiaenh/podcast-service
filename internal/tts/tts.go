package tts

import (
	"context"
	"errors"
	"fmt"
	"podcast-service/internal/httpx"
	"regexp"
	"strings"
	"time"
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

var providerClient = httpx.NewClient(3*time.Minute, false)
var voicePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func Validate(providerName, elevenlabsKey, voxtralKey, voiceID string) error {
	switch providerName {
	case "elevenlabs":
		if strings.TrimSpace(elevenlabsKey) == "" {
			return errors.New("ELEVENLABS_API_KEY is not set")
		}
	case "voxtral":
		if strings.TrimSpace(voxtralKey) == "" {
			return errors.New("VOXTRAL_API_KEY is not set")
		}
	default:
		return errors.New("unknown TTS provider")
	}
	if !voicePattern.MatchString(voiceID) {
		return errors.New("TTS voice ID must contain only letters, digits, underscores or hyphens (1-128 characters)")
	}
	return nil
}
func validateAudio(audio []byte) error {
	if len(audio) < 128 {
		return errors.New("TTS audio is empty or truncated")
	}
	if !(string(audio[:3]) == "ID3" || (audio[0] == 0xff && audio[1]&0xe0 == 0xe0)) {
		return errors.New("TTS response is not MP3 audio")
	}
	return nil
}

// ValidateAudio rejects empty or obviously invalid MP3 artifacts before reuse.
// It is a format sanity check, not a complete audio decoder.
func ValidateAudio(audio []byte) error { return validateAudio(audio) }
