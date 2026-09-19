# CommPeak managed-carrier SIP configuration

CommPeak supplies Leamout-managed termination and optional origination; DIDWW remains the initial Leamout-managed DID acquisition and DID inbound provider.

## Provisioning and activation

1. Confirm SIP hostnames, transport, port, source/destination IP ranges, carrier authentication, codecs, number presentation, and approved caller IDs from the actual CommPeak account. This repository does not assert any account-specific SIP destination or source network.
2. Use existing internal platform operations to create an active `commpeak` carrier provider, a platform-scoped connection (organization ID NULL), and a platform-managed outbound trunk. Start disabled. Set `managed_default` only after verifying connectivity, authentication and commercial authorization.
3. Validate termination endpoints with `commpeak.TerminationConfig.Validate()`; populate `trunk_endpoints` using existing `CreatePlatformTrunkEndpoint` and keep each endpoint disabled until verified. Configure outbound credentials in the protected platform credential store when digest is required. Do not store them in customer trunk records.
4. Verify `commpeak.OriginationConfig.Validate()` and keep the account-confirmed source CIDRs ready for the future backoffice-managed platform provisioning workflow. `CreateCarrierConnectionSourceIP` remains restricted to organization-owned BYOC connections and must not be used with a NULL organization ID. Inbound authorization remains disabled until a separately authorized platform operation persists provider IPs and the DID format, ownership/binding and SIP transport checks pass.
5. OpenSIPS authorizes a FreeSWITCH-provided carrier URI against the active selected connection and enabled outbound endpoint before forwarding. The existing carrier ingress flow matches inbound source networks to exactly one eligible connection and then matches the DID to that connection and a voice binding.
6. Confirm that the platform outbound credential view used by OpenSIPS exists and contains the expected scoped, non-plaintext auth material; this patch does not create or deploy that view. Exercise real SIP INVITE, challenge, media, source allowlist and negative authorization tests before activation.

## Safety boundaries

The CommPeak HTTP API client is not a SIP call controller. Routing and admission remain in Leamout; FreeSWITCH and OpenSIPS handle SIP signaling. Provider records and source networks require internal authorization and do not belong on customer-facing BYOC management endpoints. The existing source-network insertion query remains organization-scoped and cannot create platform-owned networks. A separate, internally authorized platform operation belongs in the future backoffice. Do not use raw CDRs to debit customer balances before the account-specific CDR contract is confirmed.
