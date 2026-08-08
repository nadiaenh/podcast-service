package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"podcast-service/internal/pipeline"
)

type requestBody struct {
	URL string `json:"url"`
}

type responseBody struct {
	MP3Url string `json:"mp3Url,omitempty"`
	Title  string `json:"title,omitempty"`
	Error  string `json:"error,omitempty"`
}

func jsonResponse(status int, body responseBody) events.LambdaFunctionURLResponse {
	raw, _ := json.Marshal(body)
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(raw),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func HandleRequest(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	apiKey := os.Getenv("API_KEY")
	provided := req.Headers["x-api-key"]
	if apiKey == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
		return jsonResponse(http.StatusUnauthorized, responseBody{Error: "unauthorized"}), nil
	}

	var body requestBody
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.URL == "" {
		return jsonResponse(http.StatusBadRequest, responseBody{Error: "missing or invalid \"url\" in request body"}), nil
	}

	cfg := pipeline.Config{
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		ElevenLabsAPIKey: os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoice:  os.Getenv("ELEVENLABS_VOICE_ID"),
		VoxtralAPIKey:    os.Getenv("VOXTRAL_API_KEY"),
		VoxtralVoice:     os.Getenv("VOXTRAL_VOICE_ID"),
		TTSProvider:      envOr("TTS_PROVIDER", "elevenlabs"),
	}

	title, _, audio, err := pipeline.RunStateless(cfg, body.URL)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, responseBody{Error: err.Error(), Title: title}), nil
	}

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return jsonResponse(http.StatusInternalServerError, responseBody{Error: "S3_BUCKET not configured"}), nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, responseBody{Error: fmt.Sprintf("aws config: %v", err)}), nil
	}
	s3Client := s3.NewFromConfig(awsCfg)
	key := fmt.Sprintf("%d.mp3", time.Now().UnixNano())

	if _, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(audio),
		ContentType: awsStringPtr("audio/mpeg"),
	}); err != nil {
		return jsonResponse(http.StatusInternalServerError, responseBody{Error: fmt.Sprintf("s3 upload: %v", err)}), nil
	}

	presignClient := s3.NewPresignClient(s3Client)
	presigned, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(24*time.Hour))
	if err != nil {
		return jsonResponse(http.StatusInternalServerError, responseBody{Error: fmt.Sprintf("presign: %v", err)}), nil
	}

	return jsonResponse(http.StatusOK, responseBody{MP3Url: presigned.URL, Title: title}), nil
}

func awsStringPtr(s string) *string { return &s }

func main() {
	log.SetFlags(0)
	lambda.Start(HandleRequest)
}
