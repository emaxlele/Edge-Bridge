@echo off
chcp 65001 >nul 2>nul
setlocal enabledelayedexpansion

for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "MCP_DIR=!ROOT!\mcp-server"
set "LAUNCHER_DIR=!ROOT!\launcher"
set "ASSETS_DIR=!LAUNCHER_DIR!\assets"
set "BUILD_DIR=!ROOT!\build"

echo.
echo   ==================================================
echo     EdgeBridge  -  Build v6.0
echo     Fully automatic - no extension ID needed!
echo   ==================================================
echo.

:: [1/10] Pre-flight
echo [1/10] Pre-flight checks...

go version >nul 2>nul
if !ERRORLEVEL! neq 0 (
    echo   X Go not installed! https://go.dev/dl/
    goto :fail
)
for /f "delims=" %%v in ('go version') do echo   OK %%v

where goversioninfo >nul 2>nul
if !ERRORLEVEL! neq 0 (
    echo   - goversioninfo not found, installing...
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    if !ERRORLEVEL! neq 0 (
        echo   X Failed to install goversioninfo
        goto :fail
    )
    echo   OK goversioninfo installed
) else (
    echo   OK goversioninfo found
)

set "WIKIPEDIA_ICON=!LAUNCHER_DIR!\wikipedia.ico"
set "EDGE_ICON=!LAUNCHER_DIR!\edge.ico"

if exist "!WIKIPEDIA_ICON!" (
    echo   OK wikipedia.ico found
    set "HAS_WIKIPEDIA_ICON=1"
) else (
    echo   - wikipedia.ico not found, Wikipedia builds without icon
    set "HAS_WIKIPEDIA_ICON=0"
)

if exist "!EDGE_ICON!" (
    echo   OK edge.ico found
    set "HAS_EDGE_ICON=1"
) else (
    echo   - edge.ico not found, Edge builds without icon
    set "HAS_EDGE_ICON=0"
)

if not exist "!MCP_DIR!" (echo   X mcp-server/ not found & goto :fail)
if not exist "!LAUNCHER_DIR!\main.go" (echo   X launcher/main.go not found & goto :fail)

set "EXT_COUNT=0"
for /d %%D in ("!ROOT!\extensions\*") do (
    if exist "%%D\manifest.json" set /a EXT_COUNT+=1
)
if !EXT_COUNT! equ 0 (
    echo   - No extensions found in extensions/ (continuing without extensions)
) else (
    echo   OK !EXT_COUNT! extension(s) found
)

:: [2/10] Clean + Build bridge.exe
echo.
echo [2/10] Building bridge.exe (MCP server)...

if exist "!BUILD_DIR!" (
    echo   Cleaning previous build...
    taskkill /F /IM WikipediaBridge.exe >nul 2>nul
    taskkill /F /IM WikipediaBridge-debug.exe >nul 2>nul
    taskkill /F /IM EdgeBridge.exe >nul 2>nul
    taskkill /F /IM EdgeBridge-debug.exe >nul 2>nul
    taskkill /F /IM bridge.exe >nul 2>nul
    timeout /t 1 /nobreak >nul
    rmdir /s /q "!BUILD_DIR!"
)
mkdir "!BUILD_DIR!"
mkdir "!BUILD_DIR!\Wikipedia"
mkdir "!BUILD_DIR!\Edge"

pushd "!MCP_DIR!"
go mod tidy >nul 2>nul
go build -ldflags="-s -w" -o "!BUILD_DIR!\bridge.exe" .
if !ERRORLEVEL! neq 0 (popd & echo   X bridge.exe failed! & goto :fail)
popd
echo   OK bridge.exe

:: [3/10] Stage assets for embedding
echo.
echo [3/10] Staging assets...

if exist "!ASSETS_DIR!" rmdir /s /q "!ASSETS_DIR!"
mkdir "!ASSETS_DIR!"

copy "!BUILD_DIR!\bridge.exe" "!ASSETS_DIR!\bridge.exe" >nul
if exist "!MCP_DIR!\config.json" copy "!MCP_DIR!\config.json" "!ASSETS_DIR!\config.json" >nul

echo   Copying extensions...
for /d %%D in ("!ROOT!\extensions\*") do (
    if exist "%%D\manifest.json" (
        mkdir "!ASSETS_DIR!\extensions\%%~nxD"
        xcopy "%%D\*" "!ASSETS_DIR!\extensions\%%~nxD\" /E /Y /Q >nul
        echo     + %%~nxD
    )
)
echo   OK assets staged

