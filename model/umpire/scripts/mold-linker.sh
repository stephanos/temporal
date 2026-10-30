#!/usr/bin/env bash
set -euo pipefail
export MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false
mold=$(mise -C "${BASH_SOURCE[0]%/*}/.." where github:rui314/mold)
[ -x "$mold/bin/ld.mold" ]
exec cc -B"${mold}/bin/" -fuse-ld=mold "$@"
