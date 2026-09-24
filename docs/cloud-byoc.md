# Cloud + BYOC operations guide

## Scope

In Cloud + BYOC, Monogo operates the API, SIP edge, media plane, application
runtime, and call lifecycle. The organization owns its carrier account and
supplies the carrier endpoints and authentication material.

The supported signaling matrix is:

| Direction | Authentication | Transport |
| --- | --- | --- |
| Carrier to Monogo | Source IPv4/IPv6 CIDR | UDP, TCP, TLS |
| Carrier to Monogo | SIP digest | UDP, TCP, TLS |
| Monogo to carrier | Source IP / no digest | UDP, TCP, TLS |
| Monogo to carrier | SIP digest challenge | UDP, TCP, TLS |

TLS validates the carrier certificate with the system trust store. Install an
explicit carrier CA bundle at `/etc/opensips/tls/carrier-ca.pem` only when the
carrier uses a private CA.

## Provisioning order

Provision resources in this order so no number becomes routable through an
unvalidated connection:

1. Create or select a non-platform carrier provider.
2. Create the organization carrier connection.
3. Configure inbound authentication and source CIDRs, where applicable.
4. Configure outbound authentication.
5. Create a `byoc` trunk attached to the carrier connection.
6. Add at least one enabled endpoint for every outbound trunk.
7. Wait for the endpoint health state to become `healthy`.
8. Call `POST /v1/carrier-connections/{id}/validate`.
9. Import the E.164 number with `POST /v1/numbers/`.
10. Bind the number to an active voice application.
11. Run the inbound and outbound acceptance calls below.

Carrier credentials are write-only. API responses expose the username and a
boolean indicating that a secret exists, but never return the secret.
Digest configuration also requires the exact challenge realm advertised by the
carrier (outbound) or configured on the Monogo SIP edge (inbound). After
upgrading an existing deployment to the digest-credential migration, re-enter
legacy digest credentials once so their one-way HA1 values are populated.

## Connection validation

The validation endpoint checks:

- Digest credentials exist when digest authentication is selected.
- Inbound source-IP authentication has at least one CIDR.
- Inbound unauthenticated mode is reported as a warning.
- Disabled connections are reported as a warning.
- Active outbound/bidirectional trunks have an enabled outbound endpoint.
- At least one endpoint on each outbound trunk is not marked unhealthy.
- A connection without an active BYOC trunk is reported as a warning.

Warnings describe incomplete or intentionally unusual configurations. Errors
set `valid` to `false` and must be resolved before production traffic is sent.

## Endpoint health checks

The worker claims due endpoints and sends an unauthenticated SIP `OPTIONS`
request every 30 seconds. A syntactically valid SIP response, including `401`
or `407`, proves transport reachability. Digest authentication is exercised by
real call acceptance tests rather than by storing carrier secrets in the probe.

Default behavior:

- Probe timeout: 3 seconds.
- Failure threshold: 3 consecutive failures.
- Cooldown after threshold: 2 minutes.
- Maximum endpoints per pass: 100.
- TLS minimum: TLS 1.2 with hostname verification.

The endpoint API exposes health status, consecutive failures, the last SIP
response code, latency, error, and cooldown. Editing host, port, or transport
resets this state to `unknown` so the new destination is probed.

Routing excludes `unhealthy` endpoints. A new or recently edited endpoint is
eligible while its status is `unknown`; use connection validation and wait for
`healthy` when operating in strict production mode.

## Inbound acceptance tests

### Source-IP authentication

1. Configure the carrier's exact egress CIDRs. Avoid broad networks.
2. Send an INVITE for the imported E.164 number from an allowed source.
3. Confirm it reaches the assigned voice application.
4. Repeat from an address outside every configured CIDR; it must not enter the
   carrier route.
5. Configure overlapping CIDRs on two connections at equal specificity and
   confirm the edge fails closed rather than choosing an arbitrary tenant.

### Digest authentication

1. Configure a unique inbound username and secret.
2. Send an INVITE without credentials and confirm the edge challenges it.
3. Retry with valid credentials and confirm routing succeeds.
4. Retry with an invalid secret and confirm routing fails.
5. Confirm an authenticated identity cannot route a number owned by another
   organization.

For both methods, test malformed and unknown called numbers. Carrier DIDs must
arrive in canonical E.164 format (`+[country code][subscriber number]`); the SIP
edge intentionally does not guess provider-specific number formats.

## Outbound acceptance tests

### Source-IP / no digest

1. Allow the deployment's public signaling address at the carrier.
2. Place a call through the BYOC trunk.
3. Confirm the selected destination exactly matches an enabled endpoint stored
   for that trunk.
4. Confirm the carrier receives the expected asserted identity.

### Digest challenge

1. Configure the outbound username and secret supplied by the carrier.
2. Place a call and confirm the first request receives `401` or `407`.
3. Confirm OpenSIPS retries the same authorized next hop with credentials.
4. Confirm an invalid secret fails without looping or selecting another tenant's
   endpoint.

For both methods, verify answer, media in both directions, hangup from each side,
call lifecycle completion, admission counter release, and recording completion
when recording is enabled.

## End-to-end release gate

Run at least these calls for every carrier/transport combination before enabling
customer traffic:

| Scenario | Expected result |
| --- | --- |
| Allowed inbound IP, known DID | Application answers |
| Disallowed inbound IP | Rejected before tenant routing |
| Valid inbound digest, known DID | Application answers |
| Invalid inbound digest | Authentication failure |
| Inbound known carrier, unknown DID | `404` |
| Outbound no-auth | Carrier accepts call |
| Outbound valid digest | Challenge retry succeeds |
| Outbound invalid digest | Call fails without a loop |
| Disabled connection or trunk | No route |
| Unhealthy endpoint | Endpoint is not selected |
| Concurrent-call limit reached | Additional call is rejected |
| Call ends from either leg | State and admission counters reconcile |

Record the application call UUID, SIP Call-ID, carrier connection ID, trunk ID,
endpoint ID, timestamps, and final SIP status for every acceptance call. These
identifiers will also form the correlation contract for Homer/SIPcapture.

## Troubleshooting

- **Validation reports no source CIDR:** add at least one CIDR or select digest.
- **Endpoint remains unknown:** ensure the worker is running and DNS/network
  policy permits the worker to reach the endpoint.
- **OPTIONS gets 401/407:** this is healthy; the endpoint is reachable.
- **TLS probe fails:** confirm hostname, SNI, certificate chain, and carrier CA.
- **Inbound returns 404:** confirm E.164 formatting, active number, carrier
  connection, voice binding, voice application, and organization.
- **Outbound returns 403 at the edge:** confirm the route URI corresponds to an
  enabled endpoint on the selected carrier connection.
- **Digest retry loops:** confirm the upstream realm and credentials and inspect
  OpenSIPS transaction logs before retrying production traffic.

Never log or copy digest secrets into tickets, SIP traces, or acceptance-test
artifacts.
