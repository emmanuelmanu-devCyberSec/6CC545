# Threat Model

## 1. Introduction

For this part of the project, I have identified the main things that need to be protected in the three-node application and the possible threats that could affect them. The aim is to make sure that only authenticated users and trusted nodes can access the operations they are allowed to perform.

The application uses TLS, mutual TLS (mTLS), JWT authentication and role-based access control to provide these protections.

## 2. Assets to Protect

The main assets I identified are:

- **Stored content:** Files uploaded to the application and the information stored by each node.
- **Metadata:** File names, file details and other information maintained by the application.
- **Node identities:** The identities used by the three nodes when communicating with each other.
- **Authentication keys:** The private keys, certificate authority key and JWT signing key used by the security mechanisms.
- **Service availability:** Making sure that the application and its nodes remain available to authorised users.

These assets are important because losing control of them could affect the confidentiality, integrity or availability of the application.

## 3. Trust Boundaries

The application runs in a closed teaching environment with three nodes communicating with clients and each other.

I have identified two main communication boundaries. The first is between clients and the application servers, where verified TLS is needed to protect communication. The second is between the nodes, where mutual TLS is used so that both sides can verify each other's certificates.

There is also an authentication boundary around protected application operations. A request must have a valid JWT with the correct permissions before it can access protected functionality. Similarly, a node must prove its identity using its certificate rather than simply claiming to be a particular node in a request.

Private keys and the JWT signing key must remain within the protected local environment and should not be committed to the Git repository.

## 4. Possible Threats

I considered the following situations when identifying the main threats:

- **Unauthenticated client:** Someone without a valid token may try to access stored files or other protected information.
- **Read-only client:** A user with a valid reader token may try to perform a write operation even though their role does not allow it.
- **Untrusted peer:** An unknown node may try to connect to one of the application nodes while pretending to be a trusted member of the cluster.
- **Malformed input:** A client or peer may send invalid tokens, incorrect information or unexpected data that could affect the application.
- **Compromised trusted node:** An attacker who gains access to a legitimate node's private key may be able to impersonate that node when connecting to other nodes.

There is an important difference between a compromised trusted node and an outsider without credentials. An outsider should fail the authentication checks because they do not have valid credentials. However, an attacker with a compromised node key may be able to pass certificate authentication because they possess legitimate identity material. This makes protecting private keys particularly important.

## 5. User Roles and Permissions

The application uses two roles: `reader` and `writer`.

A reader can view permitted files and metadata but cannot perform write operations. A writer can perform both read and write operations. This means the writer role includes the permissions available to a reader.

The following table shows how I intend to protect the main operations:

| Operation | Required protection | Permission |
|---|---|---|
| Minimal health response | HTTPS without sensitive information | No token required |
| REST list or read | Verified HTTPS and valid JWT | Reader or writer |
| REST write | Verified HTTPS and valid JWT | Writer only |
| Protected peer RPC or transfer | Verified mTLS identity and operation authorisation, with the required token or approved equivalent | Depends on the operation |
| Local token generation | Access to the private JWT signing-key file | Authorised local access |

For communication between nodes, the certificate identity must also match the node identity being claimed. For example, if a connection is authenticated as node A, the application must not accept a request claiming that the sender is node B.

I will not rely on a role supplied through an HTTP header or request body because this information can be changed by the client. Permissions must come from a successfully validated token.

## 6. Shared JWT Signing Key

The teaching environment uses the HS256 algorithm to sign JWTs. One limitation of this approach is that the same secret key is used for signing and verifying tokens.

This means that anyone who obtains the shared signing key can create new valid tokens as well as verify existing ones. For example, someone with access to the key could create a token claiming to have the writer role.

As a result, there is no production-level separation between the token issuer and the services that verify tokens. The shared key must therefore be kept private and restricted to the teaching environment.

## 7. Remaining Security Risks

Although the security controls help protect the application, they do not remove every possible risk.

- **TLS:** It protects information while it travels between the client and server, but it does not make uploaded files safe. The application still needs to check file sizes, names and storage operations.
- **mTLS:** If a trusted node's private key is compromised, an attacker may be able to impersonate that node until the certificate is revoked or expires.
- **JWT:** Anyone who obtains the shared HS256 key can generate tokens, including tokens with writer permissions.
- **Role-based access control:** If a protected route is missing its permission check or the permissions are mapped incorrectly, a user may gain access to an operation they should not be allowed to perform.
- **Peer identity verification:** A valid certificate does not automatically mean that every requested peer operation should be allowed. The application must still check the node identity and whether the operation is permitted.
- **Service availability:** These security controls do not completely prevent repeated requests or other activity that could affect the availability of the application.

## 8. Conclusion

This threat model has helped me identify the main assets that need protection and the security controls required by the three-node application. Readers are limited to read operations, while writers can perform both reading and writing.

TLS protects client communication, mTLS helps verify the identity of connecting nodes, and JWT authentication is used to check access to protected operations. However, risks such as compromised private keys and the shared JWT signing key still need to be considered.

Overall, these controls provide a suitable security foundation for the CW2 teaching environment, while also highlighting some limitations that would need further attention in a production system.