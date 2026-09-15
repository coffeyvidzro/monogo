# Leamout Self-Hosted

The self-hosted release bundle contains the Leamout installer plus the Docker Compose deployment assets, database migrations, and service configuration required to install without cloning the source repository.

## Requirements

- Linux host with Docker Engine and the Docker Compose plugin
- an apt-based distribution for automatic Certbot installation in the initial release
- public DNS records pointing directly to the host:
  - `api.<domain>`
  - `sip.<domain>`
  - `turn.<domain>`
- TCP ports `80`, `443`, `5060`, `5061`, and `5062` available
- UDP ports `443`, `5060`, `3478`, `5349`, `23000-32768`, and `49152-65535` available

Caddy manages HTTPS for `api.<domain>`. Leamout installs Certbot and obtains a Let's Encrypt certificate for `sip.<domain>` and `turn.<domain>` automatically. Caddy serves the ACME HTTP challenge for certificate issuance and renewal.

## Install

The intended installation command is:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo sh
```

The bootstrap script detects Linux and the supported CPU architecture, resolves the latest release unless `LEAMOUT_VERSION` is provided, downloads the matching release archive and SHA-256 checksum, verifies the archive, installs the `leamout` CLI to `/usr/local/bin/leamout`, and starts the interactive installer.

To install an exact release:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo env LEAMOUT_VERSION=1.0.0 sh
```

Release archives use this naming scheme:

```text
leamout_<version>_linux_<arch>.tar.gz
leamout_<version>_linux_<arch>.tar.gz.sha256
```

The initial release target is `linux/amd64`. Other architectures should only be published once all Leamout runtime images are available for them.

The installer asks only for:

- base domain
- public IPv4 address

The filesystem layout is a Leamout contract and is not configurable.

## Canonical filesystem

```text
/usr/local/bin/leamout

/opt/leamout/
├── current -> releases/<version>
└── releases/
    └── <version>/
        ├── VERSION
        ├── compose.yaml
        ├── Caddyfile
        ├── migrations/
        └── config/
            ├── nats-server.conf
            └── turnserver.conf

/etc/leamout/
├── leamout.env
├── certs/
│   ├── fullchain.pem
│   ├── privkey.pem
│   └── carrier-ca.pem        # optional
└── license/
    └── license.jwt           # when licensing is configured

/var/lib/leamout/
├── state/
│   └── installation.json
├── acme/
├── backups/
└── staging/

/var/log/leamout/
/run/leamout/
```

Docker named volumes hold service-owned persistent data such as PostgreSQL, Redis, NATS, recordings, and Caddy state.

The path roles are fixed:

- `/usr/local/bin/leamout` — executable CLI; contains no secrets
- `/opt/leamout` — versioned, replaceable, non-secret release assets
- `/etc/leamout` — operator configuration, runtime secrets, TLS material, and licensing
- `/var/lib/leamout` — durable Leamout-owned mutable host state
- `/var/log/leamout` — protected installer and operational logs
- `/run/leamout` — ephemeral runtime coordination

## Permissions and secrets

Leamout installation runs as root. Sensitive directories are created as `root:root` with mode `0700`. Secret-bearing files such as `/etc/leamout/leamout.env`, TLS private keys, license material, installation state, logs, and backups are mode `0600` unless a service requires a less restrictive public-certificate mode.

The following values must never be world-readable or stored under `/opt/leamout`:

- telecom or carrier credentials
- private keys
- activation credentials
- carrier credential encryption keys
- operator/API secrets
- database passwords
- TURN shared secrets
- other authentication or encryption material

`/etc/leamout/leamout.env` is persistent installation state. Back it up securely and never regenerate it during upgrades because it contains the carrier credential encryption key and other long-lived secrets.

## TLS renewal

Certbot keeps its own renewal state under `/etc/letsencrypt`. Renewed SIP/TURN certificates are deployed into `/etc/leamout/certs` through a root-owned deploy hook. The hook atomically replaces the deployed certificate and private key and restarts OpenSIPS and Coturn when those services are running.

`carrier-ca.pem` is optional. Publicly trusted carrier TLS uses the container system CA store by default; a carrier-specific CA file should only be installed when required by that carrier.

The application/runtime is the same Leamout runtime used by Cloud. Self-Hosted changes deployment and operations only; it does not introduce a separate application mode or BYOC/Managed behavior.
