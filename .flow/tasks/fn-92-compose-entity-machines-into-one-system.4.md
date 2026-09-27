---
satisfies: [R6, R9, R10]
---
# fn-92-compose-entity-machines-into-one-system.4 restrict: and extend: keys, Control over Pair with byte-identical Case

## Description
Add `from:`, `restrict:`, and `extend:` keys to `machine` with the filtering, ordering, and inheritance rules the spec states, and rewrite `Nexus/Control` as `from: pair extend: handlerReply: forgedStep` such that its Case fixture and the recorded control Run stay byte-identical.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (machine keys and elaborator), `model/Umpire/Command/Derived.lean` (new), `model/Umpire/Command/Tests/Derived.lean` (new), `model/Umpire/Command/Tests.lean`, `model/Temporal/Feature/Nexus/Control/Model.lean`, `model/Temporal/Feature/Nexus/Control/Tests.lean`
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/Derived.lean, model/Umpire/Command/Tests/Derived.lean, model/Umpire/Command/Tests.lean, model/Temporal/Feature/Nexus/Control/**]

### Approach
- `restrict:` keeps listed actions' rows and drops the rest from the catalog (else `CheckedTable.action_executable` fails); filters `evidence:` for unreturned facts, `timers:`, `unobservable:`. `extend:` appends the author's results with located errors for a disabled source or a duplicate; results sorted by `stepOrderKey`; `restrict:` before `extend:`; derived machines drop `refines:` and the abstract field and re-own their catalogs under their own namespace so Definition IDs stay under the derived machine's family.
- Control: `machine nexusControl from: pair extend: handlerReply: controlForgedStep`, with `controlForgedStep` returning the one forged result for `handlerError false`; keep the name `nexusControl`, the state, outcome and fact spellings Pair and Control already share, its set, Query, and `case` block. Today's result order is already sorted, so the fixture must not change.
- Verify: `make umpire-gen-case-runtime-conformance` produces no diff; `go test ./tools/umpire/replay/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-assess/...` pass unchanged (they read the recorded control Run keyed by the fixture's canonical bytes).

### Investigation targets
**Required:**
- `model/Umpire/Command/Syntax.lean:1656-1700, 1866-2000, 2411-2457`
- `model/Umpire/Model/Table.lean:93-120`
- `model/Temporal/Feature/Nexus/Pair/Model.lean:30-120`, `Control/Model.lean:38-130`
- `tools/umpire/replay/key_test.go:13-17`

### Key context
- fn-89 and fn-90 touch the pair fixture; Pair itself is unchanged here, and the Control rewrite lands after fn-90.

## Acceptance
- [ ] `from:`/`restrict:`/`extend:` implemented with the stated rules; located errors pinned
- [ ] Control declares one extra-results function and no other step function; its Query outcome and witness unchanged
- [ ] Control and Pair fixtures byte-identical; the three Go tests reading the recorded control Run pass unchanged
- [ ] `make umpire-check-case-runtime-conformance` and `make lint-model` pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
