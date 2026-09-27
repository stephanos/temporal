---
satisfies: [R11, R18, R19]
---
# fn-88-veil-concrete-checker-as-the-umpire.10 Exploration and Replay keys, the one-commit golden re-pin, and CI determinism

## Description
Make Exploration ledger credit and Replay violation keys compare by outcome and witness rather than by `explored` or `artifactChecksum` alone; rebuild the list of goldens the cutover flips and re-pin them in one reviewed commit; confirm determinism twice locally and once in CI. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Exploration/**` (ledger credit), `model/Umpire/Replay.lean` (`admitKept` key), `model/Temporal/Tool/ExplorationBridge*.lean`, `model/Temporal/Tool/ReplayBridge*.lean`, bridge tests, and the goldens on the R18 list
**Touches:** [model/Umpire/Exploration/**, model/Umpire/Replay.lean, model/Temporal/Tool/**, model/Umpire/Inventory/Tests/**, model/Umpire/Query/Tests/**, model/Temporal/Feature/**/Fixtures/**]

### Approach
- R19: where `candidateDigest` and the Replay `digest` are `artifactChecksum` (`model/Umpire/Exploration/Target.lean`, `model/Umpire/Replay.lean` `admitKept`), key on outcome plus witness; add a bridge test where one Query falls back to `reference` and neither a ledger status nor a Replay key changes.
- R18: build the flip list from the callers of `AdmittedQuery.search` and `searchWithIntent` (not the Search fixture tests); for each golden record old and new outcome in the task evidence; re-pin in one commit; verify every other golden is byte-identical with `make umpire-check-goldens`.
- R11: run the differential and the goldens twice locally and once in CI; compare Plan bytes, receipt JSON, and witnesses.

### Investigation targets
**Required:**
- `model/Umpire/Exploration/Target.lean`; `model/Umpire/Replay.lean` (`admitKept`)
- `model/Temporal/Tool/ExplorationBridge*.lean`, `model/Temporal/Tool/ReplayBridge*.lean`
- `model/Umpire/Command/Authoring.lean:810` — `boundWasHit`

### Key context
- The Plan artifact codec is unchanged; only `explored` values move, which is why checksums move.

## Acceptance
- [ ] Ledger credit and Replay keys compare by outcome and witness; the fallback bridge test passes
- [ ] R18 flip list with old and new outcomes in task evidence; re-pinned in one commit; all other goldens byte-identical
- [ ] Two local runs and one CI run produce identical Plan bytes, receipt JSON, and witnesses
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
