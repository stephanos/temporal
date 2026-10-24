The earlier claim that zero pointer arguments made raw clone argument ordering
immaterial on every Linux architecture was false. SIGCHLD is nonzero and must
occupy the architecture's flags slot. The primary review identified s390x's
stack-first layout; this report corrects that claim from kernel source. It also
checks the child return convention, rather than assuming argument parity alone
establishes a working fork.

All source references below are Linux v6.12, with exact Git blob SHAs and observed
command exits retained in the probe logs. The local Rust 1.99.1 target-list query
exited zero; rust-targets.txt is its unmodified output. No s390x, SPARC or other
non-host execution is claimed. Native Linux/aarch64 execution belongs to the
owner's test receipts, not these source inspections.

| Kernel clone family | Argument positions after syscall number | Native Rust/Linux families |
| --- | --- | --- |
| Normal | flags, stack, parent_tid, child_tid, tls | x86_64 including x32; csky, hexagon, loongarch64, m68k |
| CLONE_BACKWARDS | flags, stack, parent_tid, tls, child_tid | aarch64, arm/thumb, x86 32-bit, MIPS 32/64/n32, PowerPC 32/64, RISC-V 32/64 |
| CLONE_BACKWARDS2 | stack, flags, parent_tid, child_tid, tls | s390x |
| CLONE_BACKWARDS3 | flags, stack, stack_size, parent_tid, child_tid, tls | MicroBlaze kernel ABI; not present in the queried Rust native Linux target list |
| SPARC custom | flags, stack, parent_tid, tls, child_tid | sparc, sparc64; its return convention also needs normalization |

The executable definitions are in kernel/fork.c, lines 2895–2916. The
CLONE_BACKWARDS2 selection is explicit in arch/s390/Kconfig, and its syscall table
routes clone to sys_clone. The x86 CLONE_BACKWARDS selection is inside X86_32;
x86_64 uses the normal layout. MicroBlaze selects CLONE_BACKWARDS3. The generic
Kconfig help for BACKWARDS3 does not describe the current executable six-argument
prototype accurately, so the report follows SYSCALL_DEFINE6 rather than that help.
SPARC has an architecture-specific implementation in arch/sparc/kernel/process.c,
which reads flags from I0, stack from I1, parent_tid from I2, TLS from I3 and
child_tid from I4. Its zero stack value explicitly selects the parent's stack.

A sufficient argument vector for these non-sharing fork calls is six explicit
machine-word arguments. On s390x use (0, SIGCHLD, 0, 0, 0, 0); elsewhere use
(SIGCHLD, 0, 0, 0, 0, 0). Five-argument ABIs ignore the sixth value. The explicit
sixth zero also prepares the BACKWARDS3 TLS argument rather than relying on an
uninitialized variadic register. With all non-flags fields zero, tail position
swaps do become immaterial. That narrower statement does not excuse the s390x
flags/stack swap.

SPARC exposes a separate return issue: both process_32.c and process_64.c place
the parent's PID in the child's primary result register and one in I1, while the
parent gets zero in I1. The generic glibc syscall.S returns the primary register
without translating that second register. Consequently a direct raw clone result
cannot be tested for zero to identify a SPARC child. A small normalization using
raw getpid before and after a successful clone can identify the child by its
changed current PID, without assembly, atfork callbacks or a new executable.
Linux kernel/sys.c implements getpid as the current task's PID with no error
branch. This is a source-supported implementation direction, not an executed
SPARC control. The owner owns the actual correction and its focused tests.

The local target list also contains wasm32-wali-linux-musl, a virtual syscall
interface rather than a native Linux kernel architecture. This native kernel ABI
inspection does not assert WALI support or supply a virtual-host runtime result.

Evidence:
- probe.py / probe.log: generic kernel clone and top-level architecture Kconfig
  fetches, each exact command/status, URLs and Git blob SHAs; all fetches exit 0.
- custom-probe.py / custom-probe.log: SPARC custom entry/copy_thread definitions,
  s390/x86 syscall tables and MIPS routing; all fetches exit 0.
- returns-probe.py / returns-probe.log: both SPARC libc syscall shims and the
  kernel PID query, plus observed register-return definitions; every command 0.
- raw.githubusercontent.com returned HTTP 429/exit 22 for initial direct source
  fetches. The official GitHub contents API supplied the retained source objects;
  failed direct requests are not treated as passing observations.

No project source, index, Flow state or Cargo process was modified by this lane.
