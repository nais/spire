# Workload Go Server

A Go HTTP server that validates incoming requests using SPIFFE JWT-SVIDs obtained from a SPIRE Agent.

## What It Does

1. Connects to the SPIRE Agent's Workload API to obtain JWT trust bundles via a `JWTSource`.
2. Listens on port `8082` for HTTP requests.
3. An authentication middleware extracts the `Authorization: Bearer` token from each request, validates it against the SPIRE trust bundle and expected audience (`spiffe://example.nais.io/server`), and rejects unauthorized requests with HTTP 401.
4. On success, responds with `"Success!!!"` and logs the authenticated SVID claims.

This is the server half of a pair -- see [workload-go-client](../workload-go-client/) for the client that sends the JWT-authenticated requests.

## Usage

The recommended way to run this example is via the [Docker Compose setup](../docker-compose/), which starts the full SPIRE infrastructure alongside both workloads:

```bash
cd ../docker-compose
docker compose up --build
```

To run standalone (requires a running SPIRE Agent):

```bash
export SPIFFE_ENDPOINT_SOCKET="unix:///path/to/workload_api.sock"
go run main.go
```

## Configuration

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `SPIFFE_ENDPOINT_SOCKET` | `unix:///tmp/agent.sock` | SPIRE Agent Workload API socket path |

The server listens on `0.0.0.0:8082` and expects JWTs with the audience `spiffe://example.nais.io/server`.

## Dependencies

- [`github.com/spiffe/go-spiffe/v2`](https://github.com/spiffe/go-spiffe) -- SPIFFE Workload API client, JWT-SVID parsing, and validation.
