#!/usr/bin/env bash
set -euo pipefail

docker compose exec samba-ad-dc samba-tool domain info 127.0.0.1
docker compose exec samba-ad-dc bash -lc 'for srv in _ldap._tcp _kerberos._tcp _kerberos._udp _kpasswd._udp; do domain="$(hostname -d)"; echo "${srv}.${domain}"; dig @127.0.0.1 +short -t SRV "${srv}.${domain}"; done'
