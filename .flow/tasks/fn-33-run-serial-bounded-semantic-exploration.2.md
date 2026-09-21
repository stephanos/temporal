---
satisfies: [R1, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.2 Temporal bridge: whole Cases from target Queries under the caller Realization

## Description
Define the canonical `initialize`, `next`, `observe`, and `finish` frames and bind them to the campaign of task .1 for the caller Model. Each `next` carries one whole Lean-produced Case plus the opaque candidate identity and expected target keys; each `observe` carries only the exact closed Run/Verdict for the outstanding candidate and returns what was credited.

**Size:** M
**Files:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `model/Temporal/Tool/ExplorationBridgeMain.lean`, `model/lakefile.lean`, `Makefile`
**Touches:** [model/Temporal/Tool/ExplorationBridge.lean, model/Temporal/Tool/ExplorationBridgeTests.lean, model/Temporal/Tool/ExplorationBridgeMain.lean, model/lakefile.lean, Makefile]

### Approach
- Frames are canonical JSON on stdin/stdout of `lean_exe umpire-explore` (non-default), one frame per line; every frame names the set, the candidate identity and a frame sequence number, and a duplicate, stale, crossed or out-of-order frame rejects before any campaign call.
- `next` produces the Case with `Umpire.Case.Producer.produce` from the target Query's Plan under `Temporal.Feature.Nexus.Caller`'s existing `Realization` value, the machine's claims, evidence catalog and relations, exactly as the `case … realizes` block does for a functional set; the Case ID is `temporal.case.<set>.<candidate identity>` and no fixture is registered.
- `observe` reads the Run through the Case's evidence rules (`Umpire.Evidence`) into a decisive reading for the campaign's ledger; a Verdict that is not `satisfied` or `violated`, a Run whose cleanup is not closed, or a Run for another Case credits nothing.
- `finish` renders the campaign summary and the counterexamples with their promotion-source SHA-256 (task .5 compiles the source; the bridge names it).
- Early proof point: the first row target of `nexusCallerExploration` crosses the bridge as a Case that `testpilot.Prepare` accepts.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Case/Syntax.lean:150-240` — how a functional set's Case is produced; the bridge calls the same Producer with a Plan instead of a declared Query.
- `model/Temporal/Feature/Nexus/Caller/Model.lean:680` — `nexusCallerExploration`; `Caller/Fixtures/CallerExploratoryCoverage.json` — the target order.
- `model/ModelLint/ModuleIndexExporter.lean` and `Temporal/Tool/InventoryMain.lean` — the executable-boundary conventions (streams injected, buffered writes, exit codes).
- `model/Umpire/Evidence/Check.lean:232-280` — reading a Run against a Query.

### Quick commands
`cd model && lake build umpire-explore Temporal.Tool.ExplorationBridgeTests && lake exe umpire-explore-tests && cd .. && LEAN_NUM_THREADS=1 make lint-model`

### Re-plan note (2026-09-21)
Re-planned on fn-85's exploratory set after fn-86 R6 deleted the variation Space this task was first written against; see the spec's **Re-plan on fn-85** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Candidate, Case, budget, Profile/catalog and Limit bindings are canonical and exact in every frame; duplicate, stale, crossed, incomplete and N+1 frames fail before production or credit.
- [ ] `next` returns a whole Case that `testpilot.Prepare` accepts for the first row target of `nexusCallerExploration`; Go sees no target, coordinate or Case-family API.
- [ ] `observe` credits only from a decisive closed Run for the outstanding candidate and returns the credited target keys.
- [ ] The bridge executable is non-default, quiet on success and non-zero with empty stdout on any failure.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
