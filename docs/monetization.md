# Monetization

Leamout is the carrier-grade runtime and control plane for autonomous voice agents.

The commercial model should preserve one architectural principle: **Leamout Core, Leamout Cloud, and Leamout Enterprise run the same voice-agent runtime.** The products differ primarily in who operates the control plane, how runtime fleets are managed, and the level of support, governance, and deployment isolation provided.

Leamout is not a managed carrier and does not resell telephony as part of the core product model. Customers connect their own SIP trunks, carriers, PBXs, SBCs, and AI providers.

## Product segmentation

| Dimension | Leamout Core | Leamout Cloud | Leamout Enterprise |
| --- | --- | --- | --- |
| Primary user | Developers and teams operating Leamout themselves | Product teams that want a managed control plane | Organizations requiring a private control plane and enterprise operations |
| Runtime | Customer-operated | Customer/edge runtime nodes managed from Leamout Cloud | Customer-operated private runtime fleet |
| Control plane | Local | Leamout-hosted | Customer-hosted private control plane |
| Deployment | Docker, VM, bare metal, local infrastructure | Hosted SaaS control plane connected to customer runtimes | Customer VPC, Kubernetes, private cloud, or on-premises |
| Telephony | BYOC SIP trunks | BYOC SIP trunks | Enterprise SIP, SBC, PBX, and carrier connectivity |
| AI providers | Customer credentials | Customer credentials | Customer credentials or private/local models |
| Recording storage | Bundled MinIO or operator-configured S3-compatible storage | Leamout-managed storage or organization BYOS | Customer-controlled storage |
| Observability | Local | Hosted fleet and conversation observability | Private observability |
| Runtime management | Local/manual | Managed fleet control plane | Private fleet control plane |
| Support | Community | Standard commercial support | Enterprise support and deployment assistance |
| Commercial model | Free software distribution | Subscription | Annual enterprise contract |

## 1. Leamout Core

Leamout Core is the self-hosted voice-agent runtime and developer adoption layer.

A developer or team runs the runtime themselves:

```text
SIP trunk / PBX / softphone
        ↓
OpenSIPS
        ↓
RTPengine
        ↓
FreeSWITCH
        ↓
Media Runtime
        ↓
Agent Runtime
        ↓
STT / LLM / TTS or Realtime provider
```

The core runtime includes the infrastructure needed to build and operate autonomous voice agents without depending on Leamout Cloud for the call path.

Core capabilities include:

- SIP trunk connectivity;
- OpenSIPS signaling;
- RTPengine media anchoring;
- FreeSWITCH call and media control;
- Media Runtime;
- Agent Runtime;
- STT, LLM, TTS, and Realtime provider SDK contracts;
- tool execution;
- recording;
- events and telemetry;
- PostgreSQL, Redis, and NATS-based runtime services;
- bundled MinIO recording storage for the default self-hosted deployment.

### Commercial role

Leamout Core is the adoption surface.

The goal is to make the runtime easy to evaluate, develop against, and operate without requiring a commercial relationship with Leamout.

The software distribution is free. Operators remain responsible for their own infrastructure, carrier charges, AI-provider charges, storage, and related operating costs.

The exact open-source or source-available license is a separate product/legal decision and is intentionally not defined by this document.

## 2. Leamout Cloud

Leamout Cloud is the hosted control plane for production Leamout runtime fleets.

The Cloud product should not turn Leamout into another CPaaS that owns the carrier or media path. Instead, Leamout Cloud manages runtimes that execute close to the customer's telephony and infrastructure boundary.

```text
                         Leamout Cloud
                              │
              ┌───────────────┼───────────────┐
              │               │               │
          Runtime A       Runtime B       Runtime C
              │               │               │
          SIP / RTP       SIP / RTP       SIP / RTP
              │               │               │
        voice agents     voice agents     voice agents
```

A customer runtime may contain:

```text
OpenSIPS
RTPengine
FreeSWITCH
Media Runtime
Agent Runtime
```

while Leamout Cloud provides the hosted management layer.

### Cloud responsibilities

Leamout Cloud may provide:

- organization and user management;
- agent definitions and configuration;
- provider configuration and credential management;
- runtime registration;
- runtime fleet inventory;
- deployment and configuration distribution;
- runtime health and version visibility;
- call and conversation telemetry;
- latency and failure observability;
- event and audit history;
- recording metadata;
- Leamout-managed recording storage;
- optional organization BYOS;
- API keys and access control;
- hosted dashboard and operational tooling.

### Data and media boundary

The target model keeps the latency-sensitive voice path in the customer runtime:

```text
SIP / RTP
    ↓
Media Runtime
    ↓
Agent Runtime
    ↓
AI provider
```

The Cloud control plane receives the configuration, state, events, health, and telemetry required to manage those runtimes.

Specific data-retention and telemetry policies should remain configurable as the product matures.

### Cloud storage

