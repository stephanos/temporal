#!/usr/bin/env bash
set -euo pipefail
framework=$(/bin/bash "$(dirname "$0")/framework-root.sh")
exec /bin/bash "$framework/tools/scripts/xtask.sh" "$@"
