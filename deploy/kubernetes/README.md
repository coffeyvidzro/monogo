# Leamout Cloud Kubernetes

This directory contains the Kubernetes deployment path for Leamout Cloud.

Cloud and Self-Hosted run the same application/runtime. Kubernetes is a deployment concern; the Go application does not switch into a separate cloud mode.

## Architecture

The base installs the durable control-plane dependencies and the application
processes that are shared with Self-Hosted:

- namespace
- application ConfigMap
- secret template
- PostgreSQL
- Redis
- NATS JetStream
- an immutable Atlas migration Job
- FreeSWITCH's private ESL control endpoint
- two API server replicas behind a ClusterIP Service
- one asynchronous worker replica

Application workloads and telecom workloads are intentionally added separately so their runtime and networking contracts stay explicit.

## Apply

Create the real secret from `secret.example.yaml` without committing secret values. Apply the namespace and secret first, then the base. Wait for migrations before allowing an application rollout to receive traffic:

```sh
kubectl apply -f deploy/kubernetes/namespace.yaml
kubectl apply -f deploy/kubernetes/secret.yaml
kubectl apply -k deploy/kubernetes
kubectl wait --for=condition=complete job/leamout-migrate -n leamout --timeout=5m
kubectl rollout status deployment/leamout-server -n leamout
kubectl rollout status deployment/leamout-worker -n leamout
```

`secret.yaml` should remain local or be supplied by the cluster's secret-management mechanism.

The `preview` image tags are safe defaults for development only. Production
overlays must pin all Leamout images to the same release tag (or digest), set a
unique deployment ID and public URLs, provide an ingress for
`leamout-server:8080`, and use a cluster secret manager for credentials.

## Telecom boundary

FreeSWITCH is included only for the private ESL contract required by the API and
worker. OpenSIPS, RTPengine, Coturn, SIP ingress, media port ranges, recordings,
and public DNS belong in a provider-specific telecom overlay. They are not safe
to model as portable ClusterIP services because public IP advertisement, UDP
load-balancer behavior, topology, and certificate provisioning vary by cloud.

The Docker Compose deployment remains the reference all-in-one Self-Hosted
topology. This Kubernetes base is the portable Cloud control plane; overlays
own the public HTTP and telecom edges. Both paths use the same application
images, migrations, configuration contract, and runtime processes.

Terraform is not part of the Leamout deployment model.
