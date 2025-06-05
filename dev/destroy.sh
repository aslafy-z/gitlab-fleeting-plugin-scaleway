#!/usr/bin/env bash

set -eu -o pipefail

error() {
  echo >&2 "error: $*"
  exit 1
}

command -v scw > /dev/null || error "scw command not found!"

servers=$(scw instance server list tags.0=instance-group=dev-docker-autoscaler -o template="{{ .ID }}")

if [[ -n "$servers" ]]; then
  # shellcheck disable=SC2086
  scw instance server terminate $servers with-block=true with-ip=true
fi
