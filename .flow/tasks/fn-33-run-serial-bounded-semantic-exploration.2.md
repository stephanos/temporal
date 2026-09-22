---
satisfies: [R1, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.2 Temporal bridge: whole Cases from target Queries under the caller Realization

## Description
Define the canonical `initialize`, `next`, `observe`, and `finish` frames and bind them to the campaign of task .1 for the caller Model. Each `next` carries one whole Lean-produced Case plus the opaque candidate identity and expected target keys; each `observe` carries only the exact closed Run/Verdict for the outstanding candidate and returns what was credited.

**Size:** M
**Files:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `model/Temporal/Tool/ExplorationBridgeMain.lean`, `model/Temporal/Case/Syntax.lean`, `model/Umpire/Command/Registry.lean`, `model/Umpire/Command/Syntax.lean`, `model/Temporal/Feature/Nexus/Caller/Model.lean`, `model/lakefile.lean`, `Makefile`
**Touches:** [model/Temporal/Tool/ExplorationBridge.lean, model/Temporal/Tool/ExplorationBridgeTests.lean, model/Temporal/Tool/ExplorationBridgeMain.lean, model/Temporal/Case/Syntax.lean, model/Umpire/Command/Registry.lean, model/Umpire/Command/Syntax.lean, model/Temporal/Feature/Nexus/Caller/Model.lean, model/lakefile.lean, Makefile]

### Approach
- Frames are canonical JSON on stdin/stdout of `lean_exe umpire-explore` (non-default), one frame per line; every frame names the set, the candidate identity and a frame sequence number, and a duplicate, stale, crossed or out-of-order frame rejects before any campaign call.
- Record the machine's declaration `Name` on `Umpire.Command.Registry.SetEntry` in `recordSet` (today the entry carries only name, purpose, queries and repeat, and an exploratory set has no queries to reach its machine through), so the block can resolve the set's machine.
- Extend the `case … realizes` block to accept an exploratory set: it emits `<block>.realization`, `.claims`, `.catalog` and `.relations` for the set's machine and registers nothing in the Temporal Case Registry (today it rejects any set that is not functional or canary and reads claims, catalog and relations from the elaboration Registry, which a runtime bridge cannot). `nexusCallerCases.realization` is the value; the caller Model gains the exploratory block.
- `next` produces the Case with `Umpire.Command.produce checked identity realization evidence (claims := …) (evidenceCatalog := …) (relations := …)` from the candidate's `CheckedModel`; the identity is Case ID `temporal.case.<set>.<candidate identity>` with fixture `<set>-<candidate identity>`, so each candidate's run scope and workflow type are its own; nothing is registered in the Temporal Case Registry.
- `observe` decodes the Run and Verdict (`Protobuf.Json.fromJson`), checks the Case ID, disposition, cleanup status and Verdict status, and hands the campaign a decisive observation; a Verdict that is not `satisfied` or `violated`, a Run whose cleanup is not closed, or a Run for another Case credits nothing. The model reads no Run evidence: credit is the planned witness path.
- `finish` renders the campaign summary and the counterexamples with their promotion-source SHA-256 (task .5 compiles the source; the bridge names it).
- Early proof point: the first row target of `nexusCallerExploration` crosses the bridge as a Case that `testpilot.Prepare` accepts.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Case/Syntax.lean:150-240` — how a functional set's Case is produced; the bridge calls the same Producer with a Plan instead of a declared Query.
- `model/Temporal/Feature/Nexus/Caller/Model.lean:680` — `nexusCallerExploration`; `Caller/Fixtures/CallerExploratoryCoverage.json` — the target order.
- `model/ModelLint/ModuleIndexExporter.lean` and `Temporal/Tool/InventoryMain.lean` — the executable-boundary conventions (streams injected, buffered writes, exit codes).
- `proto/internal/temporal/server/api/testpilot/v1/run.proto` — what a Run and a Verdict carry (status, per-rule status, terminal state); `model/Testpilot/ProtoJSON.lean` — the codec.

### Quick commands
`cd model && lake build umpire-explore Temporal.Tool.ExplorationBridgeTests && lake exe umpire-explore-tests && cd .. && LEAN_NUM_THREADS=1 make lint-model`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Candidate, Case, budget, Profile/catalog and Limit bindings are canonical and exact in every frame; duplicate, stale, crossed, incomplete and N+1 frames fail before production or credit.
- [ ] `next` returns a whole Case that `testpilot.Prepare` accepts for the first row target of `nexusCallerExploration`; Go sees no target, coordinate or Case-family API.
- [ ] `observe` credits only from a `satisfied` closed Run for the outstanding candidate, along its planned path, and returns the credited target keys; the `case … realizes` block accepts the exploratory set through the machine recorded on `SetEntry` and registers nothing; each candidate carries its own fixture and run scope.
- [ ] The bridge executable is non-default, writes nothing to stdout beyond frames, and exits non-zero with a diagnostic on stderr for any failure outside a frame.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
