#!/bin/sh
set -eu

image="${ACCEPTANCE_MINIO_IMAGE:-docker.io/coollabsio/minio:latest}"

if docker image inspect "$image" >/dev/null 2>&1; then
    exit 0
fi

attempt=1
max_attempts=5
while [ "$attempt" -le "$max_attempts" ]; do
    echo "Pulling acceptance MinIO image (attempt $attempt/$max_attempts)..."
    if docker pull "$image"; then
        exit 0
    fi

    if [ "$attempt" -eq "$max_attempts" ]; then
        break
    fi

    sleep $((attempt * 3))
    attempt=$((attempt + 1))
done

echo "failed to pull acceptance MinIO image after $max_attempts attempts: $image" >&2
exit 1
