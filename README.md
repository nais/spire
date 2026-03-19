# spire

Proof-of-concept for evaluating [SPIRE](https://spiffe.io/docs/latest/spire-about/) (the SPIFFE Runtime Environment), which is the reference implementation of the [SPIFFE](https://spiffe.io/docs/latest/spiffe-about/overview/) (Secure Production Identity Framework For Everyone) standard for workload identity.

## Contents

- [Helm charts](charts/) for deploying a production-grade SPIRE infrastructure on Kubernetes (using the [official hardened Helm charts](https://spiffe.io/docs/latest/spire-helm-charts-hardened-about/))
- [Example workloads](examples/) that obtain SPIFFE Verifiable Identity Documents (SVIDs) from a SPIRE Agent, including a complete [end-to-end local development setup](examples/docker-compose/) with Docker Compose

## Architecture

### Deployed Components

The Helm chart deploys the following components into a Kubernetes cluster:

- **SPIRE Server**: Central authority that manages workload identities, stores registration entries, and issues SVIDs. Runs as a Deployment backed by PostgreSQL.
- **SPIRE Controller Manager**: Kubernetes-native controller that reconciles `ClusterSPIFFEID` and `ClusterFederatedTrustDomain` CRDs into SPIRE registration entries. Runs as a sidecar in the server pod.
- **SPIRE Agent**: DaemonSet that runs on every node. Performs workload attestation (verifying which workload is requesting an identity) and exposes the SPIFFE Workload API.
- **SPIFFE CSI Driver**: DaemonSet that mounts the Workload API Unix domain socket into workload pods via CSI ephemeral volumes.
- **OIDC Discovery Provider**: Serves OpenID Connect discovery documents and JWKS endpoints, enabling external services to validate JWT-SVIDs issued by this SPIRE deployment.
- **Tornjak**: Optional management UI and API for SPIRE (backend runs as a sidecar, frontend as a separate Deployment).

### Single Trust Domain per Cluster

This setup uses the default SPIRE architecture from the official Helm charts: one SPIRE Server per cluster, with a SPIRE Agent on every node. Each cluster has its own trust domain.

Workloads within the same cluster can authenticate to each other using either X.509-SVIDs (mTLS) or JWT-SVIDs.

Cross-cluster authentication is not possible with mTLS since the clusters have separate trust domains. Instead, cross-cluster communication uses OIDC federation: a workload in cluster A obtains a JWT-SVID from its local SPIRE Agent and presents it to a workload in cluster B, which validates it against cluster A's public keys via the OIDC discovery endpoint.

This is the architecture we would most likely adopt.

### Alternative Architectures

- **[Federation](https://spiffe.io/docs/latest/architecture/federation/readme/)**: Two or more SPIRE Servers with different trust domains exchange trust bundles so that workloads across domains can authenticate via mTLS. See also: [Helm chart federation guide](https://spiffe.io/docs/latest/spire-helm-charts-hardened-advanced/federation/).
- **[Nested SPIRE](https://spiffe.io/docs/latest/architecture/nested/readme/)**: SPIRE Servers are chained in a hierarchy sharing the same trust domain, suited for multi-cloud or multi-cluster deployments. See also: [Helm chart nested guide](https://spiffe.io/docs/latest/spire-helm-charts-hardened-advanced/nested-spire/).

## Use Cases

- Primary interest is in **JWT-SVIDs** for client authentication to internal and external platform services, and potentially for workload-to-workload authentication within a cluster.
- Limited need for **X.509-SVIDs** beyond potentially enabling mTLS with PostgreSQL.
- No need for trust bundle federation; OIDC federation covers our cross-cluster requirements.
- No need for workload identity outside Kubernetes; our platform runs entirely on Kubernetes.

## Strengths

- Graduated CNCF project (considered stable and used in production environments) with active development and a broad ecosystem
- Flexible architecture supporting many attestation methods (Kubernetes, TPMs, cloud provider VMs)
- Well-defined standards and APIs (SPIFFE) with support for both X.509 and JWT SVIDs
- Platform-agnostic design that works across Kubernetes, VMs, and bare metal

## Concerns

- Complex operational footprint with many moving parts: SPIRE Server, Controller Manager, Agent DaemonSet, and CSI Driver
- The official Helm charts are still under active development as of this writing. A respective amount of effort has been put into these charts, but they do sharp edges out of the box.
  - Default values do not prescribe a production-ready setup. High availability requires additional configuration.
  - See https://github.com/spiffe/helm-charts-hardened/issues/342 as an example.
- Required end-to-end mTLS between SPIRE components limits options for exposing federation endpoints out via an ingress and load balancers
- Requires privileged node access (Agent DaemonSet with `hostPID` and `hostNetwork`)
- JWT-SVIDs have minimal, non-customizable claims (unless a custom CredentialComposer plugin is implemented)
- SPIFFE identites is ultimately tied to a Kubernetes Service Account referenced by the workload.
  - Workload identities are again attested by an Agent that bootstraps and attests itself with the SPIRE Server by using its own Kubernetes Service Account Token. 
  - This makes the additional layers of indirection questionable when our platform and its workloads exclusively run on Kubernetes.

## Evaluation

SPIRE is overkill for our use cases. Its primary value proposition, platform-agnostic, attestation-based workload identity across heterogeneous environments, does not apply when our platform and all of its workloads run on Kubernetes.

### Alternative: Kubernetes Service Account Tokens

Kubernetes already provides a native equivalent through [bound service account tokens](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#bound-service-account-tokens) and [service account token volume projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#serviceaccount-token-volume-projection):

- **Bound tokens**: Short-lived JWTs issued by the API server, automatically rotated by the kubelet, and tied to the lifecycle of the pod. They expire when the pod is deleted.
- **Token volume projection**: Automatically mounts these tokens into pods with configurable audiences and expiration times.
- **OIDC discovery**: Endpoints (`/.well-known/openid-configuration` and `/openid/v1/jwks`) allow external services to validate these tokens without direct access to the Kubernetes API.

These native tokens provide comparable functionality to SPIRE JWT-SVIDs for Kubernetes-only environments: short-lived, audience-scoped JWTs with automatic rotation and OIDC-based external validation.

The following examples illustrate the structural differences between a JWT-SVID issued by SPIRE and a bound service account token issued by Kubernetes. Both are standard JWTs, but they differ significantly in the claims they carry.

**SPIRE JWT-SVID** (decoded payload):

```json
{
  "aud": [
    "foo"
  ],
  "exp": 1773858214,
  "iat": 1773854614,
  "iss": "https://example.nais.io",
  "sub": "spiffe://example.nais.io/ns/my-namespace/sa/my-workload"
}

```

JWT-SVIDs follow the [SPIFFE JWT-SVID standard](https://github.com/spiffe/spiffe/blob/main/standards/JWT-SVID.md), which mandates only three claims: `sub` (the SPIFFE ID), `aud` (audience), and `exp` (expiration). The `iat` and `iss` claims are optional. Custom claims are permitted by the spec but discouraged because they impact interoperability across SPIFFE-aware systems.

In practice, SPIRE issues tokens with just these five claims: the three required ones plus `iat` and `iss`.

The default SPIFFE ID pattern is `spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>`. This encodes the same namespace and service account information that Kubernetes uses natively.

The audience is an arbitrary string chosen by the requesting workload (in this case `"foo"`, passed as a parameter when fetching the token).

**Kubernetes bound service account token** (decoded payload):

```json
{
  "aud": [
    "https://container.googleapis.com/v1/projects/some-project/locations/some-location/clusters/some-cluster"
  ],
  "exp": 1805390756,
  "iat": 1773854756,
  "iss": "https://container.googleapis.com/v1/projects/some-project/locations/some-location/clusters/some-cluster",
  "jti": "7e086a2e-bc03-497f-b98f-e18e05525d23",
  "kubernetes.io": {
    "namespace": "my-namespace",
    "node": {
      "name": "gke-foo-bar-baz-node-pool-1-abcde",
      "uid": "6f5d06e0-a949-47c4-81bb-deb0c23002a0"
    },
    "pod": {
      "name": "my-workload-66d5ff6847-txzjn",
      "uid": "4d1c6a46-e180-44be-9a3b-62f18dc5f080"
    },
    "serviceaccount": {
      "name": "my-workload",
      "uid": "4033e269-9d96-44c8-a00d-224f8c31b799"
    },
    "warnafter": 1773858363
  },
  "nbf": 1773854756,
  "sub": "system:serviceaccount:my-namespace:my-workload"
}
```

Kubernetes bound tokens include rich contextual claims in the `kubernetes.io` namespace.
These claims encode more detailed information about the workload and its environment than SPIRE's minimal JWT-SVIDs, which only include the SPIFFE ID as the `sub` claim.

On GKE, the issuer and default audience are the cluster's resource URL rather than the generic `https://kubernetes.default.svc.cluster.local`, though the audience is configurable when projecting the token into the pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-workload
spec:
  containers:
    - image: my-workload
      name: my-workload
      volumeMounts:
        - mountPath: /var/run/secrets/tokens
          name: some-token
  volumes:
  - name: some-token
    projected:
      sources:
      - serviceAccountToken:
          path: some-token
          expirationSeconds: 600
          audience: some-audience
```

### Alternative: Kubernetes Pod Certificates (KEP-4317)

[KEP-4317: Pod Certificates](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/4317-pod-certificates/README.md) introduces native Kubernetes machinery for issuing X.509 certificates directly to pods. It reached beta in Kubernetes 1.35.

The KEP adds two primitives:

- **PodCertificateRequest**: A new API type in `certificates.k8s.io`, analogous to `CertificateSigningRequest` but scoped specifically to pods. It encodes the statement "pod X is requesting a certificate from signer Y" and includes the pod's full identity (namespace, pod name/UID, service account name/UID, node name/UID).
- **PodCertificate projected volume source**: A new projected volume type that instructs the kubelet to generate a private key, request a certificate from a named signer, write both into the pod filesystem, and automatically rotate the certificate before expiry.

The kubelet handles the entire lifecycle: key generation, certificate request, writing the credential bundle (private key + certificate chain in a single PEM file), and renewal. Pods block startup until their certificates are issued. The signer controls the certificate format, lifetime (1 hour to 91 days), and refresh schedule.

This is the X.509 counterpart to the bound service account token mechanism described above. Where bound SA tokens provide native JWT-based workload identity, Pod Certificates provide native X.509-based workload identity with mTLS support.

A pod using Pod Certificates looks like this:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-workload
spec:
  containers:
  - name: main
    image: my-app
    volumeMounts:
    - name: spiffe-credentials
      mountPath: /run/workload-spiffe-credentials
  volumes:
  - name: spiffe-credentials
    projected:
      sources:
      - podCertificate:
          signerName: "mysigner.example"
          keyType: ED25519
          credentialBundlePath: credentialbundle.pem
```

The KEP explicitly lists SPIFFE certificate issuance as a target use case for third-party signer implementations. A SPIFFE-compatible signer could issue X.509-SVIDs via this mechanism, replacing the need for a full SPIRE deployment (Server, Agent, CSI Driver) with a single signer controller and the built-in kubelet machinery.

For our use cases, Pod Certificates would eliminate the last remaining argument for SPIRE: native X.509 mTLS support. Combined with bound service account tokens for JWT-based identity, Kubernetes would natively cover both credential types that SPIRE provides.

## Conclusion

Native Kubernetes service account tokens with projected volumes are the simpler and more appropriate solution for our use cases. SPIRE adds significant operational complexity without proportional benefit in a Kubernetes-only environment.

Federation with other runtime environments such as on-premises or other cloud providers isn't a current concern as of this writing.
Such concerns could be solved by OIDC federation through a dedicated Security Token Service that supports token exchanges with federated client authentication, such as:

- Kubernetes Service Account Tokens
- SPIFFE JWT SVIDs
- OpenID Connect ID tokens from any compliant provider

If we need mTLS or X.509 certificate-based workload identity in the future, Pod Certificates (KEP-4317) provides a native Kubernetes path forward without requiring a full SPIRE deployment.
