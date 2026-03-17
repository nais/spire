# Kubernetes Example

`workload.yaml` contains an example Kubernetes Deployment that spins up the a spire-agent container.
It can be used to debug and interact with the cluster's spire-agent.

The manifests don't specify a namespace.

See https://spiffe.io/docs/latest/deploying/spire_agent/#command-line-options for additional commands.

## Deploy the workload

```shell
kubectl apply -f workload.yaml
```

## Fetch X.509 SVID

```shell
kubectl exec -it $(kubectl get pods \
  -o=jsonpath='{.items[0].metadata.name}' \
  -l app=example-workload) \
  -- /opt/spire/bin/spire-agent \
    api fetch \
    -socketPath '/spiffe-workload-api/spire-agent.sock'
```

## Fetch JWT SVID

```shell
kubectl exec -it $(kubectl get pods \
  -o=jsonpath='{.items[0].metadata.name}' \
  -l app=example-workload) \
  -- /opt/spire/bin/spire-agent \
    api fetch jwt \
    -audience 'foo' \
    -socketPath '/spiffe-workload-api/spire-agent.sock'
```

## Clean up

```shell
kubectl delete -f workload.yaml
```

## Custom deployments

To use SPIRE in your own Deployment (e.g. with the [SPIFFE SDKs](https://spiffe.io/docs/latest/deploying/libraries/),
mount the SPIRE Agent socket in your container:

```yaml
spec:
  template:
    spec:
      containers:
        - ...
          volumeMounts:
            - name: spiffe-workload-api
              mountPath: /spiffe-workload-api
              readOnly: true
      ...
      volumes:
        - name: spiffe-workload-api
          csi:
            driver: "csi.spiffe.io"
            readOnly: true
```

The above example makes the socket available at `/spiffe-workload-api/spire-agent.sock`
