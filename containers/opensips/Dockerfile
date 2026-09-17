FROM debian:bookworm-slim

ARG OPENSIPS_VERSION=4.0.1

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl openssl \
    && curl -fsSL https://apt.opensips.org/keyring.gpg \
        -o /usr/share/keyrings/opensips.gpg \
    && printf '%s\n' \
        'deb [signed-by=/usr/share/keyrings/opensips.gpg] https://apt.opensips.org bookworm 4.0-releases' \
        > /etc/apt/sources.list.d/opensips.list \
    && apt-get update \
    && package_version="$(apt-cache madison opensips | awk -v wanted="$OPENSIPS_VERSION" '$3 ~ ("^" wanted "([.-]|$)") {print $3; exit}')" \
    && test -n "$package_version" || { \
        echo "OpenSIPS $OPENSIPS_VERSION is unavailable in the official 4.0 release repository" >&2; \
        exit 1; \
       } \
    && apt-get install -y --no-install-recommends \
        "opensips=$package_version" \
        "opensips-auth-modules=$package_version" \
        "opensips-postgres-module=$package_version" \
        "opensips-tls-module=$package_version" \
        "opensips-wss-module=$package_version" \
    && installed_version="$(dpkg-query -W -f='${Version}' opensips)" \
    && test "$installed_version" = "$package_version" \
    && for module in \
        auth.so \
        auth_db.so \
        db_postgres.so \
        gflags.so \
        mi_fifo.so \
        pike.so \
        rtpengine.so \
        sqlops.so \
        tls_mgm.so \
        tls_openssl.so \
        proto_tls.so \
        proto_wss.so \
        uac.so \
        uac_auth.so; do \
        find /usr/lib -type f -name "$module" -print -quit | grep -q . || { \
            echo "required OpenSIPS module missing: $module" >&2; \
            exit 1; \
        }; \
       done \
    && apt-get purge -y --auto-remove curl \
    && rm -rf /var/lib/apt/lists/*

COPY opensips.cfg /etc/opensips/opensips.cfg
COPY tls.cfg /etc/opensips/tls.cfg
COPY drain.cfg /etc/opensips/drain.cfg
COPY apply-drain-config.sh /usr/local/bin/apply-opensips-drain-config
COPY drain-control.sh /usr/local/bin/leamout-opensips-drain
COPY entrypoint.sh /usr/local/bin/leamout-opensips-entrypoint

RUN chmod 0755 \
        /usr/local/bin/apply-opensips-drain-config \
        /usr/local/bin/leamout-opensips-drain \
        /usr/local/bin/leamout-opensips-entrypoint \
    && /usr/local/bin/apply-opensips-drain-config /etc/opensips/opensips.cfg \
    && install -d -m 0700 /run/opensips \
    && install -d -m 0700 /etc/opensips/tls \
    && openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
        -subj '/CN=opensips-build-check.invalid' \
        -keyout /etc/opensips/tls/privkey.pem \
        -out /etc/opensips/tls/fullchain.pem >/dev/null 2>&1 \
    && cp /etc/ssl/certs/ca-certificates.crt /etc/opensips/tls/carrier-ca.pem \
    && opensips -C -f /etc/opensips/opensips.cfg \
    && rm -rf /etc/opensips/tls

EXPOSE 5060/udp 5060/tcp 5061/tcp 5062/tcp

ENTRYPOINT ["/usr/local/bin/leamout-opensips-entrypoint"]
CMD ["opensips", "-F", "-f", "/etc/opensips/opensips.cfg"]
