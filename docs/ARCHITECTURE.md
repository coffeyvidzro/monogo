# Leamout Architecture

Leamout is one communications platform delivered as two products:

- **Leamout Self-Hosted** runs in infrastructure operated by the customer.
- **Leamout Cloud** runs in infrastructure operated by Leamout and adds the
  commercial and fleet services required for a managed SaaS product.

They share communications domain contracts and runtime components, but they do
not share the same control-plane responsibilities.

## Product model

Leamout supports three delivery modes. Self-Hosted is always BYOC: the customer
selects and configures the carrier, which may be a third party or Leamout
Carrier. "Managed" specifically means Cloud owns telecom selection,
provisioning, rating, and charging.

| Delivery mode | Software / platform | Telecom relationship |
| --- | --- | --- |
| Self-Hosted + BYOC | Enterprise Self-Hosted license | Customer-selected carrier; may be a third party or Leamout Carrier |
| Leamout Cloud + BYOC | Prepaid PAYG | Customer-selected carrier cost remains outside Cloud-managed usage |
| Leamout Cloud + Managed | Prepaid PAYG | Leamout-managed telecom usage is prepaid PAYG |

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
                BYOC                           +------+------+
                 |                             |             |
       +---------+---------+                 BYOC          Managed
       |         |         |                  |             |
    carrier   carrier   Leamout           customer       Leamout
       A         B      Carrier           carrier        supplies
```

The provider's brand does not determine the mode. When a Self-Hosted customer
selects Leamout Carrier, the relationship remains BYOC from the software's
perspective: the customer configures the carrier relationship and the
Enterprise runtime does not become Cloud-managed.

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

Commercial is authoritative for Cloud PAYG. Self-Hosted uses an Enterprise
license rather than the Cloud wallet and rating path. Any separate agreement
with Leamout Carrier is a carrier relationship, not a switch into the Cloud
Managed delivery mode. Self-Hosted must continue operating when Leamout Cloud
is unavailable.

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
- prepaid authorization for Cloud platform and managed telecom usage; and
- regional and regulatory constraints.

The routing engine consumes commercial authorization; it does not calculate or
mutate balances itself.

## Connectivity boundaries

### BYOC

The organization selects the carrier and supplies the connection credentials.
The carrier may be a third party or Leamout Carrier. Credentials belong to the
runtime serving that organization: locally encrypted in Self-Hosted or held in
the Cloud secret boundary for Leamout Cloud.

In Cloud + BYOC, Leamout rates and charges Cloud platform usage through prepaid
PAYG, but upstream telecom cost remains between the customer and its selected
carrier and is excluded from Leamout-managed telecom usage.

### Managed Carrier

Managed Carrier is available in Leamout Cloud. Leamout selects and provisions
connectivity, is the commercial counterparty, and includes telecom usage in
prepaid PAYG charging. DIDWW, CommPeak, and future provider credentials exist
only inside Leamout Cloud and are never distributed to customers.

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

- Self-Hosted + BYOC has no Cloud call-path or prepaid-wallet dependency,
  including when the customer selects Leamout Carrier.
- Cloud + BYOC depends on the Leamout Cloud runtime, but not managed-carrier
  telecom authorization; customer carrier charges remain external.
- Cloud + Managed requires current entitlement and prepaid commercial
  authorization for managed telecom usage.
- A Cloud outage must not disable local Self-Hosted administration or traffic.
- Usage delivery is idempotent and retryable. Managed usage is reconciled with
  carrier records before financial settlement is considered final.

## Implementation direction

1. Keep the existing identity, tenancy, communications, routing, and runtime
   packages as the shared runtime foundation.
2. Add explicit Cloud modules for commercial, fleet, usage, and provider
   orchestration instead of putting those concerns in shared runtime modules.
3. Move Stripe, Paystack, DIDWW, and CommPeak configuration into Cloud-only
   process configuration.
4. Model installation identity and Enterprise entitlements separately from
   Cloud usage events, the append-only wallet ledger, credit reservations, and
   managed-resource assignments.
5. Deliver Self-Hosted + BYOC and Cloud + BYOC first, then Cloud + Managed.
