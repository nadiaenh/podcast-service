package script

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestScriptRejectsIncompleteOutput(t *testing.T) {
	old := scriptClient
	defer func() { scriptClient = old }()
	for _, body := range []string{`{"stop_reason":"max_tokens","content":[{"type":"text","text":"partial"}]}`, `{"stop_reason":"end_turn","content":[{"type":"text","text":" "}]}`, `garbage`} {
		scriptClient = &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		if _, e := Generate(context.Background(), "test", "claude-opus-5", "https://example.com", "article"); e == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
