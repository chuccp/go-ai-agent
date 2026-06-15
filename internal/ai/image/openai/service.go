package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/internal/ai/image"
	"github.com/chuccp/go-ai-agent/internal/ai/types"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// ProviderDefaults maps provider name → {baseURL, model}.
var ProviderDefaults = map[string][2]string{
	"openai": {"https://api.openai.com", "dall-e-3"},
}

// Provider implements image.Provider for OpenAI-compatible image generation.
type Provider struct {
	name         string
	configPrefix string
	config       Config
	mu           sync.RWMutex
	initialized  bool
}

func NewProvider(name string) *Provider { return &Provider{name: name} }

func (p *Provider) Name() string { return p.name }

func (p *Provider) SetConfigPrefix(prefix string) { p.configPrefix = prefix }

func (p *Provider) Init(_ context.Context, cfg config.IConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.initialized {
		return nil
	}
	key := p.configPrefix
	if key == "" {
		key = "chat." + p.name
	}
	var oc Config
	if err := cfg.UnmarshalKey(key, &oc); err != nil {
		return fmt.Errorf("load %s config failed: %w", key, err)
	}
	if def, ok := ProviderDefaults[p.name]; ok {
		if oc.BaseURL == "" {
			oc.BaseURL = def[0]
		}
		if oc.Model == "" {
			oc.Model = def[1]
		}
	}
	if oc.BaseURL == "" {
		return fmt.Errorf("%s.baseUrl is required", key)
	}
	p.config = oc
	p.initialized = true
	log.Info("OpenAI image provider initialized",
		zap.String("name", p.name),
		zap.String("baseUrl", oc.GetBaseURL()),
		zap.String("model", oc.GetModel()))
	return nil
}

func (p *Provider) GetModels() []string { return []string{p.config.GetModel()} }

func (p *Provider) GetProviderInfo() types.ProviderInfo {
	if def, ok := ProviderDefaults[p.name]; ok {
		return types.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return types.ProviderInfo{Model: p.config.GetModel(), BaseURL: p.config.GetBaseURL()}
}

// Generate implements image.Provider.Generate.
// It calls the OpenAI-compatible /v1/images/generations endpoint.
func (p *Provider) Generate(prompt string, opts *image.GenerateOptions) (*image.GenerateResult, error) {
	if err := p.checkInitialized(); err != nil {
		return nil, err
	}

	count := 1
	model := p.config.GetModel()
	size := "1024x1024"

	if opts != nil {
		if opts.NumImages > 0 {
			count = opts.NumImages
		}
		if opts.Model != "" {
			model = opts.Model
		}
		if opts.Width > 0 && opts.Height > 0 {
			size = fmt.Sprintf("%dx%d", opts.Width, opts.Height)
		}
	}

	body := map[string]any{
		"model":  model,
		"prompt": prompt,
		"n":      count,
		"size":   size,
	}

	data, err := p.postJSON(context.Background(), p.config.GetBaseURL()+"/v1/images/generations", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []struct {
			URL           string `json:"url"`
			B64JSON       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse image response: %w", err)
	}

	dir := filepath.Join(".", "data", "gen", "images")
	_ = os.MkdirAll(dir, 0755)

	var urls []string
	var b64List []string
	for i, item := range resp.Data {
		var content []byte
		ext := "png"
		if item.B64JSON != "" {
			b64List = append(b64List, item.B64JSON)
			content, err = base64.StdEncoding.DecodeString(item.B64JSON)
			if err != nil {
				continue
			}
		} else if item.URL != "" {
			content, err = p.download(context.Background(), item.URL)
			if err != nil {
				continue
			}
		} else {
			continue
		}
		filename := fmt.Sprintf("%d_%d_%d.%s", time.Now().Unix(), i, time.Now().Nanosecond(), ext)
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, content, 0644); err == nil {
			urls = append(urls, "/api/files/gen/images/"+filename)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no image returned")
	}
	return &image.GenerateResult{URLs: urls, Base64Data: b64List, Count: len(urls)}, nil
}

func (p *Provider) checkInitialized() error {
	if !p.initialized {
		return fmt.Errorf("OpenAI image provider %s not initialized", p.name)
	}
	return nil
}

func (p *Provider) postJSON(ctx context.Context, url string, body map[string]any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("image API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (p *Provider) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 20<<20))
}
