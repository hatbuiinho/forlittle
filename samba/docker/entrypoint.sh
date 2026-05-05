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
SAMBA_INTERFACES="${SAMBA_INTERFACES:-lo tailscale0}"
SAMBA_REALM_UPPER="$(echo "${SAMBA_REALM}" | tr '[:lower:]' '[:upper:]')"
SAMBA_DOMAIN_UPPER="$(echo "${SAMBA_DOMAIN}" | tr '[:lower:]' '[:upper:]')"
SAMBA_DNS_DOMAIN="$(echo "${SAMBA_REALM}" | tr '[:upper:]' '[:lower:]')"
SAMBA_HOST_FQDN="${SAMBA_HOSTNAME}.${SAMBA_DNS_DOMAIN}"
SAMBA_MARKER_FILE="/var/lib/samba/.forlittle-provisioned"

export KRB5_CONFIG=/etc/krb5.conf

reset_partial_state() {
  echo "Resetting incomplete Samba provision state"
  rm -rf /etc/samba/*
  rm -rf /var/lib/samba/*
  rm -rf /var/cache/samba/*
  rm -rf /var/log/samba/*
}

needs_provision=true
if [[ -f "${SAMBA_MARKER_FILE}" ]]; then
  if [[ -f /var/lib/samba/private/sam.ldb && -f /var/lib/samba/private/secrets.ldb && -f /etc/samba/smb.conf ]]; then
    needs_provision=false
  else
    echo "Provision marker exists but required Samba files are missing."
    exit 1
  fi
elif [[ -e /var/lib/samba/private/sam.ldb || -e /var/lib/samba/private/secrets.ldb || -e /etc/samba/smb.conf ]]; then
  reset_partial_state
fi

if [[ "${needs_provision}" == "true" ]]; then
  echo "Provisioning Samba AD DC: realm=${SAMBA_REALM_UPPER} domain=${SAMBA_DOMAIN_UPPER} hostname=${SAMBA_HOSTNAME}"

  rm -f /etc/samba/smb.conf

  provision_args=(
    domain provision
    --server-role=dc
    --use-rfc2307
    --dns-backend=SAMBA_INTERNAL
    --realm="${SAMBA_REALM_UPPER}"
    --domain="${SAMBA_DOMAIN_UPPER}"
    --host-name="${SAMBA_HOSTNAME}"
    --adminpass="${SAMBA_ADMIN_PASSWORD}"
    --option="dns forwarder = ${SAMBA_DNS_FORWARDER}"
  )

  if [[ -n "${SAMBA_HOST_IP}" ]]; then
    provision_args+=(--host-ip="${SAMBA_HOST_IP}")
  fi

  samba-tool "${provision_args[@]}"

  cp -f /var/lib/samba/private/krb5.conf /etc/krb5.conf
  touch "${SAMBA_MARKER_FILE}"
else
  echo "Existing Samba AD database found. Skipping provision."
fi

if [[ -f /etc/samba/smb.conf ]]; then
  if grep -qE '^[[:space:]]*dns forwarder[[:space:]]*=' /etc/samba/smb.conf; then
    sed -i "s/^[[:space:]]*dns forwarder[[:space:]]*=.*/        dns forwarder = ${SAMBA_DNS_FORWARDER}/" /etc/samba/smb.conf
  else
    sed -i "/^\[global\]/a \        dns forwarder = ${SAMBA_DNS_FORWARDER}" /etc/samba/smb.conf
  fi

  if grep -qE '^[[:space:]]*interfaces[[:space:]]*=' /etc/samba/smb.conf; then
    sed -i "s/^[[:space:]]*interfaces[[:space:]]*=.*/        interfaces = ${SAMBA_INTERFACES}/" /etc/samba/smb.conf
  else
    sed -i "/^\[global\]/a \        interfaces = ${SAMBA_INTERFACES}" /etc/samba/smb.conf
  fi

  if grep -qE '^[[:space:]]*bind interfaces only[[:space:]]*=' /etc/samba/smb.conf; then
    sed -i "s/^[[:space:]]*bind interfaces only[[:space:]]*=.*/        bind interfaces only = yes/" /etc/samba/smb.conf
  else
    sed -i "/^\[global\]/a \        bind interfaces only = yes" /etc/samba/smb.conf
  fi
fi

cat >/etc/resolv.conf <<EOF
nameserver 127.0.0.1
search ${SAMBA_DNS_DOMAIN}
EOF

echo "Starting Samba AD DC as ${SAMBA_HOST_FQDN}"
exec samba -i --debug-stdout -d "${SAMBA_LOG_LEVEL}"
