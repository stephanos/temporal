Stream isolation (R5): a checked-in, classified inventory of every seeded draw site, rerouting of any host-timed site still on the seeded stream, and a diagnostic-mode runtime check.

**Size:** M
**Files:** new `tools/gomad3/toolchain/draw_inventory_test.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, a new toolchain test and fixture directory, generated protocol consumers, seeded conformance helpers, and the nested/root Makefile seeded launchers
**Touches:** [tools/gomad3/toolchain/draw_inventory_test.go, tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/runner/internal/execution/draw_check_toolchain_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/**, tools/gomad3/internal/gomadtool/conformance/runtime*.go, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/Makefile, Makefile]

### Approach
- First step (R3): re-anchor assessment findings Q2 and Q3 against the current patch.
- Mirror the clock inventory test: enumerate references to the runtime rand helpers in the patched source, require each to carry a reviewed classification (target-ordered or host-timed), and fail on an unclassified reference.
- Known host-timed sites already on the M-local stream: lock-profile sample, anti-starvation wake, steal order, symtab cache. Known seeded sites: runnext flip, run-queue shuffle and pick, select, timer rand.
- Reroute any host-timed site still on the seeded stream. A site inside a collector file is not edited; record it and raise spec Open Question 3.
- Runtime check, diagnostics only: fail the process when a path classified host-timed draws from the seeded stream. Trigger it with the task 3 fault switch in a negative fixture run through the toolchain-test launcher in `runner/internal/execution`.
- Regenerate the patch with the governed `patch-regenerate` command; do not hand-edit hunks.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/clock_inventory_test.go:86-212` — inventory pattern to mirror
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:89-99`, `:274-275`, `:745-746` — host-timed reroutes
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:556-594`, `:693-694`, `:780-806` — seeded sites
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:614-644` — rand helpers

**Optional** (reference as needed):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:369-477` — overlay rand helpers
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:121-127` — the one-shot syscall wait in `suspendG` (Q3, owned by fn-105 D12)

### Key context
- Rerouting a draw shifts every seed's schedule. Seed-specific expectations (seeds 11 and 17 in the manifests, D16 seed lists) must be requalified, so batch this with the other runtime edits named in spec Open Questions 2.
- fn-105 D26 moves forward-clock draws; if it has landed, the inventory reflects it.

### Re-anchor (2026-10-02)
Q2 is changed: purpose streams already exist, but runqputbatch still routes target/global admission and host netpoll/no-P admission through the same seeded shuffle helper. Preserve target-purpose diversification and route only host-timed batches to the M-local stream. Q3's one-shot wait remains confirmed by source and belongs to D12; this task does not change suspendG.

Go 1.27.1 enables Green Tea by default. Runner preparation explicitly selects the qualified classic collector, while the current raw seeded conformance builders and Make launchers omit that compile profile. Inventory classification must reflect the qualified profile; seeded Green Tea activation is refused outside it without editing collector files. Seeded harness/helper builds and the supported seeded launchers must establish the classic profile themselves. Keep a raw Green Tea rejection and unseeded control; a one-off environment override is not qualification evidence. These launcher and generated-protocol updates are directly implied by the runtime acceptance boundary.

Extend the existing trusted diagnostic fault switch with an explicit host:<ordinal> mode for the host-path negative control. The numeric ordinal mode continues to perturb a target draw and must retain task 3's successful diagnostic localization. Per-M host-path scope must be checked before seeded counters or state change.
