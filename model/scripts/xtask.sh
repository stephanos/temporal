#!/usr/bin/env bash
set -euo pipefail
exec /bin/bash "$(dirname "$0")/../umpire/scripts/xtask.sh" "$@"
