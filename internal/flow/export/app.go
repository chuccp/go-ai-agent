package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chuccp/go-ai-agent/internal/entity"
)

// AppType identifies the export file format.
const AppType = "go-ai-agent-app"

// CurrentVersion is the current app format version.
const CurrentVersion = 3

// Meta is the metadata stored in meta.json inside the ZIP.
type Meta struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	Kind       string `json:"kind"`
	Name       string `json:"name,omitempty"`
	VersionStr string `json:"version_str,omitempty"`
	ExportedAt string `json:"exported_at"`
	PrimaryApp string `json:"primary_app"`
}

// AppData is the app payload stored in app.json inside the ZIP.
type AppData struct {
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

// BuildAppPackage creates an app export ZIP.
func BuildAppPackage(app *entity.FlowDefinition, nodes []*entity.FlowNode, edges []*entity.FlowEdge) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	appPath := "app.json"

	// meta.json
	meta := Meta{
		Type:       AppType,
		Version:    CurrentVersion,
		Kind:       "app",
		Name:       app.Name,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		PrimaryApp: appPath,
	}
	if err := writeJSON(w, "meta.json", meta); err != nil {
		return nil, err
	}

	// app.json
	appData := AppData{
		Name:        app.Name,
		Description: app.Description,
		Category:    app.Category,
		Config:      app.Config,
		FormSchema:  app.FormSchema,
		Settings:    app.Settings,
		Icon:        app.Icon,
		Nodes:       stripIDs(nodes),
		Edges:       edges,
	}
	if err := writeJSON(w, appPath, appData); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseAppPackage reads an app ZIP and extracts the data.
func ParseAppPackage(data []byte) (*AppData, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open package: %w", err)
	}

	// Try to read meta.json to find primary app path
	var appPath string
	if meta, err := readMeta(r); err == nil && meta.PrimaryApp != "" {
		appPath = meta.PrimaryApp
	}

	// Fall back to default path
	if appPath == "" {
		appPath = "app.json"
	}

	var app AppData
	if err := readJSON(r, appPath, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

// readMeta reads meta.json from the ZIP.
func readMeta(r *zip.Reader) (*Meta, error) {
	var meta Meta
	if err := readJSON(r, "meta.json", &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// writeJSON writes a JSON value to the ZIP.
func writeJSON(w *zip.Writer, name string, v any) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// readJSON reads a JSON value from the ZIP.
func readJSON(r *zip.Reader, name string, v any) error {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", name, err)
			}
			defer rc.Close()

			if err := json.NewDecoder(rc).Decode(v); err != nil {
				return fmt.Errorf("failed to decode %s: %w", name, err)
			}
			return nil
		}
	}
	return fmt.Errorf("%s not found in package", name)
}

// stripIDs removes database IDs from nodes for export.
func stripIDs(nodes []*entity.FlowNode) []*entity.FlowNode {
	result := make([]*entity.FlowNode, len(nodes))
	for i, n := range nodes {
		clone := *n
		clone.Id = 0
		clone.FlowId = 0
		result[i] = &clone
	}
	return result
}
