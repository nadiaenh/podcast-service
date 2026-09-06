package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestValidate(t *testing.T) {
	good := Credentials{ElevenLabsKey: "key", ElevenLabsVoice: "voice"}
	if err := Validate("elevenlabs", good); err != nil {
		t.Fatalf("rejected valid credentials: %v", err)
	}
	if Validate("nope", good) == nil {
		t.Error("accepted unknown provider")
	}
	if Validate("elevenlabs", Credentials{ElevenLabsVoice: "voice"}) == nil {
		t.Error("accepted missing key")
	}
	for _, v := range []string{"", "../bad", "voice?q=bad", "voice/next"} {
		if Validate("elevenlabs", Credentials{ElevenLabsKey: "key", ElevenLabsVoice: v}) == nil {
			t.Errorf("accepted voice ID %q", v)
		}
	}
}

func TestCheckMP3(t *testing.T) {
	for _, b := range [][]byte{nil, []byte("ID3"), []byte(strings.Repeat("x", 200))} {
		if checkMP3(b) == nil {
			t.Error("accepted invalid audio")
		}
	}
	if err := checkMP3([]byte("ID3" + strings.Repeat("a", 200))); err != nil {
		t.Errorf("rejected valid ID3 audio: %v", err)
	}
}

func TestSynthesizeVoxtral(t *testing.T) {
	old := httpClient
	defer func() { httpClient = old }()

	audio := "ID3" + strings.Repeat("a", 200)
	httpClient = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["voice_id"] != "voice" || body["response_format"] != "mp3" || body["voice"] != nil {
			t.Fatalf("unexpected request body %v", body)
		}
		payload := `{"audio_data":"` + base64.StdEncoding.EncodeToString([]byte(audio)) + `"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload))}, nil
	})}

	got, err := Synthesize(context.Background(), "voxtral", Credentials{VoxtralKey: "key", VoxtralVoice: "voice"}, "script")
	if err != nil || string(got) != audio {
		t.Fatalf("got %q err %v", got, err)
	}
}
