@echo off
title Factory Lab exe build
cd /d "%~dp0"

where go >nul 2>nul
if not %errorlevel%==0 (
  echo Go is not installed. Install it from https://go.dev/dl/ then run this again.
  pause
  exit /b 1
)

set STAGE=%TEMP%\factorylab-exe
if exist "%STAGE%" rmdir /s /q "%STAGE%"
mkdir "%STAGE%\site"

rem copy site files only (exclude repo/deploy stuff)
robocopy . "%STAGE%\site" /e /xd .git .github dist tools /xf deploy-exe.bat *.zip *.exe .gitignore .assetsignore wrangler.toml README.md >nul

copy /y tools\portable\main.go "%STAGE%" >nul
copy /y tools\portable\go.mod "%STAGE%" >nul

if not exist dist mkdir dist

rem embed icon + version info (unsigned exes with blank version fields get flagged)
copy /y tools\portable\icon.ico "%STAGE%" >nul
copy /y tools\portable\versioninfo.json "%STAGE%" >nul
where goversioninfo >nul 2>nul
if not %errorlevel%==0 go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0
pushd "%STAGE%"
goversioninfo -64 -o resource.syso versioninfo.json
popd

pushd "%STAGE%"
go build -ldflags="-s -w" -o "%~dp0dist\FactoryLab.exe" .
set ERR=%errorlevel%
popd
rmdir /s /q "%STAGE%"

if %ERR%==0 (
  echo Done: dist\FactoryLab.exe
  echo Copy this single file to any Windows PC and double-click.
) else (
  echo Build failed
)
pause
