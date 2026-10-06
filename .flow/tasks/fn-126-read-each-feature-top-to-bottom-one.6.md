---
satisfies: [R1, R10, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.6 Product and system folders, zoom-ins flattened, structure lint (a)(c)

## Description
Task 6a of the split closing batch (decisions 16, 22; R20 (a) and (c); R10 folders). Give `standaloneactivity/`, `nexuscaller/` and `shared/taskqueue/` a `product/` and a `system/` folder and flatten `record/`, `withTaskQueue/` and `closepolicy/` into `system/` files, under today's pins so every ID is frozen: the reader projection (`projtool`) must be byte-identical and the IR diff only paths, lines, Function symbols and source roots. Land the R20 structure lint (a) and (c) in a sibling pass (`model/irgen/Structure.scala`), the template fixture `model/irgen/testdata/layout/` and one refusal fixture per rule; teach `Order.scala` the level-file role and the R10 layout test the new and retired folders. Re-record pinned Runs if Case identities move. Plan section 2.

Plan: `.flow/tmp/fn-126/plan6.md` (read-only planning of 2026-10-05, built after task 4 against task 5 in progress; re-verify against the merged task 5). Host decisions on its open points: IDs follow decision 23's rule literally (section objects included, e.g. `…ActivityRecord.monitors.atMostOneActive`); the Kit's family comes from the lifter substituting the realization's package when it folds a Kit call (no macro); level files keep their subject's own types and signature; `TrustingCaller`'s `inspect` moves into the feature's `object caller` (party unchanged), ending the shadowing; a `shared/` folder has at most one `object exports`. The golden harness is retired by fn-124.7 before these tasks: every proof is by projection (`projtool` + a before/after IR projection), never a golden re-capture.
## Acceptance
- [ ] Every item of this task's description is done, with the plan's verification list for its section passed.
- [ ] Equality proved by projection as described; any table, answer, verdict or fingerprint difference beyond the stated renames stops the task.
- [ ] All gates of the spec's Verification pass.
## Done summary
The work is on branch `umpire-fn126-6` in `/Users/stephan/Workspace/skunkworks/umpire/wt/lane-a`, base `395b38c6dc`. `umpire` is merged in at `19d1486f83`.

### Commits
| Commit | What it does |
| --- | --- |
| `0a97a3ec72` | `fix(umpire)`: restores `tools/umpire/model/fixtures_test.go`. The base's MILESTONES commit `395b38c6dc` had emptied it, which broke the gate. The host made the same fix on `umpire` (`6d3e81dfb6`) and the merge was clean, so this commit is now redundant but harmless. |
| `94a22fab15` | `refactor(model)`: moves the Models into `product/` and `system/` and flattens the zoom-in folders. It also regenerates IR, Cases, functional fixtures, canary Case and the lifter's expected IR, fixes Go tests that name paths, lines or function symbols, and re-records the pinned Runs. |
| `2fffbb2da7` | `feat(irgen)`: turns R20 on for every Temporal feature and adds the level-file role to the order lint. |
| `a3c2ee4864` | `test(umpire)`: the R10 layout test learns the retired folders and `product`/`system`; stale mentions it found are fixed. |
| `19d1486f83` | Merges `umpire`. |

Each commit was gated in sequence.

### Final tree
**`features/standaloneactivity/`**
- `StandaloneActivity.scala` (229 lines): types, signature, families, `object exports`.
- `product/Product.scala`: `ActivityProduct`.
- `system/System.scala`: `ActivityProtocol`, `ActivityWorker`, `StandaloneActivity`.
- `system/Record.scala` (was `record/`): its own types, `history` section and choices, plus `CurrentAdmission`, `StaleAdmission`, `HeldAdmission`, `AdmissionResponseLoss`.
- `system/WithTaskQueue.scala` (was `withTaskQueue/`): its own types, `CurrentRecord`, `StaleRecord`, and the `CurrentOver*` and `StaleOver*` compositions.

**`features/nexuscaller/`**
- `NexusCaller.scala` (254 lines): types, signature, `CallerFamily`, `ControlFamily`, and `exports`. The exports are `nexusCaller`, `nexusControl` and `nexusClose`, the last merged in from `closepolicy`.
- `product/Product.scala`: `NexusProduct`.
- `system/System.scala`: `NexusProtocol`, `HandlerWorker`, `NexusCaller`.
- `system/ForgedCompletion.scala`: its pin, an `inspection` section, the `forged`/`sent` choices, and `ForgedCompletion`.
- `system/ClosePolicy.scala` (was `closepolicy/`): `ClosePolicyFamily`, `RejectAfterClose` and its 8 derived designs.

**`shared/taskqueue/`**
- `TaskQueue.scala`: types and signature, no exports.
- `product/Product.scala`: `DispatchQueue` and its storage-loss variant.
- `system/System.scala`: `MatchingQueue` and the lossy, forgetful and volatile variants.

`nexusoperation` and `shared/worker` have one level each and are unchanged.

### Lint and R20's switch
- **Switch removed:** `Structure.everyFeature` and its gate in `held` are gone. Every feature under `model/temporal/` is now held to R20; a lifter fixture is held once it has a level folder. Today's Models raise no refusals.
- **Level-file role in `Order.scala`:** a level folder's file reads in feature-file order without `object exports`. The structure lint keeps exports to the root file. Rule (d) was already limited to the root's siblings, because every level file reads as a feature file; I documented this in the code.
- **Fixture:** `layoutRefusals/c/pump/system/System.scala` gains a type after its machine, and the test asserts the order refusal for it.
- **README:** the "until fn-126.6" sentence in `model/README.md` is replaced.
- **R20 (b)** is still task 8's.

### R10 (`tools/umpire/model/layout_test.go`)
- **Retired directories:** `standaloneactivity/{record,withTaskQueue}` and `nexuscaller/closepolicy`.
- **Also retired:** these as root paths, as paths joined from parts, as a bare folder in prose (`record/`), as package clauses and imports, and as qualified packages or function symbols. Pinned former-owner strings such as `temporal.nexuscaller.closepolicy.Model$package$` stay legal.
- **Per-kind files:** retired under `product/` and `system/` as well.
- **Tests:** 16 new test cases.
- **Stale mentions fixed:**
  - `model/README.md`: the layout tree and prose.
  - `.plans/UMPIRE_MODULES.md`: the Models row.
  - Three Scala comments and two Go comments.
  - Eight acceptance reasons in `model/ir/activity-system.lint.json`. This is prose only; kinds, owners and subjects are unchanged.

### Equality proof
`prove.sh --paths-move` against the base snapshot (`.flow/tmp/fn-126/fn126-6/prove-final.log`):
```
[OK] projection (byte-identical, 89956213 bytes)
[OK] casegen (21/21)  [OK] ir (7 files; notes: roots differ, Function names read as tokens)
[OK] lint (notes: 8 prose deltas)  [OK] laws  [OK] cases 21/21  [OK] generated 9/9  [OK] canary 2/2  [OK] names
RESULT: OK
```
- **New `--paths-move` flag:** I added it to the shared `tools/compare.py` and `prove.sh`; it is opt-in and the originals are backed up as `.orig`. Without it, prove.sh reports DIFF, because Case `source.path` values change in this task and the tool only dropped lines. The default behaviour, which tasks 7 and 8 use, is unchanged.
- **Independent check:** `fn126-6/casediff.py` finds only path and line differences (92/41/5 paths; 2 lines; 1 manifest position) and no fingerprint changes.

### Pinned Runs
The control and canary Case identities moved because their source path changed. I refreshed the replay package's pinned copy of the control Case and repinned the canary policy from `cf9d8da0…` to `c9280a98…`. `make umpire-rerecord-pinned-runs` then re-recorded both Runs live and re-rendered the receipt goldens. A final run reports both as current.

### Gates
All logs are in `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-126/fn126-6/`.

| Gate | Result | Log |
| --- | --- | --- |
| `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | exit 0 | `model-gate.log` |
| `make lint-model` | exit 0, all files formatted | `lint-model.log` |
| `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast` | 0 issues | `lint-code-fast.log` |
| `go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/...` | 17/17 packages pass | `go-suite.json` |
| `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` | exit 0 | `<target>.log` |
| `make umpire-rerecord-pinned-runs` (final) | exit 0, both current | `rerecord-final.log` |

`lint-model` also prints scalafix `NoSuchFieldException` stack traces; the same traces appear in fn-126.2's log.

### Decisions
1. **`inspect` keeps its ID; it is not moved into the feature's `caller` (needs your call).** Moving it into `object caller`, as the host decision and carry-forward step 1 say, would change its ID from `temporal.nexuscaller.Control$.inspect` to `…Model$package$.inspect`, which hits the stop condition. Instead it sits in a top-level `object inspection extends Section` in `system/ForgedCompletion.scala`, whose file pin is `Control$`. The ID stays the same, R20(c) has nothing to refuse, and it no longer shadows the feature's `caller`. Task 7, where every ID changes anyway, can move it into `caller` at no extra cost. The control object's own pin then had nothing to pin, so I removed it.
2. **Named pins.** Two anonymous `given DefinitionScope` in one package don't compile, so the pins of Record, WithTaskQueue, ClosePolicy and ForgedCompletion are now named givens such as `given recordScope: DefinitionScope`. All pins go away with decision 23.
3. **Givens in a shared level package.** The close policy's package-level `Ok[Answer]` was being picked up by `NexusProtocol`, so compilation failed. It now lives in `Answer`'s companion and only `ClosePolicy.scala` imports it. The close policy's `Family` moved into `object ClosePolicyFamily`. Family strings in the IR are unchanged.
4. **Each subject's own types and signature stay in its level file.** The task queue's types stay in its root because both levels use them.
5. **File named after today's object:** `ForgedCompletion.scala`, not `TrustingCaller.scala`. Task 8 renames the object (decision 17) and should rename the file with it.
6. **One `object exports` per feature:** `nexusClose` is now declared in `NexusCaller.scala`. `IrFiles.test.scala` expects 3 declaring exports objects.
7. **ClosePolicy's `four`, `five` and `twelve` stay top-level.** The explicit `shared.Bounds` imports resolve correctly, so moving them into an object wasn't needed.

### For you
- **Decision 1** needs your approval or a different direction.
- **Docs outside R10's roots** still describe the old folders: six `.plans` files (QUINT_MODULE_LAYOUT, DSL_SIMPLIFICATION, ACTIVITY_MODEL_COMPARISON, DYNAMIC_CONFIG, SEMANTIC_PROTOCOLS, TEMPORAL_PATTERNS). They're left for task 8's docs pass.
- **R11 hop cost of decision 16:** following `activityProduct` from its state type to a Query now goes through `StandaloneActivity.scala`, `product/Product.scala` and `shared/Bounds.scala`. References between files of one package come back in `system/` by design (decision 22).
- **Task 7 note (carry-forward step 6):** not done here, as instructed.
### Review
claude-opus-5-5 at high, fresh context (host-dispatched; single lane). Writer and reviewer are the same family (Opus).
- **Round 1: SHIP.** The reviewer reran prove.sh with `--paths-move` against a verified base snapshot: OK, 0/63 IDs and 0/65 types changed, projection byte-identical.
- **P2, fixed in 9dc2577acc:** about ten README and UMPIRE_MODULES lines the move had made wrong.
- **P3s, fixed in 2e32b623a8, 74ab0089f4, ed4edda1de and 69c03cdb62:**
  - rule (a) refuses a single-level feature's subfolders, and a package that does not mirror its folder;
  - the bare zoom-in folder check is limited to `model/` and the layout docs;
  - lint reasons name the level files;
  - formatting.
- **Proof tooling (untracked):** `--paths-move` now takes the explicit old→new path map and fails on any pair outside it.
- **Reruns, logs `review-*.log`, all pass:** model gate, `./tools/umpire/model` and `./tools/umpire/lint`, lint-code-fast, `prove-review.log` RESULT OK.
- `make lint-model` was confirmed by the host: exit 0, `host-lint-model.log`.

### Host decisions (for the owner)
- `inspect` stays in a top-level `object inspection` in `system/ForgedCompletion.scala` for now, to keep IDs frozen; task 7 moves it into `caller`.
- The named `given …Scope` pins and `…Family` objects are removed in task 7, with decision 23.

### Note
During its review round the implementer also received a request, marked as coming from a non-user source and probably sent by another local session, to compare the Umpire IR with Quint's. It answered with one read-only subagent, whose notes are in the session scratchpad (`quint-research/`). Nothing in this task's tree came from it.

### Commit hygiene, recorded
- `0a97a3ec72` duplicates the host's `6d3e81dfb6` restore of `fixtures_test.go`, which a MILESTONES commit had emptied. The merge is clean.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0a97a3ec72, 94a22fab15, 2fffbb2da7, a3c2ee4864, 19d1486f83, 9dc2577acc, 2e32b623a8, 74ab0089f4, ed4edda1de, 69c03cdb62
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (ok), make lint-model (ok), lint-code-fast (0 issues), go test -count=1 -json -tags test_dep -p 2 ./tools/umpire/... (17/17), umpire-check-cases, umpire-check-fixtures, canary-check-case (ok), make umpire-rerecord-pinned-runs (both current), prove.sh --paths-move: RESULT OK (projection byte-identical), review reruns: model gate, ./tools/umpire/model and ./tools/umpire/lint, lint-code-fast, prove.sh with the path map (RESULT OK), make lint-model (host rerun, exit 0)
- PRs: