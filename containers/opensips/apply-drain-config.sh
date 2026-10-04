#!/bin/sh
set -eu

config=${1:-/etc/opensips/opensips.cfg}

if ! grep -Fq 'include_file "drain.cfg"' "$config"; then
  if ! grep -Fq 'include_file "tls.cfg"' "$config"; then
    echo "tls include anchor is missing" >&2
    exit 1
  fi
  sed -i '/include_file "tls.cfg"/a include_file "drain.cfg"' "$config"
fi

if ! grep -Fq 'route(LEAMOUT_DRAIN);' "$config"; then
  tmp=$(mktemp)
  if ! awk '
    /^[[:space:]]*if \(!is_method\("INVITE\|MESSAGE"\)\)/ && !inserted {
      print "    route(LEAMOUT_DRAIN);"
      print ""
      inserted=1
    }
    { print }
    END { if (!inserted) exit 42 }
  ' "$config" > "$tmp"; then
    rc=$?
    rm -f "$tmp"
    if [ "$rc" -eq 42 ]; then
      echo "out-of-dialog admission anchor is missing" >&2
    fi
    exit "$rc"
  fi
  cat "$tmp" > "$config"
  rm -f "$tmp"
fi

trunk_dialog_line=$(
  awk '
    /^route\[TRUNK_INGRESS\] \{/ { inside=1 }
    inside && /if \(!create_dialog\(\)\)/ { print NR; exit }
  ' "$config"
)
trunk_record_line=$(
  awk '
    /^route\[TRUNK_INGRESS\] \{/ { inside=1 }
    inside && /^[[:space:]]*record_route\(\);[[:space:]]*$/ { print NR; exit }
  ' "$config"
)

[ -n "$trunk_dialog_line" ] || {
  echo "trunk ingress create_dialog() anchor is missing" >&2
  exit 1
}
[ -n "$trunk_record_line" ] || {
  echo "trunk ingress record_route() anchor is missing" >&2
  exit 1
}

# The dialog module can only embed its dialog identifier into the route set if
# Record-Route already exists when create_dialog() runs. Keep this ordering so
# sequential requests such as BYE match the tracked dialog and release media.
if [ "$trunk_record_line" -gt "$trunk_dialog_line" ]; then
  tmp=$(mktemp)
  awk '
    /^route\[TRUNK_INGRESS\] \{/ { inside=1 }
    inside && /if \(!create_dialog\(\)\)/ && !inserted {
      print "    # Establish the route set before dialog creation so OpenSIPS can"
      print "    # attach its dialog identifier for reliable in-dialog BYE matching."
      print "    record_route();"
      print ""
      inserted=1
    }
    inside && inserted && /^[[:space:]]*record_route\(\);[[:space:]]*$/ { next }
    { print }
    /^####### Relay Route #########/ { inside=0 }
  ' "$config" > "$tmp"
  cat "$tmp" > "$config"
  rm -f "$tmp"
fi

trunk_dialog_line=$(
  awk '
    /^route\[TRUNK_INGRESS\] \{/ { inside=1 }
    inside && /if \(!create_dialog\(\)\)/ { print NR; exit }
  ' "$config"
)
trunk_record_line=$(
  awk '
    /^route\[TRUNK_INGRESS\] \{/ { inside=1 }
    inside && /^[[:space:]]*record_route\(\);[[:space:]]*$/ { print NR; exit }
  ' "$config"
)

[ "$trunk_record_line" -lt "$trunk_dialog_line" ] || {
  echo "trunk ingress must record-route before create_dialog()" >&2
  exit 1
}

grep -Fq 'include_file "drain.cfg"' "$config" || {
  echo "drain config include is missing" >&2
  exit 1
}
grep -Fq 'route(LEAMOUT_DRAIN);' "$config" || {
  echo "drain admission hook is missing" >&2
  exit 1
}
