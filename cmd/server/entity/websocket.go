package entity

// WebSocket message type constants.
const (
	ChatType = "chat"
	PingType = "ping"
	PongType = "pong"
	StopType = "stop"
)

// RevMessage is the incoming WebSocket message structure for the chat protocol.
type RevMessage struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	SessionId string `json:"session_id"`
}
