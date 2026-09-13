---
satisfies: [R9]
---
# fn-87-tighten-the-testpilot-protocol-glossary.11 Instructions run in entrypoint order by default; after names explicit dependencies

## Description
Make ordering and success-gating the default (R9, spec "Defaults and derived fields"): an instruction depends on the previous instruction of its entrypoint and runs only when every dependency succeeded; `after` states any other dependency set; an explicit guard is written only for any other condition. The checked-in Cases do not all fit the default (inventory below), so the wire needs explicit spellings for "no dependency", "several dependencies" and "run regardless", and the default guard must assert the same success facts an explicit success guard does today.

**Size:** M
**Files:** `proto/.../v1/instruction.proto` (`InstructionNode.dependencies` → `after`), `api/testpilot/v1/*`, `common/testing/testpilot/internal/execution/{prepare.go,dataflow.go,scheduler.go,dependencies_test.go,prepare_test.go,scheduler_test.go}`, `model/Testpilot/Authoring.lean` (`node`), `model/Temporal/Testpilot/CaseSupport.lean` (`succeeded`), `model/Temporal/Case/Template/{Workflow,NexusOperation}.lean`, `model/Temporal/Testpilot/WorkerOutage.lean`, `model/Temporal/Feature/Nexus/Success/{TypedNexus,TypedUnary}.lean`, fixtures, mapping, `common/testing/testpilot/internal/execution/README.md:100-106`, `model/Umpire/ARCHITECTURE.md:212-220`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, model/Testpilot/**, model/Temporal/**, tests/testcore/testpilot/**]

### Approach
- Inventory (from today's fixtures): async-nexus, worker-outage, typed-unary and every conformance Case are linear chains whose guarded nodes use exactly `all[present(dep.status), equals(dep.status, SUCCEEDED)]` (`CaseSupport.succeeded`). Exceptions: `await-nexus-operation`/`await-nexus-complete`/`await-nexus-confirm` depend on their predecessor with no guard (run regardless of success); typed-nexus has a second root in the same entrypoint (`start-nexus-confirm`, `await-authority-confirm` depends on `start-workflow` not its predecessor) and diamonds (`history`, `finish-workflow` with two dependencies); handler entrypoints have single roots.
- Wire: `InstructionNode.after` is a message with presence, `After { repeated InstructionReference instructions = 1; }`: absent means "the previous instruction of this entrypoint" (none for the first); present means exactly the listed dependencies, possibly empty (a second root). A `repeated` field alone cannot tell empty from absent, which the typed Nexus second root needs.
- Scope of `after`: the scheduler orders only within one entrypoint graph and rejects cross-entrypoint dependencies today (`execution/prepare.go:512`), and the spec's Boundaries add no new capabilities. So `after` names instructions of the same entrypoint; an `after` naming another entrypoint's instruction rejects at preparation as `unsupported` with a located path. This is the amended R9 (Planning decisions, "`after` scope"); cross-entrypoint waiting stays with `AwaitInstruction`.
- Default guard: a node with no explicit guard runs only when every dependency's outcome status is `SUCCEEDED`. It must also make dependency outcomes statically available exactly as today's explicit success guard does: availability is `previous.source.Guard == nil` today (`execution/dataflow.go:476`) plus success facts from `successScope`/`successFacts` (`dataflow.go:483-541`); derive the same facts from the default so inputs such as `complete.result = await.outcome.value` still bind.
- Run regardless: an explicit guard `literal(bool_value: true)` expresses today's unguarded dependent nodes, so their behavior is unchanged (declared in the mapping). An explicit guard replaces the default condition; it does not conjoin with it.
- Errors (R9 plus gap analysis): `after` naming an unknown instruction, itself, a duplicate entry, or forming a cycle rejects at preparation with a located path; cleanup graph instructions follow the same default within the cleanup graph; a dependency into or out of cleanup rejects as today.
- Lean: `Testpilot.Authoring.node` drops the explicit dependency list for the default and takes optional `after` and guard; `CaseSupport.succeeded` guards disappear from Producers where they equal the default; the unguarded await nodes get the explicit `true` guard.
- Mapping: a validated step: for each old node, compute the default expansion (predecessor dependency plus success guard); when the old dependencies and guard equal it, drop both; when the old node had dependencies and no guard, emit `guard: literal true` and `after` only if the dependency set differs from the predecessor; otherwise emit `after` with the old set. Any old guard that is neither absent nor the success pattern stays explicit. Retire `dependencies` only if it is a compound spelling in Go (`GetDependencies` on InstructionNode is generated; no token if bare).

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/prepare.go:487-550` — `addGraph`, `orderGraph`
- `common/testing/testpilot/internal/execution/dataflow.go:460-545` — outcome availability and success facts
- `common/testing/testpilot/internal/execution/scheduler.go` — dependency release and false guards
- `model/Temporal/Testpilot/CaseSupport.lean:60-80` — `succeeded`
- `model/Temporal/Feature/Nexus/Success/TypedNexus.lean:714-839` — the non-linear Program

**Optional:**
- `common/testing/testpilot/internal/execution/dependencies_test.go`
- `common/testing/testpilot/internal/execution/README.md:100-106`

### Key context
- The typed Nexus live test (tenfold load, two concurrent operations) is the pin that ordering did not change: event sequences and Verdicts must match.

## Acceptance
- [ ] an instruction without `after` depends on its entrypoint predecessor; without a guard it runs only when every dependency succeeded, and dependency outcomes bind as they do under today's explicit success guard
- [ ] `after` (with presence) states any other same-entrypoint dependency set, including none; unknown, self, duplicate, cyclic and cross-entrypoint `after` entries reject at preparation with located paths (unit tests), as amended R9 requires
- [ ] no checked-in Case writes a success guard or a predecessor dependency; the formerly unguarded await nodes carry an explicit `true` guard
- [ ] equivalence test passes with the validated default-expansion step; `expected.json`, correlated expectations and live Verdicts (typed Nexus included) unchanged
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

