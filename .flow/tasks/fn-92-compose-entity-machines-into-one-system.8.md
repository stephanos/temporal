---
satisfies: [R7]
---
# fn-92-compose-entity-machines-into-one-system.8 Make the composed-table kernel agreement scale

## Description
Make the per-composition kernel agreement check scale. fn-92.5 measured the check `compose` emits (`decide +kernel` over `composedTableAgrees`) growing super-linearly: 98 states / 113 rows took 8.7 s of kernel time, 146 states / 585 rows took 398 s (5.2x rows, 46x time), and the full `nexusCaller` (every protocol action and all four timers) was killed after 30 minutes at a 15 GB footprint. Turning the protocol table into the positional index costs 11.2 s once; the cost is in the check itself (`rowsFrom`/`sourceComplete`/`reach` scan every row per state). Measurements and the ready `nexusCaller` draft are in `.flow/tmp/fn-92.5-summary.md`.

**Size:** M
**Files:** `model/Umpire/Command/Syntax.lean` (`elabComposedAgreement` and the member/literal terms it emits), `model/Umpire/Command/ComposeProofs.lean`, `model/Umpire/Command/Tests/Compose.lean`
**Touches:** [model/Umpire/Command/Syntax.lean, model/Umpire/Command/ComposeProofs.lean, model/Umpire/Command/Tests/Compose.lean, model/Umpire/Command/Compose.lean]

### Approach
- Keep the trust basis: the agreement stays a kernel-checked theorem (no `native_decide`, no `Lean.ofReduceBool`), per R4/R7.
- Candidates (the spec's first fallback, proof shape): compute member tables and the composed literal to literals once at elaboration and decide equality on literals once; group rows by source state so `rowsFrom`/`sourceComplete`/`reach` index instead of scanning; split into per-row or per-source-state lemmas combined by a structural lemma in `ComposeProofs.lean`.
- Measure kernel seconds (`set_option profiler true` / `trace.profiler`) for the fn-92.3 `workerOutage` composition, the two fn-92.5 scratch compositions, and a full-`nexusCaller`-sized scratch composition over the committed Caller module; target: the full size within a few minutes of kernel time and memory well under 8 GB.
- `workerOutage`'s agreement and every existing composition test keep passing; fn-92.3's recorded seconds may only go down.

### Key context
- The Caller module stays untouched here; fn-92.5 lands the `nexusCaller` draft after this task.

## Acceptance
- [ ] The composed-table agreement is still a kernel-checked theorem, and a full-`nexusCaller`-sized composition checks within a few minutes of kernel time and under 8 GB, measured and recorded.
- [ ] `workerOutage` and every existing composition test pass; their kernel seconds do not grow.


## Done summary
The composed-table agreement stays a kernel-checked theorem (`ComposedAgreement.ofLiterals` over five `decide +kernel` decisions, axioms propext/Classical.choice/Quot.sound only). The kernel decides that each reading of the member tables, the candidates, and the composed literal equals a literal computed once at elaboration, that the literal grouped by source state flattens back to it, and that the check holds over the grouping, which indexes rows by state instead of scanning them. A full nexusCaller-sized composition (316 states, 1468 rows) now decides in 86.5 s of kernel time at 5.15 GB peak RSS; the flat check was killed at 30 min and 15 GB. The 585-row case dropped from 398 s to 25.1 s. In a same-run comparison, workerOutage's agreement went from 135 ms (old) to 109 ms (new). The 113-row scratch case costs 15.8 s against 8.7 s recorded earlier under different host load, because reading the member table dominates at that size.

Review found one issue: the state view took its binders from the author's field names, so a member field named `Umpire` shadowed the namespace. The binders are now hygienic (a111afe3f3), and a fixture in `Tests/Compose.lean` pins that case. The regression gate was already failing before this task on a `workflow.` prefix in a Syntax.lean comment from 881a7b413ad; 6ed2bb8f4b fixes it. The earlier commit 7c2392c51d also changed `Tests/ComposeProofs.lean`, which is outside the declared Touches.

stage: impl-review - ran [codex fan-out bd85d449 NEEDS_WORK, refunded on head change .. fan-out bc69ca49 SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7c2392c51da6a350aab28dc514dae1ebd7f4a4dd, 6ed2bb8f4b99649dcdd5500dfdf409a75da786e1, a111afe3f39d9c83868ee47fc55e19f109ba661e
- Tests: baseline: green (cd model && lake build Umpire.Command Temporal.Feature rc=0 pre-edit); make umpire-check-regression red pre-edit (retired-vocabulary scan hit a workflow. prefix in a Syntax.lean comment from 881a7b413ad), fixed in 6ed2bb8f4b, cd model && lake build UmpireTests TemporalModelTests (rc=0 at a111afe3f3, 1823 s), make umpire-check-regression (rc=0 at a111afe3f3, 1759 s; includes umpire-check-goldens, canary-check-case, umpire-check-case-runtime-conformance), LEAN_NUM_THREADS=1 make lint-model-builtin LINT_MODEL_MODULES="Umpire.Command.Syntax Umpire.Command.ComposeProofs Umpire.Command.Tests.Compose Umpire.Command.Tests.ComposeProofs" (rc=0), GATE_SKIPPED:lint-model:scoped - whole-model lint runs once at fn-92.6 per conductor policy, measure (lake env lean -Dprofiler=true, /usr/bin/time -l, Lean 4.32.0, host swapping): full nexusCaller-sized scratch (316 states, 1468 rows, 23 actions) agreement kernel 86.5 s = members 17.5 + candidates 0.13 + literal 7.6 + flatten 13.3 + check 48.0; file type checking 111 s, wall 163 s, peak RSS 5.15 GB (baseline: killed at 30 min, 15 GB), measure: 146 states / 585 rows agreement kernel 25.1 s (12.8 + <0.1 + 1.77 + 1.68 + 8.88), peak RSS 2.49 GB, wall 93 s (baseline 398 s), measure: 98 states / 113 rows agreement kernel 15.8 s (9.56 + <0.1 + 0.85 + 0.70 + 4.69), peak RSS 3.40 GB (baseline 8.7 s under other load; the members reading dominates at this size), measure: workerOutage same-run old flat check 135 ms vs new five decisions 108.7 ms (17.8 + 12 + 15.3 + 11.4 + 52.2); fn-92.3 recorded 78 ms under other load, review: codex fan-out round 1 at 5c10a0d789 NEEDS_WORK (field binder shadowing, fixed a111afe3f3; round refunded on head change); re-dispatch at a111afe3f3 SHIP x3
- PRs: