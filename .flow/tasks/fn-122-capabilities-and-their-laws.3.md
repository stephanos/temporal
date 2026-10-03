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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
