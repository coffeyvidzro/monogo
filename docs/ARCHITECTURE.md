# Leamout Architecture

Leamout is one communications platform delivered as two products:

- **Leamout Self-Hosted** runs in infrastructure operated by the customer.
- **Leamout Cloud** runs in infrastructure operated by Leamout and adds the
  commercial and fleet services required for a managed SaaS product.

They share communications domain contracts and runtime components, but they do
not share the same control-plane responsibilities.

## Product model

Runtime ownership and connectivity ownership are independent choices. Leamout
supports all four combinations:

| Runtime | Connectivity | Delivery mode |
| --- | --- | --- |
| Self-Hosted | Customer carrier account | Self-Hosted + BYOC |
| Self-Hosted | Leamout carrier account | Self-Hosted + Managed Carrier |
| Leamout Cloud | Customer carrier account | Leamout Cloud + BYOC |
| Leamout Cloud | Leamout carrier account | Leamout Cloud + Managed Carrier |

```text
                         Leamout Control Plane
                                   |
             +---------------------+---------------------+
             |                     |                     |
          Identity             Commercial              Fleet
             |                     |                     |
             +---------------------+---------------------+
                                   |
                             Organizations
                                   |
                    Communications Resources
                                   |
          +------------------------+------------------------+
          |                        |                        |
        Voice                    Messaging               Numbers
          |                        |                        |
          +------------------------+------------------------+
                                   |
                              Applications
                                   |
                    +--------------+--------------+
                    |              |              |
                  Calls         Messages       Number Flows
                    |              |              |
                    +--------------+--------------+
                                   |
                           Routing / Policy Engine
                                   |
                                   v
                                Runtime
                                   |
                 +-----------------+-----------------+
                 |                                   |
            Self-Hosted                         Leamout Cloud
                 |                                   |
          +------+------+                     +------+------+
          |             |                     |             |
        BYOC          Managed                BYOC          Managed
          |             |                     |             |
      customer       Leamout              customer       Leamout
      carrier        carrier              carrier        carrier
      account        service              account        service
```

"Managed" is deliberately a peer of "BYOC" under both runtimes. A
Leamout-managed carrier is not a customer-selected BYOC carrier.

## Control-plane domains

### Identity

Identity owns users, authentication, sessions, API credentials, organization
membership, roles, and permissions. It is present in both products. Cloud may
add centralized account and support identities without making a Self-Hosted
installation dependent on Cloud for local authentication.

### Commercial

Commercial owns plans, entitlements, prices, prepaid wallets, credit
reservations, rating, invoices, and payment-provider integration. Stripe and
Paystack are payment rails; Leamout's append-only ledger remains the source of
truth for prepaid credit.

Commercial is authoritative in Leamout Cloud. A Self-Hosted installation uses
it only when the customer enables a Leamout-managed service. Self-Hosted + BYOC
must continue operating when Leamout Cloud is unavailable.

### Fleet

Fleet owns registration, identity, version, health, entitlement synchronization,
and revocation for Self-Hosted installations and Cloud runtime cells. An
installation generates and retains its private key; Fleet stores its public
identity and issues short-lived credentials.

Fleet is not a remote administrator by default. Customer consent and auditable,
scoped authorization are required for support access or remote operations.

## Communications model

Organizations own communications resources. Voice, messaging, and numbers are
capabilities used by applications. Applications create calls, messages, and
number flows. The routing and policy engine resolves those operations onto an
eligible runtime and connectivity route.

Routing decisions must consider:

- organization and application policy;
- resource ownership and entitlement;
- BYOC versus managed connectivity;
- destination, capability, health, priority, and capacity;
- prepaid authorization for managed usage; and
- regional and regulatory constraints.

The routing engine consumes commercial authorization; it does not calculate or
mutate balances itself.

## Connectivity boundaries

### BYOC

The organization supplies carrier credentials and pays the carrier directly.
Credentials belong to the runtime serving that organization: locally encrypted
in Self-Hosted or held in the Cloud secret boundary for Leamout Cloud.

### Managed Carrier

Leamout is the commercial counterparty and supplies normalized connectivity.
DIDWW, CommPeak, and future provider credentials exist only inside Leamout
Cloud. Self-Hosted installations never receive Leamout master provider or
payment credentials.

For Self-Hosted + Managed Carrier, the local runtime authenticates to a narrow
Leamout commercial and carrier gateway. Cloud authorizes purchases and usage,
provisions provider resources, and returns installation-scoped assignments and
short-lived call authorization. Local BYOC traffic does not traverse this
gateway.

## Deployment boundaries

The shared runtime includes the public communications API, routing contracts,
SIP and media control, event production, webhooks, and BYOC support. Leamout
Cloud additionally runs commercial, provider-orchestration, usage-ingestion,
reconciliation, fraud-control, and fleet workloads.

Cloud-only provider adapters and secrets must not be enabled by an environment
mode in the Self-Hosted binary. They belong to separately composed Cloud
processes with separate configuration and deployment permissions.

The portable Kubernetes base in `deploy/kubernetes` is only a runtime-cell
foundation. It is not the complete Leamout Cloud product until the Cloud
control-plane services and provider-specific telecom edge are deployed.

## Availability rules

- Self-Hosted + BYOC has no call-path dependency on Leamout Cloud.
- Cloud + BYOC depends on the Leamout Cloud runtime, but not managed-carrier
  credit authorization.
- Managed Carrier requires current entitlement and commercial authorization.
- A Cloud outage may suspend new managed purchases or calls; it must not disable
  local Self-Hosted BYOC administration or traffic.
- Usage delivery is idempotent and retryable. Managed usage is reconciled with
  carrier records before financial settlement is considered final.

## Implementation direction

1. Keep the existing identity, tenancy, communications, routing, and runtime
   packages as the shared runtime foundation.
2. Add explicit Cloud modules for commercial, fleet, usage, and provider
   orchestration instead of putting those concerns in shared runtime modules.
3. Move Stripe, Paystack, DIDWW, and CommPeak configuration into Cloud-only
   process configuration.
4. Model installation identity, capabilities, managed-resource assignments,
   usage events, an append-only wallet ledger, and credit reservations.
5. Deliver Self-Hosted + BYOC and Cloud + BYOC first, then Cloud + Managed
   Carrier, and finally Self-Hosted + Managed Carrier.

