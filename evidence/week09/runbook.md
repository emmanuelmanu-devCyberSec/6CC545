# Week 09 Security Runbook

## 1. Purpose and Overview

This runbook explains how to reproduce and verify the security work completed during Week 09 of the 6CC545 coursework.

The implementation focuses on securing communication between distributed nodes using TLS and mutual TLS (mTLS), protecting REST API access with JWT authentication, verifying peer identities, and running the application through a secure Docker Compose deployment.

The aim is to provide a clear procedure for testing these features, understanding the expected results, and recognising the limitations identified during verification.

Testing should be carried out using a separate working copy of the Week 09 checkpoint. This keeps the submitted work unchanged and allows fresh development credentials to be generated.

**Security note:** Private keys, JWT signing keys, generated access tokens and other sensitive credentials must remain in the local test environment. They must not be included in the submitted checkpoint or screenshots.

## 2. Requirements

The following tools are required to reproduce the Week 09 tests:

- **Go:** A version compatible with the project's `go.mod` file.
- **Docker Desktop:** Required to build and run the secure multi-node environment.
- **Docker Compose:** Used to validate and manage the secure deployment.
- **C compiler:** Required for Go race detection when CGO is enabled.
- **Project dependencies:** The required Go dependencies are defined in `go.mod` and `go.sum`.

The implementation uses pinned versions of its main security-related dependencies, including `golang-jwt/jwt/v5`, gRPC and Protocol Buffers. The Docker build uses the Go image specified in the Week 09 Dockerfile.

## 3. Test Environment Preparation

Before running the tests, create a separate working copy of the submitted Week 09 checkpoint.

Use the `source` directory within this copy as the working directory for the Go commands. The corresponding `deployment` directory must also be available because the secure Docker Compose configuration is stored there.

All procedures in this runbook should be performed against the temporary working copy rather than the original development repository or submitted checkpoint.

This separation prevents test-generated credentials and other temporary files from being added to the submitted work.

## 4. Generate Development Credentials

Week 09 includes a development certificate utility for generating the local certificate authority and the certificates required by the nodes.

From the `source` directory, run:

```bash
go run ./cmd/devcert -out ../secrets
```

The utility generates the local CA, node certificates, private keys and JWT signing key required by the lab environment. It is designed to refuse to overwrite existing credentials.

A short-lived test JWT can also be generated using:

```bash
go run ./cmd/devtoken -key ../secrets/jwt.key -role reader -sub lab-user -ttl 5m -iss cw2-lab -aud cw2-lab
```

The token is intended for local testing. Do not commit it, include it in screenshots or copy it into the submitted checkpoint.

## 5. Run the Go Tests

Start with the standard Go test suite. This checks the application packages before moving on to the secure deployment.

From the `source` directory:

```bash
go test ./... -count=1
```

Next, run the acceptance tests:

```bash
go test -tags acceptance ./... -count=1 -timeout 120s
```

The security package can also be tested separately to make the security-specific results easier to review:

```bash
go test -tags acceptance -v ./internal/security -count=1 -timeout 60s
```

For additional concurrency checking, run the race detector with CGO enabled:

```bash
go test -race -tags acceptance ./internal/security -count=1 -timeout 120s
```

The race detector requires CGO and a suitable C compiler. If the local environment does not have the required compiler, use a Go environment that supports CGO.

The expected result is that the relevant tests complete successfully. A failed test should be investigated rather than treated as evidence of a successful security check.

## 6. Start the Secure Docker Deployment

Week 09 provides a separate secure Compose configuration. This allows the security-focused deployment to be tested without replacing the existing Compose configuration.

From the `source` directory, validate the configuration:

```bash
docker compose -f ../deployment/compose.secure.yaml config
```

If the configuration is valid, build and start the services:

```bash
docker compose -f ../deployment/compose.secure.yaml up --build -d
```

Check the status of the running services:

```bash
docker compose -f ../deployment/compose.secure.yaml ps
```

The deployment contains three nodes. Each node uses its own certificate and data volume. The private peer communication listeners are kept separate from the main HTTPS API.

The secure configuration expects the generated `secrets` directory to be available alongside the `source` and `deployment` directories in the working copy.

## 7. Verify HTTPS and Mutual TLS

The secure deployment should be checked from inside its Docker network. A successful connection between containers does not necessarily mean that the service is also reachable through a host-published port.

### 7.1 HTTPS health check

The following command checks the HTTPS health endpoint from inside node A and validates the server certificate against the local CA:

```bash
docker exec cw2-secure-lab-node-a-1 /app/probe -url https://127.0.0.1:8443/healthz -ca /run/secrets/ca.crt
```

The recorded check completed successfully inside the Docker network.

This confirms that the HTTPS endpoint was reachable and that its certificate could be validated using the supplied CA certificate.

### 7.2 Mutual TLS rejection check

The peer file endpoint requires a valid client certificate. The following check attempts to access it from node B without presenting a client certificate:

