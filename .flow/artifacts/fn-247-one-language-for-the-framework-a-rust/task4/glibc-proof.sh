#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/umpire/temporal || exit 1
capture() {
    printf 'COMMAND:'
    printf ' %q' "$@"
    printf '\n'
    "$@"
    command_rc=$?
    printf 'command_rc=%s\n' "$command_rc"
    return "$command_rc"
}
capture ldd --version || exit $?
capture curl -fsSL -o /tmp/fn2474-glibc-sigaction.c https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/signal/sigaction.c || exit $?
capture sed -n 23,42p /tmp/fn2474-glibc-sigaction.c || exit $?
capture curl -fsSL -o /tmp/fn2474-glibc-spawni.c https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/sysdeps/unix/sysv/linux/spawni.c || exit $?
capture rg -n '__abort_lock|__libc_sigaction' /tmp/fn2474-glibc-spawni.c || exit $?
capture curl -fsSL -o /tmp/fn2474-glibc-kernel-sigaction.h https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/sysdeps/unix/sysv/linux/kernel_sigaction.h || exit $?
capture sed -n 7,18p /tmp/fn2474-glibc-kernel-sigaction.h || exit $?
capture curl -fsSL -o /tmp/fn2474-glibc-libc-sigaction.c https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/sysdeps/unix/sysv/linux/libc_sigaction.c || exit $?
capture sed -n 42,56p /tmp/fn2474-glibc-libc-sigaction.c || exit $?
printf 'Inference: raw clone can inherit the posix_spawn abort read lock; public sigaction(SIGABRT) then blocks acquiring the copied write lock. Raw rt_sigaction for the zero/default SIGABRT action avoids that lock.\n'
