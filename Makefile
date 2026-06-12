.PHONY: desktop-dev desktop-build desktop-build-mac desktop-build-win desktop-build-linux desktop-dmg

# ── Dev ─────────────────────────────────────────────────────────────

desktop-dev:
	cd view && pnpm install
	wails dev

# ── Build ───────────────────────────────────────────────────────────

desktop-build:
	wails build

desktop-build-mac:
	wails build -platform darwin/universal

desktop-build-mac-arm:
	wails build -platform darwin/arm64

desktop-build-mac-amd:
	wails build -platform darwin/amd64

desktop-build-win:
	wails build -platform windows/amd64

desktop-build-linux:
	wails build -platform linux/amd64

# ── Package ─────────────────────────────────────────────────────────

desktop-dmg: desktop-build-mac
	@echo "Creating DMG..."
	hdiutil create -volname "Go AI Agent" -srcfolder "build/bin/Go AI Agent.app" -ov -format UDZO "build/bin/go-ai-agent.dmg" 2>/dev/null || \
	create-dmg "build/bin/Go AI Agent.app" "build/bin/" 2>/dev/null || \
	echo "Install 'create-dmg' (brew install create-dmg) for nicer DMG"

# ── Web (existing) ──────────────────────────────────────────────────

web-build:
	cd view && pnpm build
	go build -tags web -o go-ai-agent .

web-run: web-build
	./go-ai-agent -web
