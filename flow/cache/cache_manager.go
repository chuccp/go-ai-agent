package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/chuccp/go-web-frame/util"
)

// CacheManager LLM result cache manager
type CacheManager struct {
	cachePath string
	enabled   bool
	mu        sync.RWMutex
}

// NewCacheManager Create cache manager
func NewCacheManager(cachePath string, enabled bool) *CacheManager {
	m := &CacheManager{
		cachePath: cachePath,
		enabled:   enabled && cachePath != "",
	}
	if m.enabled {
		util.CreateDirIfNoExists(cachePath)
	}
	return m
}

// IsEnabled Whether enabled
func (m *CacheManager) IsEnabled() bool {
	return m.enabled
}

// GenerateKey Generate cache key
func GenerateKey(parts ...string) string {
	return util.MD5Str(joinKey(parts))
}

func joinKey(parts []string) string {
	key := ""
	for _, p := range parts {
		key += p + "|"
	}
	return key
}

// cacheEntry Cache entry
type cacheEntry struct {
	Result string `json:"result"`
}

// Get Get cache
func (m *CacheManager) Get(cacheKey string) (string, bool) {
	if !m.enabled {
		return "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := util.ReadFileBytes(filepath.Join(m.cachePath, cacheKey))
	if err != nil {
		return "", false
	}
	var entry cacheEntry
	if json.Unmarshal(data, &entry) != nil {
		return "", false
	}
	return entry.Result, true
}

// Set Write cache
func (m *CacheManager) Set(cacheKey string, result string) error {
	if !m.enabled {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := cacheEntry{Result: result}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return util.WriteFile(data, filepath.Join(m.cachePath, cacheKey))
}

// GetOrCompute Get or compute cache
func (m *CacheManager) GetOrCompute(cacheKey string, fn func() (string, error)) (string, error) {
	if cached, ok := m.Get(cacheKey); ok {
		return cached, nil
	}
	result, err := fn()
	if err != nil {
		return "", err
	}
	_ = m.Set(cacheKey, result)
	return result, nil
}

// Clear Clear cache
func (m *CacheManager) Clear() {
	if !m.enabled || m.cachePath == "" {
		return
	}
	files, _ := os.ReadDir(m.cachePath)
	for _, f := range files {
		os.Remove(filepath.Join(m.cachePath, f.Name()))
	}
}
