#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOST_ARCH="$(uname -m)"
case "$HOST_ARCH" in
  arm64|aarch64)
    DEFAULT_WINDOWS_ARCH="arm64"
    ;;
  x86_64|amd64)
    DEFAULT_WINDOWS_ARCH="amd64"
    ;;
  *)
    echo "Unsupported build host architecture: $HOST_ARCH" >&2
    exit 1
    ;;
esac

WINDOWS_ARCH="${FORLITTLE_WINDOWS_ARCH:-$DEFAULT_WINDOWS_ARCH}"
case "$WINDOWS_ARCH" in
  amd64|arm64)
    ;;
  *)
    echo "FORLITTLE_WINDOWS_ARCH must be amd64 or arm64." >&2
    exit 1
    ;;
esac

DEFAULT_OUTPUT="$ROOT_DIR/dist/time-control-$WINDOWS_ARCH-$(date +%Y%m%d-%H%M%S)"
OUTPUT_DIR="${1:-$DEFAULT_OUTPUT}"

if [[ -e "$OUTPUT_DIR" ]]; then
  echo "Output directory already exists: $OUTPUT_DIR" >&2
  exit 1
fi

command -v go >/dev/null || { echo "Go is required." >&2; exit 1; }
command -v dotnet >/dev/null || { echo ".NET SDK 8 is required." >&2; exit 1; }

mkdir -p "$OUTPUT_DIR"
cd "$ROOT_DIR"
GOOS=windows GOARCH="$WINDOWS_ARCH" go build -o "$OUTPUT_DIR/forlittle-time-control.exe" "$ROOT_DIR/cmd/forlittle-time-control"
dotnet publish "$ROOT_DIR/ui-agent/ForLittle.TimeControl.Agent.csproj" \
  --configuration Release \
  --runtime "win-$WINDOWS_ARCH" \
  --self-contained true \
  --output "$OUTPUT_DIR/agent"

cp "$ROOT_DIR/config.time-control.example.json" "$OUTPUT_DIR/config.time-control.example.json"
cp "$ROOT_DIR/scripts/install-time-control.ps1" "$OUTPUT_DIR/install-time-control.ps1"
cp "$ROOT_DIR/scripts/deploy-time-control.ps1" "$OUTPUT_DIR/deploy-time-control.ps1"
cp "$ROOT_DIR/scripts/uninstall-time-control.ps1" "$OUTPUT_DIR/uninstall-time-control.ps1"
cp "$ROOT_DIR/scripts/install-time-control.cmd" "$OUTPUT_DIR/install-time-control.cmd"
cp "$ROOT_DIR/scripts/uninstall-time-control.cmd" "$OUTPUT_DIR/uninstall-time-control.cmd"

cat <<EOF
Release created: $OUTPUT_DIR
Architecture: win-$WINDOWS_ARCH

Copy this directory to the target Windows computer, create config.json from
config.time-control.example.json, then run deploy-time-control.ps1.
EOF
