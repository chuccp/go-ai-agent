@echo off
REM ─────────────────────────────────────────────────────────────────────────────
REM run.bat — Windows CMD launcher for go-ai-agent CLI
REM Usage:    run.bat
REM ─────────────────────────────────────────────────────────────────────────────
setlocal enabledelayedexpansion

echo.
echo [go-ai-agent CLI Launcher]
echo   OS: Windows
echo.

REM ── Preflight: Go ───────────────────────────────────────────────────────────
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go is not installed.
    echo   Install from: https://go.dev/dl/
    exit /b 1
)

for /f "tokens=3" %%v in ('go version') do echo [OK] Go %%v

REM ── Preflight: project root ─────────────────────────────────────────────────
if not exist "go.mod" (
    echo [ERROR] go.mod not found — are you in the project root?
    exit /b 1
)

echo [OK] Project root: %CD%

REM ── Run ─────────────────────────────────────────────────────────────────────
echo.
echo Starting CLI...
echo.

go run ./cmd/cli/

echo.
echo CLI exited.
endlocal
