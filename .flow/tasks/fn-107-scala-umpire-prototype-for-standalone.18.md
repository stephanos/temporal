---
satisfies: [R3, R4]
---
# fn-107-scala-umpire-prototype-for-standalone.18 Keep a replacing member's assumptions and admit action-scoped composition Properties

## Description
**Touches:** [model/go/umpire/compose.go, model/go/umpire/composekeys.go, model/go/umpire/*_test.go, model/scalav2/goir/load.go, model/scalav2/goir/claims.go, model/scalav2/goir/*_test.go, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/ir/activity*.json, model/scalav2/run.sh, model/scalav2/SEMANTICS.md]

Close the two pre-existing defects that the task 4 review found outside its Touches, then restore the parity they blocked. Review text: `.flow/tmp/fn-107/task4/t4-r1.md` (findings 2 and 3).

**Size:** M

### Approach
- **Replacement keeps the member's own assumptions (P1).** In `model/go/umpire/compose.go` (~:236) replacement removes any assumption of the replacing member whose name also appears on the replaced interface. `SEMANTICS.md` says the replacing member's assumptions stay and only the replaced machine's are discharged, so a receipt can claim an assumption-gated fault was checked unconditionally. Keep every assumption the replacing member declares; do not import the replaced non-member's. The fix applies to the typed and the key-level composition, which share the core; typed tables that have no such name clash stay byte-identical.
- **A `when` on a composition's Property (P2).** `model/scalav2/goir/load.go` (~:969) refuses `when_action`/`when_class` on a Property of a composition although the semantics allow it and the Go baseline answers it. Admit it against composed action keys and bind it through the key-level Property; keep the located refusal for a key that names no composed class.
- **Restore what the defects blocked.** Lift `stoppedWorkerStartsNothing`, move it from the parity test's exclusion list into the compared domain (the list is then empty), and rename `dispatchQueue.storageLoss` back to the shared assumption name with the receipts still listing it.
- Test-first for each. No feature policy in Go.

### Investigation targets
**Required:** `model/go/umpire/compose.go`, `composekeys.go`, `compose_test.go`, `composekeys_test.go`; `model/scalav2/goir/load.go` (~:960-975), `claims.go`, `admission_test.go` (~:650); `model/scalav2/goir/activity_parity_test.go` (`TestActivityCrossEntityClaimIsOutsideTheDomain`), `activity_system_test.go`; `model/scalav2/scala/temporal/standaloneactivity/{Claims,System}.scala`; `model/go/standaloneactivity/claims.go`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/...`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; `make lint-code LINT_CODE_TARGETS='./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT_FIX=false`.

## Acceptance
- [ ] A replacing member keeps every assumption it declares, also one the replaced interface names; the replaced machine's own assumptions are not imported. A receipt for a check gated by such an assumption names it. Typed compositions without a name clash are byte-identical (tables, IDs, fingerprints, existing tests unmodified).
- [ ] goir admits a Property of a composition with `when_action` or `when_class` over composed class keys and answers it through the generic key-level search; a key that names no composed class is a located admission error.
- [ ] `stoppedWorkerStartsNothing` is lifted and compared with the Go baseline; the activity parity test's exclusion list is empty and the test still fails on any uncompared baseline declaration.
- [ ] The storage-loss assumption has one name on the interface and the provider, and every receipt that relies on it lists it.
- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing.


## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: