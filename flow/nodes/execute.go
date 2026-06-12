package nodes

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chuccp/go-ai-agent/flow/engine"
)

// ExecuteNodeConfig is the configuration for an execute node.
// Runs a shell command. Supports {{node.output}} placeholders for upstream data.
type ExecuteNodeConfig struct {
	Command string `json:"command"` // shell command to run
	Timeout int    `json:"timeout"` // timeout in seconds, 0 = no timeout (default 30)
}

// ExecuteNode runs a local shell command and returns its output.
type ExecuteNode struct{}

func NewExecuteNode() *ExecuteNode { return &ExecuteNode{} }

func (n *ExecuteNode) Type() string { return TypeExecute }

func (n *ExecuteNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ExecuteNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Command == "" {
		return nil, fmt.Errorf("execute: command is required")
	}

	// Render {{node.output}} placeholders
	command := renderPrompt(cfg.Command, ctx)

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 // default 30s
	}

	output, err := runCommand(command, time.Duration(timeout)*time.Second)
	if err != nil {
		return &engine.NodeOutput{
			Data: map[string]any{
				KeyOutput: err.Error(),
				"success": false,
			},
			Status: engine.StatusSuccess, // don't block the flow on command failure
		}, nil
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: output,
			"success": true,
		},
		Status: engine.StatusSuccess,
	}, nil
}

// runCommand executes a shell command. timeout <= 0 means no timeout.
func runCommand(command string, timeout time.Duration) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	done := make(chan struct {
		output string
		err    error
	}, 1)

	go func() {
		out, err := cmd.CombinedOutput()
		done <- struct {
			output string
			err    error
		}{strings.TrimSpace(string(out)), err}
	}()

	if timeout <= 0 {
		result := <-done
		if result.err != nil {
			if result.output == "" {
				return "", result.err
			}
			return result.output, nil
		}
		if result.output == "" {
			return "", fmt.Errorf("command produced no output")
		}
		return result.output, nil
	}

	select {
	case result := <-done:
		if result.err != nil {
			if result.output == "" {
				return "", result.err
			}
			return result.output, nil
		}
		if result.output == "" {
			return "", fmt.Errorf("command produced no output")
		}
		return result.output, nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return "", fmt.Errorf("command timed out after %v", timeout)
	}
}