```bash
docker exec cw2-secure-lab-node-b-1 /app/probe -url https://node-a:9444/v1/peer/files/test.txt -ca /run/secrets/ca.crt
```

The recorded result was:

```text
remote error: tls: certificate required
```

This is the expected rejection. It demonstrates that the peer listener refuses a connection that does not provide the required client certificate.

Together, these checks demonstrate a successful HTTPS health request and enforcement of client-certificate authentication on the peer listener.

## 8. Stop the Deployment

After completing the checks, stop the running services:

```bash
docker compose -f ../deployment/compose.secure.yaml stop
```

Remove the containers and Compose network:

```bash
docker compose -f ../deployment/compose.secure.yaml down
```

Keep the generated credentials within the temporary working environment. They must not be copied into the submitted checkpoint.

## 9. Verification Results and Known Limitations

The following results distinguish between what was successfully verified during Week 09 and what remains unverified.

### 9.1 Secure Docker deployment

The secure Compose configuration was validated, and all three node containers were built and started.

The configuration declares host mappings for ports 8441, 8442 and 8443. However, during testing on Docker Desktop for Windows, the expected host port publishing was not available. Although the Compose configuration contained the port mappings, the corresponding host ports were not reachable.

For this reason, the successful HTTPS health check was performed from inside the Docker network. Host-based HTTPS access should not be described as verified.

### 9.2 Health endpoint

The `/healthz` endpoint is intentionally unauthenticated and provides a basic health response. In the required-security deployment, it is served over HTTPS.

The legacy HTTP health-only application lifecycle remains for regression compatibility with existing tests. It should not be confused with the required-security deployment.

### 9.3 Deployment checker

The Week 09 `cmd/deploycheck` utility remains a scaffold. Its reported state is `deployment-scaffold-only`, with `assessment_ready: false`.

It should not be presented as a completed deployment assessment or as proof that all deployment security requirements have been automatically checked.

### 9.4 Peer file transfer

The peer file transfer route and supporting helper are implemented. The peer listener enforces client-certificate authentication and checks the peer identity.

However, a complete end-to-end file transfer workflow was not verified during this stage. The implementation should therefore not be described as a fully demonstrated distributed file-transfer system.

### 9.5 Root endpoint

The application root path `/` returns HTTP 404. This is separate from the health endpoint and does not indicate that the HTTPS health check has failed.

## 10. Certificate Identity Information

The Week 09 development certificates were generated using a local teaching certificate authority and use ECDSA with the P-256 curve.

The recorded certificate metadata includes:

| Certificate field | Recorded configuration |
|---|---|
| Certificate authority | Local teaching CA |
| Node identities | `node-a`, `node-b`, `node-c` |
| Common Names | Corresponding node identifiers |
| Key algorithm | ECDSA P-256 |
| Extended Key Usage | ServerAuth and ClientAuth |
| DNS SANs | Node names and localhost |
| IP SAN | `127.0.0.1` |
| SPIFFE identities | `spiffe://cw2.local/node-a`, `spiffe://cw2.local/node-b`, `spiffe://cw2.local/node-c` |

The SPIFFE URI identities bind a verified certificate to the expected node identity. This helps prevent a peer from being accepted simply because it presents a certificate issued by the trusted CA.

Further details about the certificate metadata are available in:

`docs/certificate-inventory.md`

## 11. Security Considerations

The Week 09 implementation uses separate security controls for API access and peer identity.

JWT authentication protects the REST API and distinguishes between reader and writer permissions. TLS protects communication channels, while mTLS requires both sides of a peer connection to present valid certificates.

These controls serve different purposes and should not be treated as interchangeable. A valid API token does not replace the need to verify a peer certificate. Likewise, a trusted peer certificate does not automatically grant the application permissions associated with a JWT role.

The credentials generated for this development environment are not production credentials. A production deployment would require an appropriate certificate issuance and renewal process, secure key management and operational monitoring.

The identified threats, protected assets, trust boundaries and security considerations are discussed in:

`docs/threat-model.md`

## 12. Cleanup

Once testing is complete, confirm that the Docker services have been stopped and the temporary working copy is no longer required.

Remove the temporary working copy together with its generated development credentials. The original development repository and submitted Week 09 checkpoint should remain unchanged.

Do not add generated private keys, JWT signing keys, access tokens or local environment files to the coursework submission.

## 13. Supporting Evidence

The Week 09 checkpoint contains two supporting security documents:

- **`docs/threat-model.md`** – describes the threats considered during implementation, the protected assets, trust boundaries and relevant security considerations.
- **`docs/certificate-inventory.md`** – records the certificate identities and relevant certificate metadata used during testing.

The Week 09 `evidence` directory contains the screenshots captured during implementation and verification. These provide supporting records of the development credentials, TLS and mTLS tests, JWT validation, REST middleware, gRPC identity checks, negative security tests, secure startup, Docker deployment, HTTPS health check, peer mTLS rejection and graceful shutdown.

