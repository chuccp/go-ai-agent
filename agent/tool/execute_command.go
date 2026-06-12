package tool

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func init() {
	Register(&ExecuteCommand{})
}

// ExecuteCommand executes local commands
type ExecuteCommand struct{}

func (t *ExecuteCommand) Definition() Definition {
	return Definition{
		Name:        "execute_command",
		Description: "Run a Windows cmd command on the local machine. Use this to open apps (start chrome, notepad, calc, explorer), manage files, or run programs.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The Windows cmd command to run. e.g. start chrome, notepad, calc, explorer, dir",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *ExecuteCommand) Execute(call Call) (string, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", err
	}

	if params.Command == "" {
		return "Error: command cannot be empty", nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", params.Command)
	} else {
		cmd = exec.Command("sh", "-c", params.Command)
	}

	// Execute with timeout
	done := make(chan struct {
		output []byte
		err    error
	}, 1)

	go func() {
		out, err := cmd.CombinedOutput()
		done <- struct {
			output []byte
			err    error
		}{out, err}
	}()

	select {
	case result := <-done:
		output := strings.TrimSpace(string(result.output))
		if result.err != nil {
			if output == "" {
				return "", result.err
			}
			return output, nil
		}
		if output == "" {
			return "Command executed successfully (no output)", nil
		}
		return output, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return "Command execution timed out (30s)", nil
	}
}
