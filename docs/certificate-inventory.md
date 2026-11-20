# Development Certificate Inspection and Trust Matrix

## 1. Introduction

For this task, I generated development certificates for the three nodes using the supplied `devcert` tool. The certificates are used to establish trust between the nodes and allow them to verify each other's identities when communicating.

The generator created a local certificate authority (CA), individual certificates for node A, node B and node C, and a JWT signing key. All generated credentials are stored in the outer `secrets` directory, outside the `source` folder.

I also checked the generated certificates to understand their identities, who issued them and how long they remain valid.

## 2. Certificate Authority

The CA certificate is named `ca.crt`. It acts as the root of trust for the three node certificates.

From my inspection, the CA has the following details:

| Property | Observed value |
|---|---|
| Subject | CN=CW2 local teaching CA |
| Issuer | CN=CW2 local teaching CA |
| Valid from | 19 November 2026 |
| Expires | 26 November 2026 |
| Public key algorithm | ECDSA P-256 |
| Key usage | Certificate Signing and CRL Signing |

The subject and issuer are the same, which confirms that the CA certificate is self-signed. The three node certificates were issued by this CA.

## 3. Node Certificate Inspection

I inspected the public certificate for each node using the Windows `certutil -dump` command.

All three certificates have the same issuer, which is `CN=CW2 local teaching CA`. They also use ECDSA P-256 and have both Server Authentication and Client Authentication as their Extended Key Usage (EKU).

The individual certificate details are shown below.

### Node A

- **Subject:** CN=node-a
- **Issuer:** CN=CW2 local teaching CA
- **DNS SANs:** node-a, localhost
- **IP SAN:** 127.0.0.1
- **URI SAN:** spiffe://cw2.local/node-a
- **Valid from:** 19 November 2026
- **Expires:** 21 November 2026
- **EKUs:** Server Authentication and Client Authentication

### Node B

- **Subject:** CN=node-b
- **Issuer:** CN=CW2 local teaching CA
- **DNS SANs:** node-b, localhost
- **IP SAN:** 127.0.0.1
- **URI SAN:** spiffe://cw2.local/node-b
- **Valid from:** 19 November 2026
- **Expires:** 21 November 2026
- **EKUs:** Server Authentication and Client Authentication

### Node C

- **Subject:** CN=node-c
- **Issuer:** CN=CW2 local teaching CA
- **DNS SANs:** node-c, localhost
- **IP SAN:** 127.0.0.1
- **URI SAN:** spiffe://cw2.local/node-c
- **Valid from:** 19 November 2026
- **Expires:** 21 November 2026
- **EKUs:** Server Authentication and Client Authentication

One thing I noticed is that each node has its own URI SAN. This is useful because the application can use that identity to work out which node is connecting, rather than relying only on a name supplied in a request.

The certificates are short-lived development certificates. The node certificates expire on 21 November 2026, while the CA expires on 26 November 2026. I will need to keep these expiry dates in mind when testing the application.

## 4. Trust Matrix

The three nodes need to verify each other when establishing secure connections. This involves checking the certificate chain and confirming that the server name matches the certificate's SAN.

The receiving node must also verify the connecting node's client certificate and map its URI SAN to the correct node identity.

The following matrix shows the expected trust relationships:

| Connection | Server certificate verification | Client certificate identity |
|---|---|---|
| Node A to Node B | A verifies B's certificate chain and expected server name | B verifies A using `spiffe://cw2.local/node-a` |
| Node A to Node C | A verifies C's certificate chain and expected server name | C verifies A using `spiffe://cw2.local/node-a` |
| Node B to Node A | B verifies A's certificate chain and expected server name | A verifies B using `spiffe://cw2.local/node-b` |
| Node B to Node C | B verifies C's certificate chain and expected server name | C verifies B using `spiffe://cw2.local/node-b` |
| Node C to Node A | C verifies A's certificate chain and expected server name | A verifies C using `spiffe://cw2.local/node-c` |
| Node C to Node B | C verifies B's certificate chain and expected server name | B verifies C using `spiffe://cw2.local/node-c` |

This means that a node should not be trusted just because it claims to be another node. Its certificate must be issued by the trusted CA, and its URI identity must match the node it claims to represent.

## 5. Host and Container Names

The certificates contain both node DNS names and `localhost`, as well as the loopback IP address `127.0.0.1`.

I need to use the correct name depending on how the connection is made.

For container-to-container communication, the node DNS name should be used, for example `node-b` when connecting to node B.

For host-to-container communication, `localhost` or `127.0.0.1` can be used where the application is accessed through a host-published port.

The name used by the client must match an appropriate SAN in the server certificate. I will keep normal certificate and hostname verification enabled rather than disabling verification to work around a name mismatch.

## 6. Protecting the Generated Credentials

The generated credentials are stored in the outer `secrets` directory. This directory is excluded from Git using the `/secrets/` rule in `.gitignore`.

The directory contains private keys as well as public certificates. The private keys and JWT signing key must remain private and must not be committed to Git, included in evidence archives or copied into container images.

For the evidence, I captured a screenshot showing the generated directory structure and certificate filenames without opening or exposing any private key contents.

## 7. Conclusion

This task helped me understand how the three nodes establish trust using certificates issued by the same local CA. Each node has its own certificate identity, including a URI SAN that can be used to identify it during mutual TLS authentication.

The trust matrix also makes it clear that both sides of a connection have responsibilities. The connecting node checks the server's certificate and name, while the receiving node checks the client's certificate and identity.