:: Generate a unique per-run build tag so the linker always runs fresh
for /f "usebackq" %%t in (`powershell -NoProfile -Command "(Get-Date).Ticks"`) do set "BUILD_TS=%%t"

:: [4/10] Generate icon resource + prepare isolated build dir
echo.
echo [4/10] Generating Wikipedia icon resource...

if exist "!LAUNCHER_DIR!\resource_windows.syso" del "!LAUNCHER_DIR!\resource_windows.syso"
set "WIKIPEDIA_RES_HASH="

if !HAS_WIKIPEDIA_ICON! equ 1 (
    pushd "!LAUNCHER_DIR!"
    goversioninfo -icon="wikipedia.ico" -o="resource_windows.syso" "versioninfo-wikipedia.json"
    if !ERRORLEVEL! neq 0 (
        popd
        echo   ! Wikipedia icon embedding failed, continuing without icon
        set "HAS_WIKIPEDIA_ICON=0"
    ) else (
        popd
        echo   OK Wikipedia resource_windows.syso generated
    )
) else (
    echo   - Skipped (no wikipedia.ico)
)

if exist "!LAUNCHER_DIR!\resource_windows.syso" (
    for /f "usebackq" %%h in (`powershell -NoProfile -Command "(Get-FileHash '!LAUNCHER_DIR!\resource_windows.syso' -Algorithm MD5).Hash"`) do set "WIKIPEDIA_RES_HASH=%%h"
    echo   OK icon hash: !WIKIPEDIA_RES_HASH!
) else (
    set "WIKIPEDIA_RES_HASH=noicon"
)

:: Create an isolated Wikipedia build directory with its own module name.
:: This gives Go a fresh package identity and prevents the linker from
:: reusing any cached binary that was built without the icon syso.
set "WIKIPEDIA_TMP=!BUILD_DIR!\tmp_wikipedia"
mkdir "!WIKIPEDIA_TMP!"
copy "!LAUNCHER_DIR!\main.go" "!WIKIPEDIA_TMP!\" >nul
echo module launcher-wikipedia>"!WIKIPEDIA_TMP!\go.mod"
echo.>>"!WIKIPEDIA_TMP!\go.mod"
echo go 1.21>>"!WIKIPEDIA_TMP!\go.mod"
type nul>"!WIKIPEDIA_TMP!\go.sum"
mkdir "!WIKIPEDIA_TMP!\assets"
xcopy "!ASSETS_DIR!\*" "!WIKIPEDIA_TMP!\assets\" /E /Y /Q >nul
if exist "!LAUNCHER_DIR!\resource_windows.syso" copy "!LAUNCHER_DIR!\resource_windows.syso" "!WIKIPEDIA_TMP!\" >nul
echo   OK Wikipedia isolated build dir ready

:: [5/10] Build WikipediaBridge.exe
echo.
echo [5/10] Building WikipediaBridge.exe (app mode - production)...

pushd "!WIKIPEDIA_TMP!"
go build -ldflags="-s -w -H windowsgui -X main.debugMode=false -X main.appMode=true -X main.buildResourceHash=!WIKIPEDIA_RES_HASH!_!BUILD_TS!" -o "!BUILD_DIR!\Wikipedia\WikipediaBridge.exe" .
if !ERRORLEVEL! neq 0 (popd & echo   X WikipediaBridge build failed! & goto :fail)
popd
echo   OK WikipediaBridge.exe

:: [6/10] Build WikipediaBridge-debug.exe
echo.
echo [6/10] Building WikipediaBridge-debug.exe (app mode - debug)...

pushd "!WIKIPEDIA_TMP!"
go build -ldflags="-s -w -X main.debugMode=true -X main.appMode=true -X main.buildResourceHash=!WIKIPEDIA_RES_HASH!_!BUILD_TS!" -o "!BUILD_DIR!\Wikipedia\WikipediaBridge-debug.exe" .
if !ERRORLEVEL! neq 0 (popd & echo   X WikipediaBridge debug build failed! & goto :fail)
popd
echo   OK WikipediaBridge-debug.exe

:: [7/10] Generate Edge icon resource + prepare isolated build dir
echo.
echo [7/10] Generating Edge icon resource...

