package entity

import "time"

// FlowDefinition Flow definition
type FlowDefinition struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:256" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Category    string    `gorm:"size:128" json:"category"` // "picture_book", "story_video" etc
	Config      string    `gorm:"type:text" json:"config"`  // JSON: global config
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (FlowDefinition) TableName() string {
	return "flow_definitions"
}

// FlowNode Flow node
type FlowNode struct {
	Id        uint    `gorm:"primaryKey" json:"id"`
	FlowId    uint    `gorm:"index" json:"flow_id"`
	Type      string  `gorm:"size:64" json:"type"`
	Label     string  `gorm:"size:256" json:"label"`
	Config    string  `gorm:"type:text" json:"config"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
	GroupId   *uint   `json:"group_id,omitempty"` // Parent container node ID (nil=top level)
}

func (FlowNode) TableName() string {
	return "flow_nodes"
}

// FlowEdge Flow edge
type FlowEdge struct {
	Id           uint   `gorm:"primaryKey" json:"id"`
	FlowId       uint   `gorm:"index" json:"flow_id"`
	SourceNodeId uint   `json:"source_node_id"`
	TargetNodeId uint   `json:"target_node_id"`
	SourceHandle string `gorm:"size:64" json:"source_handle"` // "output" / "yes" / "no"
	TargetHandle string `gorm:"size:64" json:"target_handle"` // "input"
	Label        string `gorm:"size:256" json:"label"`
}

func (FlowEdge) TableName() string {
	return "flow_edges"
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
