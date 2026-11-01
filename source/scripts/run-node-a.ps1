$ErrorActionPreference = "Stop"
Push-Location (Join-Path $PSScriptRoot "..")
try {
    New-Item -ItemType Directory -Force "bin" | Out-Null
    go build -o bin/node.exe ./cmd/node
    if ($LASTEXITCODE -ne 0) { throw "Node build failed; complete the weekly TODOs/setup." }
    & ./bin/node.exe -id "node-a" -http "127.0.0.1:8081" -seed "testdata/node-a-files.json" -tcp "127.0.0.1:7001" -udp "127.0.0.1:7101" -grpc "127.0.0.1:9091"
    if ($LASTEXITCODE -ne 0) { throw "Node exited with code $LASTEXITCODE" }
} finally {
    Pop-Location
}
