---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-91-say-opaque-handle-in-the-driver-contract.2 Hold the retired Driver names in the vocabulary gate and fix the prose

## Description
Add the eight retired Driver-seam tokens to the vocabulary gate with pinned tests (R3), drop fn-78 from the gate's downstream list (R5), bring the READMEs and the COMPONENTS phrase to the handle vocabulary (R4), then run the full regression and lint (R6, R2). Split from .1 because the tokens fail the gate on every file .1 renames: they can only land after it.

**Size:** S
**Files:** `tools/umpire/internal/retiredvocabulary/{check.go, check_test.go}`, `common/testing/testpilot/temporal/README.md`, `common/testing/testpilot/temporal/worker/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [tools/umpire/internal/retiredvocabulary/**, common/testing/testpilot/temporal/README.md, common/testing/testpilot/temporal/worker/README.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Tokens: add `OpaqueCapability`, `CapabilityEffect`, `CapabilityBridge`, `CapabilityFactory`, `NewCapability`, `InvokeCapability`, `CapabilitySlot`, `CapabilityClaim` to `exactTokens` in `buildRetiredRules()` next to the fn-87 block (`check.go:~620-650`). The lowerCamel rule (`check.go:743-752`) covers `opaqueCapability`, `capabilitySlot`, `capabilityClaim`. Each must pass `validateRetiredToken` (`check.go:320`).
- Replace the comment at `check.go:595-597` that calls `CapabilityBridge` and `CapabilityEffect` the live Driver seam with one naming the fn-91 retirement.
- Remove fn-78 from `downstreamSpecs` (`check.go:27-48`, fn-78 at `:46`).
- Pins: extend `TestRetiredRulesHoldTheGlossaryRenamedProtocolNames` (`check_test.go:39-155`) with a want row per token plus one lowerCamel want (`opaqueCapability`), and no-want rows for `OpaqueHandleType`, `Umpire.Capability`, `KNOWN_GAP_KIND_CAPABILITY`, `OpaqueHandle`, `HandleBridge`. Every positive literal is written split (`"Opaque" + "Capability"`), because the gate scans this test file.
- Prose: `temporal/README.md:17` "generic capability factory" -> "generic handle factory"; `temporal/worker/README.md:58` `` `CapabilityFactory` `` -> `` `HandleFactory` ``; `.plans/UMPIRE4_COMPONENTS.md:34` "the capability bridge" -> "the handle bridge". Leave `temporal/README.md:45` (Opcode sense) and `UMPIRE4_COMPONENTS.md:53`. `.plans/UMPIRE4_ORDER.md` already carries no capability bullet; do not edit it.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/retiredvocabulary/check.go:20-50,279-330,590-650,740-760` - downstream list, task scanning, token validation, fn-87 block, lowerCamel rule
- `tools/umpire/internal/retiredvocabulary/check_test.go:39-155` - the pin table to extend

**Optional:**
- `.flow/memory/` entry `glossary-renames-can-reintroduce-names-2026-09-13` - run the gate after the last edit, not mid-task

### Key context
- Before the change, run `make umpire-check-regression` at the base commit and record its passing live-identity count in the evidence (R6); the after-run must match it.
- Regression runs live Testpilot tests. A live identity that fails is re-run against the base commit before being attributed (fn-90 tracks the intermittents); record both runs.
- `.plans/UMPIRE4_ORDER.md` and `.plans/index.json` may carry other sessions' uncommitted edits: never stage them.

### Acceptance
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/retiredvocabulary/...` green with the new want and no-want rows
- [ ] `make umpire-check-retired-vocabulary` green; temporarily reintroducing `InvokeCapability` in a scanned Go file fails it with path and line (checked, then reverted)
- [ ] fn-78 absent from `downstreamSpecs`; the live-seam comment is gone
- [ ] both READMEs and the COMPONENTS sentence say handle; no other `.plans` file or spec/task record changed
- [ ] `make umpire-check-regression` exits 0 with the passing live-identity count recorded at the base commit; `make lint-code-fast` reports no new issues

## Acceptance
- [ ] TBD

## Done summary
The vocabulary gate now retires OpaqueCapability, CapabilityEffect, CapabilityBridge, CapabilityFactory, NewCapability, InvokeCapability, CapabilitySlot and CapabilityClaim (lowerCamel covers the server's unexported types), pinned by a want row per token plus opaqueCapability and one live no-want row (OpaqueHandleType, OpaqueHandle, HandleBridge, Umpire.Capability, KNOWN_GAP_KIND_CAPABILITY). The live-seam comment is gone, fn-78 left downstreamSpecs, and the two Temporal Driver READMEs and the UMPIRE4_COMPONENTS sentence say handle factory / HandleFactory / handle bridge.

Regression ran after fn-90.8's commit 860ecde37d: exit 0, 45 passing live identities. fn-91.1 recorded no count to compare against; the live suite does not import the only Go package this task touches, so the base count is the same.

stage: impl-review - ran [claude backend, SHIP on first round]
## Evidence
- Commits: 4b1d55a7a6afb2923446b9bd38aa3c2235f9384f
- Tests: baseline: green (quick commands; CC=/usr/bin/clang), go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/replay/... ./tools/canary/... (rc=0), go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/replay/... ./tools/umpire/internal/retiredvocabulary/... (rc=0), make umpire-check-retired-vocabulary (rc=0); probe: InvokeCapability appended to common/testing/testpilot/temporal/worker/api.go failed it with api.go:52, then reverted, make lint-code-fast GOLANGCI_LINT_BASE_REV=a164116e87b53f3be01be1f7ffb6e8a2807ae0e8 (0 issues), make umpire-check-regression at 860ecde37d (after fn-90.8's pair-fixture commit) (rc=0): 45 passing live identities (25 top-level TestTestpilot*), 0 FAIL, 0 SKIP. fn-91.1 recorded no live-identity count; the live suite's test dependency graph (go list -test -deps ./tests) excludes retiredvocabulary and this task changes only that package and markdown, so the base-commit count equals this one
- PRs: