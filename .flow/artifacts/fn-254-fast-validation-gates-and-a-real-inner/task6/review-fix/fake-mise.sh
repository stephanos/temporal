#!/bin/bash
set -eu
printf '%s|%s|%s\n' "${MISE_AUTO_INSTALL-unset}" "${MISE_EXEC_AUTO_INSTALL-unset}" "$*" >> "$QUICK_MAKE_CONTROL_LOG"
if [ "${MISE_AUTO_INSTALL-}" = false ] && [ "${MISE_EXEC_AUTO_INSTALL-}" = false ]; then
    printf 'required pinned rust unavailable; auto-install disabled\n' >&2
    exit 78
fi
printf 'automatic-install-attempt\n' > "$QUICK_MAKE_CONTROL_PUBLICATION"
exit 0
