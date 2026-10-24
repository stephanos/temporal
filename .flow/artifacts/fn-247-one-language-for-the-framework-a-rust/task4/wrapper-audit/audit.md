The audit distinguishes a kernel system call from its public libc wrapper. Public
async-signal-safe labels alone did not establish the stronger no-lock/no-unwind
contract after a raw fork. Observed cold concurrent tests hung before target fork;
glibc source explains an inherited abort lock in sigaction(SIGABRT). Source review
also found cancellable wrappers and reserved signal filtering, requiring changes.

Pinned sources and exact fetch/inspection commands are in probe.py/probe.log.
The glibc revision is 04e750e75b73957cf1c791535a3f4319534a52fc;
Apple xnu f6217f891ac0bb64f3d375211650a4c1ff8ca1ea;
Apple Libc 71bbe350ab79eef58113991d817ccc6165061a64;
Apple libplatform 2512ffd8bb5c6caff3c8ea83331ab8a0adc820c3;
Apple libpthread 42d026df5b07825070f60134b980a1ec2552dfee.
These are source observations, not native Darwin execution or SDK link receipts.

| Child entry | Linux audit and implementation | Darwin audit and implementation |
| --- | --- | --- |
| fork | Raw SYS_clone(SIGCHLD, zero remaining parameters), bypassing application atfork callbacks. | Direct __fork assembler shim; parent primes its rare cerror/EINTR cancellation query under temporary cancellation disable. Each child immediately disables kernel cancellation, making the error unwind branch unreachable. Private ABI limitation retained. |
| setpgid | Generated scalar non-cancelling stub in Linux syscalls.list. | Generated scalar non-cancelling stub in syscalls.master/create-syscalls.pl. |
| sigaction reset | Public wrapper takes abort lock for SIGABRT and rejects NPTL-reserved signals. Replace ALL default resets with raw rt_sigaction and parent-prepared zero storage. Every kernel field zero means default handler, flags/mask/restorer zero. | Libc delegates to libplatform's scalar stack conversion to __sigaction; no allocation/lock in that conversion. Public fatal-signal restriction branch is compile-time conditional; no native certification. |
| sigaction stop handlers | Four non-SIGABRT calls to libc wrapper: source bypasses abort lock, then scalar kernel action conversion/syscall. | Same scalar trampoline conversion; masks/structures already prepared in parent. |
| dup2 | Direct dup2 or dup3 syscall. The fallback fcntl branch requires fd==fd2; all actual child mappings have unequal descriptors (prepared source>=10 to 0..3, then 3 to 0). | Generated non-cancelling syscall stub. |
| syscall | AArch64/x86_64 assembly only moves arguments, enters kernel and stores errno on error; no cancellation wrapper. | No generic Darwin syscall fork ABI is invented. Private/nocancel generated stubs use verified named entries. |
| prctl | Generated scalar non-cancelling stub; Linux only. | Not called. |
| open | Public open uses SYSCALL_CANCEL. Child instead uses raw openat for fixed proc pathname/flags. | Not called by forked production paths. |
| sigprocmask | Public wrapper delegates pthread_sigmask, which strips reserved internal bits. Parent and children instead use raw rt_sigprocmask with full prepared kernel bits, saving/restoring original bytes exactly. | Generated non-cancelling syscall stub. |
| waitpid | Public waitpid calls cancellable wait4; child instead uses raw wait4 with null rusage. | waitpid selects wait4 variants. Child binds __wait4_nocancel directly with proven four-argument signature. |
| poll | Public poll uses cancellation; child raw ppoll uses its existing pollfd and a fixed 5ms kernel timespec. | poll$NOCANCEL alias maps to __poll_nocancel generated stub. |
| read | Public wrapper uses cancellation; child raw read. | read$NOCANCEL alias maps to __read_nocancel. |
| kill | Generated scalar non-cancelling stub in generic Unix syscalls.list. | Scalar wrapper adds UNIX03 POSIX flag then __kill, which is not in cancellation list. |
| close | Public wrapper uses cancellation; child raw close. | close$NOCANCEL alias maps to __close_nocancel. |
| setsid | Generated scalar non-cancelling stub. | Generated non-cancelling stub. |
| chdir | Generated scalar non-cancelling stub. | Generated non-cancelling stub. |
| setrlimit | Scalar rlimit conversion to prlimit64; not a Linux production child call. The isolated simulation calls the same NPROC setter after normal exec. | Scalar UNIX03 flag conversion then __setrlimit, absent from cancellation list; actual target invokes NPROC(0,0). |
| execve | Generated scalar non-cancelling syscall stub; prepared argv/env/path only. | Direct resolved __execve generated kernel stub avoids unproven public wrapper behavior. |
| nanosleep | Public nanosleep/clock_nanosleep uses cancellation; child raw clock_nanosleep with fixed relative duration. | Not called by forked production paths. |
| lseek | Scalar native/legacy offset conversion and raw syscall; no cancellation. | Not called by forked production paths. |
| write | Public wrapper uses cancellation; child raw write for fixed error text only. | write$NOCANCEL alias maps to __write_nocancel. |
| _exit | exit_group syscall loop; no exit handlers/destructors. | Generic syscall map aliases _exit to generated __exit stub; no userspace exit handlers. |
| errno | __errno_location returns existing thread-local errno address. Raw fork retains that TLS mapping; syscall error assembly stores directly in libc's existing TLS. | __error and cerror_nocancel read the fixed __TSD_ERRNO slot directly and fall back to static errno. They do not allocate a TLS key or storage. |

