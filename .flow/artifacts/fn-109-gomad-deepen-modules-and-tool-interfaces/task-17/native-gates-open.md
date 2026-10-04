# Task 17 acceptance awaits qualified native gates

The network-owner source candidate and canonical simulation-gate correction
are frozen, independently reviewed and pass feasible conductor checks; see
handover.md, evidence.json, source-audit.md and conductor-verification.md.
This is not task completion or R12 acceptance.

Actual host is linux/arm64. The pinned patched executable is absent. With
stock Go 1.27.1 on PATH, the native builder exits 2 because complete mode
requires darwin/arm64 or linux/amd64 (native-toolchain.log). The unchanged
Mach-O linter cannot execute on this host; task-16 metadata remains applicable.

Required patched rebuild, native overlay/network/process tests, real Runner
transport, full host/test-host and supported-platform qualification remain
open. Scratch adapters and link-only/compile-only checks are developmental,
not IPC, timers, replay or isolation proof. Preserve D12, resolved D14 and
the existing strict-delay watchdog disposition. The scoped filter fix selects
new process cases without relaxing those expectations.

Keep Flow acceptance blocked until its actual native commands qualify the
integrated source. MILESTONES item 4 permits downstream source advancement
after review; it does not permit completion. User owns commits; commits [].
