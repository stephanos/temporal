# Primary integration and next source work

The reviewed task60–62 source checkpoints and their retained evidence are integrated at
primary merge `87388d61e05871346803b8fef26118302b0288b6`. Its second parent is
`28e4ad07aa7df8b8ec0661e7315da2117ccca98d`; task60 was already present through
`6723dfd8a293f60aae22085cf19788f31ab9443c`. The history-preserving merge required no
product conflict resolution. Owner changes under `.flow/specs`, `.flow/tasks` and `.turbo`
were left unstaged and unchanged.

Read-only `python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-60-62/verify_packet.py`
returned 0 in primary after integration. All 1,070 source inputs, five product hashes,
11 terminal receipts and their raw streams still verify. These are retained execution
receipts for `95354f677162df9cd76383569b11fe1f902870cd`, not newly executed primary gates.
Their source fingerprint remains
`fbe7657b7a8decc05f2873d04d24abafca179e1ec52fc9fd7dfc9d9c6f9a3297`.
The [gate results](gate-results.md) and [independent review](independent-review.md)
retain the passing boundaries, Runner failures, RED53 lint and open acceptance.

## Parallel work

- fn109.63 owns decomposition of `runLocal` in isolated worktree
  `/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/worktree`,
  admitted base `c58418ee1dc42f0760824f798e5714477cfab0d2`.
  Its worker holds the shared Go/build/test/lint/vet/generator lane.
- fn112.10 owns current documentation and command-source reconciliation in
  `/Users/stephan/Workspace/skunkworks/.gomad-fn10963-admission.Ho6KcW4t/task11210`,
  admitted base `18bcc34b07bd2a82edce969526fe362875674c57`.
  Source/evidence and document checks may run independently; executable gates require
  an explicit lane handoff.

Both remain in progress. Independent source-progress review precedes any source
checkpoint; neither worker may certify completion. No native qualification, CI,
PR or push action is authorized.

## Retained research decisions

The fn112.10 acceptance audit found its original-base lint prerequisite still operative.
Filtered fast-lint zero and standalone errortype do not satisfy that gate. Current
command reconciliation must account for changes in `compatibility_pack.go`, CLI `cli.go`
and gomadtool `main.go`; retained old binaries alone do not establish current help.
Its six direct predecessors are Done. Formal task acceptance remains open pending
source gates and review; native soak execution and measured bounds remain transferred.

The complete Runner audit classified the 123 failed top-level cases as 109 fixture
correction candidates and 14 real-boundary exclusions. Candidates are not automatically
portable units: five mix public/default and private substitution assertions. Follow
fn109.63 before admitting private per-call preparation, bootstrap, adapter verification
and toolchain-identity operations. Do not infer substitutions from executor type or
globally replace `testConfig`. Retain actual default paths, isolated-request rejection,
filesystem and artifact preflight, World validation, error precedence and lifetimes.
The complete failed-test inventory and first-blocker counts remain in the gate results.

The literal-message audit identifies 42 ST1005 findings, two deliberate-spin findings,
eight forbidden panics and one unchecked watchdog ready-marker write. The format
amendment supersedes encoded-format/identity equality, not clearly all diagnostic
wording; task44 explicitly preserves three schema messages and task53 excludes
application casing. Wording repairs require bounded admission, not a blanket lint
suppression or weaker error assertions. An immediate watchdog fixture exit on a real
readiness-write failure is a proposed behavior change, not an admitted correction.

The fn110 source audit finds all 87 retained task2 runtime-input hashes unchanged,
including 79 overlays. Tasks3 and4 already have their intended implementation.
They need current-source acceptance, not speculative runtime rework. Preserve fn109.9–12
source predecessor acceptance, then task3 before task4 and task5. Current archive
comparison must include now-unpatched crypto/rand initialization, syscall declarations
and source selection. Whole-toolchain receipt reuse remains unproved because builder
and conformance inputs changed. The required `.toolchain/fn-110/final-U3.patch` handoff
is missing; the current primary `.flow/tmp/fn1102-source/source-U3.patch` is retained.

Current retained U3 is 33,294 bytes/1,026 lines; checked U1 is 24,117 bytes/692 lines,
both touching 20 upstream files. U1 saves 9,177 bytes/334 lines against current U3.
Current U3 exceeds the original 32,652-byte baseline by 642 bytes under its explicit
waiver. Relocation removed 1,475 patch bytes and introduced 1,520 overlay bytes;
it is not a net deletion claim. The existing
[task3 preparation](../../fn-110-gomad-minimize-the-runtime-patch/task-3/conductor-preparation-20261008/findings.md)
and task2 source-acceptance handover retain the relevant provenance.

Native fn149 and fn128 stay deferred and unverified. Their missing transferred
execution evidence does not waive independent source acceptance and does not require
revival for source closeout.
