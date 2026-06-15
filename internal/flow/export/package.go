package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/chuccp/go-ai-agent/internal/entity"
)

// PackageType identifies the file format.
const PackageType = "go-ai-agent-package"

// CurrentVersion is the current package format version.
const CurrentVersion = 3

// Meta is the metadata stored in meta.json inside the ZIP.
type Meta struct {
	Type        string   `json:"type"`
	Version     int      `json:"version"`
	Kind        string   `json:"kind"`
	Name        string   `json:"name,omitempty"`
	VersionStr  string   `json:"version_str,omitempty"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	ExportedAt  string   `json:"exported_at"`
	PrimaryFlow string   `json:"primary_flow"`
	Flows       []string `json:"flows"`
	Skills      []string `json:"skills,omitempty"`
	Resources   []string `json:"resources,omitempty"`
	Config      string   `json:"config,omitempty"`
}

// FlowData is the flow payload stored in flow.json inside the ZIP.
type FlowData struct {
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

// SkillData is the skill payload stored inside the skills/ directory.
type SkillData struct {
	SkillId      string `json:"skill_id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Icon         string `json:"icon,omitempty"`
	Inputs       string `json:"inputs,omitempty"`
	Outputs      string `json:"outputs,omitempty"`
	DefaultModel string `json:"default_model,omitempty"`
	Prompts      []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	} `json:"prompts,omitempty"`
}

// FullPackage holds everything extracted from an imported package ZIP.
type FullPackage struct {
	Meta      Meta
	Flow      FlowData
	Skills    []SkillData
	Resources map[string][]byte
	Config    []byte
}

// BuildFlowPackage creates a Package ZIP centered around a single primary flow.
// The package also contains a config file, description, and placeholder dirs.
func BuildFlowPackage(flow *entity.FlowDefinition, nodes []*entity.FlowNode, edges []*entity.FlowEdge) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	flowPath := "flows/main/flow.json"

	// meta.json
	meta := Meta{
		Type:        PackageType,
		Version:     CurrentVersion,
		Kind:        "package",
		Name:        flow.Name,
		Description: flow.Description,
		Icon:        flow.Icon,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		PrimaryFlow: flowPath,
		Flows:       []string{flowPath},
		Config:      "config/config.yml",
	}
	if err := writeJSON(w, "meta.json", meta); err != nil {
		return nil, err
	}

	// description.md
	if err := writeString(w, "description.md", flow.Description); err != nil {
		return nil, err
	}

	// config/config.yml
	if err := writeString(w, "config/config.yml", "# Package runtime configuration\n# Example:\n# openai_api_key: sk-xxx\n"); err != nil {
		return nil, err
	}

	// flow.json
	fd := FlowData{
		Name:        flow.Name,
		Description: flow.Description,
		Category:    flow.Category,
		Config:      flow.Config,
		FormSchema:  flow.FormSchema,
		Settings:    flow.Settings,
		Icon:        flow.Icon,
		Nodes:       stripIDs(nodes),
		Edges:       stripEdgeIDs(edges),
	}
	if err := writeJSON(w, flowPath, fd); err != nil {
		return nil, err
	}

	// Placeholder dirs for future use
	_ = writeEmptyDir(w, "skills/")
	_ = writeEmptyDir(w, "resources/")

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseFlowPackage reads a Package ZIP and extracts the primary flow definition.
// It supports both the new package format and the legacy single-flow format.
func ParseFlowPackage(data []byte) (*FlowData, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip file: %w", err)
	}

	// Try new package format first.
	primaryFlow := "flows/main/flow.json"
	var meta Meta
	metaFound := false
	for _, f := range r.File {
		if f.Name == "meta.json" {
			if err := readJSON(r, f.Name, &meta); err != nil {
				return nil, err
			}
			metaFound = true
			if meta.PrimaryFlow != "" {
				primaryFlow = meta.PrimaryFlow
			}
			break
		}
	}

	flowPath := primaryFlow
	if !metaFound {
		// Legacy format: flow.json at root.
		flowPath = "flow.json"
	}

	var fd FlowData
	if err := readJSON(r, flowPath, &fd); err != nil {
		return nil, err
	}
	if fd.Name == "" {
		return nil, fmt.Errorf("flow name is required")
	}
	return &fd, nil
}

