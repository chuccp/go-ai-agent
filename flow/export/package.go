package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chuccp/go-ai-agent/entity"
)

// PackageType identifies the file format.
const PackageType = "go-ai-agent-package"

// CurrentVersion is the current package format version.
const CurrentVersion = 2

// Meta is the metadata stored in meta.json inside the ZIP.
type Meta struct {
	Type       string `json:"type"`
	Version    int    `json:"version"`
	Kind       string `json:"kind"`
	ExportedAt string `json:"exported_at"`
}

// FlowData is the flow payload stored in flow.json inside the ZIP.
type FlowData struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Category    string             `json:"category"`
	Nodes       []*entity.FlowNode `json:"nodes"`
	Edges       []*entity.FlowEdge `json:"edges"`
}

// BuildFlowPackage creates a ZIP file containing the flow definition and metadata.
func BuildFlowPackage(name string, nodes []*entity.FlowNode, edges []*entity.FlowEdge, desc, category string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// meta.json
	meta := Meta{
		Type:       PackageType,
		Version:    CurrentVersion,
		Kind:       "flow",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeJSON(w, "meta.json", meta); err != nil {
		return nil, err
	}

	// flow.json
	fd := FlowData{
		Name:        name,
		Description: desc,
		Category:    category,
		Nodes:       stripIDs(nodes),
		Edges:       stripEdgeIDs(edges),
	}
	if err := writeJSON(w, "flow.json", fd); err != nil {
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

// ParseFlowPackage reads a ZIP and extracts the flow definition.
// Returns the parsed flow data and any error.
func ParseFlowPackage(data []byte) (*FlowData, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip file: %w", err)
	}

	var flowJSON []byte
	for _, f := range r.File {
		if f.Name == "flow.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to read flow.json: %w", err)
			}
			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				rc.Close()
				return nil, err
			}
			rc.Close()
			flowJSON = buf.Bytes()
			break
		}
	}

	if flowJSON == nil {
		return nil, fmt.Errorf("flow.json not found in package")
	}

	var fd FlowData
	if err := json.Unmarshal(flowJSON, &fd); err != nil {
		return nil, fmt.Errorf("invalid flow.json: %w", err)
	}
	if fd.Name == "" {
		return nil, fmt.Errorf("flow name is required")
	}
	return &fd, nil
}

func writeJSON(w *zip.Writer, name string, v any) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(v)
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

// stripEdgeIDs removes DB IDs and external references from edges.
func stripEdgeIDs(edges []*entity.FlowEdge) []*entity.FlowEdge {
	out := make([]*entity.FlowEdge, len(edges))
	for i, e := range edges {
		clone := *e
		clone.Id = 0
		clone.FlowId = 0
		// Keep SourceNodeId / TargetNodeId — they're positional references within nodes array
		// The frontend or importer needs to re-map these after creation
		out[i] = &clone
	}
	return out
}
