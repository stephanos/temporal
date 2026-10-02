---
satisfies: [R3, R5]
---
# fn-112-gomad-determinism-assurance-and-test.5 Inventory seeded-stream draw sites and check host-timed paths at runtime

## Description
Stream isolation (R5): a checked-in, classified inventory of every seeded draw site, rerouting of any host-timed site still on the seeded stream, and a diagnostic-mode runtime check.

**Size:** M
**Files:** new `tools/gomad3/toolchain/draw_inventory_test.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, a new toolchain test and fixture directory
**Touches:** [tools/gomad3/toolchain/draw_inventory_test.go, tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/runner/internal/execution/draw_check_toolchain_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/**]

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
## Acceptance
- [ ] Findings Q2 and Q3 re-anchored and marked confirmed, changed, or refuted
- [ ] The inventory test lists every seeded-stream reference with a classification and fails on an unclassified one; a deliberate unclassified reference demonstrates the failure
- [ ] Every host-timed site draws from the M-local stream, or is recorded as blocked by the collector prohibition
- [ ] The diagnostic-mode check stops the process in the negative fixture
- [ ] If any site was rerouted: core and smoke qualification sets pass on the new toolchain identity
- [ ] The inventory's location and the reroutes made are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime` pass on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
