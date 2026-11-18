
# Week 8 - Task 8.1: Component and Configuration Boundaries

## 1. Introduction

This document describes how the main components of the distributed application are organised, how they depend on each other, and how their configuration is managed.

The application is designed to run as multiple independent nodes. Each node has its own identity, network listeners, storage directory and local state. Nodes communicate with one another over the network rather than relying on a permanent central coordinator.

The aim is to keep the responsibilities of each component clear and make it possible to configure and run multiple nodes without mixing their resources.

## 2. Components and Their Responsibilities

The application already contains several components, each responsible for a particular part of the system.

| Component | Responsibility |
|---|---|
| HTTP handlers | Receive HTTP requests and provide access to operations such as peer management and file-related functionality. |
| gRPC service | Handles remote file-related operations between nodes when gRPC is enabled. |
| State registry | Keeps track of known peers and their information within a node. |
| Catalogue | Maintains information about files known to the local node. |
| Local event bus | Allows components within the same node to publish and receive events without directly depending on each other. |
| Storage | Manages the files stored locally by a node. |
| TCP and UDP listeners | Provide optional communication endpoints for the node. |
| Configuration | Loads and validates the settings needed when a node starts. |

Each component has a defined responsibility. For example, the storage component manages file content, while the catalogue maintains the associated file information. This separation helps prevent one component from taking responsibility for unrelated operations.

## 3. Component Lifetimes

Not every object needs to be created every time a request arrives. Some components must remain available throughout the lifetime of a node, while others are only needed while processing an individual request.

### 3.1 Components created once per node

The following components are constructed during node startup and remain available while that node is running:

- **HTTP server and handlers:** The HTTP routes and their handler dependencies are registered during startup.
- **gRPC server and service:** Created once when a gRPC listening address is configured.
- **State registry:** Maintains the node's shared peer information.
- **Catalogue:** Holds the file information managed by the node.
- **Local event bus:** Provides communication between components within the same node.
- **Storage:** Created using the configured data directory.
- **Message inbox:** Provides the node's local message-handling functionality.
- **Network listeners:** Optional TCP and UDP listeners are started during node setup when configured.
- **Event consumers and forwarding worker:** Run in the background to process local events and forward supported events to other peers.

These objects are shared by the relevant components within the same node. They are not recreated for every incoming request.

### 3.2 Objects created per request

Other objects are short-lived and are created as requests are processed. Examples include:

- HTTP request objects and their request contexts.
- Request bodies and JSON decoders.
- Request-specific input and response data.
- Individual gRPC request messages and call contexts.
- Temporary values used while validating or processing an operation.

For example, the HTTP handlers are registered once during startup, but each incoming HTTP request is processed separately.

This approach allows the node to reuse its main components while handling multiple requests.

## 4. Component Dependencies

The application uses `internal/app` as its composition point. Its responsibility is to connect the existing components during startup and provide them with the dependencies they need.

The main startup flow is:

1. `cmd/node` reads the command-line arguments and loads the configuration.
2. The configuration is validated before the application starts.
3. `internal/app` creates and connects the node's main components.
4. The HTTP handlers are registered with the HTTP router.
5. The event bus, storage, registry and catalogue are made available to the relevant components.
6. Optional communication services, including gRPC, TCP and UDP, are started when configured.

The main dependencies can be described as follows:

- HTTP handlers depend on the relevant state, catalogue, storage and event-related functionality.
- The gRPC service uses the catalogue and storage-related functionality required for remote file operations.
- The event bus connects local publishers and consumers.
- The forwarding worker uses peer information to communicate with other nodes.
- The storage component manages locally stored file content.
- The configuration package supplies the settings used to construct the application.

### 4.1 Dependency overview

```text
                 cmd/node
                    |
                    v
              Configuration
                    |
                    v
                app.Run
                    |
                    v
               app.Wire
                    |
        +-----------+-----------+
        |           |           |
        v           v           v
     HTTP API     State      Storage
        |        Registry       |
        |           |           |
        +-----------+-----------+
                    |
               Local Event Bus
                    |
              Event Consumers
                    |
             Forwarding Worker
                    |
              Other Peer Nodes

             Optional Services
              /           \
             v             v
          gRPC          TCP / UDP
```

This is a logical overview of the main relationships rather than a strict execution sequence. The actual components are connected during application startup.

There is no permanent central coordinator managing all nodes. Each node constructs and manages its own components. Communication between nodes takes place through their configured network endpoints.

