#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UPDATE_BASE_URL="${UPDATE_BASE_URL:-https://little-be.hatbuinho.me/extensions/forlittle}"
PEM_PATH="${PEM_PATH:-${ROOT_DIR}/extension.pem}"
CRX_PATH="${CRX_PATH:-${ROOT_DIR}/extension.crx}"
MANIFEST_PATH="${MANIFEST_PATH:-${ROOT_DIR}/extension/manifest.json}"
RELEASE_DIR="${RELEASE_DIR:-${ROOT_DIR}/server/extension-releases/forlittle}"
SKIP_PACK="${SKIP_PACK:-false}"
CHROME_BIN="${CHROME_BIN:-}"

usage() {
  cat <<'EOF'
Usage:
  ./release-extension.sh

Optional environment variables:
  UPDATE_BASE_URL   Public base URL for update.xml and .crx
  PEM_PATH          Path to extension private key (.pem)
  CRX_PATH          Path to existing .crx file
  MANIFEST_PATH     Path to extension manifest.json
  RELEASE_DIR       Destination release folder
  SKIP_PACK         Set to true to reuse CRX_PATH instead of packing from source
  CHROME_BIN        Chrome/Chromium binary to use when packing from source

Examples:
  ./release-extension.sh
  SKIP_PACK=true ./release-extension.sh
  UPDATE_BASE_URL='https://example.com/extensions/forlittle' ./release-extension.sh
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ ! -f "${MANIFEST_PATH}" ]]; then
  echo "Manifest not found: ${MANIFEST_PATH}" >&2
  exit 1
fi

if [[ ! -f "${PEM_PATH}" ]]; then
  echo "PEM file not found: ${PEM_PATH}" >&2
  exit 1
fi

VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${MANIFEST_PATH}" | head -n1)"
if [[ -z "${VERSION}" ]]; then
  echo "Could not parse version from ${MANIFEST_PATH}" >&2
  exit 1
fi

if [[ "${SKIP_PACK}" != "true" ]]; then
  rm -f "${CRX_PATH}"
fi

PACKAGE_ENV=(
  "UPDATE_BASE_URL=${UPDATE_BASE_URL}"
  "PEM_PATH=${PEM_PATH}"
  "CRX_PATH=${CRX_PATH}"
)

if [[ -n "${CHROME_BIN}" ]]; then
  PACKAGE_ENV+=("CHROME_BIN=${CHROME_BIN}")
fi

if [[ "${SKIP_PACK}" == "true" ]]; then
  if [[ ! -f "${CRX_PATH}" ]]; then
    echo "SKIP_PACK=true but CRX file not found: ${CRX_PATH}" >&2
    exit 1
  fi
else
  PACKAGE_ENV+=("CRX_PATH=")
fi

(
  cd "${ROOT_DIR}"
  env "${PACKAGE_ENV[@]}" ./scripts/package-extension.sh
)

echo
echo "Release directory:"
echo "  ${RELEASE_DIR}"
echo
echo "Next steps:"
echo "1. Sync ${RELEASE_DIR} to the remote host if this was run locally."
echo "2. Restart the backend container if needed: docker compose restart server"
echo "3. Verify:"
echo "   - https://little-be.hatbuinho.me/extensions/forlittle/update.xml"
echo "   - https://little-be.hatbuinho.me/extensions/forlittle/forlittle-${VERSION}.crx"
echo "4. On a managed Windows client, open chrome://extensions and confirm version ${VERSION}."
