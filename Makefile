.PHONY: desktop-dev desktop-build web-build web-run

# ── Desktop ──────────────────────────────────────────────────────────

desktop-dev:
	wails dev

desktop-build:
	wails build

# ── Server (web) ─────────────────────────────────────────────────────

server-build:
	cd view && pnpm build
	go build -o go-ai-agent-server.exe ./cmd/server/

server-run: server-build
	./go-ai-agent-server.exe

# ── Legacy aliases ────────────────────────────────────────────────────

web-build: server-build
web-run: server-run
