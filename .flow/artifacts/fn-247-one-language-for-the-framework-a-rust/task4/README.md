# Native process supervisor evidence

The process library now owns its supervisor in process, with parent-prepared executable,
argv/environment, descriptor, signal, buffer and function-binding state. Safe support keeps
its original public API, capture handles and diagnostics. Only the audited syscall subtree
admits unsafe code. The workspace forbid and every other mirrored lint value are unchanged.
`libc =0.2.191` is the only promoted dependency; package/version/checksum and feature
inventories remain equal, including the guest closure. No runtime Cargo, Python bootstrap
or sibling supervisor executable is required. The fixture binary is test support only.

`control-parity.json` maps all eight retained Rust and six Python process controls to named
Rust tests. The fresh-target focused run passed 22 controls, including actual Linux direct
and escaped descendant disappearance before fallback cleanup, owner-group TERM/KILL,
supervisor TERM, owner EOF, timeout and cancellation. The allocator control runs 48 supervised
execs across four callers while four allocator threads and a std::process spawn thread run.
The two Python process files were deleted only after this counterpart run passed. Other
Model Python helpers belong to the later tasks and remain outside this port.

`observations.json` records actual commands, exits and source hashes. Final focused evidence
is 22 process tests, 21 TLC/Apalache unit tests and one required real-tool control per backend:
45 distinct passing test identities. Clippy passed for every process target. The direct cold
consumer was built in a fresh target with lazy linking, has no BIND_NOW flag or sibling fixture,
and runs with PATH pointing to no tools. Its baseline failed ENOENT before the port and its
replacement passed. `run-cold-consumer.sh` retains the actual command recipe. Both direct and
TLA-compatibility invocation corpora match the retained task1 stdout/stderr byte-for-byte
across 13 records; see `run-corpus.sh`. All 752 protected artifact hashes remain unchanged.

The safety audit found real defects while implementing the port. Separate libc-fork mutants
failed the atfork guard with child status 99 for each fork. Raw clone then exposed an inherited
glibc abort-lock deadlock under actual concurrent testing; the stuck owned PIDs, deliberate
cleanup and observed exit 101 are retained, not rounded up to green. The final implementation
uses cancellation-free child calls, complete Linux kernel signal masks/defaults and
parent-resolved typed entry points. `fork-safety-audit.md` and `wrapper-audit/audit.md` give the
per-entry argument and source receipts. Intermediate Clippy/format and source-fetch failures
are recorded honestly. Later passing runs followed the corresponding source fixes.

Native evidence is Linux/aarch64, glibc 2.43, non-root uid/euid 1000. Darwin policy and the
actual shared NPROC helper have Linux simulation/source controls; no native macOS or SDK
link is claimed. Apple private syscall entries have pinned source/export evidence, not a
stability guarantee. The Darwin fork-error cancellation guard and fresh-exec state reset are
supported by the retained kernel/thread/LibSystem source chain. No supervisor-SIGKILL recovery,
kernel memory cap or strict kernel output cap is claimed.

The fixed pre-edit base is 60c65fcb2922e953a558541343c65a4998519e2f. Root's separately owned
metadata-only commit 2c9a6959086b4ac4ce736fe3d14f276705bff6ab is included in the range and is
not this worker's implementation. Task lane branches were integrated, and their clean
worktrees/branches removed normally. No full Model validation belongs to this task; final
 task 3 retains its one combined TLC/Apalache required validation pipeline. Documentation
that temporarily described Python ownership is also assigned to final task 3.
