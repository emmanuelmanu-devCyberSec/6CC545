# 6CC545 CW1 - Week 01 student wireframe

This is a cumulative scaffold, not a completed solution. Read the supplied weekly TASKS.md and the weekly workbook. Earlier TODOs remain deliberately unimplemented. Merge new files into your own repository; never overwrite completed earlier work.

## Build status

The source compiles, but assessment tests intentionally fail until TODOs are implemented.

```text
go build ./...
go test ./... -run "^TestSmoke_"
go test ./... -count=1 -timeout 20s
go vet ./...
```

Green smoke tests only validate supplied infrastructure. They are not checkpoint completion. Do not skip, delete or weaken acceptance tests. Add your own tests. Use a fresh supported toolchain approved by the module team; the go.mod language compatibility baseline is Go 1.23.

## Safety

Use loopback and synthetic test data. These CW1 services have no TLS or authentication and must not be exposed to other networks.

## Start nodes after completing TODOs

Bash: `sh scripts/run-node-a.sh` and, in another terminal, `sh scripts/run-node-b.sh`.
PowerShell: `./scripts/run-node-a.ps1` and `./scripts/run-node-b.ps1`.
The scripts rebuild the executable and start it in the foreground. Use Ctrl+C to stop it.
HTTP addresses: A `127.0.0.1:8081`, B `127.0.0.1:8082`.

# 6CC545 CW1 - Week 02 student wireframe

This is a cumulative scaffold, not a completed solution. Read the supplied weekly TASKS.md and the weekly workbook. Earlier TODOs remain deliberately unimplemented. Merge new files into your own repository; never overwrite completed earlier work.

## Build status

The source compiles, but assessment tests intentionally fail until TODOs are implemented.

```text
go build ./...
go test ./... -run "^TestSmoke_"
go test ./... -count=1 -timeout 20s
go vet ./...
```

Green smoke tests only validate supplied infrastructure. They are not checkpoint completion. Do not skip, delete or weaken acceptance tests. Add your own tests. Use a fresh supported toolchain approved by the module team; the go.mod language compatibility baseline is Go 1.23.

## Safety

Use loopback and synthetic test data. These CW1 services have no TLS or authentication and must not be exposed to other networks.

## Start nodes after completing TODOs

Bash: `sh scripts/run-node-a.sh` and, in another terminal, `sh scripts/run-node-b.sh`.
PowerShell: `./scripts/run-node-a.ps1` and `./scripts/run-node-b.ps1`.
The scripts rebuild the executable and start it in the foreground. Use Ctrl+C to stop it.
HTTP addresses: A `127.0.0.1:8081`, B `127.0.0.1:8082`.

# 6CC545 CW1 - Week 03 student wireframe

This is a cumulative scaffold, not a completed solution. Read the supplied weekly TASKS.md and the weekly workbook. Earlier TODOs remain deliberately unimplemented. Merge new files into your own repository; never overwrite completed earlier work.

## Build status

The source compiles, but assessment tests intentionally fail until TODOs are implemented.

```text
go build ./...
go test ./... -run "^TestSmoke_"
go test ./... -count=1 -timeout 20s
go vet ./...
```

Green smoke tests only validate supplied infrastructure. They are not checkpoint completion. Do not skip, delete or weaken acceptance tests. Add your own tests. Use a fresh supported toolchain approved by the module team; the go.mod language compatibility baseline is Go 1.23.

## Safety

Use loopback and synthetic test data. These CW1 services have no TLS or authentication and must not be exposed to other networks.

## Start nodes after completing TODOs

Bash: `sh scripts/run-node-a.sh` and, in another terminal, `sh scripts/run-node-b.sh`.
PowerShell: `./scripts/run-node-a.ps1` and `./scripts/run-node-b.ps1`.
The scripts rebuild the executable and start it in the foreground. Use Ctrl+C to stop it.
HTTP addresses: A `127.0.0.1:8081`, B `127.0.0.1:8082`.

# 6CC545 CW1 - Week 04 student wireframe

