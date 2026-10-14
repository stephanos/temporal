---
satisfies: [R10]
---
# fn-89-one-contract-rule-per-entity.7 One command to re-record catalog-pinned Runs after a protocol change

## Description
Every Testpilot protocol change moves the Driver catalog identity, and `tools/umpire/evaluation`'s admission then rejects every pinned recorded Run and receipt as `stale` until each is re-recorded live (fn-89.1 found this; its R1 work is saved as `.flow/tmp/fn-89.1-wip.patch`). Re-recording is today scattered and manual: the replay control Run through its live test, the canary Run through `tests/testpilot_canary_lifecycle_test.go` with `UMPIRE_CANARY_RECORD`, then the receipts testdata and the catalog pins in `tools/canary/assessment/admission_test.go`, `tools/umpire/evaluation` (`TestTheControlRecordIsCurrent`) and `tools/umpire/cmd/umpire-assess/run_test.go`. Make it one command, so every fn-89 task (and fn-94) that changes the wire re-records in one step instead of widening its scope.

**Size:** M
**Files:** `Makefile` (a `umpire-rerecord-pinned-runs` target), the live tests' record hooks, the pinned records and receipts testdata under `tools/umpire/replay/testdata`, `tools/canary/**/testdata`, `tools/umpire/evaluation`, `tools/umpire/cmd/umpire-assess`, and a pin source the Go tests read instead of literals where that removes a hand-edited identity
**Touches:** [Makefile, tests/testpilot_canary_lifecycle_test.go, tests/testpilot_*replay*_test.go, tools/umpire/replay/**, tools/umpire/evaluation/**, tools/umpire/cmd/umpire-assess/**, tools/canary/**]

### Approach
- Inventory every test that pins a catalog identity or a recorded Run (grep `stale`, `catalog`, `recorded under catalog`, the pinned hashes) and every live test that can write a record.
- Add `make umpire-rerecord-pinned-runs`: runs each recording live test with its record hook, regenerates the derived receipts testdata, and rewrites the identity pins; idempotent on an unchanged protocol (no diff).
- Where Go tests compare against a hard-coded catalog identity, read it from the pinned record or one shared pin file, so the target updates one place.
- Keep the stale check itself unchanged: a catalog change must still stale old records; this task only makes refreshing them one step.
- Prove it: apply `.flow/tmp/fn-89.1-wip.patch` on a scratch tree state, run the target, and show the previously failing packages pass; then remove the patch (it lands in fn-89.1).

### Key context
- `make proto` currently fails at `lint-api` on `case.proto:43` (`class_name`, `core::0122::name-suffix`), pre-existing; regenerate with `make lint-protos protoc proto-codegen` as fn-89.1 did.

## Acceptance
- [ ] `make umpire-rerecord-pinned-runs` refreshes every catalog-pinned recorded Run and receipt in one step, is a no-op on an unchanged protocol, and leaves the stale check unchanged.
- [ ] With fn-89.1's patch applied, running it makes every previously stale-failing package pass.


## Done summary
Added `make umpire-rerecord-pinned-runs`: it probes each catalog-pinned recorded Run (the replay control and the canary) offline, re-records only a stale or crossed one through its live test, re-renders the receipt goldens from the control record, and runs the pinned packages; on an unchanged protocol it changes nothing. The evaluation and umpire-assess tests now read the control catalog from the record instead of a literal, and the publication test no longer depends on hash order. Proven with fn-89.1's Go/proto patch applied: every stale-failing package passed after one run.

stage: impl-review - ran [claude, SHIP; its P2/P3 notes applied in e6f8dd025a]
## Evidence
- Commits: db2a343633b9c9d65344e4dddc7fe2481eae8382, e6f8dd025acaa87e341d73f080ecb04a1a4906a8
- Tests: baseline: green (focused go test ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/... ./tools/canary/... ./tools/umpire/replay/... ./common/testing/testpilot/...), fn-89.1 patch (Go/proto half, model/* excluded to spare the concurrent fn-88.4 lake builds) applied: go test ./tools/... ./common/testing/testpilot/... failed stale in tools/canary, canary/assessment, canary/controller, canary/publication, umpire/evaluation, umpire/replay, make umpire-rerecord-pinned-runs (patched): re-recorded both Runs live, receipt goldens rewritten, all pinned packages ok; second run a no-op, go test ./tools/... ./common/testing/testpilot/... (patched, after target): all ok except tools/tests CQL suites (need Cassandra; unrelated), patch reverted with git apply -R; records and goldens restored from HEAD; make umpire-rerecord-pinned-runs (unpatched): both records current, no diff under tools/, go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/..., not run: make proto / umpire-check-testpilot-protocol / lint-model / umpire-check-regression (diff touches no proto, Lean or generated surface)
- PRs: