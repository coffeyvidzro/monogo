# Security Policy

Security is a core requirement for Leamout. Monogo handles telephony signaling, realtime media, AI-provider credentials, tenant-scoped configuration, tool execution, and other infrastructure that may sit directly in a production voice path.

Please report security vulnerabilities privately and give maintainers a reasonable opportunity to investigate and remediate them before public disclosure.

## Supported versions

Monogo is under active development. Security fixes are applied to the current `main` branch unless a maintained release line is explicitly documented.

| Version | Supported |
| --- | --- |
| `main` | Yes |
| Older commits, previews, and unmaintained branches | No |

## Reporting a vulnerability

Do **not** open a public GitHub issue, discussion, or pull request for an undisclosed vulnerability.

Report security issues privately by emailing:

**security@leamout.com**

Include as much of the following as you can:

- a concise description of the vulnerability;
- the affected component, endpoint, protocol, or runtime path;
- the commit, branch, or release you tested;
- reproduction steps or a minimal proof of concept;
- the security impact and realistic attack scenario;
- relevant logs, traces, SIP messages, requests, or screenshots with secrets removed;
- any remediation ideas you have already identified.

Never include production API keys, carrier credentials, SIP passwords, access tokens, private keys, customer audio, or other third-party secrets in a report.

## Security-sensitive areas

Examples of issues that should be reported privately include:

- authentication or authorization bypasses;
- cross-organization or cross-tenant data access;
- exposure of AI-provider, SIP-trunk, database, webhook, or infrastructure credentials;
- unsafe credential storage, encryption, rotation, or secret handling;
- SIP authentication bypass, registration abuse, spoofing, or routing-policy bypass;
- unauthorized call control, recording access, or media-session attachment;
- media injection, session hijacking, or isolation failures in realtime voice paths;
- remote code execution, command injection, SQL injection, SSRF, or path traversal;
- tool-execution authorization escapes or untrusted model actions reaching unauthorized systems;
- webhook signature verification failures or replay vulnerabilities;
- privilege escalation between users, organizations, services, or runtime components;
- vulnerabilities that permit meaningful denial of service against the control plane, telephony runtime, or media runtime.

## Safe research guidelines

Only test systems and accounts you own or have explicit permission to test.

Do not:

- place abusive or unauthorized calls;
- probe third-party carriers, PBXs, SIP infrastructure, or customer systems without permission;
- access, retain, or publish other users' data;
- degrade production availability;
- perform social engineering, phishing, or physical attacks;
- use a vulnerability to move beyond the minimum access needed to demonstrate impact.

When testing telephony or media behavior, use isolated numbers, trunks, credentials, and environments wherever possible.

## Disclosure process

After receiving a report, maintainers will validate the issue, assess its impact, coordinate a fix, and determine an appropriate disclosure timeline with the reporter when possible.

Please avoid public disclosure until a fix is available or maintainers have confirmed that disclosure can proceed.

## Security hardening

Security-related changes should preserve Leamout's architectural boundaries:

- customer-owned SIP connectivity remains isolated by organization;
- secrets must not be embedded in logs, events, snapshots, or client-visible payloads;
- the live audio path must not expose unrestricted control-plane access;
- models and tool calls must not receive unrestricted infrastructure or database access;
- authorization must be enforced at the server boundary and not delegated to UI behavior;
- provider and trunk credentials must remain scoped to the owning organization.

Security fixes should include regression tests whenever practical.
