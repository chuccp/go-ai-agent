#!/usr/bin/env python3
import json
import time
import websocket

URL = "ws://localhost:19009/ws/chat"
SESSION_ID = 1


def send_agent(ws, text):
    req = {
        "type": "agent",
        "session_id": SESSION_ID,
        "model": "",
        "messages": [{"role": "user", "content": text}],
        "stream": True,
        "options": {},
    }
    ws.send(json.dumps(req))
    print(f"\n>>> User: {text}\n")


def receive_turn(ws):
    """Receive one complete agent turn (chunks until done + any tool events)."""
    chunks = []
    done = False
    while not done:
        msg = ws.recv()
        if not msg:
            break
        data = json.loads(msg)
        t = data.get("type")
        if t == "chunk":
            chunks.append(data.get("content", ""))
            if data.get("done"):
                done = True
        elif t == "tool_call":
            print(f"[TOOL_CALL] {data.get('message', '')}")
        elif t == "tool_result":
            print(f"[TOOL_RESULT] {data.get('message', '')}")
        elif t == "error":
            print(f"[ERROR] {data.get('message', '')}")
            done = True
    reply = "".join(chunks).strip()
    if reply:
        print(f"<<< Agent: {reply}\n")
    return reply


def main():
    ws = websocket.create_connection(URL)

    turns = [
        "我想创建一个简单的 flow，输入一句话就能生成一个故事。",
        "输入就是一句话，输出是一篇 300 字左右的短篇故事。",
        "用当前可用的 deepseek 模型就行。",
        "prompt 就写：请根据下面这句话，展开成一个富有想象力的短篇故事（300字左右）。",
        "可以，按你说的创建吧。",
    ]

    for user_msg in turns:
        send_agent(ws, user_msg)
        time.sleep(0.5)
        receive_turn(ws)

    ws.close()


if __name__ == "__main__":
    main()
