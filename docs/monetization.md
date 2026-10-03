# Deployment and monetization

Leamout is a voice-agent platform that can be self-hosted or used as a managed
cloud service. The underlying voice-agent platform and product concepts are the
same in both deployment models:

```text
Leamout
├── Self-Hosted
└── Cloud
```

Enterprise is optional commercial packaging for either deployment model. It is
not a third product architecture, runtime fleet, or separate control plane.

Leamout is BYOC. Customers connect and pay for their own SIP trunks, carriers,
PBXs, SBCs, and AI providers. Leamout does not resell carrier services or
introduce itself into the customer's carrier billing relationship.

## One voice-agent platform

Both deployment models expose the same core execution path:

```text
SIP Trunk
    ↓
Call
    ↓
Voice Agent
    ↓
Realtime Media
    ↓
AI Providers
    ↓
Tools
    ↓
Recording
    ↓
Conversation History
    ↓
Observability
```

The deployment choice determines who operates Leamout's infrastructure. It
does not create a different Voice Agent, Media Runtime, Agent Runtime, provider,
or telephony architecture.

## Leamout Self-Hosted

With Leamout Self-Hosted, a customer or operator deploys and runs Leamout on
their own infrastructure. It is suitable for developers, teams, and
organizations that want full control, and it can operate without Leamout Cloud
in the call path.

A self-hosted stack may include:

- OpenSIPS;
- RTPengine;
- FreeSWITCH;
- Media Runtime;
- Agent Runtime;
- PostgreSQL;
- Redis;
- NATS JetStream;
- MinIO.

Operators connect their own SIP trunks and carrier accounts, supply their own
AI-provider credentials, and may configure an external S3-compatible recording
store. They are responsible for infrastructure, carrier, AI-provider, and
storage costs.

The software may be distributed as free/open-source or source-available
software. The exact license is a separate product and legal decision and is not
defined here.

## Leamout Cloud

Leamout Cloud is Leamout operated and hosted by us. Customers use the managed
service rather than operating its underlying infrastructure.

```text
Customer
    ↓
Leamout Cloud
├── Voice Agents
├── SIP Trunks
├── Calls
├── Recordings
├── Tools
├── AI Providers
├── Conversation History
└── Observability
```

Leamout Cloud also provides organization and user management. Customers do not
need to run customer-side nodes: we operate the infrastructure beneath these
product concepts, including the established SIP, media, and agent execution
components.

Cloud is subscription-led. Subscription packaging may vary by product
capabilities, capacity, retention, governance, and support, but this document
does not define exact prices. Customers continue to connect and pay their own
carrier and AI-provider accounts.

## Enterprise packaging

Enterprise is an optional commercial contract layered on Leamout Cloud or
Leamout Self-Hosted. Depending on customer requirements, it may include:

- SSO;
- advanced RBAC;
- audit logging;
- longer retention;
- deployment assistance;
- private networking support;
- security and compliance documentation;
- support SLAs;
- high-availability guidance;
- custom contractual terms.

These capabilities do not create an Enterprise runtime architecture or an
Enterprise control plane. An Enterprise customer chooses Cloud or Self-Hosted
and adds the appropriate support, security, governance, and contractual terms.

## AI providers

AI-provider credentials remain organization-scoped and customer-owned in both
deployment models unless a future commercial offering explicitly changes that
policy. Voice Agents bind credentials for STT, LLM, TTS, or Realtime providers;
provider-specific settings remain part of the agent/provider configuration.

```text
Organization
    ↓
AI Provider Credentials
    ↓
Voice Agent Provider Bindings
├── STT
├── LLM
├── TTS
└── Realtime
    ↓
Voice Agent Session
    ↓
Media Runtime
```

Leamout does not move these credentials into deployment-global environment
variables as part of the Cloud or Self-Hosted distinction.

## Recording storage

Recording storage preserves the existing managed-storage and tenant BYOS
boundaries.

### Self-Hosted

```text
Leamout Self-Hosted
├── Default: bundled local MinIO
└── Optional: operator-configured external S3-compatible storage
```

### Cloud

```text
Leamout Cloud
├── Default: Leamout-managed recording storage
│   └── bundled MinIO
│       └── organization prefix isolation
└── Optional: organization BYOS
    └── customer's S3-compatible storage
```

Tenant BYOS is organization-owned. Credentials remain encrypted, endpoints
must be public HTTPS S3-compatible endpoints, and SSRF protections remain in
force. Operator configuration of the Self-Hosted default store remains
separate from an organization's tenant-scoped BYOS integration. See
[storage.md](storage.md) for the complete security model.

## Commercial boundaries

```text
Customer pays carrier
Customer pays AI providers
Customer owns BYOC connectivity
Customer may own recording storage through BYOS

Leamout may charge for:
    Cloud subscriptions
    Enterprise support and SLAs
    security and governance features
    deployment assistance
    custom contractual requirements
```

Leamout's commercial model does not include:

- telecom wallet charging;
- telecom consumption billing;
- managed carrier resale;
- retail carrier rates;
- an online charging system (OCS);
- per-minute telecom billing infrastructure.

This boundary keeps monetization focused on operating the platform and on
optional support, security, and governance rather than recreating carrier
commerce.
