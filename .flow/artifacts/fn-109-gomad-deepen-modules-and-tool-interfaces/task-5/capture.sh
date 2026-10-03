#!/bin/bash
# usage: capture.sh BINARY -> task-4 captures plus plan/explore parsing captures with exit statuses
B=$1
D=$(dirname "$0")
bash "$D/../task-4/capture.sh" "$B"
run() { echo "### gomad $*"; env -u GOMAD3_TOOLCHAIN_DIR "$B" "$@" >/tmp/t5.out 2>/tmp/t5.err </dev/null; echo "status=$?"; echo "--- stdout"; cat /tmp/t5.out; echo "--- stderr"; cat /tmp/t5.err; }
R=--toolchain-root=relative
run plan $R go-run ./x
run plan $R --output /tmp/p --strategy=random go-run ./x
run plan $R --output /tmp/p --strategy= --count=0 go-run ./x
run plan $R --output /tmp/p --coverage= go-run ./x
run plan $R --output /tmp/p --coverage=all go-run ./x
run plan $R --output /tmp/p --coverage=semantic --require-probe=unknown.probe go-run ./x
run plan $R --json --output /tmp/p --choices --choice-bytes=1 go-run ./x
run plan $R --output /tmp/p --on-failure=first go-run ./x
run plan $R --output /tmp/p go-run ./x extra
run plan $R --__plan --output /tmp/p go-run ./x
run explore $R --__plan go-run ./x
run explore $R --json --__plan go-run ./x
run explore $R --strategy= go-run ./x
run explore $R --coverage= go-run ./x
run explore $R --strategy=bogus --coverage=bogus go-run ./x
run explore $R --coverage=semantic --require-probe=unknown.probe go-run ./x
