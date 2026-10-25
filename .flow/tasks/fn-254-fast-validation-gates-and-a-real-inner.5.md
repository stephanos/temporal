---
satisfies: [R5]
---
# fn-254-fast-validation-gates-and-a-real-inner.5 Gate composition: Apalache out, overlapping stages, trybuild trim, remove dead target

Touches: [Makefile, model/scripts/**, model/umpire/**/tests/ui/**, model/temporal/annotate/tests/ui/**]

## Description
Remove waste from the gate's structure without dropping a check (spec R5).

**Size:** M
**Files:** `Makefile` (model targets), `model/scripts/time-gate.sh`, `test-model-validate.sh`, trybuild UI cases.
**Touches:** [Makefile, model/scripts/**, model/umpire/**/tests/ui/**, model/temporal/annotate/tests/ui/**]

### Approach
- Apalache: routine `model-check`/`model-validate` never require it while fn-211 is deferred; `UMPIRE_APALACHE=require` stays available and documented. Its cost is reported separately.
- Overlap: run the Go host modules, guest freshness, retired-names, capability and protobuf scripts concurrently with the cargo stages (they do not take cargo's build lock, or use their own target). Keep fail-fast: any failing stage fails the gate with its name and exit code. Keep `UMPIRE_TIME_STEPS` per-stage timing working under concurrency.
- trybuild: of the remaining UI cases, convert those with no rustc type-check stage into table rows (fn-252.5 pattern). Keep the rustc-dependent ones; list both.
- Remove `model-check-changed` and `model-compiler-inputs` if only Scala uses them; check references first.
- `test-model-validate.sh` and `test-time-gate.sh` controls still pass, extended for concurrent stages.

### Investigation targets
**Required:**
- `Makefile:695-705` — dead Scala target
- `Makefile:1276-1315` — stage list
- `model/scripts/test-model-validate.sh`, `test-time-gate.sh` — gate controls
- `.flow/specs/fn-252-quick-consolidation-wins-in-model-code.md` (B5) — UI-case conversion rules

### Acceptance
- [ ] Every previous stage still runs in `model-validate`; a planted failure in each stage fails the gate with its name.
- [ ] Apalache is not required by routine gates; the opt-in still works.
- [ ] UI case count and trybuild time recorded before and after; every converted case keeps its semantic error and location.
- [ ] `model-check-changed` removed or justified.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
