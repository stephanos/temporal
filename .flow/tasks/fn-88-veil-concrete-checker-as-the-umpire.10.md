---
satisfies: [R11, R18, R19]
---
# fn-88-veil-concrete-checker-as-the-umpire.10 Exploration and Replay keys, the one-commit golden re-pin, and CI determinism

## Description
Make Exploration ledger credit and Replay violation keys compare by outcome and witness rather than by `explored` or `artifactChecksum` alone; rebuild the list of goldens the cutover flips and re-pin them in one reviewed commit; confirm determinism twice locally and once in CI. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search/Selection.lean` (set `Selection.cutover := true` in the same commit as the golden re-pin), `model/Umpire/Exploration/**` (ledger credit), `model/Umpire/Replay.lean` (`admitKept` key), `model/Temporal/Tool/ExplorationBridge*.lean`, `model/Temporal/Tool/ReplayBridge*.lean`, bridge tests, and the goldens on the R18 list
**Touches:** [model/Umpire/Search/Selection.lean, model/Umpire/Exploration/**, model/Umpire/Replay.lean, model/Temporal/Tool/**, model/Umpire/Inventory/Tests/**, model/Umpire/Query/Tests/**, model/Temporal/Feature/**/Fixtures/**]

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
Cut `Umpire.Search` over to `veil` (`Selection.cutover := true`), keyed Exploration and Replay on the Plan's witness instead of its checksum (R19), and pinned a `veil` Plan and receipt that the golden check renders again on every run, now in CI too (R11).

- **R18 flip list** (commit ec07f5afe9, one commit with the cutover). The list comes from the callers of `AdmittedQuery.search` and `searchWithIntent`: command Queries, the Promotion replan, the Variations compiler, and the Switch runs. Regenerating every golden directory and the case-runtime conformance fixtures under the cutover changed no byte. Over the checked-in Queries, `veil` visits exactly as many product states as `reference` enumerates paths, so no `explored` value and no outcome moved. Only four Lean pins flipped:
  - `Search/Tests/Replay`: `cutover == false` → `true`.
  - `Search/Tests/Replay`: `Selection.search` used to equal the reference search; now it equals `searchWith .veil`. Outcome is `found` both times.
  - `Search/Tests/Replay`: the Switch exact-action backend changed from (reference, default) to (veil, default). Outcome is `found` both times.
  - `Search/Tests/Admission`: `AdmittedQuery.search` used to equal the reference `search`; now it equals `Selection.search`. Outcome is `found`.
- **R19** (f7adebb647). `Command.Promotion.witnessKey` is the Plan's checksum resealed with `explored` cleared. It now keys `Candidate.identity`, campaign history, Case IDs, `Replay.Admitted.of` (used by `admitKept` and the replay bridge subject), and proposal names.
  - Every checked-in exploration or replay Query fixes its schedule, and both backends explore those identically. `Exploration/Tests/Classed` therefore gains a free-schedule lamp Query: `veil` reports 5 states and `reference` 6 paths, with the same witness.
  - The new `fallback` checks in the exploration and replay bridge tests search that Query on both backends. They assert equal identity, Case ID, every credited ledger, and Replay digest. Both checks fail when the keys are the checksum.
- **R11** (d157f623ef). The golden writer now renders the Caller `retry` Plan and its planning receipt (`searchBackend: veil`, `veilCommit`). The Umpire workflow's canary job runs `make umpire-check-goldens`, and `ci_workflow_test.go` pins that step.
  - Two local golden runs, two renders compared with `diff -r`, and two elaborations of each differential were identical.
  - The CI leg has not run, because push is forbidden here. It runs on the next push of `stephanos/umpire`.

Deviations:
- **Edits outside Touches:**
  - `Umpire/Command/Promotion.lean`: `propose` named proposals by `artifactChecksum`, so leaving it would have split the key.
  - `Umpire/Search/Tests/{Replay,Admission}.lean`: the flipped pins.
  - `Umpire/Replay/Tests.lean`: the digest pin.
  - `.github/workflows/umpire.yml` and `tools/umpire/regression/ci_workflow_test.go`: the CI golden step and its pin.
- **Follow-up for .7:** `model/ARCHITECTURE.md:208,212` still says Replay candidates are named by "Plan checksum". They are now named by `witnessKey`.
- The review base was ace45a1002, not the recorded base 6515338620. ace45a1002 is another session's `.flow` commit that landed between them.
- Defer mode: not applicable (R22 adopt).
- Baseline: `umpire-check-goldens` was green before any edit.
- `lint-model` ran green with no OOM.

stage: impl-review - ran [2026-09-27..2026-09-27] (codex fan-out, three draws SHIP, zero findings)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ec07f5afe9, f7adebb647, d157f623ef
- Tests: baseline: green (make umpire-check-goldens pre-edit), cd model && lake build (whole model, cutover on), make umpire-gen-goldens + make umpire-gen-case-runtime-conformance under cutover: zero byte changes, make umpire-check-goldens (run 1 and run 2, both green); umpire-goldens rendered to two dirs: diff -r identical (14 files), lake env lean Umpire/Search/Tests/Differential.lean and TemporalModelTests/SearchDifferential.lean, twice each: rc=0, identical, make umpire-check-exploration-bridge, make umpire-check-replay-bridge, make umpire-check-case-runtime-conformance, make umpire-check-regression, LEAN_NUM_THREADS=1 make lint-model (whole-model builtin lint completed, no OOM; 2616s, peak RSS 5.2 GB), make lint-code-fast, fallback tests red with checksum keys (identity/digest mismatch), green with witnessKey, CI: not run (push forbidden); canary job now runs make umpire-check-goldens
- PRs: