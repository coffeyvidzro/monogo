# syntax=docker/dockerfile:1

ARG FREESWITCH_REF=3f13ad1b1d323e435757fc43a2bb829d4ea5b9a8

# SignalWire's public CI image provides the maintained Bookworm build
# toolchain and native development dependencies. It is not a FreeSWITCH
# runtime: build the pinned FreeSWITCH release into the image below.
FROM signalwire/freeswitch-public-ci-base:bookworm-amd64@sha256:4f77a9baa0fb253d810e1139238fb75f01d06f6d41b538dcb1c3e8f7cfaa75f6

ARG FREESWITCH_REF
ARG DEP_BUILD_NUMBER=42

USER root
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/freeswitch/bin:${PATH}"
ENV LD_LIBRARY_PATH="/usr/local/freeswitch/lib:/usr/local/lib"

# The CI foundation can carry SignalWire APT sources that require credentials.
# Public Leamout builds must use only public package sources; FreeSWITCH native
# dependencies that are not in Debian are built from source below.
RUN while IFS= read -r -d '' file; do \
      if grep -Eq '(^|[[:space:]]|=)(https?://)?(packages|freeswitch)\.signalwire\.com/' "$file"; then \
        echo "Removing authenticated SignalWire APT source: $file"; \
        rm -f "$file"; \
      fi; \
    done < <(find /etc/apt -type f -print0) \
    && rm -f /etc/apt/auth.conf /etc/apt/auth.conf.d/*

WORKDIR /usr/src

# Pin the complete upstream FreeSWITCH 1.11.2 release commit instead of a
# mutable branch or package repository.
RUN git init freeswitch \
    && cd freeswitch \
    && git remote add origin https://github.com/signalwire/freeswitch.git \
    && git fetch --depth 1 origin "${FREESWITCH_REF}" \
    && git checkout --detach FETCH_HEAD

WORKDIR /usr/src/freeswitch

# mod_sofia needs Sofia-SIP. Use FreeSWITCH's maintained dependency builder,
# but intentionally avoid --all: Voice v1 does not need the optional V8,
# SignalWire-C, SpanDSP, and other dependency graph that made the old source
# build slow and fragile.
RUN ./scripts/packaging/build/dependencies/build-dependencies.sh \
      --build-number "${DEP_BUILD_NUMBER}" \
      --repo \
      --clone \
      --git-https \
      sofia-sip \
    && apt-get update \
    && find /var/local/deb -maxdepth 1 -type f -name '*.deb' -print0 \
      | xargs -0 -r apt-get install -y --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

RUN ./bootstrap.sh -j

# Compile the smallest module set that implements Leamout Voice v1: SIP,
# dialplan/control commands, ESL, conference, playback/recording formats, and
# the codecs advertised by the SIP profile. PCMU/PCMA/G722 are core codecs;
# Opus is the only codec module required here.
RUN printf '%s\n' \
      'applications/mod_commands' \
      'applications/mod_conference' \
      'applications/mod_db' \
      'applications/mod_dptools' \
      'applications/mod_hash' \
      'codecs/mod_opus' \
      'dialplans/mod_dialplan_xml' \
      'endpoints/mod_loopback' \
      'endpoints/mod_sofia' \
      'event_handlers/mod_event_socket' \
      'formats/mod_local_stream' \
      'formats/mod_native_file' \
      'formats/mod_sndfile' \
      'formats/mod_tone_stream' \
      'loggers/mod_console' \
      'loggers/mod_logfile' \
      > modules.conf \
    && ./configure \
      --prefix=/usr/local/freeswitch \
      --sysconfdir=/etc \
      --localstatedir=/var \
      --disable-dependency-tracking \
    && make -j"$(nproc)" \
    && make install \
    && make samples

# Keep Leamout configuration in the image and overlay it onto the complete
# FreeSWITCH sample configuration at container start.
COPY autoload_configs/ /leamout-config/autoload_configs/
COPY sip_profiles/ /leamout-config/sip_profiles/
COPY dialplan/ /leamout-config/dialplan/
COPY entrypoint.d/10-leamout-config.sh /docker-entrypoint.d/10-leamout-config.sh
COPY entrypoint.sh /usr/local/bin/leamout-freeswitch-entrypoint
COPY drain-control.sh /usr/local/bin/leamout-freeswitch-drain

RUN chmod 0755 \
      /docker-entrypoint.d/10-leamout-config.sh \
      /usr/local/bin/leamout-freeswitch-entrypoint \
      /usr/local/bin/leamout-freeswitch-drain \
    && mkdir -p \
      /etc/freeswitch/autoload_configs \
      /etc/freeswitch/sip_profiles \
      /etc/freeswitch/dialplan \
      /var/lib/freeswitch/recordings \
    && command -v freeswitch >/dev/null \
    && command -v fs_cli >/dev/null \
    && test -x /usr/local/freeswitch/bin/freeswitch \
    && test -x /usr/local/freeswitch/bin/fs_cli

WORKDIR /etc/freeswitch

# SIP and RTP are intentionally documented here. Docker Compose decides which
# ports are actually published. ESL 8021 is deliberately NOT exposed/published.
EXPOSE 5060/udp
EXPOSE 5060/tcp
EXPOSE 16384-32768/udp

ENTRYPOINT ["/usr/local/bin/leamout-freeswitch-entrypoint"]
CMD ["freeswitch", "-nonat", "-nf"]
