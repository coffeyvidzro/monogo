# Leamout Self-Hosted

The self-hosted release bundle contains the Leamout installer plus the Docker Compose deployment assets and service configuration required to install without cloning the source repository. Database migrations are versioned inside the matching `leamout-migrate` image so Self-Hosted and Cloud execute the exact same schema artifact.

Self-Hosted is a distinct product operated in customer infrastructure, not a
single-node edition of Leamout Cloud. It is licensed as Enterprise Self-Hosted
and uses BYOC connectivity selected and configured by the customer. That
carrier may be a third party or Leamout Carrier; selecting Leamout Carrier does
not turn the installation into the Cloud Managed delivery mode.

## Requirements

- Linux host with Docker Engine and the Docker Compose plugin
- an apt-based systemd distribution for automatic Certbot installation and renewal in the initial release
- public IPv4 DNS records pointing directly to the host:
  - `api.<domain>`
  - `sip.<domain>`
  - `turn.<domain>`
- no AAAA records for those names while the Self-Hosted runtime is IPv4-only
- TCP ports `80`, `443`, `5060`, `5061`, and `5062` available
- UDP ports `443`, `5060`, `3478`, `5349`, `23000-32768`, and `49152-65535` available

Caddy manages HTTPS for `api.<domain>`. Leamout installs Certbot and obtains one Let's Encrypt certificate containing `sip.<domain>` and `turn.<domain>`. Caddy keeps TCP port 80 available for the ACME HTTP-01 webroot used for initial issuance and renewal.

## Install

The intended installation command is:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo sh
```

The bootstrap script detects Linux and the supported CPU architecture, resolves the latest release unless `LEAMOUT_VERSION` is provided, downloads the matching release archive and `checksums.txt`, verifies the archive SHA-256, installs the `leamout` CLI to `/usr/local/bin/leamout`, and starts the interactive installer.

To install an exact release:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo env LEAMOUT_VERSION=1.0.0 sh
```

Release assets use this naming scheme:

```text
leamout_<version>_linux_<arch>.tar.gz
checksums.txt
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

Leamout installation runs as root and the bootstrap starts with `umask 077`. Sensitive directories are `root:root` with mode `0700`. Secret-bearing files such as `/etc/leamout/leamout.env`, license material, installation state, logs, and backups are mode `0600` unless a runtime service needs a narrower group-readable exception.

The deployed SIP/TURN certificate is `0644 root:root`. Its private key is `0640 root:65534`: the containing `/etc/leamout/certs` directory remains `0700 root:root`, so ordinary host users cannot traverse it, while the bind-mounted key is readable by the official Coturn image which runs as `nobody:nogroup`.

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

## TLS issuance and renewal

Certbot keeps its ACME account, certificate lineage, and renewal configuration under `/etc/letsencrypt`. Leamout uses the fixed Certbot certificate name `leamout-sip-turn` and requests SANs for `sip.<domain>` and `turn.<domain>`. The initial installer does not invent an operator email address; Certbot registers non-interactively without an email contact.

Caddy serves only `/.well-known/acme-challenge/*` for the SIP and TURN hostnames over HTTP. Certbot uses `/var/lib/leamout/acme` as its webroot. The installer enables `certbot.timer`, so TCP port 80 must remain reachable after installation for future HTTP-01 renewals.

After successful issuance or renewal, a root-owned Certbot deploy hook atomically copies the certificate into `/etc/leamout/certs` and restarts OpenSIPS and Coturn when those containers are running.

`carrier-ca.pem` is optional. Publicly trusted carrier TLS uses the container system CA store by default; a carrier-specific CA file should only be installed when required by that carrier.

## Release authenticity

`checksums.txt` is the canonical SHA-256 manifest for downloadable release archives. The next release-hardening step is to sign this manifest with Minisign and make signature verification mandatory in the bootstrap. The Minisign private key must never be committed to the repository; only the public verification key belongs in public distribution material.

Self-Hosted shares communications runtime components and contracts with Cloud,
but has a distinct product and operational boundary: Enterprise licensing,
customer-operated infrastructure, and customer-selected BYOC connectivity.
