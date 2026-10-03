---
satisfies: [R3, R5]
---
# fn-112-gomad-determinism-assurance-and-test.5 Inventory seeded-stream draw sites and check host-timed paths at runtime

## Description
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
## Acceptance
- [ ] Findings Q2 and Q3 re-anchored and marked confirmed, changed, or refuted
- [ ] The inventory test lists every seeded-stream reference with a classification and fails on an unclassified one; a deliberate unclassified reference demonstrates the failure
- [ ] Every host-timed site draws from the M-local stream, or is recorded as blocked by the collector prohibition
- [ ] The diagnostic-mode check stops the process in the negative fixture
- [ ] If any site was rerouted: core and smoke qualification sets pass on the new toolchain identity
- [ ] The inventory's location and the reroutes made are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime` pass on darwin/arm64; linux status recorded
## Done summary
Delivered R3/R5 on Darwin runtime key `2ecdbd330bb5928f360739fdd80cb3eb0f0764e513a5cc208cd83714fff0a457`. Q2 changed: purpose streams already existed, but target admission and host netpoll/no-P batch admission shared a seeded shuffle. Explicit batch origins retain target seeded diversification and route host batches to M-local entropy; the Linux CPU-profiler timer sample is also rerouted. Q3 confirmed: suspendG still performs its one-shot syscall wait before its retry loop; D12 owns that correction.

The checked-in inventory is `tools/gomad3/toolchain/draw_inventory_test.go`: 269 platform references, 135 classified rows, both qualified platforms, declarations/calls/function values and package-scoped linkname aliases. Unclassified references, count drift, and disappeared references fail closed. Scratch controls demonstrated an unclassified direct draw and alias uses in another file; the cross-file control failed before the fix and passes afterwards. The classic collector's `runtime/mgcpacer.go:enlistWorker` remains an explicitly recorded host-timed seeded draw blocked by the collector-file prohibition. No collector file was edited. Green Tea's worker draw is outside the qualified profile: seeded activation is rejected before user code, with a passing unseeded control.

Diagnostics use a per-M host scope and check each seeded entry before counters or state change. Existing numeric perturbation still localizes ordinal 5; `host:5` fails with the host-timed seeded-draw diagnostic. Seeded raw builders and supported Make launchers own `GOEXPERIMENT=nogreenteagc`. The full gate exposed another simulation integration builder, fixed at its compile boundary without changing any assertion or simulation selection. Additional existing test owners used instead of the initially proposed new fixture file: diagnostics_toolchain_test.go, simulation_root_integration_test.go, and target/internal/livecap/toolchain_test.go; generated protocol/descriptor consumers follow the changed runtime identity. Parent diagnostics choices golden changed in seven identity-derived fields only; the plain golden and complete byte comparison remain.

Verification: required validate/test-toolchain/test-runtime passed. The final census fix passed its focused and full toolchain suites and scoped vet; the parent retained a final patched-driver toolchain run. All components of the nested full test gate passed on stable inputs, with already green components reused after the integration-only helper correction; `fn-114/qualification/darwin-2ecdbd33/combined-gates.md` lists each component and log. Host gate: 45 packages. Core: 7/7 qualified and replayed. Smoke: 4/4 qualified and replayed. Scoped vet, integration-tagged execution vet, gofmt, diff check, and 16/16 source-hash verification passed. Linux/amd64 classic runtime cross-build passed, but native Linux qualification remains unavailable. The existing root lint cannot analyse nested-module paths; its prior environment-failure log is retained, rather than reporting lint green.

Independent working-tree implementation review: SHIP, codex gpt-6-sol at high, same-family reviewer with fresh context, no introduced findings. The reviewer inspected the uncommitted patch, 16 bound sources, generated/test changes, and gate evidence. `working-tree-review.json` is the acceptance receipt. The earlier task-scoped empty HEAD..HEAD review explicitly excluded uncommitted work; its receipt is preserved as `review-empty-committed-scope.json` and is not acceptance evidence. The recorded collector blocker and a possible census extraction were nonblocking; no speculative refactor was added.

Task-10 documentation facts: inventory path/counts; host batch and Linux profiler reroutes; diagnostic host mode; classic-only seeded activation and raw disabled control; remaining classic collector blocker; Q3/D12 ownership. Shared candidate qualification stays with fn-114.14. Its first representative run stopped at the unchanged 2 GiB space bound after 1/28; caches were cleared and the unchanged manifest retry is running. Representative and native Linux acceptance remain unproven. No disposition was weakened; no staging, commit, or push was performed.

Evidence: task-5/handover.md, source-hashes.txt, working-tree-review.json, and the shared fn-114 qualification/darwin-2ecdbd33 directory. HEAD is d635e23f00d926a43b942f25a9d05bd0ccb72025 with uncommitted sources.

stage: wave-dispatch - ran (model: gpt-6-sol at high)
stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate test-toolchain test-runtime (Darwin, exit 0; required-gates.log), Final cross-file alias census: focused and full toolchain tests, stock Go1.27.1, -tags test_dep -count=1, exit0; scoped toolchain vet exit0, make -C tools/gomad3 -o toolchain test-toolchain (candidate patched driver, -tags test_dep -count=1; census-final.log), Focused numeric diagnostic perturb, host:5 rejection, GreenTea seeded rejection/unseeded control, collector-profile unit control: exit0, make -C tools/gomad3 -o test-toolchain -o test-runtime test: harness/interception/45-package host/nine-overlay packages passed; simulation integration exposed a raw builder collector-profile gap, make -C tools/gomad3 -o test-harness -o test-toolchain -o intercept-test -o test-host -o overlay-test -o test-runtime test: exit0, 252.304s; simulation/World race/builder/live capability/upstream passed; unchanged green components reused explicitly, make -C tools/gomad3 core-qualification: exit0,192.040s,7/7 qualified and replayed on2ecdbd33, make gomad3-smoke-qualification: exit0,402.453s,4/4 qualified and replayed on2ecdbd33, Patched Go vet -tags test_dep ./toolchain ./internal/gomadtool/conformance ./runner/internal/execution ./runner ./target/internal/livecap: exit0, Patched Go vet -tags test_dep,integration ./runner/internal/execution: exit0, GOOS=linux GOARCH=amd64 GOEXPERIMENT=nogreenteagc CGO_ENABLED=0 patched Go build runtime: exit0; cross-compilation only, Pinned gofmt: no output; git diff --check: exit0; 16/16 source SHA256 bindings match, flowctl codex impl-review standalone working tree --base HEAD --spec codex:gpt-6-sol:high: SHIP; working-tree-review.json, NOT VERIFIED: native linux/amd64; representative qualification is still running under fn114.14, ROOT LINT UNAVAILABLE: root module cannot load nested module paths; reuse fn114/task-13/integrated-root-lint.log environment failure
- PRs: