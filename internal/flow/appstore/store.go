package appstore

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chuccp/go-ai-agent/internal/entity"
)

const (
	AppType    = "go-ai-agent-app"
	AppVersion = 4
	FlowFile   = "flow.json"
	MetaFile   = "meta.json"
	SkillsDir  = "skills"
	AssetsDir  = "assets"

	maxPackageSize = 100 << 20 // 100 MB total
	maxEntrySize   = 50 << 20  // 50 MB per file
)

// FlowContent is the on-disk structure stored in flow.json inside each app directory.
type FlowContent struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    string             `json:"category"`
	Config      string             `json:"config,omitempty"`
	FormSchema  string             `json:"form_schema,omitempty"`
	Settings    string             `json:"settings,omitempty"`
	Icon        string             `json:"icon,omitempty"`
	Nodes       []*entity.FlowNode `json:"nodes"`
	Edges       []*entity.FlowEdge `json:"edges"`
}

// Meta is stored in meta.json inside each app directory.
type Meta struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	Name       string `json:"name,omitempty"`
	ExportedAt string `json:"exported_at"`
}

// Store manages app directories on disk.
type Store struct {
	baseDir string
}

// New creates a Store rooted at baseDir (e.g. "./data/apps").
func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// BaseDir returns the base apps directory.
func (s *Store) BaseDir() string { return s.baseDir }

// EnsureBaseDir creates the base apps directory if it doesn't exist.
func (s *Store) EnsureBaseDir() error {
	return os.MkdirAll(s.baseDir, 0755)
}

// randomID generates a 16-char hex string for directory naming.
func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// CreateAppDir creates a new unique app directory with standard subdirectories
// and returns its path relative to the working directory.
func (s *Store) CreateAppDir() (string, error) {
	if err := s.EnsureBaseDir(); err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		id := randomID()
		relPath := filepath.Join(s.baseDir, id)
		if _, err := os.Stat(relPath); os.IsNotExist(err) {
			if err := os.MkdirAll(relPath, 0755); err != nil {
				return "", err
			}
			_ = os.MkdirAll(filepath.Join(relPath, SkillsDir), 0755)
			_ = os.MkdirAll(filepath.Join(relPath, AssetsDir), 0755)
			return relPath, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique app directory")
}

// SaveFlow writes flow.json into the app directory.
func (s *Store) SaveFlow(relPath string, content *FlowContent) error {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal flow.json: %w", err)
	}
	return os.WriteFile(filepath.Join(relPath, FlowFile), data, 0644)
}

// LoadFlow reads flow.json from the app directory.
func (s *Store) LoadFlow(relPath string) (*FlowContent, error) {
	data, err := os.ReadFile(filepath.Join(relPath, FlowFile))
	if err != nil {
		return nil, fmt.Errorf("read flow.json: %w", err)
	}
	var content FlowContent
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("parse flow.json: %w", err)
	}
	return &content, nil
}

// SaveMeta writes meta.json into the app directory.
func (s *Store) SaveMeta(relPath string, meta *Meta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(relPath, MetaFile), data, 0644)
}

// LoadMeta reads meta.json. Returns nil, err if not present.
func (s *Store) LoadMeta(relPath string) (*Meta, error) {
	data, err := os.ReadFile(filepath.Join(relPath, MetaFile))
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SaveIcon saves uploaded icon data and returns the icon filename (e.g. "icon.png").
// Any previous icon file is removed so only one icon exists at a time.
func (s *Store) SaveIcon(relPath string, data []byte, ext string) (string, error) {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	if ext == "" {
		ext = "png"
	}
	filename := "icon." + ext
	if err := os.WriteFile(filepath.Join(relPath, filename), data, 0644); err != nil {
		return "", err
	}
	for _, e := range []string{"png", "jpg", "jpeg", "svg", "webp", "gif"} {
		if e != ext {
			_ = os.Remove(filepath.Join(relPath, "icon."+e))
		}
	}
	return filename, nil
}

// SaveSVGIcon generates a deterministic SVG icon from the app name, writes it
// to the app directory, and returns the filename ("icon.svg").
func (s *Store) SaveSVGIcon(relPath string, name string) (string, error) {
	svg := GenerateSVGIcon(name)
	filename := "icon.svg"
	if err := os.WriteFile(filepath.Join(relPath, filename), []byte(svg), 0644); err != nil {
		return "", err
	}
	for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif"} {
		_ = os.Remove(filepath.Join(relPath, "icon."+e))
	}
	return filename, nil
}

