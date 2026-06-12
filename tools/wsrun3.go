//go:build ignore
package main
import ("github.com/bytedance/sonic";"fmt";"time";"github.com/gorilla/websocket")
func main() {
	conn,_,_:=websocket.DefaultDialer.Dial("ws://127.0.0.1:19009/ws/chat",nil);defer conn.Close()
	msg:=map[string]interface{}{"type":"flow_start","session_id":7.0,"model":"deepseek.default","stream":true,"messages":[]map[string]string{{"role":"user","content":"Bunny looks for mom"}},"options":map[string]interface{}{"flow_id":16.0,"execution_id":10.0}}
	d,_:=sonic.Marshal(msg);conn.WriteMessage(websocket.TextMessage,d)
	conn.SetReadDeadline(time.Now().Add(120*time.Second))
	for{_,m,err:=conn.ReadMessage();if err!=nil{fmt.Println("\nend:",err);break}
	var ev map[string]interface{};sonic.Unmarshal(m,&ev)
	switch ev["type"]{
	case "flow_node_start":fmt.Printf("\n[%v]",ev["node_label"])
	case "flow_node_chunk":if c,ok:=ev["content"].(string);ok{fmt.Print(c)}
	case "flow_node_done":fmt.Printf(" (%v)",ev["content"])
	case "flow_complete":fmt.Println("\n\nDONE!");return
	case "flow_error":fmt.Printf("\nERR:%v\n",ev["message"]);return
	}}
}