// LoadPackageResource reads an arbitrary file from the package by path.
func LoadPackageResource(data []byte, filePath string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if path.Clean(f.Name) == path.Clean(filePath) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("resource not found: %s", filePath)
}

func writeJSON(w *zip.Writer, name string, v any) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readJSON(r *zip.Reader, name string, v any) error {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", name, err)
			}
			defer rc.Close()
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return err
			}
			if err := json.Unmarshal(buf.Bytes(), v); err != nil {
				return fmt.Errorf("invalid %s: %w", name, err)
			}
			return nil
		}
	}
	return fmt.Errorf("%s not found in package", name)
}

func writeString(w *zip.Writer, name, content string) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func writeEmptyDir(w *zip.Writer, name string) error {
	_, err := w.Create(name)
	return err
}

// stripIDs removes DB IDs from nodes so they can be re-imported cleanly.
func stripIDs(nodes []*entity.FlowNode) []*entity.FlowNode {
	out := make([]*entity.FlowNode, len(nodes))
	for i, n := range nodes {
		clone := *n
		clone.Id = 0
		clone.FlowId = 0
		out[i] = &clone
	}
	return out
}

// BuildFullPackage creates a complete package ZIP including flow, skills, resources and config.
func BuildFullPackage(meta Meta, fd FlowData, skills []SkillData, resources map[string][]byte, config []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	if err := writeJSON(w, "meta.json", meta); err != nil {
		return nil, err
	}
	if err := writeString(w, "description.md", meta.Description); err != nil {
		return nil, err
	}
	if config == nil {
		config = []byte("# Package runtime configuration\n")
	}
	if err := writeBytes(w, "config/config.yml", config); err != nil {
		return nil, err
	}
	if err := writeJSON(w, meta.PrimaryFlow, fd); err != nil {
		return nil, err
	}

	for _, s := range skills {
		dir := path.Join("skills", s.SkillId)
		if err := writeJSON(w, path.Join(dir, "skill.json"), s); err != nil {
			return nil, err
		}
	}

	for name, data := range resources {
		if err := writeBytes(w, path.Join("resources", name), data); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}
	return buf.Bytes(), nil
}

// ParseFullPackage reads a Package ZIP and extracts meta, flow, skills, resources and config.
func ParseFullPackage(data []byte) (*FullPackage, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip file: %w", err)
	}

	var meta Meta
	if err := readJSON(r, "meta.json", &meta); err != nil {
		return nil, err
	}

	flowPath := meta.PrimaryFlow
	if flowPath == "" {
		flowPath = "flows/main/flow.json"
	}
	var fd FlowData
	if err := readJSON(r, flowPath, &fd); err != nil {
		return nil, err
	}
	if fd.Name == "" {
		return nil, fmt.Errorf("flow name is required")
	}

	full := &FullPackage{Meta: meta, Flow: fd, Resources: make(map[string][]byte)}

	for _, f := range r.File {
		if path.Dir(f.Name) == "skills/"+path.Base(path.Dir(f.Name)) && path.Base(f.Name) == "skill.json" {
			var s SkillData
			if err := readJSON(r, f.Name, &s); err == nil {
				full.Skills = append(full.Skills, s)
			}
		}
		if path.Dir(f.Name) == "resources" {
			if data, err := readBytes(r, f.Name); err == nil {
				full.Resources[path.Base(f.Name)] = data
			}
		}
		if f.Name == "config/config.yml" {
			full.Config, _ = readBytes(r, f.Name)
		}
	}

	return full, nil
}

func writeBytes(w *zip.Writer, name string, data []byte) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func readBytes(r *zip.Reader, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("%s not found in package", name)
}

// stripEdgeIDs removes DB IDs and external references from edges.
func stripEdgeIDs(edges []*entity.FlowEdge) []*entity.FlowEdge {
	out := make([]*entity.FlowEdge, len(edges))
	for i, e := range edges {
		clone := *e
		clone.Id = 0
		clone.FlowId = 0
		out[i] = &clone
	}
	return out
}
