// Package ai provides the unified generation service that ties together
// image, speech, and video providers, following the same provider-registry
// pattern as chat.UnifiedChatService.
package ai

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/internal2/ai/image"
	imgopenai "github.com/chuccp/go-ai-agent/internal2/ai/image/openai"
	"github.com/chuccp/go-ai-agent/internal2/ai/speech"
	speechopenai "github.com/chuccp/go-ai-agent/internal2/ai/speech/openai"
	"github.com/chuccp/go-ai-agent/internal2/ai/video"
	videoopenai "github.com/chuccp/go-ai-agent/internal2/ai/video/openai"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// GenService manages generation providers (image, audio, video) by model id.
// It reads config from the same "chat.{id}.*" keys used by the chat service,
// since generation endpoints share the same API credentials.
type GenService struct {
	mu              sync.RWMutex
	config          config.IConfig
	imgProviders    map[uint]image.Provider
	speechProviders map[uint]speech.Provider
	videoProviders  map[uint]video.Provider
}

func NewGenService() *GenService {
	return &GenService{
		imgProviders:    make(map[uint]image.Provider),
		speechProviders: make(map[uint]speech.Provider),
		videoProviders:  make(map[uint]video.Provider),
	}
}

// Init implements IService interface.
func (s *GenService) Init(ctx *core.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = ctx.GetConfig()
	return nil
}

// RegisterImageProvider registers an image provider for the given id.
func (s *GenService) RegisterImageProvider(id uint, provider image.Provider) error {
	configPrefix := "chat." + strconv.FormatUint(uint64(id), 10)
	switch p := provider.(type) {
	case *imgopenai.Provider:
		p.SetConfigPrefix(configPrefix)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imgProviders[id] = provider
	return nil
}

// RegisterSpeechProvider registers a speech provider for the given id.
func (s *GenService) RegisterSpeechProvider(id uint, provider speech.Provider) error {
	configPrefix := "chat." + strconv.FormatUint(uint64(id), 10)
	switch p := provider.(type) {
	case *speechopenai.Provider:
		p.SetConfigPrefix(configPrefix)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.speechProviders[id] = provider
	return nil
}

// RegisterVideoProvider registers a video provider for the given id.
func (s *GenService) RegisterVideoProvider(id uint, provider video.Provider) error {
	configPrefix := "chat." + strconv.FormatUint(uint64(id), 10)
	switch p := provider.(type) {
	case *videoopenai.Provider:
		p.SetConfigPrefix(configPrefix)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.videoProviders[id] = provider
	return nil
}

// resolvePath splits "id.model" into provider id and model name.
func resolvePath(path string) (uint, string, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid model path format, expected: id.model")
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid provider id in path: %s", parts[0])
	}
	return uint(id), parts[1], nil
}

// ImageSizeFromAspectRatio maps an aspect ratio to pixel dimensions for image generation.
func ImageSizeFromAspectRatio(ratio string) (width, height int) {
	switch ratio {
	case "16:9":
		return 1792, 1024
	case "9:16":
		return 1024, 1792
	case "4:3":
		return 1024, 768
	case "3:4":
		return 768, 1024
	default:
		return 1024, 1024
	}
}

// GenerateImage generates images using the provider referenced by modelPath.
// It returns the first URL and all URLs of saved image files.
func (s *GenService) GenerateImage(ctx context.Context, modelPath, prompt string, count int, aspectRatio string) (string, []string, error) {
	id, _, err := resolvePath(modelPath)
	if err != nil {
		return "", nil, err
	}

	s.mu.RLock()
	prov, ok := s.imgProviders[id]
	s.mu.RUnlock()
	if !ok {
		return "", nil, fmt.Errorf("image provider not found for id: %d", id)
	}

	w, h := ImageSizeFromAspectRatio(aspectRatio)
	opts := &image.GenerateOptions{
		Width:     w,
		Height:    h,
		NumImages: count,
	}

	result, err := prov.Generate(prompt, opts)
	if err != nil {
		return "", nil, err
	}

	first := ""
	if len(result.URLs) > 0 {
		first = result.URLs[0]
	}
	return first, result.URLs, nil
}

// GenerateAudio synthesizes speech using the provider referenced by modelPath.
// It returns the URL of the saved audio file.
func (s *GenService) GenerateAudio(ctx context.Context, modelPath, text, voice string) (string, error) {
	id, _, err := resolvePath(modelPath)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	prov, ok := s.speechProviders[id]
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("speech provider not found for id: %d", id)
	}

	if voice == "" {
		voice = "alloy"
	}

	opts := &speech.SynthesizeOptions{
		Voice: voice,
	}

	result, err := prov.Synthesize(text, opts)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// GenerateVideo generates a video using the provider referenced by modelPath.
// It returns the URL of the saved video file.
func (s *GenService) GenerateVideo(ctx context.Context, modelPath, prompt string, duration int, aspectRatio string) (string, error) {
	id, _, err := resolvePath(modelPath)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	prov, ok := s.videoProviders[id]
	s.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("video provider not found for id: %d", id)
	}

	opts := &video.GenerateOptions{
		Duration: duration,
	}

	result, err := prov.Generate(prompt, opts)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// ConfigureProvider sets API credentials and re-initializes providers from DB-stored config.
func (s *GenService) ConfigureProvider(id uint, apiKey, baseURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config == nil {
		return fmt.Errorf("gen service not initialized")
	}

	configKey := "chat." + strconv.FormatUint(uint64(id), 10)
	s.config.Put(configKey+".apiKey", apiKey)
	if baseURL != "" {
		s.config.Put(configKey+".baseUrl", baseURL)
	}

	// Re-init all providers for this id
	if prov, ok := s.imgProviders[id]; ok {
		if err := prov.Init(context.Background(), s.config); err != nil {
			log.Warn("image provider configure failed", zap.Uint("id", id), zap.Error(err))
		}
	}
	if prov, ok := s.speechProviders[id]; ok {
		if err := prov.Init(context.Background(), s.config); err != nil {
			log.Warn("speech provider configure failed", zap.Uint("id", id), zap.Error(err))
		}
	}
	if prov, ok := s.videoProviders[id]; ok {
		if err := prov.Init(context.Background(), s.config); err != nil {
			log.Warn("video provider configure failed", zap.Uint("id", id), zap.Error(err))
		}
	}

	log.Info("generation providers configured from DB", zap.Uint("id", id))
	return nil
}

// NewOpenAIImageProvider creates an OpenAI-compatible image provider.
func NewOpenAIImageProvider(name string) image.Provider {
	return imgopenai.NewProvider(name)
}

// NewOpenAISpeechProvider creates an OpenAI-compatible speech provider.
func NewOpenAISpeechProvider(name string) speech.Provider {
	return speechopenai.NewProvider(name)
}

// NewOpenAIVideoProvider creates an OpenAI-compatible video provider.
func NewOpenAIVideoProvider(name string) video.Provider {
	return videoopenai.NewProvider(name)
}
