# Leamout Self-Hosted

The self-hosted release bundle contains the Leamout installer plus the Docker Compose deployment assets, database migrations, and service configuration required to install without cloning the source repository.

## Requirements

- Linux host with Docker Engine and the Docker Compose plugin
- public DNS records pointing to the host:
  - `api.<domain>`
  - `sip.<domain>`
  - `turn.<domain>`
- TCP ports `80`, `443`, `5060`, `5061`, and `5062` available
- UDP ports `443`, `5060`, `3478`, `5349`, `23000-32768`, and `49152-65535` available
- a TLS certificate and private key trusted for both `sip.<domain>` and `turn.<domain>`; a wildcard certificate such as `*.example.com` is suitable

Caddy manages HTTPS certificates for `api.<domain>` automatically. The supplied SIP/TURN certificate is copied into the deployment for OpenSIPS and Coturn.

## Install

The intended installation command is:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo sh
```

The bootstrap script detects Linux and the supported CPU architecture, resolves the latest release unless `LEAMOUT_VERSION` is provided, downloads the matching release archive and SHA-256 checksum, verifies the archive, installs the `leamout` CLI to `/usr/local/bin/leamout`, and starts the interactive installer.

To install an exact release:

```bash
curl -fsSL https://get.leamout.com/install.sh | sudo LEAMOUT_VERSION=1.0.0 sh
```

Release archives use this naming scheme:

```text
leamout_<version>_linux_<arch>.tar.gz
leamout_<version>_linux_<arch>.tar.gz.sha256
```

The initial release target is `linux/amd64`. Other architectures should only be published once all Leamout runtime images are available for them.

The installer asks only for deployment-specific values:

- base domain
- public IP address
- SIP/TURN TLS certificate path
- SIP/TURN TLS private key path

The bundled release version is selected automatically. `/opt/leamout` is the default deployment directory; advanced installations can override it with `leamout install --install-dir <path>`.

The installer then:

1. validates Docker and Docker Compose
2. copies the release deployment assets to `/opt/leamout` by default
3. generates deployment secrets and writes `/opt/leamout/.env` with mode `0600`
4. copies SIP/TURN TLS material into the deployment
5. validates the Compose configuration
6. pulls the selected Leamout images
7. runs the Atlas migration job
8. starts the runtime with Docker Compose

The generated `.env` is installation state. Back it up securely and do not regenerate it during upgrades because it contains the carrier credential encryption key and other persistent secrets.

## Deployment layout

```text
/opt/leamout/
├── .env
├── deploy/docker/
│   ├── compose.yaml
│   ├── Caddyfile
│   └── certs/
│       ├── fullchain.pem
│       └── privkey.pem
├── server/migrations/
├── containers/nats/nats-server.conf
└── containers/coturn/turnserver.conf
```

The selected release version is stored in `.env` as `LEAMOUT_VERSION`.

The application/runtime is the same Leamout runtime used by Cloud. Self-Hosted changes deployment and operations only; it does not introduce a separate application mode or BYOC/Managed behavior.
