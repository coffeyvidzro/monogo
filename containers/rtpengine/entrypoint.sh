#!/bin/sh
set -eu

: "${RTPENGINE_PUBLIC_IP:?RTPENGINE_PUBLIC_IP must be set}"

config=/etc/rtpengine/rtpengine.conf
runtime_config=/tmp/rtpengine.conf

sed "s|@RTPENGINE_PUBLIC_IP@|${RTPENGINE_PUBLIC_IP}|g" \
  "$config" > "$runtime_config"

exec /usr/local/bin/rtpengine --config-file="$runtime_config"
