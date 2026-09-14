#!/bin/sh
set -eu

install -d \
  /etc/freeswitch/autoload_configs \
  /etc/freeswitch/sip_profiles \
  /etc/freeswitch/dialplan

install -m 0644 \
  /leamout-config/autoload_configs/modules.conf.xml \
  /etc/freeswitch/autoload_configs/modules.conf.xml

install -m 0644 \
  /leamout-config/autoload_configs/acl.conf.xml \
  /etc/freeswitch/autoload_configs/acl.conf.xml

install -m 0644 \
  /leamout-config/autoload_configs/event_socket.conf.xml \
  /etc/freeswitch/autoload_configs/event_socket.conf.xml

# The sample configuration provides useful global defaults, but Leamout owns
# the SIP listeners. Remove sample top-level profiles before installing the
# production internal profile so no extra SIP listener is started implicitly.
find /etc/freeswitch/sip_profiles -maxdepth 1 -type f -name '*.xml' -delete
install -m 0644 \
  /leamout-config/sip_profiles/internal.xml \
  /etc/freeswitch/sip_profiles/internal.xml

install -m 0644 \
  /leamout-config/dialplan/leamout.xml \
  /etc/freeswitch/dialplan/leamout.xml
