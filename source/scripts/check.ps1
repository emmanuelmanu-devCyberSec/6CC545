$ErrorActionPreference = "Stop"
Push-Location (Join-Path $PSScriptRoot "..")
try {
    go version
    go build ./...
    if ($LASTEXITCODE -ne 0) { throw "Build failed" }
    go test ./... -count=1 -timeout 20s
    if ($LASTEXITCODE -ne 0) { throw "Tests failed" }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "Vet failed" }
    Write-Host "Checks passed. This is not a mark or proof of complete test coverage."
} finally { Pop-Location }
