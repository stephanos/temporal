---
satisfies: [R4, R5]
---
# fn-254-fast-validation-gates-and-a-real-inner.5 Expose parallel fixture/compiler cases, trim UI work and retire dead targets


## Description
Remove waste from the gate's structure without dropping a check (spec R5).

**Size:** M
**Files:** existing fixture/compiler test binaries and helpers, parser-only UI cases, bounded nextest configuration and dead Make targets.
**Touches:** [Makefile, model/scripts/**, model/umpire/dev/xtask/**, model/dev/xtask/**, model/umpire/**/tests/ui/**, model/temporal/annotate/syntax/tests/**, model/temporal/realization/tests/**, model/umpire/author/dsl/macros/tests/fixtures.rs, model/umpire/author/dsl/macros/tests/parser_refusals/**, model/umpire/.config/nextest.toml, model/.config/nextest.toml, model/umpire/backends/tlc/src/runner.rs, model/umpire/dev/fixtures/build.rs, model/umpire/dev/fixtures/fixtures.rs, model/umpire/dev/testing/src/runner.rs, model/umpire/dev/testing/src/lib.rs, model/umpire/dev/testing/tests/consumers.rs]

### Approach
- Apalache: routine `model-check`/`model-validate` never require it while fn-211 is deferred; `UMPIRE_APALACHE=require` stays available and documented. Its cost is reported separately.
- Expose the 24 framework fixtures as separate libtest cases in the same existing binary, so nextest can schedule them. Keep the full registry/consumer/snapshot admission before selecting a case; retain unchanged CHECKER/SUITE/LIFT/TLC consumers and both renders per consumer. Remove the executing umbrella rather than duplicating its TLC work. Preserve interpreted and fixture-local tests. Temporal's activity `fixtures` selector stays unchanged.
- Break up or batch the expensive realization/compiler admission controls and reuse their owned compiler artifacts where exact source/tool/feature inputs permit it. Preserve all rustc-dependent checks and negative cases. The correct five-package quick workload took 444.027 s, including 388.875 s of realization tests; the two largest admission controls are the first targets. Never speed this up by excluding affected consumer packages or accepting cached validation without exact inputs.
- Inventory actual Cargo locks, generated outputs, consumer edges and resource costs for .17. Task .17 exclusively owns dependency-independent stage scheduling, simultaneous failure handling and sibling cleanup; do not build a second custom scheduler here. Keep named primitive failures and existing public wiring intact.
- trybuild: of the remaining UI cases, convert those with no rustc type-check stage into table rows (fn-252.5 pattern). Keep the rustc-dependent ones; list both.
- Remove `model-check-changed` and `model-compiler-inputs` if only Scala uses them; check references first.
- Existing mode, stage-presence and timing controls still pass; .17 adds concurrent-stage orchestration controls.
- Existing routine defaults already skip Apalache: report the historical opt-in cost separately rather than claiming its removal as a new default-gate saving. Keep explicit opt-in controls runnable. Register resulting stages/counts for .12 and preserve converted UI-case coverage against .1's inventory.

### Investigation targets
**Required:**
- `Makefile:695-705` — dead Scala target
- `Makefile:1276-1315` — stage list
- `model/scripts/test-model-validate.sh`, `test-time-gate.sh` — gate controls
- `.flow/specs/fn-252-quick-consolidation-wins-in-model-code.md` (B5) — UI-case conversion rules

## Owner validation cadence (2026-10-25)

Keep actual changed test/runner execution, exact coverage mappings, stage-presence/mode controls and bounded JVM-owner checks now. Run all 24 new framework cases with TLC required and caching disabled. Unrelated whole-gate execution and final measurements wait for .8; .17 owns changed scheduling controls. No fresh timing-baseline campaign is required.

## Acceptance
- [ ] The old framework umbrella maps exactly to 24 cases in the same binary, with all other identities preserved except separately listed new controls. Full fixture/consumer/output ownership, snapshots, both renders, registry failures and dynamic orphan/nondeterminism refusals remain covered. Preserve the 119 dependency tuples separately; they are not fixture counts.
- [ ] Expensive compiler controls retain all positive/negative semantics while exposing independent cases or batching/reusing owned compiler work. Record current selected/executed case counts and timings with complete consumer coverage.
- [ ] Apalache is not required by routine gates; the opt-in still works.
- [ ] UI case count and trybuild time recorded before and after; every converted case keeps its semantic error and location.
- [ ] `model-check-changed` removed or justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Bounded execution parallelism

Consume the source-grounded decision at `.flow/tmp/fn254-jvm-parallel-shape/decision.md`. Tune the JVM group from one to two test owners in both nextest configurations, with four global slots for required-backend gate invocations and the standalone child. Keep backend-skipped native execution at the existing CPU-based default. Preserve fail-fast, every filter/identity/ignored case and the separate doctest phase. Keep the TLC recovery default at two processes per owner; cap the retained TLC discovery probe with the existing default heap limit so the bound includes it. Under one cross-invocation JVM lease this admits at most four checker JVMs with 2 GiB configured model heaps; JVM/native/Go overhead still needs separate resource admission. Never call sampled RSS a hard memory bound.

Prove the candidate with current real simultaneous framework JVM owners and a Temporal case, exact membership/count checks and bounded process sampling. Reuse task .3's historical identity/workload evidence; do not launch a fresh group-one timing baseline campaign or attribute speedup across incomparable workloads. Keep heavy Cargo/Go stages separate until their combined budget is established. This stays part of the existing gate composition task and does not reopen completed .3.

For UI conversion, preserve each old diagnostic message and useful exact authored location. The read-only 63-case proposal is a candidate list, not proof of equivalence; retain any case whose full behavior cannot be reproduced in the table. New parser table source belongs under the declared nested parser_refusals tree, avoiding another test binary. Keep required parent/isolated-child suite authority and output contracts unchanged; artifact-publication or authority-deduplication redesign is outside this task.
