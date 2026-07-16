.PHONY: desktop-dev desktop-build web-build web-run

# ── Desktop ──────────────────────────────────────────────────────────

# Dev mode: start Go server + Vite frontend (open http://localhost:5173)
desktop-dev:
	@echo "Starting Go server on :19009 and Vite frontend on :5173 ..."
	@echo "Open http://localhost:5173 in your browser."
	@make -j2 _desktop-dev-server _desktop-dev-frontend

_desktop-dev-server:
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags "wails dev" -o go-ai-agent-desktop . && ./go-ai-agent-desktop

_desktop-dev-frontend:
	cd cmd/server/view && pnpm dev

desktop-build:
	cd cmd/server/view && pnpm build
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags "wails production" -o go-ai-agent-desktop .

# ── Server (web) ─────────────────────────────────────────────────────

server-build:
	cd cmd/server/view && pnpm build
	go build -o go-ai-agent-server.exe ./cmd/server/

server-run: server-build
	./go-ai-agent-server.exe

# ── Legacy aliases ────────────────────────────────────────────────────

web-build: server-build
web-run: server-run
