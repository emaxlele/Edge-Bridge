@echo off
setlocal enabledelayedexpansion

for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "USER_DATA=!APPDATA!\EdgeBridge"
set "URL=https://www.wikipedia.org/"

:: Auto-detect all extension subfolders that contain manifest.json and join them with commas
set "EXT_DIRS="
for /d %%D in ("!ROOT!\extensions\*") do (
    if exist "%%D\manifest.json" (
        if defined EXT_DIRS (
            set "EXT_DIRS=!EXT_DIRS!,%%~fD"
        ) else (
            set "EXT_DIRS=%%~fD"
        )
    )
)

:: Allow launching even if no extensions found. Warning is shown only by debug EXE binaries.
if not defined EXT_DIRS (
    set "NO_EXT=1"
    set "EXT_DIRS="
)

set "EDGE_EXE="
set "E1=!ProgramFiles(x86)!\Microsoft\Edge\Application\msedge.exe"
set "E2=!ProgramFiles!\Microsoft\Edge\Application\msedge.exe"
set "E3=!LOCALAPPDATA!\Microsoft\Edge\Application\msedge.exe"
if exist "!E1!" set "EDGE_EXE=!E1!"
if not defined EDGE_EXE if exist "!E2!" set "EDGE_EXE=!E2!"
if not defined EDGE_EXE if exist "!E3!" set "EDGE_EXE=!E3!"

if not defined EDGE_EXE (
    echo   X Edge not found!
    pause
    exit /b 1
)

echo   Launching EdgeBridge...
if defined NO_EXT (
    echo   Extensions: (none)
) else (
    echo   Extensions: !EXT_DIRS!
)
echo   URL: !URL!

if defined EXT_DIRS (
    set "LOAD_EXT=--load-extension=!EXT_DIRS!"
) else (
    set "LOAD_EXT="
)

start "" "!EDGE_EXE!" --app="!URL!" --user-data-dir="!USER_DATA!" %LOAD_EXT% --no-first-run --disable-default-apps

echo   OK Edge launched!
echo   Ctrl+Shift+P to open panel
