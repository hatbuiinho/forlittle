#!/usr/bin/env bash
set -euo pipefail

required_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: ${name}" >&2
    exit 1
  fi
}

required_env "SAMBA_REALM"
required_env "SAMBA_DOMAIN"
required_env "SAMBA_HOSTNAME"
required_env "SAMBA_ADMIN_PASSWORD"

SAMBA_DNS_FORWARDER="${SAMBA_DNS_FORWARDER:-1.1.1.1}"
SAMBA_LOG_LEVEL="${SAMBA_LOG_LEVEL:-1}"
SAMBA_HOST_IP="${SAMBA_HOST_IP:-}"
SAMBA_REALM_UPPER="$(echo "${SAMBA_REALM}" | tr '[:lower:]' '[:upper:]')"
SAMBA_DNS_DOMAIN="$(echo "${SAMBA_REALM}" | tr '[:upper:]' '[:lower:]')"
SAMBA_HOST_FQDN="${SAMBA_HOSTNAME}.${SAMBA_DNS_DOMAIN}"

export KRB5_CONFIG=/etc/krb5.conf

if [[ ! -f /var/lib/samba/private/sam.ldb ]]; then
  echo "Provisioning Samba AD DC: realm=${SAMBA_REALM_UPPER} domain=${SAMBA_DOMAIN} hostname=${SAMBA_HOSTNAME}"

  rm -f /etc/samba/smb.conf

  provision_args=(
    domain provision
    --server-role=dc
    --use-rfc2307
    --dns-backend=SAMBA_INTERNAL
    --realm="${SAMBA_REALM_UPPER}"
    --domain="${SAMBA_DOMAIN}"
    --host-name="${SAMBA_HOSTNAME}"
    --adminpass="${SAMBA_ADMIN_PASSWORD}"
    --option="dns forwarder = ${SAMBA_DNS_FORWARDER}"
  )

  if [[ -n "${SAMBA_HOST_IP}" ]]; then
    provision_args+=(--host-ip="${SAMBA_HOST_IP}")
  fi

  samba-tool "${provision_args[@]}"

  cp -f /var/lib/samba/private/krb5.conf /etc/krb5.conf
else
  echo "Existing Samba AD database found. Skipping provision."
fi

if [[ -f /etc/samba/smb.conf ]]; then
  if grep -qE '^[[:space:]]*dns forwarder[[:space:]]*=' /etc/samba/smb.conf; then
    sed -i "s/^[[:space:]]*dns forwarder[[:space:]]*=.*/        dns forwarder = ${SAMBA_DNS_FORWARDER}/" /etc/samba/smb.conf
  else
    sed -i "/^\[global\]/a \        dns forwarder = ${SAMBA_DNS_FORWARDER}" /etc/samba/smb.conf
  fi
fi

cat >/etc/resolv.conf <<EOF
nameserver 127.0.0.1
search ${SAMBA_DNS_DOMAIN}
EOF

echo "Starting Samba AD DC as ${SAMBA_HOST_FQDN}"
exec samba -i --debug-stdout -d "${SAMBA_LOG_LEVEL}"
