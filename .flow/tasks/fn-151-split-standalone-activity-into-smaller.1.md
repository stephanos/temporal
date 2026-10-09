---
satisfies: [R1, R3]
---
# fn-151-split-standalone-activity-into-smaller.1 Move existing derived activity models and worker composition into subject files

## Description
Move existing complete declarations into the subject files below (R1, R3). Preserve every declaration identifier and body, package, export root and realization standing. Seal the declaration comparison and source-coordinate map before task .2 changes ownership. Task .3 consumes that seal and performs the joined generated-artifact comparison.

**Size:** M. This is one mechanical, existing-pattern move with a shared source seal and focused checks. The larger path count includes destination files and directly coupled filename references, not new model design. Keep the existing .1 → .2 → .3 DAG.

**Files:** The three current system source files and eight destinations below; `Standalone.scala`; the three current layout documents; the parity, system-test header and layout-test consumers listed in Touches. `system/Realization.scala` and all generated artifacts are read-only in this task.

**Touches:** [model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/activity/standalone/system/Record.scala, model/temporal/features/activity/standalone/system/WithTaskQueue.scala, model/temporal/features/activity/standalone/system/RetryTimeouts.scala, model/temporal/features/activity/standalone/system/Heartbeat.scala, model/temporal/features/activity/standalone/system/ResponseByID.scala, model/temporal/features/activity/standalone/system/Reset.scala, model/temporal/features/activity/standalone/system/Dispatch.scala, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala, model/temporal/features/activity/standalone/system/DispatchWithWorker.scala, model/temporal/features/activity/standalone/Standalone.scala, model/README.md, .plans/UMPIRE_MODULES.md, .plans/STRING_SEQUENCE_DIAGRAMS.md, tools/umpire/check/activity_parity_test.go, tools/umpire/check/activity_system_test.go, tools/umpire/ir/layout_test.go, .flow/tmp/fn151/task1/**]

### Approach

1. Anchor the input at structural baseline `4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd` and the conductor's sealed fn-145 input receipts. Record source hashes before editing. If the baseline inputs differ, report the mismatch before moving declarations. Do not repeat the heavyweight fn-145 runs to reconstruct a baseline.
2. Move complete top-level declarations according to the table. Preserve the existing package clauses `temporal`, `features.activity`, `standalone`, `system`. Preserve internal member ordering, initialization expressions, behavior comments, property/scenario/query names, bounds, totals, expectations and realization-related metadata byte-for-byte. Do not introduce wrapper objects, rename declarations or split their bodies.
3. Redistribute imports outside those declarations. Each subject uses `framework.*`; keep only the additional imports it actually needs. `ActivitySystem` still uses the `actors.worker.worker as process` alias, so retain that alias in `System.scala`; `DispatchWithWorker.scala` also needs `process`, `WorkerPhase`, `WorkerState` and `Timeout.expires`. Same-package and enclosing-package declarations need no new imports. Compile with the existing `-Werror`/unused-import policy.
4. Keep `Standalone.scala` export declarations and their order unchanged. Keep `system/Realization.scala` bytes unchanged, including the attached subject declarations and their standing. Retire `Record.scala` and `WithTaskQueue.scala` only after every declaration is accounted for at its destination. Keep the existing `Phase`, `Dispatch`, `State`, `Fact` and complete `ActivitySystem` in `System.scala`.
5. Extend the explicit lifecycle source list in `TestActivityEveryClaimDeclarationIsLifted` with `system/RetryTimeouts.scala`, `system/Heartbeat.scala`, `system/ResponseByID.scala`, `system/Reset.scala` and `system/DispatchWithWorker.scala`. Keep its existing exact `ElementsMatch` equality, capability-section count and name-based parsing. Do not add the dispatch protocol, race or queue files to this lifecycle list, weaken equality to a subset, or implement the family/owner/kind/name enhancement owned by task .2.
6. Correct current file-layout references in the declared docs and source/test headers. Update the positive current-layout example in `layout_test.go`, not its intentionally retired-path examples. Do not rewrite historical comparison documents. A filename reference inside a declaration body stays unchanged for this pure-move seal; identify it for task .2 instead of breaking body equality.
7. Seal the source-oriented equivalence evidence below, run the focused Quick commands, and hand off the destination map to task .2 and the immutable .1 coordinate ledger to task .3. Do not regenerate IR, Cases, fixture mirrors or lifter goldens here.

#### Whole-declaration move map

Paths below are relative to `model/temporal/features/activity/standalone/system/`. Line references describe the sealed baseline, not required candidate line numbers.

| Source | Destination | Complete declarations |
| --- | --- | --- |
| `System.scala:23`, `:41`, `:45`, `:57`, `:70` | `System.scala` | Retain `Phase`, `Dispatch`, `State`, `Fact`, `ActivitySystem` unchanged. |
| `System.scala:1029` | `RetryTimeouts.scala` | `TimeoutRetry`. |
| `System.scala:1070`, `:1087`, `:1107` | `Heartbeat.scala` | `HeartbeatCompletion`, `HeartbeatRetry`, `HeartbeatExhaustion`. |
| `System.scala:1130`, `:1142`, `:1158` | `ResponseByID.scala` | `ByIDCompletion`, `ByIDFailure`, `ByIDCancellation`. |
| `System.scala:1188`, `:1306`, `:1333` | `Reset.scala` | `ResetSettlement`, `ResetKeepingPause`, `DeferredReset`. |
| `System.scala:65`, `:1353`, `:1358` | `DispatchWithWorker.scala` | `StandaloneActivityState`, `ActivityWorker`, `StandaloneActivity`; put the state type before its consumers. |
| `Record.scala:35–62`, `:75–106`, `:116`, `:372` | `Dispatch.scala` | `AdmissionPhase`, `Active`, `Answer`, `AdmissionState`, `AdmissionFact`, `Finality`, `AdmissionClaims`, `AdmissionCapabilities`, `admissionCommits`, `admissionCommitFails`, complete `history`, `ActivityRecord`, `TrustingActivityRecord`. |
| `Record.scala:66`, `:69`, `:109`, `:110`, `:392`, `:482` | `DispatchRaces.scala` | `AdmissionResponseState`, `AdmissionResponseFact`, `committedThenLost`, `failedThenLost`, `HeldDispatch`, `LostStartAnswer`. |
| `WithTaskQueue.scala:35–284` | `DispatchWithTaskQueue.scala` | All declarations: `OverQueue`, `OverMatching`, `OverQueueClaims`, `OverMatchingClaims`, `OverQueueCapabilities`, `OverMatchingCapabilities`, `RecordMember`, `TrustingRecordMember`, `RecordOverQueue`, `TrustingRecordOverQueue`, `RecordOverMatching`, `TrustingRecordOverMatching`, `RecordOverForgetful`, `RecordOverVolatile`, `RecordOverLossyMatching`. |

#### Source equivalence seal

Use a task-local byte-range comparison, not a new production verification framework. Write its receipts under `.flow/tmp/fn151/task1/` and retain the exact invocation and input/output hashes in the handoff.

- `baseline-declarations.json` and `candidate-declarations.json`: identical inventories keyed by fully qualified top-level declaration identity, with exact declaration-body byte hashes. Enumerate every declaration from the three input files, including those retained in `System.scala`. The comparison must fail on a missing, duplicate, extra, renamed or modified declaration. Coordinates and filenames are not part of this equal-body inventory.
- `source-coordinate-map.json`: record the baseline revision, affected source hashes and a one-to-one pre/post mapping for each declaration and nested lifted source span. Record exact old/new file, line and column boundaries, unchanged identity and body hash. Translate each baseline lifted span by its byte offset within its unchanged containing declaration, then resolve candidate line/column coordinates from the destination source. Include shifted spans in retained declarations. Every mapped span must resolve to the recorded source slice; do not authorize arbitrary position deletion or a blanket filename normalizer.
- Record exact equality of the export declarations/order and `Realization.scala` bytes separately. Seal the .1 post-move source hashes before .2 begins. Task .3 composes this .1 mapping with .2's ownership and coordinate changes, rather than treating .1 candidate coordinates as final.
- Name any deferred embedded filename comments in the handoff. Existing generated artifacts stay untouched, with their baseline hashes retained for task .3. Passing source parity against baseline IR is not a claim that generated positions have been refreshed.

The eventual artifact pin is the adapted fn-145 comparison harness at `/tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454/equivalence.go`, owned by task .3. Reuse its ordered reader-table, Check-receipt, Query-answer, Case and deterministic identity comparisons there; do not execute its historical mappings unchanged in this task.

### Investigation targets

**Required** (read before moving):

- `model/temporal/features/activity/standalone/system/System.scala:23–1391`: complete lifecycle core and the existing derived declarations.
- `model/temporal/features/activity/standalone/system/Record.scala:35–526`: dispatch protocol, negative control and race declarations.
- `model/temporal/features/activity/standalone/system/WithTaskQueue.scala:35–284`: unchanged queue-composition declarations.
- `model/temporal/features/activity/standalone/Standalone.scala:20–27` and `model/temporal/features/activity/standalone/system/Realization.scala`: layout header, export order and unchanged realization attachments.
- `tools/umpire/check/activity_parity_test.go:58`: explicit source inventory and exact parity assertion.

**Optional** (reference as needed):

- `model/irgen/Context.scala:318–344`: qualified names strip file-level `$package$` owners; do not change that mechanism.
- `tools/umpire/ir/layout_test.go:489`, `model/README.md:654–657`, `model/README.md:960`, `.plans/UMPIRE_MODULES.md:32` and `.plans/STRING_SEQUENCE_DIAGRAMS.md:30`: current filename references only.

### Boundaries

Task .2 owns new subject models, ownership changes, claim extraction and owner-key parity. Task .3 owns generated artifact updates and joined behavior/identity comparison. Preserve the inherited full-suite RED, strict Activity completion/fatal-failure/pause-resume assertions and Batch 5 disposition; fn-154 heavyweight Quint memory work remains deferred. Do not pursue fn-155 simplification, semantic cleanup, expectation changes or broader documentation rewrites.

### Quick commands

Run from the repository root using the existing shared heavy-run coordination. Record exit status and logs. The inventory diff requires the sealed source-comparison receipts described above. These focused checks do not replace task .3's canonical joined gate.

```bash
diff -u .flow/tmp/fn151/task1/baseline-declarations.json .flow/tmp/fn151/task1/candidate-declarations.json
mise exec -- scala-cli compile --server=false model/project.scala model/framework model/temporal
mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/framework model/temporal
make lint-model-models
mise exec -- go test -tags test_dep -p 2 -timeout 30m -json -count=1 ./tools/umpire/ir -run 'Test(KindGeneralFilesAndForms|RetiredModelPathsStayRetired|StandaloneActivityRealizationFollowsSystem)$'
mise exec -- go test -tags test_dep -p 2 -timeout 30m -json -count=1 ./tools/umpire/check -run '^TestActivityEveryClaimDeclarationIsLifted$'
```
## Acceptance
- [ ] The whole-declaration move map is complete and one-to-one. The eight named subject files contain exactly their listed declarations; `System.scala` retains its five core declarations; the two retired source files contain no orphan declarations. No wrapper object, new subject model or renamed declaration is introduced.
- [ ] The sealed pre/post declaration inventories compare exactly equal by fully qualified identity and declaration-body byte hash, including internal member order and behavioral comments. The comparison rejects omissions, duplicates, additions, renames and body changes. Imports and comments outside declaration bodies are the only permitted Scala-model source edits beyond relocation.
- [ ] Package identity, export declarations and export order remain unchanged; `system/Realization.scala` is byte-for-byte unchanged. Properties, Scenarios, Queries, bounds, totals, expectations, monitors, refinements, capabilities and executable/no-realization standing receive no semantic or metadata change in this task.
- [ ] The handoff includes baseline revision `4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd`, exact source/artifact input hashes, immutable .1 post-move source hashes, the declaration inventories and `source-coordinate-map.json`. Its exact pre/post file/line/column spans resolve to recorded source slices, including nested lifted and retained-core spans. Task .2 and .3 receive this ledger; no generated IR, Case, mirror or golden is regenerated before task .3.
- [ ] `TestActivityEveryClaimDeclarationIsLifted` reads the five additional lifecycle subject files while retaining exact declaration equality, the existing capability-section count and name-key parsing. Dispatch protocol/race/queue claims do not enter this lifecycle inventory. Owner-key parity enhancement remains assigned to task .2. The focused parity and layout Quick commands pass with recorded output.
- [ ] Compilation, formatting and `lint-model-models` pass with recorded output and no unused imports. Current file-layout docs and header references match the destinations, intentional retired-path examples and historical docs remain unchanged, and embedded filename comments deferred to preserve body equality are listed in the handoff. The handoff explicitly preserves inherited RED/Batch 5/fn-154 dispositions and excludes fn-155 simplification; focused passes are not reported as a green canonical suite.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
