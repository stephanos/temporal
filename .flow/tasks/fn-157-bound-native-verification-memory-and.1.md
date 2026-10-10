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
Blocked:
# Complete pre-repair oracle unavailable on current capacity

Task fn-157-bound-native-verification-memory-and.1 is blocked, not complete or reviewed.
All owned commands and the worker/research child are terminal. Production source is unchanged.
Worker HEAD/task base: f7b2e2ee7d938da6c7bdb5a37d8232e0252f4dcc; executable activation
pin: 551bf89c68f9b8dd3ac8ec219e65f96f0aa85890. Later integrated changes are overview metadata.

Recoverable worker evidence root:
`/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-157-bound-native-verification-memory-and/.flow/tmp/fn157/task1/`.
Root read the handover and independently checked all 574 evidence-seal file hashes, exit 0.
Summary SHA256: 1456bae5f2e024b5377ea3767d1ec6661e58b2f6d85b28e7aab8b25238070df9.
Evidence SHA256: 57c4a97e9a942d88903e1955f793bca1ec29a96d6db6e6c076c94cd8a7f6ad2e.
Seal-file SHA256: cb11ea018d9a9a6776671e2ca19485e2d921217a621055ad3de70f2faf09aab6.

Captured: seven complete Models, 61 raw table owners, 393 typed Check receipts, all 340
Query answer/bind/replay records (including original 154 Activity occurrences), and full
Producer/Lower outputs for 275 Queries in the six non-primary Models. The primary Model's
65 Lower results and the full independent artifact/manifest/conformance/export/error-control
joins remain missing. `oracle-completeness.json` lists 34 outstanding surfaces/controls,
with complete=false and repair_allowed=false. These partial captures earn no gate-fit credit.

The primary Producer was naturally SIGKILLed before and after the exact normal exit of the
exclusively owned historical fn-155 private Bloop daemon: 17.724510883 and 19.258385842 seconds.
The shared compiler was untouched. Kernel windows identify global OOM; the later native victim
had 7,678,400 KiB anonymous RSS. Numeric PID namespace mapping and successful capacity remain
unproven. Heap profiles demonstrate transition clones and simultaneous bindings, not a sole
owner or an accepted repair strategy.

Unchanged `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` first failed
exit 2 after 9.955093963 seconds with Java /tmp creation ENOSPC: sampled overlay free inodes
2,001 to one while roughly 29.9 GB free bytes remained. Storage then changed externally;
the same gate retry passed scratch tests but failed exit 2 after 37.690613476 seconds, with
kernel global OOM for its Case generator (5,984,192 KiB anonymous RSS). Preserve its distinct
claim.pb.go/claim.proto timestamp warning; neither result proves completed Lift/Case overlap.

Total 52 measured probes: 412.922787696 wall seconds, with preparation/private lifecycle
separate. The one-hour deferral limit was not reached; this blocker is missing mandatory
reference output, not that time limit. No smaller input, forced GC, memory knob, selected-only
fit claim, source repair, review verdict or Flow completion substitutes for the missing oracle.

Revisit when a runner can execute the unchanged whole primary Producer and independent readers
with adequate RAM under measured ordinary compiler/process overlap, and effective JVM scratch
has sufficient verified bytes/inodes. Successful memory demand is unknown; victim RSS is only
a failed lower bound. Re-anchor changed executable inputs, resume the frozen complete oracle
and outstanding controls, then review .1 before any .2/.3/.4 repair. R3 strict assertions,
canonical full Go -json/test_dep/-p 2/-timeout 30m, ordinary no-update Cases/fixtures/model
gates, fn-154 JSON work and separate Canary/Batch 5 debt remain unweakened.

Blocked:
# Deferred at the owner's explicit request

On 2026-10-10 the owner asked to defer fn-157 for now, then authorized fn-156 and fn-146 implementation in parallel. Fn-157.1 is deferred, not complete or reviewed. All worker and child commands are terminal. No constructor-partition harness, directory, build or primary probe was created. Production source is unchanged.

Retain the worktree at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-157-bound-native-verification-memory-and` and its `.flow/tmp/fn157/task1/` evidence. Frozen source HEAD is `f7b2e2ee7d938da6c7bdb5a37d8232e0252f4dcc`; executable activation pin is `551bf89c68f9b8dd3ac8ec219e65f96f0aa85890`. Later integrated commits alter overview/planning only.

The original 574-file seal and controls-resume 525-file seal remain intact. Root independently verified the new controls-remaining-v2 151-file seal. Its summary SHA256 is `401eea3d7928b1f1024bba395cd8cb788b76e165e6836dd29ad1e395c047dd4b`, evidence SHA256 `3edbecd0802eb24c34307e523c612eecbf351bbf367209b786f0f90f069c0239`, and seal-file SHA256 `f6cb67db78e2e9b9c6f2d69349266bac7cbcae38ff7914e7820f6133e0c991c6`. Manifest SHA256 is `f4ebd1f4178a7601c3d3febbbc2db820c350a98e89b712d9e1b06c85b686f67b`.

Current preservation includes all seven Models/340 Query answer/replay records, 61 claimed owners (59 complete streams and two typed refusals), 275 non-primary Lower results and twelve Cases. The latest generation covers five additional Cases with eight no-server executor Runs/139 events, all 116 Query bindings on three Models, eighteen associated tables/126,062 ordered rows and three complete through refinements/1,306 rows plus one typed refusal. Retain natural inconclusive and forged-completion negative results. The owner-qualified inventory negative passed. The frozen repeated-Build nested Row.Results alias RED remains evidence.

Cumulative native evidence is 114 probes/596.297063286 measured wall seconds. Five original surfaces remain missing, including the primary 65-Query Lower oracle and dependent generated/conformance/dump joins. Complete oracle and repair_allowed remain false. No full native/model/case gate, production repair or fn-155 closure credit follows from the partial captures. External Quint JSON remains fn-154; Activity semantic debt remains Batch 5.

Read-only research identifies overlapping Realizer/Check/replay interpretations, transition clone allocations and pre-Lower canonical fingerprint strings as candidate peaks. A future separately pinned constructor-lifetime partition must preserve typed receipt provenance and first match every ordinary output on the six complete Models before any primary attempt. One non-primary Query receipt has a typed Cause; generic JSON receipt reconstruction is not yet an accepted seam. The proposed diagnostic is now deferred, not currently authorized to run.

Resume only when the owner revives fn-157. Re-anchor executable inputs, inspect `deferral-20261010-{summary.md,evidence.json}` outside the sealed subtrees, preserve all three seals and all RED receipts, then complete the independent oracle/causal measurements before any .2/.3/.4 repair. Unchanged canonical domains, strict assertions, default memory policy, -p2 concurrency, real shared heavy lock and normal no-update gates remain required. No paid provisioning, swap, shared-daemon termination or unowned cleanup is authorized.

stage: impl-review - skipped(policy: incomplete measurement and explicit owner deferral)
## Evidence
- Commits:
- Tests:
- PRs:
