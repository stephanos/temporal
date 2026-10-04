---
satisfies: [R5, R11]
---
# fn-122-capabilities-and-their-laws.3 Declare the activity's capabilities, retire the authored twins by name and extend the harness

## Description
Roll the pilot out to the standalone activity: the product and the admission designs declare `Closable`, `Pausable` and `Pollable`, the protocol declares `Terminable`, `Cancelable` and `Describable`, both composition families read them through the member projection; `terminalIsFinal`, `pausedIsNotDispatched`, `terminalStays` and `notAdmittedWhilePaused` leave the activity's own files, retired by name and replaced by generated `<machine>.<law>` twins; the baseline harness gains the three delta categories this spec needs, allow-listed by name in `original.json`.

**Size:** M
**Files:** `model/temporal/standaloneactivity/{Model,Properties,Queries}.scala`, `admission/{Model,Properties,Queries}.scala`, `compositions/{Model,Properties,Queries}.scala`; `model/temporal/standaloneactivity/Realization.scala` (the `Describable` table is fn-112.9's; only its reference is added); `tools/umpire/internal/golden/{original.go,original.json,config.json}` and the equivalence command; the activity's IR-file declaration (fn-114.1) or `model/gate/Roots.scala`; `model/ir/activity*.json` and their `.laws.json` sidecars; `model/cases/**` (new generated Cases only).
**Touches:** [model/temporal/standaloneactivity/**, model/gate/Roots.scala, tools/umpire/internal/golden/**, model/ir/activity.json, model/ir/activity-system.json, model/ir/activity-race.json, model/ir/*.laws.json, model/cases/**]

### Approach
- Harness first (`tools/umpire/internal/golden/original.go:362-374` treats the IR and Case sets as closed): add three allow-list categories to `original.json`, each by name: a retired Property or Query with its generated `<machine>.<law>` twin and the twin's verdict; a new Case named by its generated Query; a new IR file for a new Model (task 4 uses it). Prove with mutation controls that an unlisted retirement, Case, IR file or ID change fails and that no fingerprint comparison is waived (spec Edge Cases).
- Declare the capabilities beside each machine with the vocabulary objects' named status sets (`Product.status`, `Product.terminal`, `Product.paused`, `Product.running`, `Admission.*`; fn-112 R10), the controls' `ClassRef`s and each declaration's `limits` (the retired Query's bound: `five` on the record, the product's as `terminalHolds` used). Functional-law capabilities go on `activityProtocol`, whose realization lowers their find Queries (spec Decision Context).
- Retire by name: `terminalIsFinal` (+ `terminalHolds`), `pausedIsNotDispatched` (+ `pauseHolds`, `*.product.pausedIsNotDispatched`), `terminalStays` and `notAdmittedWhilePaused` (+ their `*.any.*` Queries on the record designs and both composition families). Compare each generated twin's verdict with the retired one; a `limit reached` where the retired Query held is fixed by raising `limits`, recorded in the done summary. `atMostOneActive` and `startedByPollingWorker` stay authored (Boundaries). `terminated` and `terminate` stay unless the generated `terminateSettles` Case equals theirs byte-for-byte apart from the name (then record the rename).
- Register the `capabilities` values for lifting: in the activity's Scala IR-file declaration if fn-114.1 has landed, else in `model/gate/Roots.scala` (fn-114.1 carries them over).
- Generated find Queries lower to Cases whose awaited status comes from the `Describable` table; their intervals come from the kit until fn-118 lands (no literal wait).
- Add the violating-Model fixture for the vision's acceptance test (R11): a fixture machine declaring `Pausable` and `Pollable` whose step dispatches while paused fails its generated `<machine>.pausedIsNotDispatched` Query, and the report names the law and the two bindings from the sidecar.

### Investigation targets
**Required:**
- `tools/umpire/internal/golden/original.go:300-400` and `original.json` - the closed inventory and the allow-list to extend
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.1.md` - the equivalence harness and its allowed-delta list
- `model/temporal/standaloneactivity/Properties.scala`, `admission/Properties.scala`, `compositions/Properties.scala` (post fn-112)
- `model/temporal/standaloneactivity/Realization.scala` - the status table and `awaitStatus` (post fn-112.9)
**Optional:**
- `tools/umpire/lower/testdata/migration` - Case goldens

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model && go test -count=1 -tags test_dep ./tools/umpire/model/... ./tools/umpire/lower/... ./tools/umpire/internal/golden/...
```

### Execution constraints
- Existing Case bytes and every remaining authored declaration's tables, IDs, fingerprints and answers stay exact; retirements, added claims, new Cases and ID changes are only those allow-listed by name in `original.json`.
- fn-114 overlap: `model/gate/Roots.scala`, the activity's `Properties.scala` files and the golden config may be edited by fn-114.1/.5/.6/.7 at the same time; whichever lands second rebases.
## Acceptance
- [ ] The product and admission designs declare `Closable`, `Pausable`, `Pollable`, the protocol `Terminable`, `Cancelable`, `Describable`, with named status sets and `limits`; both composition families read them through the member projection.
- [ ] `terminalIsFinal`, `pausedIsNotDispatched`, `terminalStays`, `notAdmittedWhilePaused` and their `verify` Queries are absent from the standalone activity's files, each replaced by a generated `<machine>.<law>` twin whose verdict equals the retired one (listed in the done summary, with any `limits` raised); `atMostOneActive` and `startedByPollingWorker` stay authored.
- [ ] `original.json` allow-lists every retirement, new Case and ID change by name under the three categories; mutation controls prove an unlisted one fails and no fingerprint comparison is waived.
- [ ] Generated find Queries lower to Cases whose await reads the `Describable` table; `terminate`'s Case is unchanged or recorded as a byte-identical rename.
- [ ] The violating fixture Model fails its generated `pausedIsNotDispatched` Query naming the law and the `Pausable`/`Pollable` bindings (R11).
- [ ] Model gate, Umpire Go golden tests and lint-model pass.
## Done summary
Rolled the capabilities out to the standalone activity, retired the authored twins by name, and extended the original-baseline harness with the allow-listed delta categories.

**Declarations**
- `productCapabilities` (Properties.scala, limits `three`) declares Closable, Pausable and Pollable.
- `protocolCapabilities` (limits `three`) declares Terminable, Cancelable and Describable (`ActivityRealization.activityStatus`). Its reach is `Seq(start(), workerStop)` and it expects `inconclusive(explanationsDisagree)`.
- `admissionCapabilities(m)` (limits `five`) declares Closable, Pausable and Pollable, and waives closedIsRejectedUniformly.
- `overQueueCapabilities(c)` (`five`) and `overMatchingCapabilities(c)` (`twelve`) declare the record's three through the `activity` member's projection, and waive closedIsRejectedUniformly.
- `activityFile` registers the product and protocol declarations with `irFile`. The design and composition declarations are folded where their bundles are built, so the system file's Query roots lift them.

**Retired, with generated twins whose verdicts equal the retired ones**
- `terminalIsFinal` + `terminalHolds` → `activityProduct.terminalStatesAreFinal`, Verified.
- `pausedIsNotDispatched` + `pauseHolds` → `activityProduct.pausedIsNotDispatched`, Verified.
- On each of the 7 designs and compositions:
  - `terminalStays` + `*.any.terminalStays` → `<m>.terminalStatesAreFinal`;
  - `notAdmittedWhilePaused` + `*.any.notAdmittedWhilePaused` → `<m>.pausedIsNotDispatched`.
  - These are Verified on current\*, and Counterexample on stale\* as before. Witnesses and monitors are unchanged.

**New claims**
- `activityProduct.closedIsRejectedUniformly` (Verified).
- `activityProtocol.terminateSettles` and `activityProtocol.cancelIsRequested` (Found). They are new Cases, lowered.

**Stay authored:** `atMostOneActive`, `startedByPollingWorker`, `terminated` and its find `terminate`. The generated `terminateSettles` Case differs from it (property fingerprint and sources), so it is a new Case, not a rename. No `limits` were raised.

**Framework and lifter**
- `declared.claim(law)` lets the activity's own Queries read the generated Property. The pinned paths (`staleDelivery`, `admittedBeforePause`, `duplicateDelivery.monitored`) and `*.product.pausedIsNotDispatched` keep their names and answers, now read off the generated Property.
- `Terminable` and `Cancelable` carry `expect: RunExpectation` for their generated finds. That field was left over from task 2; the API contract change is recorded here.
- `LawViolations` (tools/umpire/model/laws.go) reads the sidecar and names a violated generated claim's law, capabilities and bindings. The fixture `rogueJob`, whose poll dispatches a paused job, fails `rogueJob.pausedIsNotDispatched` and is reported so (R11).
- The admission specimen fixture declares its own `pausedIsNotDispatched` instance, so its frozen IR keeps its names.

**Harness** (`original.go`/`original.json`/`config.json`, with mutation controls)
- `law_replacements` lists each generated claim by (file, machine, law) with its verdict, an optional `renames` (the retired Property, renamed in the expected baseline so re-pointed Queries and rows compare exactly) and the retired Queries. There are 22 entries.
- `new_cases` holds the two generated finds. `new_ir_files` is empty; task 4 adds to it.
- `Expected(baseline)` and `Ungenerated(current)` compare everything else byte for byte, re-deriving and never waiving fingerprints. `TestOriginalLawVerdicts` checks each recorded verdict and the retired Query's baseline answer.
- The inventory admits only the listed Cases and sidecars, and checks the sidecar's claim names.
- In config.json: `source_root_retirements` and `source_root_additions`, and `MatchAt` for the fn-115 migration golden.

**Findings**
- `closedIsRejectedUniformly` is false on the admission record. A delivery to a timed-out record is accepted, recorded as `admissionRejected`, and owes matching the answer (activity.go HandleStarted). The wrong universal surfaced, as R7/Edge Cases anticipate, and is waived with that reason.
- On compositions it is waived because the queue member keeps stepping after the record closes.

**Gates:** all pass (evidence.md):
- `umpire-gen-model` and `umpire-check-model`;
- the full Go suite (after `666d8b8fcb`);
- the goldens and focused activity tests, after the review fixes;
- `lint-model` and `lint-code-fast` (0 issues).

**Live run:** the two new Cases ran live under shared load. Each passed at least once with `inconclusive(explanationsDisagree)`. The failures were Contract INCONCLUSIVE after the 10 s window, which the authored `activity-terminate-case` shows in the same run, so the flakiness predates this task.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with four P3s:
- P3 1 (the test comment on terminate's Case) is fixed in d89d482d1e.
- P3 4 (the composed rejection string) is fixed: it is documented as read only by the waived law.
- P3 2 is deferred. `LawViolations` is called from a test only; wiring the sidecar's bindings into a printed report belongs to the law lint and table view (tasks 5/6), since the gate prints no Query report today.
- P3 3 is deferred to task 4, where a second Describable lands. Nothing yet ties the declared Describable table to the realization's await: the await reads `activityStatus` through the realization, the same table.
- FYI: the twins of `terminalHolds`/`pauseHolds` search the product freely rather than the protocol paths. Only the verdict carries over, by design (spec: twin over the machine's free Scenario).

**Shared files for the merge:**
- `tools/umpire/internal/golden/{config.json,golden.go,golden_test.go,original.go,original.json,original_test.go}`;
- `tools/umpire/{model,lower}/*_test.go` (activity, original, migration);
- `tools/umpire/export/quint_test.go`;
- `model/lifter/{Capabilities,Claims}.scala`, `Fixtures.test.scala`, `lifts/Admission.scala`;
- `model/README.md`, `.plans/SEMANTIC_PROTOCOLS.md`;
- regenerate `model/ir` and `model/cases` after merging.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5379f94ca3, eeee4ee1c6, 666d8b8fcb, d89d482d1e
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration|Original|TestActivity|TestEveryQuery' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, after review fixes), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases/.../activity-(terminate|activityProtocol) (flaky under load; each generated Case passed at least once; authored terminate fails alike), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 1 SHIP)
- PRs: