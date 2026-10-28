#!/usr/bin/env bash
set -euo pipefail
script_dir=${BASH_SOURCE[0]%/*}
[[ "$script_dir" != "${BASH_SOURCE[0]}" ]] || script_dir=.
tool=$(/bin/bash "$script_dir/xtask.sh")
exec "$tool" fetch-tlc "$@"
