// Package stt provides speech-to-text (STT) provider interfaces.
package stt

import (
	"context"

	"github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-web-frame/config"
)

// Provider is the interface for speech-to-text services.
type Provider interface {
	Name() string
	Init(ctx context.Context, cfg config.IConfig) error
	Transcribe(audio []byte, opts *TranscribeOptions) (*TranscribeResult, error)
	GetModels() []string
	GetProviderInfo() types.ProviderInfo
}

// TranscribeOptions holds STT parameters.
type TranscribeOptions struct {
	Model    string `json:"model"`
	Language string `json:"language"`
	Format   string `json:"format"` // mp3, wav, ogg, webm
}

// TranscribeResult holds the result of speech recognition.
type TranscribeResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Language   string  `json:"language"`
	Segments   []Segment `json:"segments,omitempty"`
}

// Segment is a timed segment of transcribed speech.
type Segment struct {
	StartMs int    `json:"start_ms"`
	EndMs   int    `json:"end_ms"`
	Text    string `json:"text"`
}
