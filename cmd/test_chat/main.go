// Test for chat mode with file attachments.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	server := flag.String("server", "ws://localhost:19009/ws/chat", "WebSocket server URL")
	model := flag.String("model", "1.default", "Model to use")
	filePath := flag.String("file", "", "Server-side file path from upload")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: test_chat -file <server-path>")
		fmt.Println("Example: test_chat -file ./data/uploads/abc123_test-doc.txt")
		os.Exit(1)
	}

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		fmt.Println("dial:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := map[string]any{
		"type":       "chat",
		"session_id": 0,
		"model":      *model,
		"messages": []map[string]any{
			{"role": "user", "content": "请总结一下我上传的文件内容"},
		},
		"stream": true,
		"attachments": []map[string]any{
			{
				"id":   "test",
				"name": "test-file",
				"type": "text/plain",
				"size": 0,
				"path": *filePath,
			},
		},
	}

	data, _ := json.Marshal(req)
	fmt.Println("=== Test: Chat with attachment ===")
	conn.WriteMessage(websocket.TextMessage, data)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			return
		}
		var resp map[string]any
		json.Unmarshal(msg, &resp)

		if resp["type"] == "chunk" && resp["content"] != nil && resp["content"] != "" {
			fmt.Print(resp["content"])
		}
		if resp["type"] == "error" {
			fmt.Printf("\nERROR: %v\n", resp["message"])
			return
		}
		if done, ok := resp["done"].(bool); ok && done {
			fmt.Println("\n--- Done ---")
			return
		}
	}
}
