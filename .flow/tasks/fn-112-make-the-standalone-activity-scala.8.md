---
satisfies: [R7, R10, R11, R14, R15, R18]
---
# fn-112-make-the-standalone-activity-scala.8 Reorganize the standalone activity feature by subject and declaration kind

Touches: [model/temporal/standaloneactivity/**, model/temporal/taskqueue/**, model/gate/Roots.scala, model/README.md, .plans/UMPIRE_MODULES.md, model/ir/**, model/cases/**]

## Description
Move the now-deduplicated feature into the specified Model/Properties/Queries layout and finish vocabulary cleanup under the stable-name rules.

**Size:** M
**Files:** model/temporal/standaloneactivity/**; gate roots/pins; README and module map.

### Approach
- Create top-level, admission and compositions Model/Properties/Queries files; finish the shared temporal/taskqueue layout from task 12, omitting genuinely empty files. Delete System.scala and Claims.scala; no activity-local dispatchqueue folder remains.
- Group feature vocabulary under Product, Protocol and Admission objects and reuse the shared taskqueue vocabulary. Remove protocol/admission step prefixes and resolve completed/completes/completion collisions.
- Expose each machine's status sets as named defs on its vocabulary object (`Product.terminal`, `Product.paused`, `Product.running`, `Protocol.held`, the admission record's on `Admission`), moving the named defs task 6 kept, so no `in(…)` list is repeated at a use site and the shared-claim defs and the realization read those names (spec R10; `.plans/SEMANTIC_PROTOCOLS.md` "What fn-112 should do now", item 2).
- Put the three parameterized shared-claim defs of task 7 in the top-level `Properties.scala`, beside `admission/` and not inside it, since they are declared on the record and on both composition families.
- Keep the parent package in subfolders where it compiles without synthetic owner collisions. Apply task 2's one-per-former-owner DefinitionScope pins so actions, monitors, assumptions, channels and realizations keep exact IDs after moving under vocabulary objects; never add per-declaration ID strings.
- Remove worker.worker stutter and the workerStop alias; retain Delivery result text while resolving its dead Scala declaration.
## Acceptance
- [ ] The final directory/file matrix matches R11, includes fn107 models and has no System.scala, Claims.scala or empty placeholder file.
- [ ] Vocabulary objects and declaration names meet R10/R14 with no worker.worker stutter; each machine's status sets are named defs on its vocabulary object and no `in(…)` phase list is repeated at a use site.
- [ ] The three parameterized shared-claim defs live in the top-level `Properties.scala`, beside `admission/`.
- [ ] DefinitionScope pins reproduce every task-1 symbol ID after relocation; all outputs match R1's original baseline plus its exact authorized metadata delta.
- [ ] Feature source is trending toward the 1,600-line and 60-literal closing targets; both standalone-only and combined standalone-plus-shared-queue counts are recorded, so moving code alone does not count as simplification.
## Done summary
Reorganized the standalone activity Model by subject and declaration kind. Commits 18306a4474, e53650fa5c, 45ecf5e5e2. Tables, Definition IDs, IR type names, Property rows, Query answers/totals and Case bytes are unchanged; model/cases differ only in Scala source paths; original.json untouched.

**Layout (R11)**
- `standaloneactivity/{Model,Properties,Queries,Realization}.scala`, `admission/{Model,Properties,Queries}.scala`, `compositions/{Model,Properties,Queries}.scala`. System.scala and Claims.scala are gone; no empty file.
- Folders are subpackages (`package standaloneactivity; package admission`): repeated file names would collide as one package's `Model$package$`, and the chained clause reads the parent vocabulary without imports.
- One `DefinitionScope("temporal.standaloneactivity.System$package$")` per subfolder Model.scala keeps the record's action/monitor IDs and the IR type names (`temporal.standaloneactivity.AdmissionState`, `OverQueue`, ...). Top-level files need no pin (their symbol-based declarations stayed in Model.scala).
- Top-level Properties.scala: product/protocol Properties, `scheduleToCloseFires`, the cross-entity claim and the three shared-claim defs. Queries.scala: `object Paths` (Scenarios), Limits three/four/five/six/eight, `object Functional` (functional Queries, `Functional.all`), `cancelRequest`, `terminalHolds`, `pauseHolds`, `competingTimers`, `stoppedWorkerStartsNothing`.
- Subject Properties are case-class bundles (`AdmissionClaims`, `OverQueueClaims`, `OverMatchingClaims`, the `queueLaws` pattern), read by field from the Query defs.

**Vocabulary (R10, R14)**
- Objects `Product`, `Protocol`, `Admission`, `ResponseLoss` hold named status sets (`Product.terminal/paused/running/held/pausable`, `Protocol.terminal/live/held/waiting`, `Admission.terminal/paused/running/twoActive/phase/ends`) and step functions named after their actions (`attemptStart ~> Protocol.attemptStart`); every `in(…)` is inside such a def. `admissionOver` folded into `Admission.terminal`; `Admission.admit` replaces three copies of the admitted record.
- completed/completes/completion and terminated/terminate: Scenarios live in `Paths`, functional Queries in `Functional`, so no package-level pair differs by inflection; no literal added.
- The `workerStop` alias is gone; features import `worker.{serve, workerStop}`. `worker.workerEntity` is now `worker.entity`. `workerStop`/`workerResume` keep their names: an action's Definition ID is its val's owner and name and a pin cannot rename, so the rename would change frozen IDs (comment in Worker.scala). `results("Delivery")` is documented as action metadata.

**Harness**: fn-115's migration golden gained `source_path_merges` (positions, Case source paths and located strings under `model/temporal/standaloneactivity/` compare as that directory; splits cannot express a file mixing Claims.scala and System.scala declarations), plus root moves; unit tests both ways. Go tests follow the new function names/files; the P/Quint `admissionOver` mutation (a contains-turned-AND that P refused) is now `overAtOnce`.

**Metrics (R18)**: standaloneactivity 2281 lines / 179 literals -> 2449 / 181; taskqueue 418 / 25; combined 2699 / 204 -> 2867 / 206. +2 literals are the two pins; the line growth is six file headers/package clauses, object docs and the three bundles. Realization.scala holds 862 lines / 94 literals: that is where task 9/10 recover.

**Review**: claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the same family (Opus). Round 1 NEEDS_WORK (worker stutter, line growth, `Admission.pause` naming); round 2 NEEDS_WORK (worker stutter); round 3 SHIP.

**Deferred P3/FYI**: OverQueue/OverMatching repeat five forwarders (lambda refusal; fn-122). nexuscaller still has `val workerStop = worker.workerStop` (out of scope; fn-114). `stoppedBeforeRetry` keeps its explicit start.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 18306a4474, e53650fa5c, 45ecf5e5e2
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection|IRInventory' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 240 s), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: