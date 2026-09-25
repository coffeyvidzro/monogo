# Leamout architecture

Leamout Cloud provides identity, tenancy, programmable voice, BYOC carrier
connections, carrier routing, number inventory, and runtime media control.

## Commercial functionality removed

The commercial implementation (wallets, checkouts, payment attempts,
pay-as-you-go pricing, and usage charging) and its source migrations have
been removed. This repository currently does not provide an active payment
gateway, prepaid billing, or a telecom usage charging service.

BYOC calling remains supported through customer-owned carrier connections;
upstream carrier costs remain between the customer and their carrier.
Managed outbound calls are disabled until a new commercial authorization and
settlement system has been implemented and verified. Managed DID purchasing is
also disabled to prevent new unapproved wholesale obligations. Existing managed
resources and provider reconciliation records remain in the telecom control
plane and require operator oversight.

## Control plane

Identity and tenancy own users, organizations, authentication, and access
control. Telecom owns routing, calling, number provisioning, provider
integrations, and call lifecycle. Platform owns events, webhooks, audit events,
and infrastructure.

## Migration history and existing installations

Commercial source migrations 028–031 have been removed. Later control-plane
and telecom migrations retain their existing filenames and versions. Removing
migration source files from Git does not drop existing database tables, erase
financial records, or refund customers. A database that already applied
migrations 028–031 requires a separate reviewed upgrade and financial record
retention/reconciliation plan. Use the revised migration set only for a new
database until such a plan is available.
