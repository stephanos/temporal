---
satisfies: [R1, R4]
---
# fn-85-model-side-effects-as-typed-actions-and.4 Entity instances and structured machine state in the Contract

## Description
Let a Model hold several instances of each entity bounded by Limits, with Search, admission and path selection over every instance's state (R1), and carry per-instance state fields in the Contract instead of one state value plus facts: the correlated transition records the fields the machine keeps, keyed by the entity's key (the structured machine state fn-87 left to this spec). Interleavings across instances are paths (R4).

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/correlated.proto` (a repeated named-value field record on the correlated transition and initial state, additive on fn-87's shapes), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean` and `Correlated.lean` (structured state in the Lean interpreter), `model/Umpire/Case/Correlated.lean` and `Projection/*.lean` (lower machine fields, not one atom), `model/Umpire/Property.lean` and `Property/Evaluate.lean` (`PropertyTraceField.state` per field; `naturalAtMost` reads a `count` field), `model/Umpire/Search.lean` (instances as candidate setups; interleavings), `model/Umpire/Query.lean` (`FiniteDomain` fingerprints over instances), `common/testing/testpilot/internal/verification/{correlated,correlated_prepare}.go` (structured state), `common/testing/testpilot/testdata/case-runtime-conformance/correlated.json` (regenerated), conformance corpus (a correlated case over two instances), `model/Umpire/Case/Tests/*.lean`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Umpire/Property.lean, model/Umpire/Property/**, model/Umpire/Search.lean, model/Umpire/Query.lean, common/testing/testpilot/**]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- Protocol: follow fn-87's extension checklist (`common/testing/testpilot/README.md`) for the new correlated shape; additive fields, no compatibility shim, fixtures regenerate through their generators, a Driver conformance case per changed message.
- State: an instance's state is a record of named fields; the Contract's correlated transition carries `prior` and `next` field records and the projection rule matches on fields, so `attempts` compares as a number and `phase` as an enum without parsing one atom.
- Instances: the fixed-width tuple from task .3 becomes the setup shape; `Query.FiniteDomain.canonicalRoleAssignments` enumerates instance assignments; a bound of zero rejects; exceeding a bound reports `limitReached`.
- Interleavings: Search treats every enabled step of every instance as a candidate; the exact-sequence Scenario still pins one order.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto` (post-fn-87) — the transition and initial-state messages
- `model/Umpire/Case/Correlated.lean:28-216` and `model/Testpilot/Correlated.lean:29-45,211-352`
- `model/Umpire/Property.lean:303-361` and `Property/Evaluate.lean:60-120` — trace fields and numeric predicates
- `model/Umpire/Search.lean:375,760-795` — setups and root traces
- `common/testing/testpilot/internal/verification/correlated_prepare.go`

**Optional:**
- `common/testing/testpilot/README.md` — the extension section fn-87 R7 wrote

### Key context
- fn-87's boundary table assigns "correlated transitions over structured machine state" to this spec; this is that task.
- Memory: check unbounded Lean numbers before protobuf narrowing (`count` crosses into a fixed-width field).

### Deferred here from task .14, 2026-09-18

`DESIGN.md` writes an instance's creation as a row from `none`, taken by the action that `creates:`
the entity. That is a different shape from a step over an existing state -- its function takes no
prior state, and its successors are the machine's *initial* states rather than transition rows -- so
`machine` takes a `starts:` key naming those states directly, as the `model` command did. Deciding
whether a creating action's step function produces the initial states, and whether that action
belongs in the machine's enumerated Action domain at all, needs the instance model this task owns.

## Acceptance
- [ ] the correlated Contract carries per-instance state fields; the Lean interpreter and the Go evaluator agree on `correlated.json` and on a new two-instance conformance case
- [ ] a Model with two operation instances admits, searches interleavings, and a Query selecting one path per instance produces a Case; an instance bound of zero rejects in place; an exhausted bound reports `limitReached`
- [ ] the async-Nexus fixture regenerates with only the structured-state diff, listed in the receipt; `make umpire-check-regression` exit 0


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
