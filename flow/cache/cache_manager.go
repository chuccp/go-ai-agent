package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/chuccp/go-web-frame/util"
)

// CacheManager LLM 结果缓存管理器
type CacheManager struct {
	cachePath string
	enabled   bool
	mu        sync.RWMutex
}

// NewCacheManager 创建缓存管理器
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

// IsEnabled 是否启用
func (m *CacheManager) IsEnabled() bool {
	return m.enabled
}

// GenerateKey 生成缓存键
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

// cacheEntry 缓存条目
type cacheEntry struct {
	Result string `json:"result"`
}

// Get 获取缓存
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

// Set 写入缓存
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

// GetOrCompute 获取或计算缓存
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

// Clear 清空缓存
func (m *CacheManager) Clear() {
	if !m.enabled || m.cachePath == "" {
		return
	}
	files, _ := os.ReadDir(m.cachePath)
	for _, f := range files {
		os.Remove(filepath.Join(m.cachePath, f.Name()))
	}
}
