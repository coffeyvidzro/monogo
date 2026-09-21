# Cloud + Managed Carrier acceptance

This suite establishes the Leamout-operated Cloud + Managed topology and proves
the customer-facing managed-number purchase workflow against a deterministic
fake DIDWW provider:

```text
customer API -> Leamout Cloud control plane -> provider operation worker
             -> fake managed-number provider -> cloud managed ingress
```

The suite proves inventory search, opaque selection, durable purchase intent,
asynchronous order execution, provider routing convergence, activation, and
suppression of provider/platform identifiers from customer responses. It then
binds the DID to a cloud voice application, receives a managed call through
local Cloud OpenSIPS and FreeSWITCH, and places a trunkless managed call through
the platform default route to a synthetic wholesale carrier. The outbound leg
uses the DIDWW-provisioned DID as caller ID on a CommPeak-attributed route, then
reconciles a provider CDR idempotently to one call and one wholesale charge.

The same gate also proves tenant B cannot read or use tenant A resources,
explicit trunk failure never falls back to managed routing, a missing managed
route cannot consume a tenant route, inactive numbers and organizations reject
inbound delivery, and mismatched/conflicting CDRs cannot alter wholesale cost.

Run it with:

```sh
sh tests/acceptance/cloud-managed/run.sh
```
