// Test for read_document tool via agent mode.
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
	filePath := flag.String("file", "", "Server-side file path to read")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: test_readdoc -file <server-path>")
		fmt.Println("Example: test_readdoc -file ./data/uploads/abc123_doc.pdf")
		os.Exit(1)
	}

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		fmt.Println("dial:", err)
		os.Exit(1)
	}
	defer conn.Close()

	prompt := fmt.Sprintf("请使用 read_document 工具读取文件 %s，总结文档内容", *filePath)
	req := map[string]any{
		"type":       "agent",
		"session_id": 0,
		"model":      *model,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}

	data, _ := json.Marshal(req)
	fmt.Printf("=== Test: read_document tool ===\nFile: %s\n", *filePath)
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
			fmt.Printf("\n[TOOL_CALL]\n")
		case "tool_result":
			msg := resp["message"].(string)
			if len(msg) > 1000 {
				msg = msg[:1000] + "..."
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
