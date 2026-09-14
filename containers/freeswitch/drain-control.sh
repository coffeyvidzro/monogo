#!/bin/sh
set -eu

: "${FREESWITCH_ESL_PASSWORD:?FREESWITCH_ESL_PASSWORD must be set}"

fs() {
  fs_cli \
    -H 127.0.0.1 \
    -P 8021 \
    -p "$FREESWITCH_ESL_PASSWORD" \
    -x "$1"
}

pause_state() {
  if response=$(fs 'fsctl pause_check' 2>&1); then
    :
  else
    rc=$?
    echo "FreeSWITCH pause_check failed (exit $rc)" >&2
    printf '%s\n' "$response" >&2
    return "$rc"
  fi

  response=$(printf '%s\n' "$response" | tr -d '\r')
  state=$(
    printf '%s\n' "$response" |
      awk '{
        for (i = 1; i <= NF; i++) {
          token = tolower($i)
          if (token == "true" || token == "false") {
            state = token
          }
        }
      }
      END {
        if (state != "") {
          print state
        }
      }'
  )

  case "$state" in
    true|false)
      printf '%s\n' "$state"
      ;;
    *)
      echo "Unable to parse FreeSWITCH pause state" >&2
      printf '%s\n' "$response" >&2
      return 1
      ;;
  esac
}

case "${1:-}" in
  enable)
    fs 'fsctl pause' >/dev/null
    echo "draining"
    ;;
  disable|resume)
    fs 'fsctl resume' >/dev/null
    echo "accepting"
    ;;
  status)
    state=$(pause_state)
    case "$state" in
      true) echo "draining" ;;
      false) echo "accepting" ;;
    esac
    ;;
  channels)
    if response=$(fs 'show channels count' 2>&1); then
      :
    else
      rc=$?
      echo "FreeSWITCH channel-count query failed (exit $rc)" >&2
      printf '%s\n' "$response" >&2
      exit "$rc"
    fi

    response=$(printf '%s\n' "$response" | tr -d '\r')
    count=$(printf '%s\n' "$response" | sed -n 's/^[[:space:]]*\([0-9][0-9]*\)[[:space:]][[:space:]]*total.*/\1/p' | head -n 1)
    [ -n "$count" ] || {
      echo "Unable to parse FreeSWITCH active channel count" >&2
      printf '%s\n' "$response" >&2
      exit 1
    }
    echo "$count"
    ;;
  *)
    echo "usage: $0 {enable|disable|resume|status|channels}" >&2
    exit 2
    ;;
esac
