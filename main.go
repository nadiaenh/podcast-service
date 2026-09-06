package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"podcast-service/internal/pipeline"
)

func loadEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := applyEnvLine(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// applyEnvLine sets one KEY=value line from a .env file.
// A variable already set in the environment is left untouched.
func applyEnvLine(raw string) error {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return nil
	}
	key, value = strings.TrimSpace(key), strings.TrimSpace(value)
	if _, present := os.LookupEnv(key); present {
		return nil
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		value = value[1 : len(value)-1]
	}
	return os.Setenv(key, value)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		return fmt.Errorf("usage: pod <url> [output.mp3]")
	}
	url := os.Args[1]
	out := "podcast.mp3"
	if len(os.Args) == 3 {
		out = os.Args[2]
	}

	if err := loadEnv(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	cfg := pipeline.Config{
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		ElevenLabsAPIKey: os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoice:  os.Getenv("ELEVENLABS_VOICE_ID"),
		VoxtralAPIKey:    os.Getenv("VOXTRAL_API_KEY"),
		VoxtralVoice:     os.Getenv("VOXTRAL_VOICE_ID"),
		TTSProvider:      envOr("TTS_PROVIDER", "elevenlabs"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := pipeline.RunContext(ctx, cfg, url)
	if err != nil {
		return fmt.Errorf("failed: %w", err)
	}
	if err := os.WriteFile(out, result.Audio, 0o644); err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
