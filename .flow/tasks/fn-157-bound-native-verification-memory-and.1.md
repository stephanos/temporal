---
satisfies: [R1, R2, R3]
---
# fn-157-bound-native-verification-memory-and.1 Seal independent full baseline and measure causal resource ownership

## Description
Seal the independent preservation baseline and measure the causal owner before repair (R1-R3). The conductor must activate this deferred spec first.

**Size:** M
**Files:** task-owned `.flow/tmp/fn157/task1/` evidence and a disposable independent comparison harness; no production edits.
**Touches:** [.flow/tmp/fn157/task1/**]

### Approach

- Preserve fn-151 task3 receipts, logs, kernel records and input pins recoverably before any worktree cleanup. Anchor to `31ac0e5e88e67a45c23a0623454ca302d5b3d18b` plus its recorded artifacts/corrections, not HEAD alone. Read `classification/native-memory-followup.md`, `classification/canonical-final.{json,md}`, `final-verification.json`, `filesystem-diagnosis.json` and `owned-temp-inventory.json` beneath that retained worker root.
- On the conductor's integrated activation baseline, discover all seven IR paths and every machine/Query/realization/manifest owner. Pin exact source, dirty authorized inputs if any, IR/Case/fixture trees, toolchain versions, environment/temp mounts, process/package concurrency, RAM/cgroup limits and available/peak bytes/inodes. Re-anchor against any completed fn-155 proof rather than inventing its results.
- Build a task-local `native-preservation` comparison harness against a frozen pre-repair reader executable/source and immutable captured complete outputs. Inventory original 154 Activity Query occurrences plus every added focused owner. Capture ordered tables/values and every R2 observable, including error precedence and failed witness replay. Frozen reference evaluation must never call the repaired helper. Reuse historical full proof only on exact matching inputs; independently regenerate the full new baseline otherwise. All seven Models may be processed separately to obtain an oracle on provisioned capacity; this earns no gate-fit credit. Missing complete reference output stops repair.
- Reproduce the named memory and storage failures or record their natural absence on a sufficiently provisioned runner. Instrument phases of admission, interpretation, binding/Realizer, Check and independent replay, Producer setup, per-Query lowering, conformance preparation and export serialization/admission. Record live/retained heap, transient allocation, per-process RSS, aggregate overlapping process pressure and kernel/cgroup scope separately; unavailable counters remain unknown. No forced GC or memory knobs.
- Measure owned scratch peak bytes/files/inodes by mount during normal no-update overlap, including Lift extraction, Bloop and retained gate history. Identify failing operation and bytes-versus-inodes cause rather than infer it from later free-space snapshots. Existing behavior sharing is baseline; test traced and failed evaluations separately.
- Name the demonstrated owner, native strategy, scratch budget/remedy and precise lane write surfaces. If data implicates only capacity, recommend provisioning and remove unsupported source edits. Serialize all real heavy probes with actual `fcntl`/`flock` on `/tmp/umpire-heavy-gates.lock`. Apply the parent's cumulative one-hour deferral policy without pass credit.

### Investigation targets

**Required:**
- `.flow/specs/fn-151-split-standalone-activity-into-smaller.md` and `.flow/tasks/fn-151-split-standalone-activity-into-smaller.3.md` - historical closure and retained evidence root.
- `tools/umpire/interp/machine.go:209`, `tools/umpire/interp/eval.go:201` - interpretation and current local sharing.
- `tools/umpire/check/claims.go:44`, `tools/umpire/check/checking.go:250` - bindings and independent witness replay.
- `tools/umpire/lower/lower.go:103`, `tools/umpire/lower/generated.go:61` - Producer and complete generator.
- `model/check/Gate.scala:394`, `model/irgen/Lift.scala:221` - ordinary overlap and extracted scratch.
- `tools/umpire/export/dump.go`, `tools/umpire/conformance/conformance.go:116` - export phase and assessor snapshots.

### Key context

Current public Model/Type/table accessors expose referenced data; no safe mutation/invalidation guarantee follows from their existence. Do not presuppose that cached tables or sequential generator iterations dominate heap. Storage and native owners are independent measurement questions.
## Acceptance
- [ ] Historical evidence and current activation pins are complete and recoverable; runner budget and unresolved counters have explicit scopes.
- [ ] Complete independent `native-preservation` oracle exists for every R2 observable and owner, with a seeded table, receipt, artifact and replay difference proving the comparison fails. Input/count mismatch or incomplete capture stops dependent repair.
- [ ] Named failures have phase-labelled measurements or natural complete results on the provisioned runner; causal native owner and scratch bytes/inodes demand are recorded without unsupported RSS conclusions.
- [ ] Reviewable native/scratch strategies and non-overlapping write seams exist, or Flow tasks are re-anchored by the conductor before repair. No repair or activation is inferred from preparation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
