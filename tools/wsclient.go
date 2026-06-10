//go:build ignore

package main

import (
	"github.com/bytedance/sonic"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:19009/ws/chat", nil)
	if err != nil {
		fmt.Println("DIAL ERR:", err)
		return
	}
	defer conn.Close()

	msg := map[string]interface{}{
		"type":       "flow_start",
		"session_id": float64(3),
		"model":      "deepseek.default",
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": "一只小兔子在森林里迷路了，它要找到回家的路"},
		},
		"options": map[string]interface{}{
			"flow_id":      float64(2),
			"execution_id": float64(2),
		},
	}
	data, _ := sonic.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
	fmt.Println(">>> flow_start 已发送")

	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("连接关闭:", err)
			break
		}
		var event map[string]interface{}
		sonic.Unmarshal(msg, &event)
		t, _ := event["type"].(string)

		switch t {
		case "flow_node_start":
			fmt.Printf("[开始] %v\n", event["node_label"])
		case "flow_node_chunk":
			if c, ok := event["content"].(string); ok {
				fmt.Print(c)
			}
		case "flow_node_done":
			fmt.Printf("\n[完成] %v %v\n", event["node_label"], event["content"])
		case "flow_waiting_user":
			fmt.Printf("[等待] %v\n>>> 自动回复确认\n", event["message"])
			resp := map[string]interface{}{
				"type":       "flow_user_response",
				"session_id": float64(3),
				"options": map[string]interface{}{
					"execution_id": float64(2),
					"response":     "确认",
				},
			}
			d, _ := sonic.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, d)
		case "flow_complete":
			fmt.Println("[流程完成]")
			return
		case "flow_error":
			fmt.Printf("[错误] %v\n", event["message"])
			return
		}
	}
}
