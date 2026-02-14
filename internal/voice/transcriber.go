package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	defaultModel   = "gpt-4o-mini-transcribe"
	defaultBaseURL = "https://api.openai.com/v1"
	defaultTimeout = 30 * time.Second
	maxInputBytes  = 25 * 1024 * 1024
)

// Input is one audio payload to transcribe.
type Input struct {
	FileName string
	MIMEType string
	Data     []byte
}

// Transcriber converts audio bytes to text.
type Transcriber interface {
	Transcribe(ctx context.Context, input Input) (string, error)
}

type openAITranscriber struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

// NewOpenAITranscriber builds an OpenAI-compatible audio transcription client.
func NewOpenAITranscriber(apiKey, baseURL, model string, timeout time.Duration) (Transcriber, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required for voice transcription")
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &openAITranscriber{
		endpoint: strings.TrimRight(baseURL, "/") + "/audio/transcriptions",
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{Timeout: timeout},
	}, nil
}

func (t *openAITranscriber) Transcribe(ctx context.Context, input Input) (string, error) {
	if len(input.Data) == 0 {
		return "", fmt.Errorf("audio data must not be empty")
	}
	if len(input.Data) > maxInputBytes {
		return "", fmt.Errorf("audio data too large: %d bytes (max %d)", len(input.Data), maxInputBytes)
	}

	body, contentType, err := createMultipartForm(input, t.model)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("transcription request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode transcription response: %w", err)
	}
	out.Text = strings.TrimSpace(out.Text)
	if out.Text == "" {
		return "", fmt.Errorf("transcription response returned empty text")
	}
	return out.Text, nil
}

func createMultipartForm(input Input, model string) (*bytes.Buffer, string, error) {
	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		fileName = "audio.bin"
	}
	mimeType := strings.TrimSpace(input.MIMEType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	partHeader.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(input.Data); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("model", strings.TrimSpace(model)); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}
