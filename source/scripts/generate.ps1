$ErrorActionPreference = "Stop"
Push-Location (Join-Path $PSScriptRoot "..")
$oldGoBin = $env:GOBIN
$oldPath = $env:PATH
try {
    if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) { throw "Install the module-approved protoc compiler first." }
    $env:GOBIN = Join-Path (Get-Location) "bin/tools"
    New-Item -ItemType Directory -Force $env:GOBIN | Out-Null
    $env:PATH = "$($env:GOBIN);$env:PATH"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
    if ($LASTEXITCODE -ne 0) { throw "protoc-gen-go installation failed" }
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
    if ($LASTEXITCODE -ne 0) { throw "protoc-gen-go-grpc installation failed" }
    protoc --version
    protoc-gen-go --version
    protoc-gen-go-grpc --version
    go generate ./api/files/v1
    if ($LASTEXITCODE -ne 0) { throw "Protocol generation failed" }
    go mod tidy
    if ($LASTEXITCODE -ne 0) { throw "Dependency resolution failed" }
    go mod verify
    if ($LASTEXITCODE -ne 0) { throw "Dependency verification failed" }
    Write-Host "Bindings generated. Complete TODOs, then run scripts/check.ps1."
} finally {
    $env:GOBIN = $oldGoBin
    $env:PATH = $oldPath
    Pop-Location
}
