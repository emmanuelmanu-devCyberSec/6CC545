$ErrorActionPreference = "Stop"
Push-Location (Join-Path $PSScriptRoot "..")
try {
    New-Item -ItemType Directory -Force "bin" | Out-Null
    go build -o bin/node.exe ./cmd/node
    if ($LASTEXITCODE -ne 0) { throw "Node build failed; complete the weekly TODOs/setup." }
    & ./bin/node.exe -id "node-b" -http "127.0.0.1:8082" -seed "testdata/node-b-files.json" -tcp "127.0.0.1:7002" -udp "127.0.0.1:7102" -grpc "127.0.0.1:9092"
    if ($LASTEXITCODE -ne 0) { throw "Node exited with code $LASTEXITCODE" }
} finally {
    Pop-Location
}
