// Test for agent mode with tool calls.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func main() {
	server := flag.String("server", "ws://localhost:19009/ws/chat", "WebSocket server URL")
	model := flag.String("model", "1.default", "Model to use")
	prompt := flag.String("prompt", "列举当前系统有多少个已配置的模型", "User prompt")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		fmt.Println("dial:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := map[string]any{
		"type":       "agent",
		"session_id": 0,
		"model":      *model,
		"messages": []map[string]any{
			{"role": "user", "content": *prompt},
		},
		"stream": true,
	}

	data, _ := json.Marshal(req)
	fmt.Printf("=== Test: Agent mode ===\nPrompt: %s\n", *prompt)
	conn.WriteMessage(websocket.TextMessage, data)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			return
		}
		var resp map[string]any
		json.Unmarshal(msg, &resp)

		switch resp["type"] {
		case "session_created":
			fmt.Printf("[session_created: %v]\n", resp["session_id"])
		case "chunk":
			done, _ := resp["done"].(bool)
			if !done && resp["content"] != nil && resp["content"] != "" {
				fmt.Print(resp["content"])
			}
		case "tool_call":
			msg := resp["message"].(string)
			// Truncate long tool calls
			if len(msg) > 200 {
				msg = msg[:200] + "..."
			}
			fmt.Printf("\n[TOOL_CALL: %s]\n", msg)
		case "tool_result":
			msg := resp["message"].(string)
			if len(msg) > 500 {
				msg = msg[:500] + "..."
			}
			fmt.Printf("[TOOL_RESULT: %s]\n", msg)
		case "error":
			fmt.Printf("\nERROR: %v\n", resp["message"])
		case "iter_start":
			fmt.Printf("[Iter %v] ", resp["iteration"])
		}

		if done, ok := resp["done"].(bool); ok && done {
			fmt.Println(strings.Repeat("-", 40))
			return
		}
	}
}
