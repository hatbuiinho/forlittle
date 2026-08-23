#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_OUTPUT="$ROOT_DIR/dist/time-control-$(date +%Y%m%d-%H%M%S)"
OUTPUT_DIR="${1:-$DEFAULT_OUTPUT}"

if [[ -e "$OUTPUT_DIR" ]]; then
  echo "Output directory already exists: $OUTPUT_DIR" >&2
  exit 1
fi

command -v go >/dev/null || { echo "Go is required." >&2; exit 1; }
command -v dotnet >/dev/null || { echo ".NET SDK 8 is required." >&2; exit 1; }

mkdir -p "$OUTPUT_DIR"
cd "$ROOT_DIR"
GOOS=windows GOARCH=amd64 go build -o "$OUTPUT_DIR/forlittle-time-control.exe" "$ROOT_DIR/cmd/forlittle-time-control"
dotnet publish "$ROOT_DIR/ui-agent/ForLittle.TimeControl.Agent.csproj" \
  --configuration Release \
  --runtime win-x64 \
  --self-contained true \
  --output "$OUTPUT_DIR/agent"

cp "$OUTPUT_DIR/agent/ForLittle.TimeControl.Agent.exe" "$OUTPUT_DIR/ForLittle.TimeControl.Agent.exe"
cp "$ROOT_DIR/config.time-control.example.json" "$OUTPUT_DIR/config.time-control.example.json"
cp "$ROOT_DIR/scripts/install-time-control.ps1" "$OUTPUT_DIR/install-time-control.ps1"
cp "$ROOT_DIR/scripts/deploy-time-control.ps1" "$OUTPUT_DIR/deploy-time-control.ps1"

cat <<EOF
Release created: $OUTPUT_DIR

Copy this directory to the target Windows computer, create config.json from
config.time-control.example.json, then run deploy-time-control.ps1.
EOF
