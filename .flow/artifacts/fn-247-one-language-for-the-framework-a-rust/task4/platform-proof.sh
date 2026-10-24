#!/bin/bash
set -u
cd /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2474-syscall || exit 1
capture() {
    printf 'COMMAND:'
    printf ' %q' "$@"
    printf '\n'
    "$@"
    command_rc=$?
    printf 'command_rc=%s\n' "$command_rc"
    return "$command_rc"
}
capture curl -fsSL -o /tmp/fn2474-platform-fork.c https://raw.githubusercontent.com/apple-oss-distributions/Libc/71bbe350ab79eef58113991d817ccc6165061a64/sys/fork.c || exit $?
capture rg -n 'extern pid_t __fork|ret = __fork|_libSystem_atfork_child_v2\(flags\)' /tmp/fn2474-platform-fork.c || exit $?
capture curl -fsSL -o /tmp/fn2474-platform-__fork.s https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/custom/__fork.s || exit $?
capture rg -n 'LEAF\(___fork|MI_ENTRY_POINT\(___fork|zero eax|mov.*w0, #0|mov.*-1' /tmp/fn2474-platform-__fork.s || exit $?
capture curl -fsSL -o /tmp/fn2474-platform-SYS.h https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/custom/SYS.h || exit $?
capture sed -n 474,479p /tmp/fn2474-platform-SYS.h || exit $?
capture curl -fsSL -o /tmp/fn2474-platform-asm_help.h https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/EXTERNAL_HEADERS/architecture/i386/asm_help.h || exit $?
capture sed -n 205,214p /tmp/fn2474-platform-asm_help.h || exit $?
capture curl -fsSL -o /tmp/fn2474-platform-Libsyscall.xcconfig https://raw.githubusercontent.com/apple-oss-distributions/xnu/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/Libsyscall.xcconfig || exit $?
capture rg -n 'PRODUCT_NAME = kernel|DYLIB_LDFLAGS|LINK_WITH_STANDARD_LIBRARIES' /tmp/fn2474-platform-Libsyscall.xcconfig || exit $?
printf 'Evidence type: source declarations, assembly and build recipe; no SDK/native Darwin observation.\n'
