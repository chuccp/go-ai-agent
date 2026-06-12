//go:build ignore

package main

import (
	"bytes"
	"github.com/bytedance/sonic"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	base := "http://127.0.0.1:19009"

	// 1. create session
	sid := apiID(post(base+"/api/sessions", `{"title":"e2e"}`), "id")
	fmt.Println("1. Session:", sid)

	// 2. create flow WITHOUT edges first
	fid := apiID(post(base+"/api/flows", `{"name":"e2e","nodes":[
		{"id":1,"type":"start","label":"S","config":"{}","position_x":100,"position_y":50},
		{"id":2,"type":"llm","label":"G","config":"{\"model\":\"deepseek.default\",\"prompt\":\"One sentence:\\n{{user_input.output}}\",\"max_tokens\":200}","position_x":100,"position_y":130},
		{"id":3,"type":"end","label":"E","config":"{}","position_x":100,"position_y":210}
	]}`), "id")
	fmt.Println("2. Flow:", fid)

	// 3. read back real node IDs
	raw := get(base + "/api/flows/" + fid)
	var res struct {
		Data struct {
			Nodes []struct {
				Id   uint   `json:"id"`
				Type string `json:"type"`
			} `json:"nodes"`
		} `json:"data"`
	}
	sonic.Unmarshal([]byte(raw), &res)

	var sidN, lid, eid uint
	for _, n := range res.Data.Nodes {
		switch n.Type {
		case "start": sidN = n.Id
		case "llm": lid = n.Id
		case "end": eid = n.Id
		}
	}
	fmt.Printf("3. Real IDs: S=%d L=%d E=%d\n", sidN, lid, eid)

	// 4. update with real edge IDs
	edgeJSON := fmt.Sprintf(`{"name":"e2e","edges":[
		{"source_node_id":%d,"target_node_id":%d,"source_handle":"output","target_handle":"input"},
		{"source_node_id":%d,"target_node_id":%d,"source_handle":"output","target_handle":"input"}
	]}`, sidN, lid, lid, eid)
	put(base+"/api/flows/"+fid, edgeJSON)
	fmt.Println("4. Edges updated")

	// 5. create execution
	eid2 := apiID(post(base+"/api/flows/"+fid+"/execute", fmt.Sprintf(`{"session_id":%s}`, sid)), "execution_id")
	fmt.Println("5. Execution:", eid2)

	// 6. WebSocket execution
	time.Sleep(300 * time.Millisecond)
	conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:19009/ws/chat", nil)
	if err != nil { fmt.Println("WS err:", err); return }
	defer conn.Close()

	msg := map[string]interface{}{
		"type":       "flow_start",
		"session_id": mustF(sid),
		"model":      "deepseek.default",
		"stream":     true,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
		"options":    map[string]interface{}{"flow_id": mustF(fid), "execution_id": mustF(eid2)},
	}
	d, _ := sonic.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, d)
	fmt.Println("\n6. >>> Flow executing >>>\n")

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	for {
		_, m, err := conn.ReadMessage()
		if err != nil { fmt.Println("Done:", err); break }
		var ev map[string]interface{}
		sonic.Unmarshal(m, &ev)
		switch ev["type"] {
		case "flow_node_start":
			fmt.Printf("  [%v] start\n", ev["node_label"])
		case "flow_node_chunk":
			if c, ok := ev["content"].(string); ok { fmt.Print(c) }
		case "flow_node_done":
			fmt.Printf("\n  [%v] done %v\n", ev["node_label"], ev["content"])
		case "flow_complete":
			fmt.Println("\n7. >>> Flow complete! <<<")
			return
		case "flow_error":
			fmt.Printf("\nERR: %v\n", ev["message"])
			return
		}
	}
}

func post(url, body string) string {
	r, _ := http.Post(url, "application/json", bytes.NewBufferString(body))
	data, _ := io.ReadAll(r.Body)
	r.Body.Close()
	return string(data)
}

func get(url string) string {
	r, _ := http.Get(url)
	data, _ := io.ReadAll(r.Body)
	r.Body.Close()
	return string(data)
}

func put(url, body string) {
	req, _ := http.NewRequest("PUT", url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r, _ := http.DefaultClient.Do(req)
	r.Body.Close()
}

func apiID(raw, key string) string {
	var v map[string]interface{}
	sonic.Unmarshal([]byte(raw), &v)
	if d, ok := v["data"].(map[string]interface{}); ok {
		return fmt.Sprintf("%.0f", d[key].(float64))
	}
	return "0"
}

func mustF(s string) float64 {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}