For recording storage, the Cloud model is:

```text
Leamout Cloud
├── Default: Leamout-managed recording storage
│   └── bundled MinIO
│       └── organization prefix isolation
│
└── Optional: BYOS
    └── organization's S3-compatible storage
```

BYOS remains customer-owned and separate from Leamout-managed storage.

### Commercial model

The initial Cloud commercial model is **subscription-led**.

Subscription tiers can eventually reflect capabilities such as:

- number of organizations or projects;
- runtime fleet size;
- concurrency or capacity bands;
- retention and observability features;
- team and governance features;
- support level.

Leamout should not reintroduce telecom wallet charging, carrier resale, or a telecom usage-rating system as part of this model.

Per-minute orchestration pricing may be evaluated later if operating data shows that usage-based pricing is necessary. It is not a requirement for the initial Cloud model.

Customers continue to pay their carrier and AI-provider costs directly.

## 3. Leamout Enterprise

Leamout Enterprise packages the control plane and runtime fleet for organizations that require the entire system inside their own infrastructure boundary.

```text
Customer private environment

Private Control Plane
├── API
├── PostgreSQL
├── Redis
├── NATS
├── dashboard
├── audit / observability
└── runtime management

Runtime Fleet
├── OpenSIPS
├── RTPengine
├── FreeSWITCH
├── Media Runtime
└── Agent Runtime
```

The deployment may run in:

- a customer VPC;
- private Kubernetes;
- on-premises infrastructure;
- regulated or disconnected environments where supported.

Enterprise telephony remains standards-based:

```text
carrier ─────┐
enterprise SBC ─┤
Cisco / Avaya PBX ─┤
other SIP systems ─┘
           ↓
        SIP trunk
           ↓
        Leamout
```

Vendor-specific adapters should only be introduced where a concrete interoperability requirement cannot be handled through SIP, SBC, or existing enterprise telephony standards.

### Enterprise capabilities

Enterprise packaging may add commercial capabilities around the same runtime, including:

- private control-plane deployment;
- multi-node runtime orchestration;
- enterprise RBAC and governance;
- audit and retention controls;
- private recording storage;
- private/local AI-model connectivity;
- SSO and enterprise identity integration;
- deployment automation;
- upgrade and release management;
- high-availability guidance;
- operational support;
- security and compliance documentation;
- support SLAs.

### Commercial model

Leamout Enterprise is sold through an annual software and support contract.

Pricing can be based on agreed capacity bands, deployment scope, support requirements, and enterprise features rather than telecom consumption.

Exact pricing bands should be defined from real customer requirements rather than hard-coded into the architecture.

## Commercial boundaries

Leamout's monetization model should keep product responsibility clear.

```text
Customer pays carrier
Customer pays AI providers
Customer owns BYOC connectivity
Customer may own storage through BYOS

Leamout charges for:
    managed control-plane software
    fleet management
    observability
    governance
    enterprise deployment
    support
```

This keeps Leamout focused on the runtime and control plane instead of rebuilding the economics of a carrier or AI reseller.

## Product progression

The three products should be viewed as progressively managed forms of the same system:

```text
                    Leamout Runtime
                          │
             ┌────────────┼────────────┐
             │            │            │
           Core         Cloud      Enterprise
             │            │            │
        local control   hosted       private
           plane       control       control
                        plane         plane
             │            │            │
          runtime       runtime       runtime
```

### Core

The developer or operator manages everything.

### Cloud

Leamout operates the control plane and manages registered customer runtime fleets.

### Enterprise

The customer operates both the private control plane and runtime fleet, with Leamout providing the enterprise software package and support relationship.

## Monetization principles

1. **Keep the voice runtime portable.** The same core runtime should operate across Core, Cloud, and Enterprise.
2. **Monetize management, not carrier resale.** BYOC remains the telephony model.
3. **Do not own AI spend by default.** Provider credentials and model costs remain customer-controlled.
4. **Keep Cloud subscription-first.** Avoid rebuilding telecom-style metering before the business requires it.
5. **Use annual contracts for private enterprise deployments.** Price around capacity, operational scope, support, and governance requirements.
6. **Keep storage ownership explicit.** Cloud may provide managed recording storage, while BYOS and Enterprise storage remain customer-controlled options.
7. **Avoid vendor-specific lock-in.** SIP, provider SDK contracts, and S3-compatible storage remain the primary interoperability boundaries.
8. **Separate licensing from monetization architecture.** The exact Core license can evolve without changing the product segmentation described here.

## Near-term commercial priority

Engineering should prioritize the runtime and control-plane capabilities that create value across all three products:

```text
provider-neutral Voice Agent Runtime
        ↓
reliable realtime execution
        ↓
runtime registration and control
        ↓
fleet observability
        ↓
Cloud management plane
        ↓
private Enterprise packaging
```

Billing complexity should follow proven product demand rather than lead the architecture.
