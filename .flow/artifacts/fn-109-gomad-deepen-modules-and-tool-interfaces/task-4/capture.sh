#!/bin/bash
# usage: capture.sh BINARY  -> prints help/usage/error captures with exit statuses
B=$1
run() { echo "### gomad $*"; env -u GOMAD3_TOOLCHAIN_DIR "$B" "$@" >/tmp/t4.out 2>/tmp/t4.err </dev/null; echo "status=$?"; echo "--- stdout"; cat /tmp/t4.out; echo "--- stderr"; cat /tmp/t4.err; }
run
run unknown
for c in plan execute-shard merge explore qualify qualify-set merge-set compare-support analyze resume recover replay minimize doctor inspect; do run $c -h; run $c --bogus; done
run __coordinator
run explore --toolchain-root relative go-run ./x
run explore --json --toolchain-root relative go-run ./x
run plan --output /tmp/p --toolchain-root relative go-run ./x
run replay --toolchain-root relative /nonexistent
run minimize --toolchain-root relative /nonexistent
run resume --toolchain-root relative /nonexistent
run execute-shard --shard 0/1 --toolchain-root relative /nonexistent
run analyze --toolchain-root relative go-run ./x
run doctor --toolchain-root relative
run qualify --toolchain-root relative go-run ./x
run explore --output x go-run ./x
run explore --__plan go-run ./x
