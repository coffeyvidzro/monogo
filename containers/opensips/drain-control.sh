#!/bin/sh
set -eu

fifo=/run/opensips/opensips_fifo
reply_dir=/run/opensips

mi() {
  method=$1
  params=${2:-[]}

  [ -p "$fifo" ] || {
    echo "OpenSIPS MI FIFO is unavailable: $fifo" >&2
    return 1
  }

  reply_name="leamout-mi-$$-$(date +%s)"
  reply_path="$reply_dir/$reply_name"
  mkfifo -m 0600 "$reply_path"

  cleanup() {
    rm -f "$reply_path"
  }
  trap cleanup EXIT HUP INT TERM

  request=$(printf '{"jsonrpc":"2.0","method":"%s","id":"leamout-drain","params":%s}' "$method" "$params")
  printf ':%s:%s\n' "$reply_name" "$request" > "$fifo" &
  writer=$!

  if ! response=$(timeout 5 cat "$reply_path"); then
    kill "$writer" 2>/dev/null || true
    wait "$writer" 2>/dev/null || true
    echo "OpenSIPS MI command timed out: $method" >&2
    return 1
  fi
  wait "$writer" 2>/dev/null || true

  cleanup
  trap - EXIT HUP INT TERM

  if printf '%s' "$response" | grep -q '"error"'; then
    printf '%s\n' "$response" >&2
    return 1
  fi

  printf '%s\n' "$response"
}

case "${1:-}" in
  enable)
    mi 'gflags:set' '[1]' >/dev/null
    echo "draining"
    ;;
  disable|resume)
    mi 'gflags:reset' '[1]' >/dev/null
    echo "accepting"
    ;;
  status)
    response=$(mi 'gflags:check' '[1]')
    if printf '%s' "$response" | grep -Eq '"result"[[:space:]]*:[[:space:]]*true'; then
      echo "draining"
    else
      echo "accepting"
    fi
    ;;
  dialogs)
    # get_statistics treats entries as statistic/group selectors. Query the
    # documented dialog: group, then extract the active_dialogs member.
    response=$(mi 'get_statistics' '{"statistics":["dialog:"]}')
    count=$(printf '%s' "$response" | sed -n 's/.*"dialog:active_dialogs"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p')
    [ -n "$count" ] || {
      echo "Unable to parse OpenSIPS active dialog count" >&2
      printf '%s\n' "$response" >&2
      exit 1
    }
    echo "$count"
    ;;
  *)
    echo "usage: $0 {enable|disable|resume|status|dialogs}" >&2
    exit 2
    ;;
esac