Public cancellable wrappers are unsafe here even when the normal Rust caller never
uses pthread_cancel: the library contract must not inherit an enabled/pending
cancellation path. Glibc's disabled-cancellation path is direct syscalls, but a
parent cancellation-state guard was rejected in favor of explicit child wrappers.
Apple cerror can call pthread_exit after EINTR; nocancel stubs select cerror_nocancel
and avoid that branch entirely.

Signal storage is prepared before fork. Linux masks and zero default action have
256-byte storage aligned to u64; the kernel copies only its supported signal bytes
(8 normally, 16 on MIPS). The byte count derives from libc SIGRTMAX in the parent.
The default action is never interpreted as libc's nonzero, differently laid-out
sigaction. Darwin continues to use its supplied libc signal structure ABI.

Time ABI is explicit rather than assuming libc::timespec matches a raw syscall.
Linux native 64-bit and x32 use two signed 64-bit slots. RISC-V 32-bit has only the
time64 ppoll/clock_nanosleep entries and uses that layout. Remaining legacy 32-bit
entries use two c_long fields, including MIPS n32; the Linux n32 syscall table maps
them to time32 handlers. The requested relative 5ms duration fits every form.
Sources: Linux v6.12 include/uapi/linux/time_types.h, arch/mips/kernel/syscalls/
syscall_n32.tbl, arch/x86/entry/syscalls/syscall_64.tbl, and the existing libc
0.2.191 syscall-number definitions. Native Linux execution does not prove 32-bit
or Darwin execution.

Binding/TLS closure is encoded in calls::Api. The parent resolves every typed
entry with dlsym(RTLD_DEFAULT), refusing a missing entry before fork. Static-CRT
builds use direct function addresses. Children call only volatile-loaded pointers;
there is no child dlsym or first-use PLT stub. The table and the existing errno
cell address are prepared in the parent and survive in the private fork copy.
Parent-only signal queries bind the signal conversion's transitive path; a harmless
setpgid(-1,-1) call binds the common errno path, and Darwin kill(getpid,0) binds its
scalar wrapper's internal syscall. No test depends on a warm executable or eager
consumer linking. The separate observed BIND_NOW binary is historical link evidence,
not the basis of the implementation guarantee. The owner also retains a real
lazy-linked cold-consumer control; this report does not fabricate its result.

Darwin's raw __fork shim still uses cerror on failure, which can test pending
pthread cancellation on EINTR. The implementation resolves the exact generated
__pthread_canceled(int) stub and calls action 2 as the first operation in each
forked process. The kernel implementation only sets UT_CANCELDISABLE; it invokes
no pthread/libSystem callback. Failure uses non-cancelling diagnostics and _exit.
This makes the supervisor's second-fork error path unable to invoke pthread_exit.
No action 1 is called before exec, so an exec failure stays in the disabled context.

Successful Darwin exec creates a replacement kernel thread rather than retaining
that disabled old uthread. The pinned xnu kern_exec.c __mac_execve path calls
cloneproc(CLONEPROC_EXEC) unconditionally. proc_exec_switch_task copies signal and
dtrace fields to the new uthread, without copying cancellation flags. thread.c
allocates the thread plus appended uthread from a CLEARMEM zone and calls
uthread_init, whose initialization does not inherit the old cancellation flags.
The new LibSystem initialization separately calls _pthread_struct_init and sets
user-space cancellation to ENABLE|DEFERRED. This source chain establishes the
fresh-image default; it remains distinct from unavailable native Darwin evidence.
See final-probe.py/final-probe.log for exact source fetches and selected observations.

The supplemental extra-probe.sh run exited 22 when raw.githubusercontent.com
returned HTTP 429 for a Linux kernel source. It is retained as a failed fetch,
not a passing audit observation. final-probe.py subsequently fetched the same
v6.12 kernel sources through the official GitHub contents API, retained their
blob SHAs, and observed exit zero for every fetch and inspection. No test suite
was rerun to replace an unobserved result.

Darwin rare-error binding is also prepared before fork. calls::Api::new resolves
parent-only pthread_setcancelstate and the OS_EXPORT _pthread_exit_if_canceled.
It temporarily disables cancellation, primes the EINTR query path, then restores
the previous state, checking both POSIX return codes. With UT_CANCELDISABLE set,
the kernel query returns EINVAL without consuming any pending cancellation, so
no pthread_exit occurs. The inspected Darwin setter restores kernel/user state
and returns normally; it has no glibc-style immediate asynchronous unwind branch.
Both symbols and DISABLE=0/ENABLE=1 are verified by pinned Apple export/public
headers. This does not add any child callback or library API. See
priming-probe.sh/priming-probe.log (observed overall and per-command exit zero).
The private Darwin syscall bindings and source-only/native-unavailable distinction
remain explicit; no extra package, assembler path or platform certification was added.
