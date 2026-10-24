The fork-path audit found that calling libc fork would invoke inherited application
pthread_atfork callbacks in the supervisor and target, before our child entry points.
Those callbacks can allocate or acquire locks, so the implementation bypasses them.

Linux uses libc's syscall shim for SYS_clone with SIGCHLD and no sharing flags; all
remaining clone parameters are zero, making architecture-specific pointer argument
ordering irrelevant. Both supervisor and target forks use this path. The existing
libc 0.2.191 registry source supplies the syscall numbers and scalar ABI. The isolated
fixture poison_atfork_child registers a callback that _exit(99)s if an ordinary libc
fork accidentally reappears; normal successful target execution distinguishes bypass.

For Darwin, Apple source proves the __fork C shim declaration and implementation:
- https://github.com/apple-oss-distributions/Libc/blob/71bbe350ab79eef58113991d817ccc6165061a64/sys/fork.c
  declares __fork as pid_t(void), calls it before LibSystem/application callbacks,
  and describes its return fixup.
- https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/custom/__fork.s
  has x86_64 and arm64 implementations normalizing the two-register kernel return
  into -1, zero or child PID; failure sets errno. The exported assembler symbol is
  ___fork (C's leading Mach-O underscore plus the __fork function name).
- https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/custom/SYS.h
  defines the arm64 entry macro with a global symbol directive.
- https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/EXTERNAL_HEADERS/architecture/i386/asm_help.h
  defines the x86_64 leaf macro with a global symbol directive.
- https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/libsyscall/Libsyscall.xcconfig
  links the kernel dylib by loading the syscall archive.

This is source evidence, not a Darwin SDK link or native execution receipt. __fork is
private API: the Go project explicitly records that limitation in its 2023 revert,
https://go.googlesource.com/go/+/0a0de0fc4241c7531860e3c1a03ab4a4643ec37f .
No claim of an ABI stability guarantee or current native macOS validation is made.
The Rust declaration names the C shim directly rather than inventing a generic
Darwin syscall fork ABI.

Actual investigation commands used curl -fsSL against these exact Apple GitHub source
URLs, the GitHub commits/main endpoints to resolve the two pinned revisions, and rg
for entry/export/link definitions. All successful source fetches exited 0. Three guessed
export-file paths and one guessed mach/arm64 header path returned curl 22/HTTP404;
those failed guesses support no platform claim. No native Darwin tool was available.
