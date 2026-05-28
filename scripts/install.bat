@echo off
chcp 65001 >nul 2>nul
setlocal enabledelayedexpansion

for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "MCP_DIR=!ROOT!\mcp-server"
set "BUILD_DIR=!ROOT!\build"
set "EXT_DIR=!ROOT!\extensions\Wikipedia-Assistant"

echo.
echo   ==================================================
echo     EdgeBridge  -  Installer v3.0
echo   ==================================================
echo.

:: [1/6] Pre-flight
echo [1/6] Pre-flight checks...

go version >nul 2>nul
if !ERRORLEVEL! neq 0 (
    echo   X Go not installed! https://go.dev/dl/
    goto :fail
)
for /f "delims=" %%v in ('go version') do echo   OK %%v

set "EDGE_EXE="
set "E1=!ProgramFiles(x86)!\Microsoft\Edge\Application\msedge.exe"
set "E2=!ProgramFiles!\Microsoft\Edge\Application\msedge.exe"
set "E3=!LOCALAPPDATA!\Microsoft\Edge\Application\msedge.exe"
if exist "!E1!" set "EDGE_EXE=!E1!"
if not defined EDGE_EXE if exist "!E2!" set "EDGE_EXE=!E2!"
if not defined EDGE_EXE if exist "!E3!" set "EDGE_EXE=!E3!"
if defined EDGE_EXE (echo   OK Edge found) else (echo   WARNING Edge not found)

if not exist "!MCP_DIR!" (
    echo   X mcp-server/ not found at !MCP_DIR!
    goto :fail
)
if not exist "!EXT_DIR!" (
    echo   X extensions/Wikipedia-Assistant/ not found
    goto :fail
)

:: [2/6] Build
echo.
echo [2/6] Building Go MCP server...

if not exist "!BUILD_DIR!" mkdir "!BUILD_DIR!"
set "BRIDGE_EXE=!BUILD_DIR!\bridge.exe"

pushd "!MCP_DIR!"
echo   go mod tidy...
go mod tidy >nul 2>nul
echo   Compiling bridge.exe...
go build -ldflags="-s -w" -o "!BRIDGE_EXE!" .
if !ERRORLEVEL! neq 0 (
    popd
    echo   X Build failed!
    goto :fail
)
popd
echo   OK bridge.exe built in build/

:: [3/6] Config
echo.
echo [3/6] Copying config.json...

set "CONFIG_DST=!BUILD_DIR!\config.json"
if not exist "!CONFIG_DST!" (
    if exist "!MCP_DIR!\config.json" (
        copy "!MCP_DIR!\config.json" "!CONFIG_DST!" >nul
        echo   OK config.json copied to build/
    ) else (
        echo   WARNING No config.json in mcp-server/
    )
) else (
    echo   OK config.json already exists
)

:: [4/6] Extension ID
echo.
echo [4/6] Extension ID...
echo.
echo   1. Open Edge - edge://extensions
echo   2. Developer Mode ON
echo   3. Load unpacked - select extensions/Wikipedia-Assistant/
echo   4. Copy the ID
echo.

set "EXT_ID="
set /p EXT_ID="  Extension ID (Enter to skip): "
if "!EXT_ID!"=="" (
    set "EXT_ID=YOUR_EXTENSION_ID_HERE"
    echo   WARNING Placeholder ID. Run update-extension-id.bat later.
) else (
    echo   OK ID: !EXT_ID!
)

:: [5/6] Native Messaging manifest
echo.
echo [5/6] Creating Native Messaging manifest...

set "MANIFEST=!BUILD_DIR!\com.edgebridge.host.json"
set "BPATH=!BRIDGE_EXE:\=\\!"

> "!MANIFEST!" (
    echo {
    echo   "name": "com.edgebridge.host",
    echo   "description": "EdgeBridge Native Messaging Host",
    echo   "path": "!BPATH!",
    echo   "type": "stdio",
    echo   "allowed_origins": ["chrome-extension://!EXT_ID!/"]
    echo }
)
echo   OK Created in build/

:: [6/6] Registry
echo.
echo [6/6] Registering Native Messaging host...

set "REG_KEY=HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.edgebridge.host"
reg add "!REG_KEY!" /ve /d "!MANIFEST!" /f >nul 2>nul
if !ERRORLEVEL! neq 0 (
    echo   X Registry failed. Try as Admin.
    goto :fail
)
echo   OK Registry key created

:: Done
echo.
echo   ==================================================
echo     Installation Complete!
echo   ==================================================
echo.
echo   build/bridge.exe              OK
echo   build/config.json             OK
echo   build/com.edgebridge.host.json OK
echo   Registry                      OK
echo.
echo   Next: load extensions/Wikipedia-Assistant/ in Edge
echo   Then run launch.bat
echo.
goto :done

:fail
echo.
echo   FAILED. Fix errors above and retry.
echo.

:done
pause
