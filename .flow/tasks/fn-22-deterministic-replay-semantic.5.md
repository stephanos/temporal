---
satisfies: [R4]
---
# fn-22-deterministic-replay-semantic.5 Compile ordered typed edits into whole candidate Cases through a Lean replay bridge

## Description
Add `Umpire.Replay`: over an `AdmittedQuery` with its checked Model, the finite ordered edit `dropPrefixStep i` (last prefix step first), each re-admitted through `Umpire.Command.checkAdmitted` and reported `inapplicable` when the Model does not admit it, and a monotonic `Reduction` value that retains a candidate, never retries a rejected edit and never reintroduces a dropped step. Lift the exploration bridge's frame parsing, rejection rules, profile echo and `Effects` into `Temporal.Tool.Bridge`, imported by both executables, with `umpire-check-exploration-bridge` kept green. Add `Temporal.Tool.ReplayBridge` (`umpire-replay-bridge`, non-default): `admit` names the set and the Query or the exploration target key, recovers the admitted Query (registry, or the fn-33 campaign replanned to that target), re-produces the Case under the set's realization and admits only when the bytes are the subject's; `next` hands out the next candidate as one whole Case under `temporal.case.<set>.<digest>` with the fixture `<set>-<digest>`, the digest being the edited Query's Plan checksum (`Umpire.Exploration.candidateDigest`, computed before the Case exists, as the exploration bridge names a candidate; the subject's own digest is its admitted Query's Plan checksum), with the edit applied, or reports the edit `rejected` with the Producer's reason; the Case checksum travels beside it as the identity and names nothing; `observe` takes the candidate's classification (task .4's class, by Go) and advances the reduction; `finish` reports minimized, irreducible or incomplete with every edit's fate. Determinism as fn-33 .5 pins it: the same frames twice give the same frames.

### Approach
- The lamp Model of `Umpire.Exploration.Tests.Classed` and the switch pin the edits; the caller Model's exploratory candidate pins `irreducible` at once; a functional caller Query with a prefix pins one admitted edit.

### Quick commands
`cd model && lake build umpire-replay-bridge umpire-replay-bridge-tests umpire-explore umpire-explore-tests UmpireTests && lake exe umpire-replay-bridge-tests && cd .. && make umpire-check-exploration-bridge`

**Size:** L
**Files:** `model/Umpire/Replay.lean`, `model/Umpire/Replay/Edits.lean`, `model/Umpire/Replay/Tests.lean`, `model/Temporal/Tool/Bridge.lean`, `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `model/Temporal/Tool/ReplayBridge.lean`, `model/Temporal/Tool/ReplayBridgeMain.lean`, `model/Temporal/Tool/ReplayBridgeTests.lean`, `model/lakefile.lean`, `Makefile`
**Touches:** `model/Umpire/Replay*`, `model/Temporal/Tool/**`, `model/lakefile.lean`, `Makefile`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review rounds one and two; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Lean owns applicability, order, candidate naming (the Plan checksum digest) and Case compilation, with the Case checksum reported beside it; an edit the Model does not admit is `inapplicable`, recorded, and produces no Case; no edit is listed twice.
- [ ] A subject whose bytes no set of the Model produces is `crossed` at `admit`; duplicate, stale, out-of-order and oversized frames are rejected before any campaign call; the exploration bridge's tests and gate stay green over the shared module.
- [ ] Go receives whole Cases and returns classes; it has no Case mutation or coordinate-editing API.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
