package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

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

func (t *ExecuteCommand) Execute(ctx context.Context, call Call) (string, error) {
	var params struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", err
	}

	if params.Command == "" {
		return "Error: command cannot be empty", nil
	}

	// Sandbox: reject dangerous commands before execution.
	if err := validateCommand(params.Command); err != nil {
		return "Error: " + err.Error(), nil
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

// dangerousPatterns is the list of regex patterns that identify commands
// considered too dangerous to execute.
var dangerousPatterns = []string{
	`rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)?/(?:\s|$)`, // rm -rf /, rm -rf /*
	`rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+/`,   // rm -rf / variants
	`mkfs`,                                       // mkfs filesystem formatting
	`format\s+[a-zA-Z]:`,                         // Windows format command
	`shutdown`,                                   // shutdown
	`:\(\)\s*\{\s*:\|:\s*&\s*\}\s*;:`,            // classic fork bomb :(){ :|:& };:
	`>\s*/dev/sd[a-z]`,                           // write directly to block device
	`dd\s+.*of=/dev/sd[a-z]`,                     // dd to block device
}

// validateCommand checks a command string against a set of dangerous patterns.
// It returns an error if the command matches any blocked pattern.
func validateCommand(command string) error {
	for _, p := range dangerousPatterns {
		if matched, _ := regexp.MatchString(p, command); matched {
			return fmt.Errorf("command rejected by sandbox: matches dangerous pattern %q", p)
		}
	}
	return nil
}
