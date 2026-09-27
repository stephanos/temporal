---
satisfies: [R19]
---
# fn-93-simplify-the-lean-model.34 Shared test fixtures and helpers (D-fixtures)

## Description
Lane D-fixtures. One checked-table builder over `FiniteTable.checkTypedModel` (inline calls in `Model/Tests/FiniteMachine.lean` ×8 and six other test files); one `Test.messageOwner` for the 7 private `owner : RpcOwner` copies; one bridge-test helper set for `Temporal/Tool/{Exploration,Replay}BridgeTests.lean` (`Captured`, `runScript`, `parseJson`, `field`, `stringField`, `natField`, `frameAt`); one IO test helper set (`fail`, `require`) for the 8 IO test mains; one `Except.error?` for the `errorKindOf` copies and same-bodied functions that survive lane B. The parameterized evidence fixture is not needed (B2 taken); if D2 was declined, add it.

**Size:** M
**Files:** `model/Umpire/Shared/Test.lean` (or sibling test-support modules), `model/Temporal/Tool/{ExplorationBridgeTests,ReplayBridgeTests,InventoryMainTests,InventoryMakeTests}.lean`, `model/ModelLint/{ImportGraphTests,ModuleIndexMainTests}.lean`, `model/Testpilot/Tests/ProtoJSONMain.lean`, `model/Umpire/Case/Tests/CorrelatedFixtureMain.lean`, owner-copy files (`Operation/Tests.lean:17`, `Property/Tests/Correlated/Fields.lean:32`, `Property/Tests/Fields.lean:44`, `Value/FieldTests.lean:18`, `Value/Tests.lean:45`, `Model/Tests/Parameterized.lean:13`, `Case/Tests/FieldLowering.lean:41`), `errorKindOf` files
**Touches:** [model/Umpire/Shared/**, model/Umpire/**/Tests/**, model/Umpire/**/*Tests.lean, model/Temporal/Tool/**, model/Temporal/System/**/Tests/**, model/Temporal/System/**/*Tests.lean, model/ModelLint/*Tests.lean, model/Testpilot/Tests/**, model/lakefile.lean]

### Approach
- Test-support modules stay inside `testSupportNamespaces`; IO helpers live where exe roots can import them (check `model/lakefile.lean` exe roots).
- Pure test refactor: every test keeps its assertions; goldens untouched.

### Investigation targets
**Required:**
- `model/Temporal/Tool/ExplorationBridgeTests.lean:20-100`, `model/Temporal/Tool/ReplayBridgeTests.lean:18-95`
- `model/Umpire/Shared/Test.lean`
- `model/ModelLint/ImportGraph.lean:165-172` (`testSupportNamespaces`)

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
```

## Acceptance
- [ ] One helper per concern listed; no private copies remain
- [ ] Every test keeps its assertions (receipt lists count of `#guard`/`#guard_msgs`/theorems before and after, equal or explained)
- [ ] `lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
