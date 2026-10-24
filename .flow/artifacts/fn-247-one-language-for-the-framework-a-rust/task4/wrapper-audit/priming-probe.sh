#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/umpire/temporal || exit 1
capture() {
 printf 'COMMAND:'; printf ' %q' "$@"; printf '\n'
 "$@"; command_rc=$?; printf 'command_rc=%s\n' "$command_rc"; return "$command_rc"
}
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/apple-exports_internal.h https://raw.githubusercontent.com/apple-oss-distributions/libpthread/42d026df5b07825070f60134b980a1ec2552dfee/src/exports_internal.h || exit $?
capture sed -n 112,114p /tmp/fn2474-wrapper-audit/apple-exports_internal.h || exit $?
capture rg -n 'declare_symbol\(pthread_setcancelstate\)' /tmp/fn2474-wrapper-audit/apple-exports_internal.h || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/apple-pthread.h https://raw.githubusercontent.com/apple-oss-distributions/libpthread/42d026df5b07825070f60134b980a1ec2552dfee/include/pthread/pthread.h || exit $?
capture rg -n 'PTHREAD_CANCEL_DISABLE|PTHREAD_CANCEL_ENABLE|pthread_setcancelstate' /tmp/fn2474-wrapper-audit/apple-pthread.h || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/apple-pthread-cancellation.c https://raw.githubusercontent.com/apple-oss-distributions/libpthread/42d026df5b07825070f60134b980a1ec2552dfee/src/pthread_cancelable.c || exit $?
capture sed -n 120,143p /tmp/fn2474-wrapper-audit/apple-pthread-cancellation.c || exit $?
capture sed -n 207,219p /tmp/fn2474-wrapper-audit/apple-pthread-cancellation.c || exit $?
capture sed -n 980,995p /tmp/fn2474-wrapper-audit/apple-kern_sig.c || exit $?
printf 'Source-only control: parent temporary DISABLE makes EINTR priming query return EINVAL without consuming pending cancellation; restoration returns normally in the inspected Darwin implementation. No native Darwin execution or link result is claimed.\n'
