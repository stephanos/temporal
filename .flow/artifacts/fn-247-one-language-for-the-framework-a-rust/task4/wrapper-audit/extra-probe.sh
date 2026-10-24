#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/umpire/temporal || exit 1
capture() {
 printf 'COMMAND:'; printf ' %q' "$@"; printf '\n'
 "$@"; command_rc=$?; printf 'command_rc=%s\n' "$command_rc"; return "$command_rc"
}
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/glibc-dlsym.c https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/dlfcn/dlsym.c || exit $?
capture rg -n 'lock|unlock|_dl_sym|dlsym' /tmp/fn2474-wrapper-audit/glibc-dlsym.c || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/glibc-errno.h https://raw.githubusercontent.com/bminor/glibc/04e750e75b73957cf1c791535a3f4319534a52fc/include/errno.h || exit $?
capture rg -n 'errno|tls_model' /tmp/fn2474-wrapper-audit/glibc-errno.h || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/apple-resource.h https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/sys/resource.h || exit $?
capture rg -n 'RLIMIT_POSIX_FLAG|RLIMIT_NPROC' /tmp/fn2474-wrapper-audit/apple-resource.h || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/linux-mipsn32.tbl https://raw.githubusercontent.com/torvalds/linux/v6.12/arch/mips/kernel/syscalls/syscall_n32.tbl || exit $?
capture rg -n 'clock_nanosleep|ppoll' /tmp/fn2474-wrapper-audit/linux-mipsn32.tbl || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/linux-time_types.h https://raw.githubusercontent.com/torvalds/linux/v6.12/include/uapi/linux/time_types.h || exit $?
capture sed -n 5,8p /tmp/fn2474-wrapper-audit/linux-time_types.h || exit $?
capture sed -n 27,30p /tmp/fn2474-wrapper-audit/linux-time_types.h || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/linux-x86.tbl https://raw.githubusercontent.com/torvalds/linux/v6.12/arch/x86/entry/syscalls/syscall_64.tbl || exit $?
capture rg -n 'clock_nanosleep|ppoll' /tmp/fn2474-wrapper-audit/linux-x86.tbl || exit $?
capture rg -n '__pthread_canceled' /tmp/fn2474-wrapper-audit/apple-syscalls.master || exit $?
capture curl -fsSL -o /tmp/fn2474-wrapper-audit/apple-kern_sig.c https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/kern/kern_sig.c || exit $?
capture sed -n 970,995p /tmp/fn2474-wrapper-audit/apple-kern_sig.c || exit $?
capture readelf -d /home/agent/.cache/codex-build/cargo-target/debug/deps/umpire_backend_process-e7d9c6953b360575 || exit $?
