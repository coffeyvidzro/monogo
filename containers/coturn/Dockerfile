# syntax=docker/dockerfile:1

# Keep the runtime on the upstream release previously used by Leamout.
ARG COTURN_REF=4.6.3

FROM debian:trixie-slim AS build

ARG COTURN_REF
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        autoconf ca-certificates g++ gcc git libtool make pkg-config \
        libevent-dev libssl-dev libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src
RUN git clone --depth 1 --branch "${COTURN_REF}" \
        https://github.com/coturn/coturn.git coturn

WORKDIR /usr/src/coturn
RUN ./configure \
        --prefix=/usr \
        --sysconfdir=/etc/coturn \
        --turndbdir=/var/lib/coturn \
        --disable-rpath \
    && make -j"$(nproc)" \
    && mkdir -p /out \
    && DESTDIR=/out make install \
    && mkdir -p /out/usr/share/licenses/coturn \
    && cp LICENSE /out/usr/share/licenses/coturn/LICENSE

FROM debian:trixie-slim AS runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates gosu libatomic1 \
        libevent-2.1-7t64 libevent-core-2.1-7t64 \
        libevent-extra-2.1-7t64 libevent-openssl-2.1-7t64 \
        libevent-pthreads-2.1-7t64 libssl3t64 libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system coturn \
    && useradd --system --gid coturn --home-dir /var/lib/coturn \
        --shell /usr/sbin/nologin coturn \
    && mkdir -p /var/lib/coturn /run/leamout-coturn \
    && chown coturn:coturn /var/lib/coturn \
    && chmod 0700 /run/leamout-coturn

COPY --from=build /out/ /
COPY turnserver.conf /etc/coturn/turnserver.conf
COPY entrypoint.sh /usr/local/bin/leamout-coturn-entrypoint

RUN chmod 0755 /usr/local/bin/leamout-coturn-entrypoint \
    && test -x /usr/bin/turnserver

EXPOSE 3478/tcp 3478/udp 5349/tcp 5349/udp

# The entrypoint briefly runs as root to copy root-readable TLS mounts to
# private runtime files, then drops privileges before executing turnserver.
ENTRYPOINT ["/usr/local/bin/leamout-coturn-entrypoint"]
