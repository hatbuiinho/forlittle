#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXTENSION_DIR="${EXTENSION_DIR:-${ROOT_DIR}/extension}"
RELEASE_SLUG="${RELEASE_SLUG:-forlittle}"
RELEASES_ROOT="${RELEASES_ROOT:-${ROOT_DIR}/server/extension-releases}"
RELEASE_DIR="${RELEASES_ROOT}/${RELEASE_SLUG}"
MANIFEST_PATH="${MANIFEST_PATH:-${EXTENSION_DIR}/manifest.json}"
UPDATE_BASE_URL="${UPDATE_BASE_URL:-}"
PEM_PATH="${PEM_PATH:-}"
CRX_PATH="${CRX_PATH:-${ROOT_DIR}/extension.crx}"
EXTENSION_ID="${EXTENSION_ID:-}"
CHROME_BIN="${CHROME_BIN:-}"

if [[ -z "${UPDATE_BASE_URL}" ]]; then
  echo "UPDATE_BASE_URL is required, for example: https://little-be.hatbuinho.me/extensions/${RELEASE_SLUG}" >&2
  exit 1
fi

if [[ -z "${PEM_PATH}" ]]; then
	echo "PEM_PATH is required and must point to the private key used to keep a stable extension ID." >&2
	exit 1
fi

derive_extension_id() {
	local pem_path="$1"
	local public_key_hex
	local public_key_hash
	local -A hex_to_id=(
		["0"]="a" ["1"]="b" ["2"]="c" ["3"]="d"
		["4"]="e" ["5"]="f" ["6"]="g" ["7"]="h"
		["8"]="i" ["9"]="j" ["a"]="k" ["b"]="l"
		["c"]="m" ["d"]="n" ["e"]="o" ["f"]="p"
	)

	public_key_hex="$(openssl pkey -pubout -outform DER -in "${pem_path}" 2>/dev/null | sha256sum | awk '{print $1}')"
	public_key_hash="${public_key_hex:0:32}"

	local id=""
	local nibble
	for ((i = 0; i < ${#public_key_hash}; i++)); do
		nibble="${public_key_hash:i:1}"
		id+="${hex_to_id[$nibble]}"
	done

	printf '%s' "${id}"
}

if [[ -z "${EXTENSION_ID}" ]]; then
	EXTENSION_ID="$(derive_extension_id "${PEM_PATH}")"
fi

if [[ -z "${EXTENSION_ID}" ]]; then
	echo "Could not derive EXTENSION_ID from ${PEM_PATH}" >&2
	exit 1
fi

if [[ ! -f "${MANIFEST_PATH}" ]]; then
  echo "Manifest not found: ${MANIFEST_PATH}" >&2
  exit 1
fi

if [[ ! -f "${PEM_PATH}" ]]; then
  echo "Private key not found: ${PEM_PATH}" >&2
  exit 1
fi

if [[ -z "${CHROME_BIN}" && ! -f "${CRX_PATH}" ]]; then
	for candidate in \
		google-chrome \
		google-chrome-stable \
    chromium \
    chromium-browser \
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
  do
    if command -v "${candidate}" >/dev/null 2>&1; then
      CHROME_BIN="$(command -v "${candidate}")"
      break
    fi

    if [[ -x "${candidate}" ]]; then
      CHROME_BIN="${candidate}"
      break
    fi
  done
fi

VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${MANIFEST_PATH}" | head -n1)"
if [[ -z "${VERSION}" ]]; then
	echo "Could not parse version from ${MANIFEST_PATH}" >&2
	exit 1
fi

mkdir -p "${RELEASE_DIR}"

RELEASE_FILENAME="${RELEASE_SLUG}-${VERSION}.crx"
RELEASE_PATH="${RELEASE_DIR}/${RELEASE_FILENAME}"
UPDATE_XML_PATH="${RELEASE_DIR}/update.xml"
UPDATE_CODEBASE="${UPDATE_BASE_URL%/}/${RELEASE_FILENAME}"

if [[ -f "${CRX_PATH}" ]]; then
  cp -f "${CRX_PATH}" "${RELEASE_PATH}"
else
  if [[ -z "${CHROME_BIN}" ]]; then
    echo "Could not find a Chrome/Chromium binary and CRX_PATH does not exist. Set CHROME_BIN or provide CRX_PATH." >&2
    exit 1
  fi

  PACK_OUTPUT="${EXTENSION_DIR}.crx"
  rm -f "${PACK_OUTPUT}"

  "${CHROME_BIN}" \
    --pack-extension="${EXTENSION_DIR}" \
    --pack-extension-key="${PEM_PATH}"

  if [[ ! -f "${PACK_OUTPUT}" ]]; then
    echo "Chrome packaging did not produce ${PACK_OUTPUT}" >&2
    exit 1
  fi

  mv -f "${PACK_OUTPUT}" "${RELEASE_PATH}"
fi

cat >"${UPDATE_XML_PATH}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<gupdate xmlns="http://www.google.com/update2/response" protocol="2.0">
  <app appid="${EXTENSION_ID}">
    <updatecheck codebase="${UPDATE_CODEBASE}" version="${VERSION}" />
  </app>
</gupdate>
EOF

cat <<EOF
Release created:
  CRX:        ${RELEASE_PATH}
  Update XML: ${UPDATE_XML_PATH}
  Version:    ${VERSION}
  App ID:     ${EXTENSION_ID}

Policy update_url:
  ${UPDATE_BASE_URL%/}/update.xml
EOF
