package nodes

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
)

// ExecuteNodeConfig is the configuration for an execute node.
// Runs a shell command. Supports {{node.output}} placeholders for upstream data.
type ExecuteNodeConfig struct {
	Command   string `json:"command"`    // shell command to run
	Timeout   int    `json:"timeout"`    // timeout in seconds, 0 = no timeout (default 30)
	WorkDir   string `json:"work_dir"`   // working directory for the command (optional)
	BlockList string `json:"block_list"` // comma-separated extra patterns to block (optional)
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

	// Sandbox: reject dangerous commands before execution.
	if err := validateCommand(command, cfg.BlockList); err != nil {
		return &engine.NodeOutput{
			Data: map[string]any{
				KeyOutput: err.Error(),
				"success": false,
			},
			Status: engine.StatusSuccess, // don't block the flow on command rejection
		}, nil
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 // default 30s
	}

	output, err := runCommand(command, time.Duration(timeout)*time.Second, cfg.WorkDir)
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
// workDir, if non-empty, sets the working directory for the command.
func runCommand(command string, timeout time.Duration, workDir string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	if workDir != "" {
		cmd.Dir = workDir
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

// dangerousPatterns is the list of regex patterns that identify commands
// considered too dangerous to execute.
var dangerousPatterns = []string{
	`rm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)?/(?:\s|$)`, // rm -rf /, rm -rf /*
	`rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+/`,  // rm -rf / variants
	`mkfs`,                            // mkfs filesystem formatting
	`format\s+[a-zA-Z]:`,              // Windows format command
	`shutdown`,                        // shutdown
	`:\(\)\s*\{\s*:\|:\s*&\s*\}\s*;:`, // classic fork bomb :(){ :|:& };:
	`>\s*/dev/sd[a-z]`,                // write directly to block device
	`dd\s+.*of=/dev/sd[a-z]`,          // dd to block device
}

// validateCommand checks a command string against a set of dangerous patterns
// and an optional caller-supplied block list. It returns an error if the
// command matches any blocked pattern.
func validateCommand(command, blockList string) error {
	for _, p := range dangerousPatterns {
		if matched, _ := regexp.MatchString(p, command); matched {
			return fmt.Errorf("command rejected by sandbox: matches dangerous pattern %q", p)
		}
	}
	// Apply caller-supplied extra block list (comma-separated substrings).
	if blockList != "" {
		for _, item := range strings.Split(blockList, ",") {
			item = strings.TrimSpace(item)
			if item != "" && strings.Contains(command, item) {
				return fmt.Errorf("command rejected by sandbox: contains blocked substring %q", item)
			}
		}
	}
	return nil
}
