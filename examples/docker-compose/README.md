# Docker Compose Example

An end-to-end SPIFFE/SPIRE demo that runs a complete trust infrastructure and two workloads communicating via JWT-SVID authentication -- all orchestrated with Docker Compose.

## What It Does

1. Starts a **SPIRE Server** (certificate authority for trust domain `example.nais.io`).
2. Runs a **bootstrap** init container that generates a join token, extracts the trust bundle, and registers a workload entry.
3. Starts a **SPIRE Agent** that attests to the server using the join token and exposes the Workload API over a Unix socket.
4. Starts the **workload-go-server**, which validates incoming JWT-SVIDs using the SPIRE trust bundle.
5. Starts the **workload-go-client**, which fetches a JWT-SVID from the agent and sends an authenticated HTTP request to the server.

```
SPIRE Server ──► Bootstrap (init) ──► SPIRE Agent
                                          │
                                  Workload API socket
                                    ┌─────┴─────┐
                                 Go Client   Go Server
                                    └───HTTP────►┘
```

## Usage

```bash
docker compose up --build
```

On success the client logs the server's `"Success!!!"` response and exits.

## Services

| Service | Description |
|---------|-------------|
| `spire-server` | SPIRE Server (binds `0.0.0.0:8081`) |
| `spire-bootstrap` | One-shot container: creates trust bundle, join token, and workload registration entry, then exits |
| `spire-agent` | SPIRE Agent using join-token attestation; exposes the Workload API socket |
| `workload-go-server` | HTTP server (port `8082`) that validates JWT-SVIDs |
| `workload-go-client` | HTTP client that fetches a JWT-SVID and calls the server |

## Configuration

Server and agent configs live in `config/`:

- `config/server/server.conf` -- Trust domain, CA settings, SQLite datastore, JWT issuer.
- `config/agent/agent.conf` -- Server address, workload socket path, Unix workload attestor.

The bootstrap script (`scripts/bootstrap.sh`) registers a single workload entry:

- **SPIFFE ID:** `spiffe://example.nais.io/workload/go-client`
- **Selector:** `unix:uid:0` (matches processes running as root)

## Shared Volumes

| Volume | Purpose |
|--------|---------|
| `spire-server-socket` | Server admin API socket shared with the bootstrap container |
| `spire-bootstrap` | Join token and trust bundle shared between bootstrap and the agent |
| `spire-workload-socket` | Agent Workload API socket shared with both Go workloads |
