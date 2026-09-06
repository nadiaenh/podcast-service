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
func TestValidation(t *testing.T) {
	for _, v := range []string{"", "../bad", "voice?query=bad", "voice/next"} {
		if Validate("elevenlabs", "key", "", v) == nil {
			t.Errorf("accepted %q", v)
		}
	}
	for _, b := range [][]byte{nil, []byte("ID3"), []byte(strings.Repeat("x", 200))} {
		if validateAudio(b) == nil {
			t.Fatal("accepted invalid audio")
		}
	}
}
func TestVoxtralRequestAndAudio(t *testing.T) {
	old := providerClient
	defer func() { providerClient = old }()
	audio := "ID3" + strings.Repeat("a", 200)
	providerClient = &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		var data map[string]any
		if e := json.NewDecoder(r.Body).Decode(&data); e != nil {
			t.Fatal(e)
		}
		if data["voice_id"] != "voice" || data["response_format"] != "mp3" || data["voice"] != nil {
			t.Fatalf("bad request %v", data)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"audio_data":"` + base64.StdEncoding.EncodeToString([]byte(audio)) + `"}`))}, nil
	})}
	got, e := (&Voxtral{APIKey: "key"}).Synthesize(context.Background(), "script", "voice")
	if e != nil || string(got) != audio {
		t.Fatalf("audio %v", e)
	}
}
