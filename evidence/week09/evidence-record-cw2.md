# CW2 evidence record

## Record 1: Development certificates

**Student number:** 100862618

**Week and task:** Week 09, development certificates.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go run ./cmd/devcert -out ../secrets
```

**Expected behaviour and requirement ID:** Generate development certificates for the three nodes.

**Observed output (attach complete, sanitised transcript):** Certificate generation completed successfully. The certificate details are recorded in the certificate inventory.



**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The certificates were available for the Week 09 security tests. They are development credentials, not production certificates.

**Failure correction and next action:** No correction required.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 2: TLS and mTLS

**Student number:** 100862618

**Week and task:** Week 09, Task 9.3, TLS and mTLS.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go test -tags acceptance -v ./internal/security -count=1 -timeout 60s
```

**Expected behaviour and requirement ID:** Verify trusted certificates and require client certificates for mTLS connections.

**Observed output (attach complete, sanitised transcript):** The TLS and mTLS acceptance tests passed.



**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The tested certificate validation and mutual authentication scenarios worked as expected.

**Failure correction and next action:** No outstanding failure.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 3: JWT and REST middleware

**Student number:** 100862618

**Week and task:** Week 09, JWT validation and REST authorisation.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go test -tags acceptance ./... -count=1 -timeout 120s
```

**Expected behaviour and requirement ID:** Validate JWT claims and enforce reader and writer permissions.

**Observed output (attach complete, sanitised transcript):** The acceptance tests passed. Invalid tokens and requests with insufficient permissions were rejected.





**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The tests cover the token validation and role checks included in the suite. The implementation uses HS256 with a shared signing key.

**Failure correction and next action:** No outstanding failure.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 4: gRPC peer identity

**Student number:** 100862618

**Week and task:** Week 09, gRPC mTLS identity verification.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go test -tags acceptance ./... -count=1 -timeout 120s
```

**Expected behaviour and requirement ID:** Verify peer certificates and match the certificate identity to the requesting node.

**Observed output (attach complete, sanitised transcript):** The identity tests passed, including cases involving missing, mismatched and unverified identities.



**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The tested peer identity checks worked as expected.

**Failure correction and next action:** No outstanding failure.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 5: Security regression and race tests

**Student number:** 100862618

**Week and task:** Week 09, regression, negative security and race tests.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go test ./... -count=1
go test -tags acceptance ./... -count=1 -timeout 120s
go test -tags acceptance -v ./internal/security -count=1 -timeout 60s
go test -race -tags acceptance ./internal/security -count=1 -timeout 120s
```

**Expected behaviour and requirement ID:** The tests should pass, invalid credentials should be rejected, and the race detector should report any races it finds.

**Observed output (attach complete, sanitised transcript):** All recorded commands passed. The negative tests covered invalid tokens, permissions and certificates. No races were detected in the security test paths.











**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** No races were detected in the executed tests. This does not guarantee that every possible execution is race-free.

**Failure correction and next action:** No outstanding test failure.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 6: Required security startup

**Student number:** 100862618

**Week and task:** Week 09, required security startup.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Run from `source/`.

```
go test -tags acceptance ./... -count=1 -timeout 120s
```

**Expected behaviour and requirement ID:** The application should reject incomplete security configuration rather than starting in an insecure mode.

**Observed output (attach complete, sanitised transcript):** The startup acceptance test passed. I corrected the test fixture to include the required peer HTTP configuration before the successful run.



**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The test confirms the startup checks covered by the acceptance test.

**Failure correction and next action:** The test fixture was corrected and the test passed.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 7: Secure deployment and network checks

**Student number:** 100862618

**Week and task:** Week 09, secure Docker Compose deployment, HTTPS and peer mTLS.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Docker Desktop 4.99.0, Engine 29.9.3, Compose 5.7.0. Image digests not recorded.

**Exact command and working directory:** Run from `source/`.

```
docker compose -f ../deployment/compose.secure.yaml config
docker compose -f ../deployment/compose.secure.yaml up --build -d
docker compose -f ../deployment/compose.secure.yaml ps
docker exec cw2-secure-lab-node-a-1 /app/probe -url https://127.0.0.1:8443/healthz -ca /run/secrets/ca.crt
docker exec cw2-secure-lab-node-b-1 /app/probe -url https://node-a:9444/v1/peer/files/test.txt -ca /run/secrets/ca.crt
```

**Expected behaviour and requirement ID:** Start the three secured nodes, verify HTTPS health access and reject peer requests without a client certificate.

**Observed output (attach complete, sanitised transcript):** Compose validation and deployment succeeded. The internal HTTPS health check succeeded. The peer request without a client certificate was rejected with `remote error: tls: certificate required`.







**Pass / fail / not run and explanation:** Pass for deployment and the internal network checks.

**Interpretation and limit of what the test establishes:** Docker Desktop did not publish the configured ports to the Windows host. I therefore verified HTTPS from inside the Docker network. Host access was not verified.

**Failure correction and next action:** The host port issue remained unresolved and is recorded as a limitation.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 8: Graceful shutdown

**Student number:** 100862618

**Week and task:** Week 09, secure deployment shutdown.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. CGO and compiler details not recorded.

**Docker/Compose versions and image digest where relevant:** Docker Desktop 4.99.0, Engine 29.9.3, Compose 5.7.0.

**Exact command and working directory:**

```
docker compose -f ../deployment/compose.secure.yaml stop
```

**Expected behaviour and requirement ID:** Stop all three containers cleanly.

**Observed output (attach complete, sanitised transcript):** All three containers exited with code `0` and no reported error.



**Pass / fail / not run and explanation:** Pass.

**Interpretation and limit of what the test establishes:** The containers shut down cleanly during this test.

**Failure correction and next action:** No outstanding failure.

**AI/source acknowledgement where applicable:** ChatGPT assisted with documentation.

---

## Record 9: Security documentation

**Student number:** 100862618

**Week and task:** Week 09, security documentation.

**Immutable tag and full commit hash:** No Week 09 tag or commit created. Changes remain uncommitted.

**Date/time and timezone:** 19 November 2026, GMT.

**OS, architecture, go version, CGO_ENABLED, C compiler:** Windows 11, amd64, Go 1.27.1. Not applicable to documentation.

**Docker/Compose versions and image digest where relevant:** Not applicable.

**Exact command and working directory:** Documentation is stored in `docs/` and `evidence/week09/`.

**Expected behaviour and requirement ID:** Document the security design, certificate identities, roles, test procedure and known limitations.

**Observed output (attach complete, sanitised transcript):** The threat model, certificate inventory, role matrix, runbook and evidence record were prepared. No private keys or tokens are included.

**Pass / fail / not run and explanation:** Documentation prepared.

**Interpretation and limit of what the test establishes:** The peer file-transfer implementation was not tested end to end. The deploycheck scaffold does not establish application readiness.

**Failure correction and next action:** Check that the two certificate inventory copies match before submission.

**AI/source acknowledgement where applicable:** ChatGPT assisted with drafting and editing. Technical details are based on the implementation and local test results.

