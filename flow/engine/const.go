package engine

// 节点执行状态
const (
	StatusSuccess      = "success"
	StatusError        = "error"
	StatusWaitingUser  = "waiting_user"
)

// 流程执行状态
const (
	ExecRunning   = "running"
	ExecCompleted = "completed"
	ExecCreated   = "created"
	ExecError     = "error"
)

// 边连接点
const (
	HandleOutput = "output"
	HandleInput  = "input"
)
