package script

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type response struct {
	Content []contentBlock `json:"content"`
}

func GenerateScript(apiKey, url, articleText string) (string, error) {
	if apiKey == "" {
		return "", errors.New("ANTHROPIC_API_KEY is not set")
	}

	userMsg := fmt.Sprintf("Resource: %s\n\nPage content:\n%s", url, articleText)
	reqBody := request{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 2048,
		System:    scriptSystemPrompt,
		Messages:  []message{{Role: "user", Content: userMsg}},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("ANTHROPIC_API_KEY is invalid or expired (claude api: %d %s)", resp.StatusCode, string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("claude api: %d %s", resp.StatusCode, string(body))
	}

	var data response
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	for _, b := range data.Content {
		if b.Type == "text" {
			return trimSpace(b.Text), nil
		}
	}
	return "", errors.New("claude api: no text block in response")
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
