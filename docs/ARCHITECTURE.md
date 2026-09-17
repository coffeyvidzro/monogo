# Leamout Architecture

Leamout is a managed communications cloud. Leamout operates its control plane,
runtime cells, telecom edges, and the infrastructure that supports them.

Customers can connect their own carrier (BYOC), or use connectivity managed by
Leamout. The deployment model remains the same in both cases: customer traffic
is served by Leamout Cloud.

## Product model

Leamout supports two connectivity modes:

| Connectivity mode | Platform | Telecom relationship |
| --- | --- | --- |
| Cloud + BYOC | Prepaid PAYG | The customer selects a carrier and supplies its credentials; carrier charges remain outside Leamout-managed telecom usage |
| Cloud + Managed | Prepaid PAYG | Leamout selects and provisions connectivity; telecom usage is included in prepaid PAYG charging |

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
                         Leamout Cloud Runtime
                                   |
                          +--------+--------+
                          |                 |
                         BYOC            Managed
                          |                 |
                    customer carrier   Leamout supplies
```

## Control-plane domains

### Identity

Identity owns users, authentication, sessions, API credentials, organization
membership, roles, and permissions. Support access requires auditable, scoped
authorization.

### Commercial

Commercial owns plans, entitlements, prices, prepaid wallets, credit
reservations, rating, invoices, and payment-provider integration. Stripe and
Paystack are payment rails; Leamout's append-only ledger remains the source of
truth for prepaid credit.

Commercial is authoritative for Cloud PAYG. BYOC carrier costs remain between
the customer and the selected carrier, while Leamout rates Cloud platform
usage. Managed connectivity includes telecom usage in Leamout's rating and
charging path.

### Fleet

Fleet owns runtime-cell registration, identity, version, health, entitlement
synchronization, and revocation. Runtime cells use short-lived credentials and
are operated as part of Leamout Cloud.

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
- prepaid authorization for platform and managed telecom usage; and
- regional and regulatory constraints.

The routing engine consumes commercial authorization; it does not calculate or
mutate balances itself.

## Connectivity boundaries

### BYOC

The organization selects the carrier and supplies the connection credentials.
The carrier may be a third party or Leamout Carrier. Credentials are encrypted
inside the Cloud secret boundary and scoped to the serving runtime.

Leamout rates Cloud platform usage, but upstream telecom cost remains between
the customer and its selected carrier and is excluded from Leamout-managed
telecom usage.

### Managed Carrier

Leamout selects and provisions connectivity, is the commercial counterparty,
and includes telecom usage in prepaid PAYG charging. DIDWW, CommPeak, and future
provider credentials stay inside Leamout Cloud and are never distributed to
customers.

## Deployment boundaries

The Cloud platform includes the public communications API, routing, SIP and
media control, event production, webhooks, commercial services,
provider-orchestration, usage-ingestion, reconciliation, fraud control, and
fleet management.

The Docker Compose model in `deploy/compose.yaml` defines the Leamout Cloud
runtime and its telecom edge. Leamout operates this deployment and supplies its
production secrets, public addresses, DNS, and SIP/TURN certificates.

## Availability rules

- Cloud + BYOC depends on Leamout Cloud, but not managed-carrier telecom
  authorization; customer carrier charges remain external.
- Cloud + Managed requires current entitlement and prepaid commercial
  authorization for managed telecom usage.
- Usage delivery is idempotent and retryable. Managed usage is reconciled with
  carrier records before financial settlement is considered final.

## Implementation direction

1. Keep identity, tenancy, communications, routing, and runtime packages as the
   runtime-cell foundation.
2. Compose commercial, fleet, usage, and provider orchestration as explicit
   Cloud modules with separate configuration and deployment permissions.
3. Keep Stripe, Paystack, DIDWW, and CommPeak credentials in Cloud secret
   boundaries.
4. Model Cloud usage events, the append-only wallet ledger, credit reservations,
   and managed-resource assignments as distinct commercial concerns.
5. Deliver Cloud + BYOC before enabling Cloud + Managed.
