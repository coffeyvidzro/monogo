# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.6

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMAND=server
RUN case "${COMMAND}" in \
      server|worker) ;; \
      *) echo "unsupported command: ${COMMAND}" >&2; exit 1 ;; \
    esac \
    && CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/leamout \
      "./cmd/${COMMAND}"

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system leamout \
    && useradd \
      --system \
      --gid leamout \
      --home-dir /nonexistent \
      --shell /usr/sbin/nologin \
      leamout

COPY --from=build /out/leamout /usr/local/bin/leamout

USER leamout

ENTRYPOINT ["/usr/local/bin/leamout"]

FROM arigaio/atlas:1.3.0-alpine AS migration-build

COPY migrations /migrations

RUN /atlas migrate hash --dir file:///migrations

FROM arigaio/atlas:1.3.0-alpine AS migrate

COPY --from=migration-build /migrations /migrations

ENTRYPOINT ["/atlas"]
