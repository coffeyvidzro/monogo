#!/bin/sh
set -eu

config=${1:-/etc/opensips/opensips.cfg}

if ! grep -Fq 'include_file "drain.cfg"' "$config"; then
  sed -i '/include_file "tls.cfg"/a include_file "drain.cfg"' "$config"
fi

if ! grep -Fq 'route(LEAMOUT_DRAIN);' "$config"; then
  sed -i '/^    # REGISTER is authenticated against PostgreSQL/i\    route(LEAMOUT_DRAIN);\
' "$config"
fi

carrier_dialog_line=$(
  awk '
    /^route\[CARRIER_INGRESS\] \{/ { inside=1 }
    inside && /if \(!create_dialog\(\)\)/ { print NR; exit }
  ' "$config"
)
carrier_record_line=$(
  awk '
    /^route\[CARRIER_INGRESS\] \{/ { inside=1 }
    inside && /^[[:space:]]*record_route\(\);[[:space:]]*$/ { print NR; exit }
  ' "$config"
)

[ -n "$carrier_dialog_line" ] || {
  echo "carrier ingress create_dialog() anchor is missing" >&2
  exit 1
}
[ -n "$carrier_record_line" ] || {
  echo "carrier ingress record_route() anchor is missing" >&2
  exit 1
}

# The dialog module can only embed its dialog identifier into the route set if
# Record-Route already exists when create_dialog() runs. Keep this ordering so
# sequential requests such as BYE match the tracked dialog and release media.
if [ "$carrier_record_line" -gt "$carrier_dialog_line" ]; then
  tmp=$(mktemp)
  awk '
    /^route\[CARRIER_INGRESS\] \{/ { inside=1 }
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

carrier_dialog_line=$(
  awk '
    /^route\[CARRIER_INGRESS\] \{/ { inside=1 }
    inside && /if \(!create_dialog\(\)\)/ { print NR; exit }
  ' "$config"
)
carrier_record_line=$(
  awk '
    /^route\[CARRIER_INGRESS\] \{/ { inside=1 }
    inside && /^[[:space:]]*record_route\(\);[[:space:]]*$/ { print NR; exit }
  ' "$config"
)

[ "$carrier_record_line" -lt "$carrier_dialog_line" ] || {
  echo "carrier ingress must record-route before create_dialog()" >&2
  exit 1
}

grep -Fq 'include_file "drain.cfg"' "$config"
grep -Fq 'route(LEAMOUT_DRAIN);' "$config"
