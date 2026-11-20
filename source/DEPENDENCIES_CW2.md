# CW2 dependency additions

Retain your existing module path `example.edu/6cc545/cw1`, CW1 go.mod/go.sum,
generated Protocol Buffer bindings and pinned gRPC/Protobuf dependencies.
Do not replace go.mod with a new module or copy another student's go.sum.

Weeks 6 to 8 additions use only the standard library. In Week 9, from source/:

```sh
go get github.com/golang-jwt/jwt/v5@v5.3.1
go mod tidy
go mod verify
```

Use the maintained JWT library for parsing and cryptographic signature checking.
The supplied test issuer is strictly local teaching infrastructure. HS256 shares
one random key with all verifiers; any compromised verifier can mint credentials.
Discuss this trust limitation. Asymmetric signing can separate issuer/verifier
privileges but is optional; preserve the token validation/role acceptance tests.

Use the staff-approved supported Go toolchain. The Docker teaching example pins
Go 1.27.3; staff must verify the corresponding image and record its digest in the
lab. The existing go 1.23.0 directive expresses source compatibility, not a demand
to run an unsupported Go 1.23 toolchain. Do not silently update all dependencies.

If modules cannot download, retain the exact proxy error and use the approved
lab/cache route. Never bypass TLS verification to work around a download error.
