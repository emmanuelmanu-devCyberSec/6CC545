# Role and endpoint matrix

This matrix records the access controls wired into the Week 09 application.

| Endpoint / operation | Identity or permission required | Notes |
|---|---|---|
| `GET /healthz` | No JWT | Minimal health response. HTTPS in required security mode. |
| `GET /v1/status` | JWT `reader` | Read-only status information. |
| `GET /v1/peers` | JWT `reader` | Read-only peer information. |
| `GET /v1/files` | JWT `reader` | Read access to file information/content through the API. |
| `POST /v1/files` | JWT `writer` | Write operation. |
| `GET /v1/messages` | JWT `reader` | Read messages. |
| `POST /v1/messages` | JWT `writer` | Publish/write messages. |
| `GET /v1/remote-files` | JWT `reader` plus secure peer communication | The route uses the secure remote-file handler and configured TLS identity. |
| `/v1/peer/announce` | Verified peer client certificate and peer identity validation | Private peer HTTPS listener; mTLS required. |
| `/v1/peer/files/{name}` | Verified peer client certificate and peer identity validation | Private peer HTTPS listener; mTLS required. |

## Role definitions

- **Reader:** may perform the protected read operations listed above.
- **Writer:** may perform the protected read and write operations listed above.
- **Peer identity:** node-to-node operations use a certificate issued by the configured CA. The certificate URI SAN is checked against the claimed node identity.

The public API uses JWT role middleware for the protected REST routes. The private peer HTTPS listener requires a verified client certificate. These are separate controls: a peer certificate establishes the connecting node identity; it does not by itself mean that a public reader or writer JWT was supplied.

## Deliberately limited endpoints and compatibility paths

- `/healthz` is intentionally a minimal public health endpoint. It is not a substitute for an authenticated application operation.
- The legacy HTTP health-only lifecycle path is retained for inherited regression tests. Required security mode uses HTTPS.
- `cmd/deploycheck` is supplied deployment scaffolding. Its `/healthz` endpoint reports `deployment-scaffold-only` and `assessment_ready: false`; it is not the secured CW2 node.
- The root `/` route returns 404. No application functionality is exposed there.

The application wiring retains compatibility branches for tests where the validator is absent. Those branches must not be interpreted as the required-security deployment configuration.