if exist "!LAUNCHER_DIR!\resource_windows.syso" del "!LAUNCHER_DIR!\resource_windows.syso"

if !HAS_EDGE_ICON! equ 1 (
    pushd "!LAUNCHER_DIR!"
    goversioninfo -icon="edge.ico" -o="resource_windows.syso" "versioninfo-edge.json"
    if !ERRORLEVEL! neq 0 (
        popd
        echo   ! Edge icon embedding failed, continuing without icon
        set "HAS_EDGE_ICON=0"
    ) else (
        popd
        echo   OK Edge resource_windows.syso generated
    )
) else (
    echo   - Skipped (no edge.ico)
)

set "EDGE_RES_HASH="
if exist "!LAUNCHER_DIR!\resource_windows.syso" (
    for /f "usebackq" %%h in (`powershell -NoProfile -Command "(Get-FileHash '!LAUNCHER_DIR!\resource_windows.syso' -Algorithm MD5).Hash"`) do set "EDGE_RES_HASH=%%h"
    echo   OK icon hash: !EDGE_RES_HASH!
) else (
    set "EDGE_RES_HASH=noicon"
)

:: Create an isolated Edge build directory
set "EDGE_TMP=!BUILD_DIR!\tmp_edge"
mkdir "!EDGE_TMP!"
copy "!LAUNCHER_DIR!\main.go" "!EDGE_TMP!\" >nul
echo module launcher-edge>"!EDGE_TMP!\go.mod"
echo.>>"!EDGE_TMP!\go.mod"
echo go 1.21>>"!EDGE_TMP!\go.mod"
type nul>"!EDGE_TMP!\go.sum"
mkdir "!EDGE_TMP!\assets"
xcopy "!ASSETS_DIR!\*" "!EDGE_TMP!\assets\" /E /Y /Q >nul
if exist "!LAUNCHER_DIR!\resource_windows.syso" copy "!LAUNCHER_DIR!\resource_windows.syso" "!EDGE_TMP!\" >nul
echo   OK Edge isolated build dir ready

:: [8/10] Build EdgeBridge.exe
echo.
echo [8/10] Building EdgeBridge.exe (browser mode - production)...

pushd "!EDGE_TMP!"
go build -ldflags="-s -w -H windowsgui -X main.debugMode=false -X main.appMode=false -X main.buildResourceHash=!EDGE_RES_HASH!_!BUILD_TS!" -o "!BUILD_DIR!\Edge\EdgeBridge.exe" .
if !ERRORLEVEL! neq 0 (popd & echo   X EdgeBridge build failed! & goto :fail)
popd
echo   OK EdgeBridge.exe

:: [9/10] Build EdgeBridge-debug.exe
echo.
echo [9/10] Building EdgeBridge-debug.exe (browser mode - debug)...

pushd "!EDGE_TMP!"
go build -ldflags="-s -w -X main.debugMode=true -X main.appMode=false -X main.buildResourceHash=!EDGE_RES_HASH!_!BUILD_TS!" -o "!BUILD_DIR!\Edge\EdgeBridge-debug.exe" .
if !ERRORLEVEL! neq 0 (popd & echo   X EdgeBridge debug build failed! & goto :fail)
popd
echo   OK EdgeBridge-debug.exe

:: [10/10] Cleanup
echo.
echo [10/10] Cleanup...
rmdir /s /q "!ASSETS_DIR!"
if exist "!WIKIPEDIA_TMP!" rmdir /s /q "!WIKIPEDIA_TMP!"
if exist "!EDGE_TMP!" rmdir /s /q "!EDGE_TMP!"
if exist "!LAUNCHER_DIR!\resource_windows.syso" del "!LAUNCHER_DIR!\resource_windows.syso"
echo   OK cleaned

:: Done
echo.
echo   ==================================================
echo     Build Complete!
echo   ==================================================
echo.
echo   build/
echo     Wikipedia/
echo       WikipediaBridge.exe          App mode
echo       WikipediaBridge-debug.exe    App mode debug
echo     Edge/
echo       EdgeBridge.exe          Clean browser
echo       EdgeBridge-debug.exe    Browser debug
echo     bridge.exe                Standalone MCP (testing)
echo.
echo   First run creates extension-key.txt next to exe.
echo   Keep it - it ensures the same extension ID every time.
echo.
goto :done

:fail
echo.
echo   FAILED. Fix errors above and retry.
echo.

:done
pause
