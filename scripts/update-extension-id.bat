@echo off
setlocal enabledelayedexpansion

for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "BUILD_DIR=!ROOT!\build"
set "MANIFEST=!BUILD_DIR!\com.edgebridge.host.json"
set "BRIDGE_EXE=!BUILD_DIR!\bridge.exe"
set "BPATH=!BRIDGE_EXE:\=\\!"

if not exist "!MANIFEST!" (
    echo   X Manifest not found. Run install.bat first.
    pause
    exit /b 1
)

set /p EXT_ID="Enter extension ID: "
if "!EXT_ID!"=="" goto :eof

> "!MANIFEST!" (
    echo {
    echo   "name": "com.edgebridge.host",
    echo   "description": "EdgeBridge Native Messaging Host",
    echo   "path": "!BPATH!",
    echo   "type": "stdio",
    echo   "allowed_origins": ["chrome-extension://!EXT_ID!/"]
    echo }
)

reg add "HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.edgebridge.host" /ve /d "!MANIFEST!" /f >nul
echo   OK Updated! Restart Edge.
pause
