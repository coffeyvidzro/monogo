# Leamout Cloud Kubernetes

This directory contains the Kubernetes deployment path for Leamout Cloud.

Cloud and Self-Hosted run the same application/runtime. Kubernetes is a deployment concern; the Go application does not switch into a separate cloud mode.

## Current MVP scope

The first cloud slice provides the shared control-plane dependencies:

- namespace
- application ConfigMap
- secret template
- PostgreSQL
- Redis
- NATS JetStream

Application workloads and telecom workloads are intentionally added separately so their runtime and networking contracts stay explicit.

## Apply

Create the real secret from `secret.example.yaml` without committing secret values, then apply the manifests:

```sh
kubectl apply -f deploy/kubernetes/secret.yaml
kubectl apply -k deploy/kubernetes
```

`secret.yaml` should remain local or be supplied by the cluster's secret-management mechanism.

## Next workloads

The next cloud step is to add the Atlas migration Job plus the `server` and `worker` workloads after their runtime entrypoints are ready for long-running Kubernetes execution.

OpenSIPS, FreeSWITCH, RTPengine, and Coturn are deliberately deferred to a separate telecom networking slice because SIP and media exposure have different requirements from the HTTP control plane.

Terraform is not part of the Leamout deployment model.
