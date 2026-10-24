# Fork boundary audit

`umpire-backend-process` runs its supervisor in process. `support.rs` and the integration
controls forbid unsafe code; only `syscall.rs` and its child module may use it. The
process Cargo policy mirrors every workspace lint except its documented crate-local
unsafe deny, following the existing runtime precedent. The workspace forbid is unchanged.

The parent prepares executable PATH resolution, byte-owned argv/environment strings,
NUL-terminated pointer tables, current directory, capture descriptors, the owner pipe,
signal actions/masks, `/proc` parsing storage and descriptor-close bounds before forking.
Prepared storage remains live until the child enters its nonreturning path; parent drops
cannot change the child's private fork copy. Every forked exit calls `_exit`.

The supervisor entry, target entry and their transitive cleanup, signal and error callees
admit only audited system interfaces, raw pointer operations and stack scalar operations.
No child code invokes formatting, allocation, collections, Rust process/filesystem helpers,
locks, indexed bounds checks or destructors. The first-signal slot is a lock-free 32-bit
atomic; compilation rejects a supported target without that intrinsic. PID parsing carries
partial decimal tokens across bounded reads and checks multiplication against `pid_t::MAX`.
Read offsets are bounded by the successful read size; diagnostic scanning uses fixed static
NUL-terminated strings. Stop-signal arithmetic and positive target PID negation are bounded
by the operating-system signal/PID ranges. libc wait-status predicates are scalar bit tests.

The target receives a separate session, null stdin and restored caller signal mask before
execve. The supervisor has a private process group, retains the owner-liveness read end,
kills the target session and waits for it. Linux subreaping then repeatedly kills and reaps
adopted descendants, including those which escaped the target group/session, before exit.
Darwin admits only a non-root real/effective identity and sets child-only NPROC zero.
The corresponding Rust helper is exercised on Linux as a simulation; this is not native
Darwin evidence. Unsupported operating systems refuse explicitly.

The callback controls independently failed with child status 99 when either fork was
changed back to libc fork; the production source was restored after each mutation. Both
forks bypass inherited application pthread_atfork callbacks. A fresh-target concurrent
run then found a genuine inherited glibc abort-lock deadlock. The exact task-owned stuck
supervisors, targeted cleanup and observed gate exit 101 are retained in
`cold-deadlock.json` and `focused-deadlock.log`. The glibc source receipts explain the
SIGABRT locking path and concurrent posix_spawn lock. The final signal setup bypasses
that path. Cancellation and reserved-signal details are recorded in the child-call source
receipts; passing concurrency tests alone are not the complete safety argument.

Capture handles, inode checks, bounded UTF-8 diagnostics, dump admission and scratch cleanup
stay in safe Rust. Java heap configuration remains a Java heap limit, and regular-file size
polling remains polling; neither is claimed as a kernel memory/output cap. Supervisor
SIGKILL recovery is not promised. Linux lifecycle controls assert direct and escaped PID
disappearance before their fallback cleanup. Native macOS and Darwin SDK linkage are
unavailable here. Apple's private `__fork` entry has source/export evidence, not an ABI
stability guarantee; see `platform-source-evidence.md`.

The primary review corrected the initial PATH and clone assumptions. Every PATH
candidate byte string and pointer is prepared before fork; the target iterates
execve attempts using only that immutable sentinel-terminated table and scalars.
Linux clone uses an explicit stack-first prefix on s390x, six zero-filled ABI
slots, and raw PID comparison for SPARC child-return normalization. These have
native host regression controls plus source/table controls for the unavailable
architectures; no non-host execution is claimed. See clone-abi/README.md.
