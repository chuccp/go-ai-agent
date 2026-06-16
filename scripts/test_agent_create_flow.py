#!/usr/bin/env python3
import json
import websocket

URL = "ws://localhost:19009/ws/chat"
SESSION_ID = 0


def send(ws, msg_type, messages=None, options=None):
    req = {
        "type": msg_type,
        "session_id": SESSION_ID,
        "model": "",
        "messages": messages or [],
        "stream": True,
        "options": options or {},
    }
    ws.send(json.dumps(req))


def recv_event(ws, timeout=120):
    ws.settimeout(timeout)
    try:
        msg = ws.recv()
        if not msg:
            return None
        return json.loads(msg)
    except websocket.WebSocketTimeoutException:
        return None


def main():
    ws = websocket.create_connection(URL)
    print(">>> Sending agent request to create flow via conversation")
    send(ws, "agent", messages=[{
        "role": "user",
        "content": "帮我创建一个输入一句话就能生成故事的流程",
    }])

    execution_id = None
    # Auto-replies to the built-in flow's questions.
    flow_replies = [
        "输入是一句话，输出是一篇300字的短篇故事",
        "用 deepseek-v4-flash 模型",
        "READY，请开始创建",
    ]
    reply_idx = 0

    while True:
        evt = recv_event(ws)
        if evt is None:
            print("[TIMEOUT]")
            break

        t = evt.get("type")
        if t == "chunk":
            if evt.get("done"):
                print("\n[CHUNK DONE]")
                break
            content = evt.get("content", "")
            if content:
                print(content, end="", flush=True)
        elif t == "tool_call":
            print(f"\n[TOOL_CALL] {evt.get('message')}")
        elif t == "tool_result":
            print(f"[TOOL_RESULT] {evt.get('message')}")
        elif t == "error":
            print(f"\n[ERROR] {evt}")
            break
        elif t in ("flow_started", "flow_node_start", "flow_node_done", "flow_complete", "flow_error"):
            print(f"[FLOW_EVENT] {evt}")
        elif t == "flow_waiting_user":
            execution_id = evt.get("execution_id")
            question = evt.get("message", "")
            print(f"\n[WAITING_USER] exec={execution_id} prompt={question}")
            if reply_idx < len(flow_replies):
                answer = flow_replies[reply_idx]
                reply_idx += 1
                print(f">>> Auto-reply: {answer}")
                send(ws, "flow_user_response", options={
                    "execution_id": execution_id,
                    "response": answer,
                })
            else:
                print("No more auto-replies")
                break

    ws.close()


if __name__ == "__main__":
    main()
