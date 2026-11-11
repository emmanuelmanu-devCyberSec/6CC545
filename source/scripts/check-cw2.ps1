param([ValidateSet("smoke","acceptance","race")][string]$Mode="smoke")
$ErrorActionPreference="Stop"
go version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go env GOOS GOARCH CGO_ENABLED CC GOMOD
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
$packages=@("./internal/events","./internal/storage")
if ($Mode -eq "smoke") { & go test @packages -count=1 }
elseif ($Mode -eq "acceptance") { & go test -tags acceptance @packages -count=1 -timeout 30s }
else { & go test -race -tags acceptance @packages -count=1 -timeout 60s }
exit $LASTEXITCODE