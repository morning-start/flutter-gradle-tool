@echo off
REM build.bat — Build fgt for the current platform
REM Usage: build.bat [version]
REM   version defaults to "dev"

setlocal
set VERSION=%1
if "%VERSION%"=="" set VERSION=dev

echo Building fgt %VERSION% for Windows...
go build -ldflags "-X main.version=%VERSION%" -o fgt.exe ./cmd/fgt/...
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%

echo Done: fgt.exe
