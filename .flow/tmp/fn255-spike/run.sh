#!/bin/bash
set -euo pipefail
unset BASH_ENV
spike_dir=$(cd -- "$(dirname -- "$0")" && pwd -P)
exec python3 "$spike_dir/runner.py" "$@"
