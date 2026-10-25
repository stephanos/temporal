---
satisfies: [R5]
---
# fn-254-fast-validation-gates-and-a-real-inner.5 Gate composition: Apalache out, overlapping stages, trybuild trim, remove dead target


## Description
Remove waste from the gate's structure without dropping a check (spec R5).

**Size:** M
**Files:** `Makefile` (model targets), `model/scripts/time-gate.sh`, `test-model-validate.sh`, trybuild UI cases.
**Touches:** [Makefile, model/scripts/**, model/umpire/dev/xtask/**, model/dev/xtask/**, model/umpire/**/tests/ui/**, model/temporal/annotate/tests/ui/**]

### Approach
- Apalache: routine `model-check`/`model-validate` never require it while fn-211 is deferred; `UMPIRE_APALACHE=require` stays available and documented. Its cost is reported separately.
- Overlap only dependency-independent stages. Guest freshness/build and protobuf scripts invoke Cargo/code generation today; inventory their actual locks, generated outputs and consumer edges instead of assuming independence. Bound shared build/JVM resources, then run safe non-cargo work beside Rust stages. Preserve named exits and deterministic timing/report order. A failing stage cancels/reaps owned siblings and preserves simultaneous failures; planted failures while siblings are active prove cleanup. Do not share a target across copied workspaces.
- trybuild: of the remaining UI cases, convert those with no rustc type-check stage into table rows (fn-252.5 pattern). Keep the rustc-dependent ones; list both.
- Remove `model-check-changed` and `model-compiler-inputs` if only Scala uses them; check references first.
- `test-model-validate.sh` and `test-time-gate.sh` controls still pass, extended for concurrent stages.
- Existing routine defaults already skip Apalache: report the historical opt-in cost separately rather than claiming its removal as a new default-gate saving. Keep explicit opt-in controls runnable. Register resulting stages/counts for .12 and preserve converted UI-case coverage against .1's inventory.

### Investigation targets
**Required:**
- `Makefile:695-705` — dead Scala target
- `Makefile:1276-1315` — stage list
- `model/scripts/test-model-validate.sh`, `test-time-gate.sh` — gate controls
- `.flow/specs/fn-252-quick-consolidation-wins-in-model-code.md` (B5) — UI-case conversion rules

## Owner validation cadence (2026-10-25)

Keep real changed orchestration wiring, stage-presence and mode/failure/cleanup controls now. Unrelated real full-stage execution waits for .8; synthetic controls never claim a production suite pass. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] Every previous stage still runs in `model-validate`; a planted failure in each stage fails the gate with its name.
- [ ] Apalache is not required by routine gates; the opt-in still works.
- [ ] UI case count and trybuild time recorded before and after; every converted case keeps its semantic error and location.
- [ ] `model-check-changed` removed or justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
