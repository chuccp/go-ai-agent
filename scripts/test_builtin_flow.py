#!/usr/bin/env python3
import json
import time
import websocket

URL = "ws://localhost:19009/ws/chat"
SESSION_ID = 1


def send(ws, msg_type, options=None, messages=None):
    req = {
        "type": msg_type,
        "session_id": SESSION_ID,
        "model": "",
        "messages": messages or [],
        "stream": True,
        "options": options or {},
    }
    ws.send(json.dumps(req))


def recv_event(ws, timeout=60):
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

    # Start the built-in create-flow flow with initial description
    print(">>> Starting built-in create_flow flow")
    send(ws, "flow_start", {
        "builtin_flow": "create_flow",
    }, messages=[{"role": "user", "content": "创建一个输入一句话就能生成故事的流程"}])

    execution_id = None
    replies = [
        "输入是一句话，输出是一篇300字的短篇故事",
        "用 deepseek-v4-flash 模型",
        "READY，请开始创建",
    ]
    reply_idx = 0

    while True:
        evt = recv_event(ws, timeout=120)
        if evt is None:
            print("[TIMEOUT]")
            break

        t = evt.get("type")
        if t == "flow_started":
            print(f"[FLOW_STARTED] {evt}")
        elif t == "flow_node_start":
            print(f"[NODE_START] {evt.get('node_label')} ({evt.get('node_type')})")
        elif t == "flow_node_done":
            print(f"[NODE_DONE] {evt.get('node_label')} status={evt.get('status')}")
        elif t == "flow_waiting_user":
            waiting_prompt = evt.get("message")
            execution_id = evt.get("execution_id")
            print(f"[WAITING_USER] exec={execution_id} prompt={waiting_prompt}")
            if reply_idx < len(replies):
                answer = replies[reply_idx]
                reply_idx += 1
                print(f">>> Auto-reply: {answer}")
                send(ws, "flow_user_response", {
                    "execution_id": execution_id,
                    "response": answer,
                })
            else:
                print("No more auto-replies")
                break
        elif t == "flow_complete":
            print(f"[FLOW_COMPLETE] {evt}")
            break
        elif t == "flow_error":
            print(f"[FLOW_ERROR] {evt}")
            break
        elif t == "error":
            print(f"[ERROR] {evt}")
            break
        else:
            print(f"[OTHER] {evt}")

    ws.close()


if __name__ == "__main__":
    main()
