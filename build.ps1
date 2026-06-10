# build.ps1 — Build fgt for the current platform
# Usage: .\build.ps1 [[-Version] <string>]
#   Version defaults to "dev"

param(
    [string]$Version = "dev"
)

Write-Host "Building fgt $Version for Windows..." -ForegroundColor Cyan

go build -ldflags "-X main.version=$Version" -o fgt.exe ./cmd/fgt/...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed with exit code $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}

Write-Host "Done: fgt.exe" -ForegroundColor Green
