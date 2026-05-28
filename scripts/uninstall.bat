@echo off
setlocal enabledelayedexpansion

for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "BUILD_DIR=!ROOT!\build"

echo.
echo   EdgeBridge - Uninstall
echo.

:: Registry
set "REG_KEY=HKCU\Software\Microsoft\Edge\NativeMessagingHosts\com.edgebridge.host"
reg query "!REG_KEY!" >nul 2>nul
if !ERRORLEVEL! equ 0 (
    reg delete "!REG_KEY!" /f >nul
    echo   OK Registry key removed
) else (
    echo   - Registry key not found
)

:: User data
set "USER_DATA=!APPDATA!\EdgeBridge"
if exist "!USER_DATA!" (
    set /p CONFIRM="  Delete Edge profile at !USER_DATA!? (y/N): "
    if /i "!CONFIRM!"=="y" (
        rmdir /s /q "!USER_DATA!"
        echo   OK User data removed
    ) else (
        echo   - Skipped
    )
)

:: Build folder
if exist "!BUILD_DIR!" (
    set /p CONFIRM2="  Delete build/ folder? (y/N): "
    if /i "!CONFIRM2!"=="y" (
        rmdir /s /q "!BUILD_DIR!"
        echo   OK build/ removed
    ) else (
        echo   - Skipped
    )
)

echo.
echo   Uninstall complete.
echo   extensions/ and mcp-server/ preserved.
echo.
pause
