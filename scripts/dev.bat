@echo off
cd /d "%~dp0.."
echo === Killing old processes ===
taskkill /f /im go-ai-agent.exe >nul 2>&1
taskkill /f /im go-ai-agent-desktop.exe >nul 2>&1
echo === Starting Wails desktop dev mode ===
wails dev
pause
