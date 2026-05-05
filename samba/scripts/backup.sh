#!/usr/bin/env bash
set -euo pipefail

mkdir -p backups
timestamp="$(date +%Y%m%d-%H%M%S)"

docker run --rm \
  -v forlittle_samba_lib:/var/lib/samba:ro \
  -v forlittle_samba_etc:/etc/samba:ro \
  -v "$(pwd)/backups:/backups" \
  ubuntu:24.04 \
  tar czf "/backups/samba-${timestamp}.tar.gz" /var/lib/samba /etc/samba

echo "Backup created: backups/samba-${timestamp}.tar.gz"