## 5. External Configuration Contract

Each node receives its configuration through environment variables and command-line flags.

Environment variables provide the initial configuration, while command-line flags override the corresponding values when supplied. The network timeout is parsed from the environment before command-line flags are processed, so an invalid environment timeout causes configuration parsing to fail.

The following table describes the supported configuration settings.

| Setting | Default or example | Purpose |
|---|---|---|
| `NODE_ID` | `node-a` | Identifies the local node. |
| `HTTP_ADDR` | `127.0.0.1:8081` | HTTP listening address. |
| `TCP_ADDR` | Optional | TCP listening address. |
| `UDP_ADDR` | Optional | UDP listening address. |
| `GRPC_ADDR` | Optional | gRPC listening address. |
| `HTTP_ADVERTISE` | Optional | HTTP address advertised to other peers. |
| `GRPC_ADVERTISE` | Optional | gRPC address advertised to other peers. |
| `DATA_DIR` | `data/<NODE_ID>` | Directory used for local file storage. |
| `PEER_SEEDS` | Optional | Comma-separated peer addresses supplied as bootstrap configuration. |
| `SEED` | Optional | Path to a JSON metadata seed file. |
| `NETWORK_TIMEOUT` | `2s` | Timeout used for network operations. |

The listening addresses and advertised addresses have different purposes.

A listening address determines where a service accepts connections on the local machine or container. An advertised address tells other peers where they should connect to reach that node.

For example, a node running inside a Docker network can listen on `0.0.0.0:8080` while advertising `node-a:8080` to other containers.

Outside a container deployment, the default HTTP listening address remains bound to `127.0.0.1:8081`. This keeps the default listener local rather than exposing it on every network interface.

The `SEED` setting is also separate from `PEER_SEEDS`. The former identifies a JSON metadata file, while the latter contains peer addresses.

At this stage, `PEER_SEEDS` is part of the configuration contract. Its presence should not be taken as proof that automatic peer discovery or bootstrap registration has already been implemented.

## 6. Supported Formats and Validation

The configuration package checks the supplied values before the application starts.

The supported formats include:

- **Node identity:** Between 1 and 32 characters, using letters, numbers, underscores or hyphens.
- **Listening addresses:** Host-and-port values with a valid port between 1 and 65535. The host must be `localhost`, a loopback IP address or an explicitly configured unspecified IP address such as `0.0.0.0`.
- **Advertised addresses:** IP addresses or valid DNS-style hostnames followed by a valid port. URL schemes such as `http://` should not be included.
- **Peer seeds:** A comma-separated list of peer host-and-port addresses. Whitespace around individual entries is removed before validation.
- **Storage directory:** A non-empty filesystem path.
- **Metadata seed:** A path to the optional JSON metadata file.
- **Network timeout:** A valid Go duration value, such as `2s` or `500ms`. It must be greater than zero and no more than 30 seconds.

Startup validation checks important configuration values, including:

- The node identity.
- The required HTTP listening address.
- Optional TCP, UDP and gRPC listening addresses when supplied.
- Optional HTTP and gRPC advertised addresses when supplied.
- The storage directory.
- Every supplied peer seed.
- The network timeout.

Invalid values cause configuration validation to fail instead of allowing the application to continue with an incorrect setup.

The timeout must be greater than zero and must not exceed 30 seconds.

Optional TCP, UDP and gRPC services are only configured when their corresponding addresses are supplied.

## 7. Environment Example and Secret Handling

The example environment file is located at:

`deployment/.env.example`

It contains the Docker build settings and commented examples of the application's runtime configuration.

The examples are provided to show the expected configuration format. The three nodes in `deployment/compose.yaml` have their own runtime settings, including separate identities, listening addresses, advertised addresses and storage volumes.

The runtime examples in `.env.example` are documentation only. Docker Compose does not automatically pass them into the containers because the corresponding runtime values are currently defined directly in `compose.yaml`.

No passwords, access tokens, private keys or other secrets should be stored in `.env.example`.

File-based secret configuration is intentionally left for Week 9, as specified in the coursework requirements.

## 8. Summary

The application is organised around independent components with clearly defined responsibilities. Each node constructs its own shared components during startup, while request-specific objects are created when individual operations are processed.

The `internal/app` package connects these components without introducing a permanent central coordinator. Configuration is kept separate from application logic, allowing each node to use its own identity, network addresses and storage directory.

This provides a clear starting point for the deployment and multi-node work in the following stages.
```