// ReadIcon reads the icon file from the app directory.
// iconRef is the value stored in FlowDefinition.Icon (a filename like "icon.png").
// Returns the raw bytes and the MIME type.
func (s *Store) ReadIcon(relPath string, iconRef string) ([]byte, string, error) {
	if iconRef == "" || isEmoji(iconRef) {
		return nil, "", fmt.Errorf("no icon file (icon is emoji or empty)")
	}
	data, err := os.ReadFile(filepath.Join(relPath, iconRef))
	if err != nil {
		return nil, "", err
	}
	return data, mimeTypeFor(iconRef), nil
}

// GenerateSVGIcon returns an SVG string with a colored rounded square and the
// first letter of the name. The color is deterministic based on the name hash.
func GenerateSVGIcon(name string) string {
	initial := "A"
	for _, r := range name {
		initial = string(r)
		break
	}
	var hash uint32
	for _, c := range name {
		hash = hash*31 + uint32(c)
	}
	hue := hash % 360
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 128 128">
  <rect width="128" height="128" rx="28" fill="hsl(%d, 65%%, 55%%)"/>
  <text x="64" y="64" font-family="Arial,sans-serif" font-size="64" font-weight="bold" fill="white" text-anchor="middle" dominant-baseline="central">%s</text>
</svg>`, hue, initial)
}

// DeleteApp removes the entire app directory.
func (s *Store) DeleteApp(relPath string) error {
	return os.RemoveAll(relPath)
}

// CopyApp duplicates an app directory into a new one and returns the new path.
func (s *Store) CopyApp(srcPath string) (string, error) {
	newPath, err := s.CreateAppDir()
	if err != nil {
		return "", err
	}
	if err := copyDir(srcPath, newPath); err != nil {
		_ = s.DeleteApp(newPath)
		return "", err
	}
	return newPath, nil
}

// AppExists returns true if the app directory contains a flow.json.
func (s *Store) AppExists(relPath string) bool {
	if relPath == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(relPath, FlowFile))
	return err == nil && !info.IsDir()
}

// PackageToZip creates a ZIP archive of the entire app directory, ensuring
// meta.json is present and up-to-date.
func (s *Store) PackageToZip(relPath string) ([]byte, error) {
	content, err := s.LoadFlow(relPath)
	meta := &Meta{
		Type:       AppType,
		Version:    AppVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err == nil {
		meta.Name = content.Name
	}
	_ = s.SaveMeta(relPath, meta)

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	err = filepath.Walk(relPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(relPath, path)
		if err != nil {
			return err
		}
		zipName := filepath.ToSlash(rel)
		writer, err := w.Create(zipName)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(writer, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}

// InstallFromZip extracts a ZIP package into a new app directory and returns
// the new path plus the flow content.
func (s *Store) InstallFromZip(data []byte) (string, *FlowContent, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", nil, fmt.Errorf("open package: %w", err)
	}

	// ZIP-bomb mitigation.
	var total uint64
	for _, f := range r.File {
		if f.UncompressedSize64 > maxEntrySize {
			return "", nil, fmt.Errorf("file %q too large: %d bytes", f.Name, f.UncompressedSize64)
		}
		total += f.UncompressedSize64
		if total > maxPackageSize {
			return "", nil, fmt.Errorf("package total size exceeds limit (%d bytes)", maxPackageSize)
		}
	}

	newPath, err := s.CreateAppDir()
	if err != nil {
		return "", nil, err
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(filepath.Join(newPath, f.Name), 0755)
			continue
		}
		target := filepath.Join(newPath, filepath.FromSlash(f.Name))
		_ = os.MkdirAll(filepath.Dir(target), 0755)

		rc, err := f.Open()
		if err != nil {
			_ = s.DeleteApp(newPath)
			return "", nil, fmt.Errorf("read %s: %w", f.Name, err)
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			_ = s.DeleteApp(newPath)
			return "", nil, fmt.Errorf("create %s: %w", f.Name, err)
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			_ = s.DeleteApp(newPath)
			return "", nil, copyErr
		}
	}

	content, err := s.LoadFlow(newPath)
	if err != nil {
		_ = s.DeleteApp(newPath)
		return "", nil, fmt.Errorf("package missing flow.json: %w", err)
	}

	return newPath, content, nil
}

// --- helpers ---

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// isEmoji returns true if the string looks like an emoji (not a filename).
func isEmoji(s string) bool {
	if s == "" {
		return false
	}
	// Filenames contain a dot; emojis don't.
	return !strings.Contains(s, ".")
}

func mimeTypeFor(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
