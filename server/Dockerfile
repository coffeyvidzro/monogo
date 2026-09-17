# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26.6

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN set -eux; \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/server \
        ./cmd/server; \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/worker \
        ./cmd/worker

FROM alpine:3.22 AS runtime

WORKDIR /leamout

RUN apk add --no-cache \
        ca-certificates \
        tzdata \
    && addgroup -S leamout \
    && adduser -S -D -H -G leamout leamout

COPY --from=build /out/server /leamout/server
COPY --from=build /out/worker /leamout/worker

USER leamout

EXPOSE 8080

CMD ["/leamout/server"]

# Atlas and the immutable migration set are published as a separate target so
# Kubernetes can run schema changes before rolling out application workloads.
FROM arigaio/atlas:1.3.0-alpine AS migration-build

COPY migrations /migrations
RUN /atlas migrate hash --dir file:///migrations

FROM arigaio/atlas:1.3.0-alpine AS migrate

COPY --from=migration-build /migrations /migrations
ENTRYPOINT ["/atlas"]
