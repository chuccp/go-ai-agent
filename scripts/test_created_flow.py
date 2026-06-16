#!/usr/bin/env python3
import json
import sqlite3
import sys
import websocket

URL = "ws://localhost:19009/ws/chat"
DB_PATH = "build/bin/go-ai-agent.app/Contents/MacOS/data/go-ai-agent.db"
SESSION_ID = 2
STORY_PROMPT = "在月球上发现了一扇门"


def latest_flow_id():
    conn = sqlite3.connect(DB_PATH)
    cur = conn.cursor()
    cur.execute("SELECT id FROM flow_definitions ORDER BY id DESC LIMIT 1")
    row = cur.fetchone()
    conn.close()
    if row is None:
        raise RuntimeError("no flow found in database")
    return row[0]


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
    flow_id = int(sys.argv[1]) if len(sys.argv) > 1 else latest_flow_id()
    print(f">>> Running created flow id={flow_id}")

    ws = websocket.create_connection(URL)
    send(ws, "flow_start", {"flow_id": flow_id})

    story_parts = []
    execution_id = None

    while True:
        evt = recv_event(ws)
        if evt is None:
            print("[TIMEOUT]")
            break

        t = evt.get("type")
        if t == "flow_started":
            print(f"[FLOW_STARTED] {evt}")
        elif t == "flow_node_start":
            print(f"[NODE_START] {evt.get('node_label')} ({evt.get('node_type')})")
        elif t == "flow_node_chunk":
            story_parts.append(evt.get("content", ""))
        elif t == "flow_node_done":
            print(f"[NODE_DONE] {evt.get('node_label')} status={evt.get('status')}")
        elif t == "flow_waiting_user":
            execution_id = evt.get("execution_id")
            print(f"[WAITING_USER] exec={execution_id} prompt={evt.get('message')}")
            print(f">>> Auto-reply: {STORY_PROMPT}")
            send(ws, "flow_user_response", {
                "execution_id": execution_id,
                "response": STORY_PROMPT,
            })
        elif t == "flow_complete":
            print(f"[FLOW_COMPLETE]")
            print("\n=== Generated Story ===")
            print("".join(story_parts))
            break
        elif t == "flow_error":
            print(f"[FLOW_ERROR] {evt}")
            break
        elif t == "error":
            print(f"[ERROR] {evt}")
            break

    ws.close()


if __name__ == "__main__":
    main()
