# Examples

Example configurations and applications demonstrating SPIRE usage.

## Contents

| Directory | Description |
|-----------|-------------|
| [docker-compose/](docker-compose/) | End-to-end SPIRE setup with Docker Compose: server, agent, bootstrap, and two Go workloads communicating via JWT-SVID authentication. Best starting point for running a complete demo locally. |
| [workload-go-client/](workload-go-client/) | Go application that fetches a JWT-SVID from a SPIRE Agent and uses it as a Bearer token to call a server. |
| [workload-go-server/](workload-go-server/) | Go HTTP server that validates incoming JWT-SVIDs against a SPIRE trust bundle. |
| [kubernetes/](kubernetes/) | Kubernetes manifests for deploying an example workload that interacts with a cluster's SPIRE Agent via the SPIFFE CSI driver. |

## Getting Started

The quickest way to see everything in action is the Docker Compose example:

```bash
cd docker-compose
docker compose up --build
```

This spins up a SPIRE Server, Agent, and both Go workloads. The client fetches a JWT-SVID and sends an authenticated request to the server, which validates it and responds with `"Success!!!"`.

For Kubernetes environments, see the [kubernetes/](kubernetes/) directory.
