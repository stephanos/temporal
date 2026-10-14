---
satisfies: [R1, R2, R4]
---
# fn-91-say-opaque-handle-in-the-driver-contract.1 Rename the Driver seam's Go from capability to opaque handle

## Description
Apply every row of the spec's Renames table (R1) to the hand-written Go in one atomic commit: nothing compiles until every `Session` implementation moves to `InvokeHandle`, so the contract, facade, Drivers, Executor and all test fakes land together. Also moves the handle-sense Go doc comments and diagnostics (R4, API Contracts) and re-splits the two `"OpaqueCapability" + "Type"` spellings so .2's tokens do not trip on them. No gate token is added here; that is .2.

**Size:** M (mechanical, wide: 23 Go files plus two file renames)
**Files:**
- Contract and facade: `common/testing/testpilot/contract/driver.go`, `common/testing/testpilot/contract.go`
- Server Driver: `common/testing/testpilot/temporal/server/{capability.go -> handle.go, capability_test.go -> handle_test.go, ownership_test.go, session.go, driver.go}`
- Worker Driver: `common/testing/testpilot/temporal/worker/{api.go, callback.go, driver.go, interpreter.go, session.go, session_test.go, typed_test.go, runtime_fixture_test.go}`
- Composite: `common/testing/testpilot/temporal/{driver.go, driver_test.go}`
- Executor and IR: `common/testing/testpilot/internal/execution/{scheduler.go, scheduler_test.go, typed.go}`, `common/testing/testpilot/internal/ir/{expression.go, path.go}`
- Activation test: `common/testing/testpilot/temporal/internal/activation/activation_test.go`
- Facade tests: `common/testing/testpilot/{conformance_test.go, protocol_test.go}`
- Other Sessions: `tests/testcore/testpilot/{artifact_test.go, workflow_start_artifact_test.go}`, `tools/umpire/replay/driver_test.go`
- Gate test re-split only: `tools/umpire/internal/retiredvocabulary/check_test.go`
**Touches:** [common/testing/testpilot/contract/**, common/testing/testpilot/contract.go, common/testing/testpilot/temporal/**, common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/ir/**, common/testing/testpilot/conformance_test.go, common/testing/testpilot/protocol_test.go, tests/testcore/testpilot/**, tools/umpire/replay/**, tools/umpire/internal/retiredvocabulary/check_test.go]

### Approach
- Order: contract leaf types and `Session.InvokeHandle`/`Bridge` (`contract/driver.go:37-94`), facade aliases (`contract.go:15-21`), then each implementation until `go vet -tags test_dep` over the Quick-commands packages is clean.
- Server: `git mv capability.go handle.go` and `capability_test.go handle_test.go` so history follows. Rename `opaqueCapability`/`capabilitySlot`/`capabilityClaim` (`capability.go:11-26`), their `capability` fields, `Session.capabilities` (`session.go:31-32,298-301`) and the `driver.go:181,203` constructions. Rename the helpers `capabilitySession`, `capabilityEffectFunc`, `successfulCapabilityEffect`, `blockingCapabilityEffect`, `capabilityValue` and the five tests: four in `capability_test.go` (`:43,:76,:92,:149`) and `TestRejectedCapabilityInvocationRestoresClaimForCleanup` (`ownership_test.go:96`).
- Local-name collisions (spec Architecture): the server `InvokeHandle` (`capability.go:59-124`) already has a parameter named `opaque` (asserted to `*capabilityClaim` at `:60`) and an EffectHandle local `handle` (`:113`). Rename the parameter to `claimed`, the local `capability := claim.capability` (`:87`) to `opaque`, and keep `handle`. Where tests already bind `handle` to an EffectHandle (`capability_test.go:63,106`, `ownership_test.go:37`), the opaque-handle local becomes `opaque`. The composite `InvokeHandle` (`temporal/driver.go:246-247`) has no competing binding, so its `capability` parameter becomes `handle`; its `Reserve` (`:172-204`) `handles` are ReservationHandles and are untouched. Elsewhere a `capability` local becomes `handle` (for example `scheduler.go:739-745`).
- Worker: `CapabilityFactory` -> `HandleFactory` and `SessionOptions.NewCapability` -> `NewHandle` (`api.go:36-43`); callers at `callback.go:81`, `driver.go:199`, `interpreter.go:278-288` and `temporal/driver.go:102`.
- Diagnostics per API Contracts: `scheduler.go:742` `nil capability` -> `nil opaque handle`; `ir/expression.go:316` and `ir/path.go:69` `capabilities cannot be inspected` -> `opaque handles cannot be inspected`; `conformance_test.go:389` and `tools/umpire/replay/driver_test.go:79` test-Session errors say handle. `scheduler.go:749` `controller capability required` is Opcode sense and stays.
- Slot literals: `"capability"` -> `"handle"` in `server/{capability,ownership}_test.go`, `worker/{typed,session,runtime_fixture}_test.go` and `activation/activation_test.go:93,117`; `"private-capability"` -> `"private-handle"` (`activation_test.go:307`); `"nil-capability"` -> `"nil-handle"` (`scheduler_test.go:533,545`). `prepare_test.go:135` (Opcode case name) and every known-gap `"capability"` stay.
- Re-split `"OpaqueCapability" + "Type"` at `protocol_test.go:111` and in the want literal at `check_test.go:80` to `"Opaque" + "CapabilityType"`, and the same line's `line:` literal `"&pb.OpaqueCapability" + "Type{}"` to `"&pb.Opaque" + "CapabilityType{}"` (same concatenated values).
- Go doc comments in the handle sense: `contract/driver.go:37,48,71`, `server/capability.go:31`, `worker/interpreter.go:278`, `internal/execution/typed.go:230` ("delivers through its capability" -> "through its opaque handle").
- Equivalence pin: this is a rename with no behavior change. The pin is the conformance corpus and generated protocol code staying byte-identical under the three R2 make targets (`git status --porcelain` clean outside the Touches list), plus the renamed Driver ownership, closure, quarantine and race tests passing under `-race`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/contract/driver.go:30-100` - the seam being renamed
- `common/testing/testpilot/temporal/server/capability.go` - unexported types and the InvokeCapability body with the local collision
- `common/testing/testpilot/temporal/worker/api.go:30-50` - factory and options
- `common/testing/testpilot/internal/execution/scheduler.go:560-760` - Executor consume/invoke path
- `common/testing/testpilot/temporal/driver.go:95-260` - composite Session and its ReservationHandle locals

**Optional:**
- `tools/canary/controller/fenced.go` - embeds `testpilot.Session`; must compile unchanged
- `.flow/tasks/fn-87-tighten-the-testpilot-protocol-glossary.3.md` - the fn-87 rename task this mirrors

### Key context
- `EffectHandle`, `ReservationHandle`, `Bridge`, `Publish`, `Await`, `Consume` keep their names (Boundaries).
- Other senses of capability stay (spec Edge Cases list); when unsure, it is the handle sense only if the value is an `OpaqueCapability` or the Slot holding one.
- Do not run `make proto` or any fixture generator: a regenerated file is an R2 failure.
- Other sessions leave uncommitted `.plans`/`.flow` edits; stage only this task's files.

### Acceptance
- [ ] `go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/replay/... ./tools/canary/...` clean
- [ ] `git grep -nE 'OpaqueCapability|CapabilityEffect|CapabilityBridge|CapabilityFactory|NewCapability|InvokeCapability|capabilitySlot|capabilityClaim|opaqueCapability' -- '*.go'` hits only the split spellings and `retiredvocabulary/check.go`; `git grep -nw -e capability -e capabilities -- common/testing/testpilot tests/testcore/testpilot tools/umpire/replay` hits only other-sense uses, listed in the done summary
- [ ] no scope binds one name to two handle kinds; the server `InvokeHandle` names its parameter `claimed`, the minted-handle local `opaque` and the EffectHandle `handle`; the composite `InvokeHandle` parameter is `handle`
- [ ] `go test -count=1 -race -tags test_dep ./common/testing/testpilot/... ./tools/umpire/replay/... ./tools/umpire/internal/retiredvocabulary/...` green, including the five renamed server tests
- [ ] `make umpire-check-case-runtime-conformance umpire-check-testpilot-protocol umpire-check-testpilot-authoring` green with no fixture, `expected.json` or generated-file diff

## Acceptance
- [ ] TBD

## Done summary
Renamed the Driver seam's Go from capability to opaque handle per the spec's Renames table (OpaqueHandle, HandleEffect, HandleBridge, Session.InvokeHandle, worker HandleFactory/NewHandle, server NewHandle, opaqueHandle/handleSlot/handleClaim, handles field), moved server capability.go/capability_test.go to handle.go/handle_test.go, renamed the five server tests, moved handle-sense diagnostics, doc comments and test Slot literals, and re-split "Opaque" + "CapabilityType" in protocol_test.go and check_test.go. The server InvokeHandle binds claimed / opaque / handle; the composite InvokeHandle parameter is handle.

Remaining word-boundary capability/capabilities hits in the touched Go trees are all other-sense: correlated capability comments (dataflow.go:308, ir/expression.go:51, verification/*, tests/testcore/testpilot artifact comments), "unsupported instruction context or Driver capability" (dataflow.go:82, typed_test.go:257, preparation_error_test.go), "controller capability required" (scheduler.go:749), the Opcode case "capability" (prepare_test.go:135). READMEs (temporal/README.md:17 "generic capability factory", worker/README.md:58 `CapabilityFactory`) are left for .2 (R4 prose).

Gates: vet, tests, -race and the protocol/authoring/vocabulary checks green. Inherited reds: (1) tools/umpire/replay/bridge_live_test.go races under -race (exec stderr copier vs the eager stderr.String() argument; untouched file) - follow-up. (2) make umpire-check-case-runtime-conformance is red at HEAD because another session's b90b63e072 (Lean API regeneration) moved two behaviorFingerprint values; this commit changes no Lean, fixture or generated file. That fixture drift needs regenerating by the owner of b90b63e072 before .2's R6 run.

stage: impl-review - ran [claude backend, verdict SHIP on first round]
## Evidence
- Commits: 61c570fc594f9ed09317225493f9515df91dbeab
- Tests: baseline: green (CC=/usr/bin/clang; without it cgo fails on stddef.h in agent shells), go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/replay/... ./tools/canary/..., go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/replay/... ./tools/umpire/internal/retiredvocabulary/..., go test -count=1 -race -tags test_dep ./common/testing/testpilot/... ./tools/umpire/replay/... ./tools/umpire/internal/retiredvocabulary/... (all green except inherited race in untouched tools/umpire/replay/bridge_live_test.go TestLiveReplayBridgeAdmitsTheControlByItsBytes; replay re-run -race with -skip of that test: green), make umpire-check-retired-vocabulary, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-case-runtime-conformance: RED, inherited from b90b63e072 (Lean API regeneration by another session); only two behaviorFingerprint values move in tests/testcore/testpilot/testdata/{nexusPairTests-bothComplete,workflowStartTests-started}-case.json; 61c570fc59 touches no Lean, fixture or generated file
- PRs: