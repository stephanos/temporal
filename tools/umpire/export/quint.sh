#!/usr/bin/env bash
set -euo pipefail
exec npm exec --yes --package=@informalsystems/quint@0.33.0 -- quint "$@"
