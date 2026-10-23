### Correctness axis

## Correctness audit addendum — `26dd66814f`

- **Original start-order finding: resolved (Conf 75).** Walked: `model/umpire/lift/src/safety.rs:387` uses ordered validity accumulators and assigns typed defaults to all fields after a fault. An earlier empty domain still prevents initialization. Cited: retained red logs show the original failure; corrected controls distinguish fault-first (`violated=true`) from empty-first (`violated=false`).

- **Original discarded-condition finding: resolved (Conf 75).** Walked: `model/umpire/front/src/body/effects.rs:194` retains each evaluated condition in existing branch checks, including guards after returns and around untaken branches. Cited: the retained red control misses division by zero; corrected controls detect the evaluated condition and suppress the untaken nested condition.

No concrete newly introduced correctness issue found in the correction delta.

Cited: `review-controls-checkpoint.log` records all nine actual-tool controls passing in 37.07 seconds. `review-lift-final.log` records focused lift tests, sequential-return growth, and field-count growth passing; `review-clippy-final.log` records successful strict clippy. Walked: step validity is bound once, and fault paths retain total state/monitor assignments.

This addendum used source and retained evidence only. No tests or tools were rerun. The required full `model-time` and terminal fresh live-gaps evidence remain pending.

### Standards axis

## Standards audit addendum — checkpoint `26dd66814f`

Original findings: **3 resolved, 0 unresolved.** No new standards findings in the correction delta.

- **Annotation ownership — resolved (walked).** `emit.rs` now emits typed annotations directly and reuses `observer_types` for declarations and metadata. `snowcat.rs` retains only type rendering; rendered-text parsing and fallback inference are removed.
- **Baseline inventory size — resolved (executed).** JSON comparison against `141df3cc41` confirms identical content, compacted to one line.
- **Compatibility inventory size — resolved (executed).** Every original entry remains unchanged; added fields explicitly mark the historical proof superseded. The file is one line. No files cross 1,000 lines against the original base.

**Verification limits (cited):** Retained focused lift/growth checks, strict changed-crate clippy and all nine actual Apalache controls pass; controls took 37.07 seconds. The final compatibility record states a fresh seven-scenario live-slice pass, gaps pending and no reused live-proof claim. The full required `model-time` remains pending. No suites were rerun or files changed during this addendum.

Correctness: 0 unresolved original findings; 0 new findings.
Standards: 0 unresolved original findings; 0 new findings.

Conductor decision: All five original findings are resolved at the source checkpoint. Normal same-cycle review returned SHIP. Completion still requires terminal fresh gap/privacy evidence and the one final required model-time; neither is claimed green by these addenda.

