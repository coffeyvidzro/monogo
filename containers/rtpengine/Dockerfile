# syntax=docker/dockerfile:1

ARG RTPENGINE_REF=mr26.2.1

FROM debian:trixie-slim AS build

ARG RTPENGINE_REF

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    default-libmysqlclient-dev \
    g++ \
    gcc \
    git \
    gperf \
    libavcodec-extra \
    libavfilter-dev \
    libbcg729-dev \
    libcurl4-openssl-dev \
    libevent-dev \
    libglib2.0-dev \
    libhiredis-dev \
    libiptc-dev \
    libjson-glib-dev \
    libjwt-dev \
    libncursesw5-dev \
    libopus-dev \
    libpcap-dev \
    libpcre2-dev \
    libspandsp-dev \
    libssl-dev \
    libwebsockets-dev \
    make \
    markdown \
    patch \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src

RUN git clone --depth 1 --branch "${RTPENGINE_REF}" \
    https://github.com/sipwise/rtpengine.git \
    rtpengine

WORKDIR /usr/src/rtpengine/daemon

RUN make -j"$(nproc)" rtpengine \
    && strip -o /usr/local/bin/rtpengine rtpengine

FROM debian:trixie-slim AS runtime

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
    libavcodec-extra \
    libavfilter10 \
    libbcg729-0 \
    libcurl4 \
    libevent-2.1-7 \
    libevent-pthreads-2.1-7 \
    libglib2.0-0 \
    libhiredis1.1.0 \
    libip4tc2 \
    libip6tc2 \
    libjson-glib-1.0-0 \
    libjwt2 \
    libmariadb3 \
    libncursesw6 \
    libopus0 \
    libpcap0.8 \
    libpcre2-8-0 \
    libspandsp2 \
    libssl3 \
    libwebsockets19 \
    net-tools \
    procps \
    sed \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --system --gid 1000 rtpengine \
    && useradd \
    --system \
    --uid 1000 \
    --gid 1000 \
    --home-dir /nonexistent \
    --shell /usr/sbin/nologin \
    rtpengine

COPY --from=build /usr/local/bin/rtpengine /usr/local/bin/rtpengine
COPY rtpengine.conf /etc/rtpengine/rtpengine.conf
COPY entrypoint.sh /usr/local/bin/rtpengine-entrypoint

RUN chmod 0755 /usr/local/bin/rtpengine-entrypoint \
    && mkdir -p /var/run/rtpengine \
    && chown -R rtpengine:rtpengine /var/run/rtpengine

WORKDIR /var/lib/rtpengine

USER rtpengine

EXPOSE 22222/udp
EXPOSE 23000-32768/udp

ENTRYPOINT ["/usr/local/bin/rtpengine-entrypoint"]
