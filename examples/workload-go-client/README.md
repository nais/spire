# Workload Go Client

A Go application that fetches a JWT-SVID from a SPIRE Agent and uses it as a Bearer token to make an authenticated HTTP request to a server.

## What It Does

1. Connects to the SPIRE Agent's Workload API via a Unix domain socket.
2. Fetches a JWT-SVID scoped to the audience `spiffe://example.nais.io/server`.
3. Sends an HTTP GET request to the server with the JWT in the `Authorization: Bearer` header.
4. Logs the server's response.

This is the client half of a pair -- see [workload-go-server](../workload-go-server/) for the server that validates these tokens.

## Usage

The recommended way to run this example is via the [Docker Compose setup](../docker-compose/), which starts the full SPIRE infrastructure alongside both workloads:

```bash
cd ../docker-compose
docker compose up --build
```

To run standalone (requires a running SPIRE Agent with a registered entry for this workload):

```bash
export SPIFFE_ENDPOINT_SOCKET="unix:///path/to/workload_api.sock"
export SERVER_URL="http://localhost:8082"
go run main.go
```

## Configuration

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `SPIFFE_ENDPOINT_SOCKET` | `unix:///tmp/agent.sock` | SPIRE Agent Workload API socket path |
| `SERVER_URL` | `http://localhost:8082` | URL of the workload-go-server |

## Dependencies

- [`github.com/spiffe/go-spiffe/v2`](https://github.com/spiffe/go-spiffe) -- SPIFFE Workload API client and JWT-SVID support.
