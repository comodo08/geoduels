#!/bin/sh

set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <service-package> <bin-name>" >&2
  exit 1
fi

SERVICE_PATH="$1"
BIN_NAME="$2"
AIR_VERSION="${AIR_VERSION:-v1.61.7}"
AIR_BIN="${GOPATH:-/go}/bin/air"
TMP_DIR="/tmp/${BIN_NAME}-air"
SERVICE_DIR="${SERVICE_PATH#./}"
INCLUDE_DIRS="${SERVICE_DIR},pkg,internal"
INCLUDE_FILES="go.mod,go.sum"

if [ ! -x "$AIR_BIN" ]; then
  echo "installing air ${AIR_VERSION}..." >&2
  GOBIN="$(dirname "$AIR_BIN")" go install "github.com/air-verse/air@${AIR_VERSION}"
fi

mkdir -p "$TMP_DIR"

exec "$AIR_BIN" \
  --build.cmd "go build -o ${TMP_DIR}/${BIN_NAME} ${SERVICE_PATH}" \
  --build.bin "${TMP_DIR}/${BIN_NAME}" \
  --build.delay "250" \
  --build.include_dir "${INCLUDE_DIRS}" \
  --build.include_file "${INCLUDE_FILES}" \
  --build.exclude_dir ".git,node_modules,tmp,dist" \
  --build.include_ext "go,tpl,tmpl,html,yaml,yml,json" \
  --misc.clean_on_exit "true"