This is a cumulative scaffold, not a completed solution. Read the supplied weekly TASKS.md and the weekly workbook. Earlier TODOs remain deliberately unimplemented. Merge new files into your own repository; never overwrite completed earlier work.

## Build status

The source compiles, but assessment tests intentionally fail until TODOs are implemented.

```text
go build ./...
go test ./... -run "^TestSmoke_"
go test ./... -count=1 -timeout 20s
go vet ./...
```

Green smoke tests only validate supplied infrastructure. They are not checkpoint completion. Do not skip, delete or weaken acceptance tests. Add your own tests. Use a fresh supported toolchain approved by the module team; the go.mod language compatibility baseline is Go 1.23.

## Safety

Use loopback and synthetic test data. These CW1 services have no TLS or authentication and must not be exposed to other networks.

## Start nodes after completing TODOs

Bash: `sh scripts/run-node-a.sh` and, in another terminal, `sh scripts/run-node-b.sh`.
PowerShell: `./scripts/run-node-a.ps1` and `./scripts/run-node-b.ps1`.
The scripts rebuild the executable and start it in the foreground. Use Ctrl+C to stop it.
HTTP addresses: A `127.0.0.1:8081`, B `127.0.0.1:8082`.

# 6CC545 CW1 - Week 05 student wireframe

This is a cumulative scaffold, not a completed solution. Read the supplied weekly TASKS.md and the weekly workbook. Earlier TODOs remain deliberately unimplemented. Merge new files into your own repository; never overwrite completed earlier work.

## Build status

Complete Week 5 toolchain preparation in DEPENDENCIES.md first: dependencies and generated Protocol Buffer bindings are required.

```text
go build ./...
go test ./... -run "^TestSmoke_"
go test ./... -count=1 -timeout 20s
go vet ./...
```

Green smoke tests only validate supplied infrastructure. They are not checkpoint completion. Do not skip, delete or weaken acceptance tests. Add your own tests. Use a fresh supported toolchain approved by the module team; the go.mod language compatibility baseline is Go 1.23.

## Safety

Use loopback and synthetic test data. These CW1 services have no TLS or authentication and must not be exposed to other networks.

## Start nodes after completing TODOs

Bash: `sh scripts/run-node-a.sh` and, in another terminal, `sh scripts/run-node-b.sh`.
PowerShell: `./scripts/run-node-a.ps1` and `./scripts/run-node-b.ps1`.
The scripts rebuild the executable and start it in the foreground. Use Ctrl+C to stop it.
HTTP addresses: A `127.0.0.1:8081`, B `127.0.0.1:8082`.

# Week 6 student wireframe

## Publish and subscribe events

This is an add-on overlay for your existing CW1 project, not a replacement repository.
Work in your existing `source/` folder with module `example.edu/6cc545/cw1`.
Keep all completed earlier work. Read MERGE.md before copying anything.

The `project/` folder contains cumulative starter templates. Assessed behaviour is
deliberately incomplete at TODOs. Supplied infrastructure compiles when dependencies
are installed; acceptance tests fail until you implement the contracts. Never submit
an unimplemented starter as a functioning milestone.

## Run from source

```sh
go test ./internal/events -count=1
go test -tags acceptance ./internal/events -count=1 -timeout 30s
go test -race -tags acceptance ./internal/events -count=1 -timeout 60s
```

The first command checks allocation/supplied safety scaffolding only. The second and
third exercise student functions. Run inherited CW1 tests too. For final submission,
run `go test ./...`, `go test -tags acceptance ./...`, `go test -race -tags acceptance ./...`
and `go vet ./...` from your integrated project using the supported lab toolchain.

Expected starter outcome: ordinary smoke checks pass; acceptance reports TODO/error
failures. After implementation, acceptance must pass and the live node evidence must
also show the required behaviour. Do not delete, skip or weaken tests to turn them green.

Use TASKS.md, docs/cw2-integration.md, docs/evidence-record-cw2.md and the Week 6
workbook together. Suggested designs may be adapted if the assessment's observable
requirements remain satisfied and the differences are justified.

