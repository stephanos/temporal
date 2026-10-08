#!/bin/sh
set -eu
unset TEST_TELEMETRY_DIR
test "$TMPDIR" = /dev/shm/.fn1097-prune-source-XjdO2ItK
test "$GOTMPDIR" = /dev/shm/.fn1097-prune-source-XjdO2ItK
test "$GOCACHE" = /Users/stephan/Workspace/skunkworks/.gomad-fn1129-cache-EseD1r
test "$(pwd -P)" = /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3/qualification/set
printf 'actual test child TMPDIR=%s GOTMPDIR=%s GOCACHE=%s telemetry_override=unset\n' "$TMPDIR" "$GOTMPDIR" "$GOCACHE"
exec "$@"
