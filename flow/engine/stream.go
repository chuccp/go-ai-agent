package engine

// Flow event types
const (
	EventNodeStart    = "flow_node_start"
	EventNodeChunk    = "flow_node_chunk"
	EventNodeDone     = "flow_node_done"
	EventWaitingUser  = "flow_waiting_user"
	EventFlowComplete = "flow_complete"
	EventFlowError    = "flow_error"
)

type FlowEvent struct {
	Type        string `json:"type"`
	ExecutionId uint   `json:"execution_id"`
	NodeId      uint   `json:"node_id,omitempty"`
	NodeLabel   string `json:"node_label,omitempty"`
	NodeType    string `json:"node_type,omitempty"`
	Content     string `json:"content,omitempty"`
	Message     string `json:"message,omitempty"`
	Status      string `json:"status,omitempty"`
}

type EventEmitter interface {
	Emit(event FlowEvent)
}

type NopEmitter struct{}

func (n *NopEmitter) Emit(event FlowEvent) {}
