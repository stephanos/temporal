---
satisfies: [R1]
---
# fn-225-umpire-lift-separate-model-shape-from.1 Split umpire-lift into shape and body lowering; tree into declarations and expressions

## Description
Pure refactor of model/umpire/lift/src: a shape module (single #[machine] module, imports, attributes, fields, starts, actions, claim/assumption/progress declarations and order) that never lowers statements or expressions, and a body/ module (expr, effects, vector, claim and progress bodies incl. path!, typer) that never walks items, taking a typed environment plus a syn node and returning Expr or Vec<Branch>. Split tree.rs into declarations and expr.rs. Public API, diagnostics text and locations unchanged; every golden byte-identical. A few lines on the seam in model/README.md and model/DESIGN.md. Verify make model-check, UMPIRE_BACKENDS=require and make model-time. Starts after fn-224 is closed in umpire's history.

## Acceptance
- [ ] Shape module never lowers bodies; body module never walks items
- [ ] tree.rs split into declarations and expressions
- [ ] All goldens byte-identical; diagnostics unchanged
- [ ] make model-check and UMPIRE_BACKENDS=require pass

## Done summary
`umpire-lift` now separates declaration discovery in `shape.rs` from typed body lowering in `body/`, with declarations in `tree.rs` and expressions in `expr.rs`. Existing APIs, diagnostics and goldens remain unchanged, and README/DESIGN describe the seam.

R1 is covered by the existing lift source, emission, vector, shadowing, sequential-check and remainder tests, the complete Model gate, and required TLC agreement. The body modules never walk items, and shape delegates supplied bodies to the typed environment.

baseline: green. Normal and required Model gates and model-time passed before editing. Baseline timing was 15.641s gate / 3.143s inner loop.
Verification passed with `make model-check`, `make model-check UMPIRE_BACKENDS=require`, `make model-time`, and `cd model && mise exec -- cargo test -p umpire-lift --quiet`. Post-change timing was 12.991s gate / 6.084s inner loop. Git comparisons confirm all expected files, refusal stderr files and expansion snapshots are byte-identical.
Logs are in `.flow/tmp/fn225/{baseline-model-check,baseline-required,baseline-time,focused,verify-model-check,verify-required,verify-time}.log`.

GATE_SKIPPED:model-check:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass
GATE_SKIPPED:model-check-required:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass
GATE_SKIPPED:model-time:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass

stage: impl-review - ran [2026-10-20T15:40:08Z..2026-10-20T15:43:16Z]
The three fresh Codex review contexts used `codex:gpt-6.1-sol:high` and returned SHIP with zero findings. Review and implementation use the same family. Reviewers could not run Cargo against its read-only build-lock directory; worker gate evidence above supplies runtime verification. Receipt is `/tmp/impl-review-receipt-8f37faba39e2-fn-225-umpire-lift-separate-model-shape-from.1.json`.

Tier: session (jev-unavailable(no_key))
The host did not expose executed implementer model metadata, so no actual-model annotation is inferred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9a6c4fb86ee674abe2e0daf052a6e45ef39121b5
- Tests: make model-check, make model-check UMPIRE_BACKENDS=require, make model-time, cd model && mise exec -- cargo test -p umpire-lift --quiet, git diff --check, git diff --exit-code 5ea50c7fa626d633d88157087b29028e7c40a60a -- model/umpire/lift/tests model/umpire/tests/fixtures model/umpire/dsl/tests model/temporal/tests, GATE_SKIPPED:model-check:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass, GATE_SKIPPED:model-check-required:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass, GATE_SKIPPED:model-time:green-receipt 9a6c4fb8 - verification reused from prior post-gate pass
- PRs: