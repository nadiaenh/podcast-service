package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

const usage = `usage:
  pod verify-keys              check every provider key is valid before any generation
  pod fetch <url> <dir>        extract the article, write <dir>/source.md (fails if too short)
  pod transcript <url> <dir>   read <dir>/source.md, write <dir>/transcript.txt
  pod speak <dir>              read <dir>/transcript.txt, write <dir>/episode.mp3`

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return errors.New(usage)
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
		TTSProvider:      envOr("TTS_PROVIDER", "voxtral"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return dispatch(ctx, cfg, args)
}

func dispatch(ctx context.Context, cfg pipeline.Config, args []string) error {
	switch {
	case args[0] == "verify-keys" && len(args) == 1:
		return cmdVerifyKeys(ctx, cfg)
	case args[0] == "fetch" && len(args) == 3:
		return cmdFetch(ctx, args[1], args[2])
	case args[0] == "transcript" && len(args) == 3:
		return cmdTranscript(ctx, cfg, args[1], args[2])
	case args[0] == "speak" && len(args) == 2:
		return cmdSpeak(ctx, cfg, args[1])
	default:
		return errors.New(usage)
	}
}

func cmdVerifyKeys(ctx context.Context, cfg pipeline.Config) error {
	if err := pipeline.VerifyKeys(ctx, cfg); err != nil {
		return err
	}
	fmt.Println("all required provider keys verified")
	return nil
}

func cmdFetch(ctx context.Context, url, dir string) error {
	res, err := pipeline.Fetch(ctx, url)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "source.md"), []byte(res.Source), 0o644); err != nil {
		return err
	}
	fmt.Println(res.Title)
	return nil
}

func cmdTranscript(ctx context.Context, cfg pipeline.Config, url, dir string) error {
	source, err := os.ReadFile(filepath.Join(dir, "source.md"))
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	scriptText, err := pipeline.Script(ctx, cfg, url, string(source))
	if err != nil {
		return fmt.Errorf("failed: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "transcript.txt"), []byte(scriptText), 0o644)
}

func cmdSpeak(ctx context.Context, cfg pipeline.Config, dir string) error {
	scriptText, err := os.ReadFile(filepath.Join(dir, "transcript.txt"))
	if err != nil {
		return err
	}
	audio, err := pipeline.Speak(ctx, cfg, string(scriptText))
	if err != nil {
		return fmt.Errorf("failed: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "episode.mp3"), audio, 0o644)
}
