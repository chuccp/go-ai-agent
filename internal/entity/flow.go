package entity

import "time"

// FlowDefinition Flow definition - an app is a flow.
//
// The DB table stores only lightweight metadata. The full flow content
// (config, form_schema, settings, nodes, edges, skills, assets, icon) lives
// on disk inside the app directory pointed to by Path.
//
// Config / FormSchema / Settings are tagged gorm:"-" so they are never
// persisted; they are hydrated from flow.json on disk when a flow is loaded
// in detail. This keeps the JSON API backward compatible.
type FlowDefinition struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:256" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Category    string    `gorm:"size:128" json:"category"` // "picture_book", "story_video" etc
	Path        string    `gorm:"size:512" json:"path"`     // relative path to the app directory under data/apps
	Icon        string    `gorm:"size:512" json:"icon,omitempty"` // emoji, or icon filename (icon.png/icon.svg) in app dir

	// --- on-disk content (loaded from flow.json, NOT persisted in DB) ---
	Config     string `gorm:"-" json:"config,omitempty"`     // JSON: global config
	FormSchema string `gorm:"-" json:"form_schema,omitempty"` // JSON: form schema for app mode
	Settings   string `gorm:"-" json:"settings,omitempty"`    // JSON: flow-level settings (default_model, ...)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (FlowDefinition) TableName() string {
	return "flow_definitions"
}

// FlowNode Flow node — a plain data structure stored inside flow.json on disk.
// No longer a DB entity.
type FlowNode struct {
	Id        uint    `json:"id"`
	FlowId    uint    `json:"flow_id"`
	Type      string  `json:"type"`
	Label     string  `json:"label"`
	Config    string  `json:"config"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	GroupId   *uint   `json:"group_id,omitempty"` // Parent container node ID (nil=top level)
}

// FlowEdge Flow edge — a plain data structure stored inside flow.json on disk.
// No longer a DB entity.
type FlowEdge struct {
	Id           uint   `json:"id"`
	FlowId       uint   `json:"flow_id"`
	SourceNodeId uint   `json:"source_node_id"`
	TargetNodeId uint   `json:"target_node_id"`
	SourceHandle string `json:"source_handle"` // "output" / "yes" / "no"
	TargetHandle string `json:"target_handle"` // "input"
	Label        string `json:"label"`
}

// FlowExecution Flow execution instance
type FlowExecution struct {
	Id            uint      `gorm:"primaryKey" json:"id"`
	FlowId        uint      `gorm:"index" json:"flow_id"`
	SessionId     uint      `json:"session_id"`               // Associated chat session
	Status        string    `gorm:"size:32" json:"status"`    // "running", "waiting_user", "completed", "error"
	CurrentNodeId *uint     `json:"current_node_id"`          // Current executing node
	Context       string    `gorm:"type:text" json:"context"` // JSON: runtime context
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (FlowExecution) TableName() string {
	return "flow_executions"
}
