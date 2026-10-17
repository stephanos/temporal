---
satisfies: [R3, R5]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.3 Capture derivation provenance without changing behavior

## Description
Collect exact derivation provenance and validate inherited/local targets (Scala groundwork for R5). This task does not change Go findings or suppress them.

**Size:** M
**Files:** derivation extraction, gap target validation and fixtures.
**Touches:** [model/irgen/Declarations.scala, model/irgen/Gaps.scala, model/irgen/Context.scala, model/irgen/Lifting.scala, model/irgen/test/Gaps.test.scala, model/irgen/testdata/gaps/**, model/framework/SEMANTICS.md]

### Approach
- Extend the existing derivation extractor before flattening destroys operation history. Preserve chained source origins and every existing drop/replacement/assumption behavior.
- Distinguish a local binding/claim change from inherited origin; refuse only inherited gaps whose relevant subject and context can be proven equivalent. Ambiguity is reported, never silently treated as equivalence.
- Supply task 6 with explicit provenance and conservative eligibility, not an assertion that all untouched bindings have the same reachability or claim judgment.

### Investigation targets
Required:
- `model/irgen/Declarations.scala:1204` — derivation extractor and flattened copies.
- `model/framework/Machine.scala:465` — restrict/drop semantics.
- `model/framework/Facet.scala:11` — facet ownership representation.
- `model/irgen/Capabilities.scala` — capability inheritance and waivers.
- `model/check/Equivalence.scala:369` — existing normalization.

### Verification
Add fixtures for restrict/rebind/extend chains, guard rebinding, monitor/refinement drops, replacement refinements, added assumptions and shared facets. Pin pre/post step tables and existing semantic projection for each operation; negative controls change a guard, outcome or claim context and must fail the pin. Record that consumer suppression is still unconfirmed until task 6's per-kind evidence.

## Acceptance
- [ ] Every existing derivation operation and chained origin is represented and tested.
- [ ] Pre/post step-table and semantic projection pins pass, with behavior mutations rejected.
- [ ] Inherited/local target controls pass conservatively; no Go finding suppression or full R5 claim occurs.

## Done summary
Captured exact derivation provenance in the existing extractor (`derivedMachine` in Declarations.scala) before flattening loses it, and added conservative inherited/local validation of gaps on derived machines (Gaps.scala `Provenance`, `DerivationOp`, `Inheritance`). Nothing written changes; no Go finding is suppressed.

baseline: green — fn-167.2 receipt state (Gap* suites 6/6; mini selected lift bytes in .flow/tmp/fn167/task2/before).
stage: impl-review - skipped(config: REVIEW_MODE=none; owner gates independent review once at Scala batch end)

Requirement evidence (R5 Scala groundwork, R3 partial):
- Every derivation operation is represented and pinned in GapResolutionSuite: assuming (kept move, added assumption), unmonitored (dropped monitor and refinement, kept move), rebind by bare binding and by guarded `on(...)` rules (rebound), restrict (kept move, dropped refinement), extend (added finish), refining (replaced refinement worker -> shadow), and a chain over a derived object (`PlainAssumed.unmonitored`: source plainAssumed, origins [plainAssumed, plain]).
- Conservative split: refused only where every operation provably keeps the finding (`assuming` for all table kinds; `unmonitored` for kinds no monitor pins and no refined state field keys), naming the root base through chains; `assumed`, `unmonitored` and two-step `chained` refusals pinned with exact messages. Negative controls: an outcome-changing rebind and a guard-changing rebind are local; dropped monitors (unconstrained-result) and dropped refinement (stuck-state) are unproven, as are restrict/extend/refining. Binding equality is never used as proof.
- Pre/post: the derived machines' lifted IR with gaps is byte-identical to the gap-blanked twin; the mini selected lift (which exercises unmonitored, restrict, rebind and restrict+extend chains) is byte-identical to the fn-167.2 base (mini-cmp.log); FEATURES=activity_mini steps-only equivalence against mini-selected-baseline-00d22597d2 is equivalent (16 tables; equivalence.log).
- `unread` is also accepted on a machine object that `refining` gave its refinement (no refinement object exists to annotate).

Verification: `umpire.irgen.Gap*` 8/8 (gaps-final.log, exit 0); selected.sh after + cmp; steps-only equivalence exit 0; scalafmt --check; git diff --check.

Failures retained: gaps-1..5.log (fixture text replacement, -Wsafe-init ordering, effects-section and effect-gives-no-step lints, an intended-inherited fixture gap correctly refused, refining classified unproven rather than local — accepted as the conservative reading).

Open obligations: task 6 owns per-kind consumer evidence and any Go suppression of inherited findings (R5 remains unconfirmed); task 5 serializes provenance (`Lifter.derivations`) through the shared IR as metadata excluded from identity. The monitor/refinement-independence of the six `unmonitored` kinds rests on the current lint/interp reading (monitors feed only pins; refinement adds a state field) and must be re-proven by task 6 per kind.

Deferred gates: complete lifter suite not re-run for this task (ran at the fn-167.2 boundary with inherited failures only); Go/proto/regeneration, real features, broad validation and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c3dedf5f92c778d4712c14d43d256233cd75f579
- Tests: timeout 1800 mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.Gap*' (8/8; gaps-final.log), /bin/bash .flow/tmp/fn167/task3/selected.sh after; cmp against .flow/tmp/fn167/task2/before/lift (identical; mini-cmp.log), FEATURES=activity_mini timeout 1200 mise exec -- scala-cli run --server=false model/check -- --check-equivalence .flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ .flow/tmp/fn161/task1/mini.map --steps-only (equivalent), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check <changed files>, git diff --check
- PRs: