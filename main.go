package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"podcast-service/internal/pipeline"
	"podcast-service/internal/store"
)

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: pod <url>")
		os.Exit(1)
	}
	url := os.Args[1]

	loadEnv(".env")

	dataDir := "./data"
	os.MkdirAll(dataDir, 0o755)

	st, err := store.New(dataDir + "/episodes.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := pipeline.Config{
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		ElevenLabsAPIKey: os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoice:  os.Getenv("ELEVENLABS_VOICE_ID"),
		VoxtralAPIKey:    os.Getenv("VOXTRAL_API_KEY"),
		VoxtralVoice:     os.Getenv("VOXTRAL_VOICE_ID"),
		TTSProvider:      envOr("TTS_PROVIDER", "elevenlabs"),
		DataDir:          dataDir,
	}

	ep, err := pipeline.Run(st, cfg, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed: %v\nrun again to resume from this step\n", err)
		os.Exit(1)
	}

	fmt.Println(ep.MP3Path)
}
