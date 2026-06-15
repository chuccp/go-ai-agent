package engine

// Node execution status
const (
	StatusSuccess      = "success"
	StatusError        = "error"
	StatusWaitingUser  = "waiting_user"
)

// Flow execution status
const (
	ExecRunning   = "running"
	ExecCompleted = "completed"
	ExecCreated   = "created"
	ExecError     = "error"
)

// Edge connection points
const (
	HandleOutput = "output"
	HandleInput  = "input"
)
