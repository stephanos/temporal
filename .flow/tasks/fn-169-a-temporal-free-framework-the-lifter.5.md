# fn-169-a-temporal-free-framework-the-lifter.5 framework/samples: one-file sample Models (Kettle, Bank, TwoPhaseCommit) comparable with P and Quint

## Description
Owner request, 2026-10-12: `model/framework/samples/` holds one-file sample Models, Temporal-free, over a range of complexity, kept very manageable. Some match well-known P and Quint examples so the three can be compared side by side.

Samples (one Scala file each, in rising complexity):
1. `Kettle.scala`: one machine with phases (off, heating, boiling), rules and one safety Property. Compare with Quint's `examples/tutorials/repl/kettle.qnt`.
2. `Bank.scala`: P's Client Server tutorial. Clients withdraw through a bank server that keeps balances in a database. Safety: the bank always responds with the correct balance. Liveness: every withdrawal gets a response. Compare with P's ClientServer tutorial and the bank in Quint's README.
3. `TwoPhaseCommit.scala`: one coordinator and a bounded number of participants (two or three), composed. Safety: atomicity (no participant commits while another aborts). Include one negative control, for example a coordinator that commits on a partial vote, which the check must refute. Compare with P's TwoPhaseCommit tutorial and Quint's `examples/classic/distributed/TwoPhaseCommit/two_phase_commit.qnt` (adapted from TLA+).

Rules:
- Each sample compiles against `framework` alone (no `temporal.*`, no Temporal API). The samples therefore also prove fn-169's build boundary, and the boundary check covers them.
- Each sample states its counterpart in a short header comment, with the upstream path, and stays small: well under 200 lines for Kettle and Bank, under 300 for 2PC.
- The gate (or fn-166.13's scoped check) compiles, exports and checks each sample with its Queries answered as expected; samples are discovered, never listed.
- Model-only: no realization. fn-169.2 may give `Bank` a tiny proto API of its own as its generic-proto proof.
- `model/README.md` links the samples as the framework's starting point; the mini activity stays the Temporal running example.

## Acceptance
- R8: `model/framework/samples/{Kettle,Bank,TwoPhaseCommit}.scala` exist, each one file, compiling with no Temporal source or API on the classpath.
- Each exports to IR and its Properties/Queries check as expected; the 2PC negative control is refuted.
- Each names its P/Quint counterpart in its header; README links them.


## Done summary
Added three Temporal-free, one-file framework samples—Kettle, Bank, and a composed two-participant TwoPhaseCommit with a refuted negative control—and linked them from the model guide. The change-scoped gate now automatically discovers, compiles, exports, and checks every sample, with coverage and syntax-lint integration that treats samples as authored Models.

TDD: `framework.check.SamplesSuite` was observed red before sample discovery existed, then green; follow-up red/green slices pinned held-Step-name normalization, exact Query-form classification, and the authored-sample syntax boundary. `temporal.check.CoverageSuite` pins selection of the registered sample suite.

Upstream comparisons were verified against `p-org/P/Tutorial/1_ClientServer/PSrc/Server.p`, `p-org/P/Tutorial/2_TwoPhaseCommit/PSrc/Coordinator.p`, `quint-co/quint/examples/tutorials/repl/kettle.qnt`, `quint-co/quint/quint/README.md`, and `quint-co/quint/examples/classic/distributed/TwoPhaseCommit/two_phase_commit.qnt`.

baseline: red inherited/deferred (`FEATURES=activity_mini make model-mini` failed before implementation on held checked-in IR drift; MILESTONES.md lines 25-40 and model/README.md lines 582-623 defer that strict gate to the batch sweep; no generated output was edited)

verification: `make model-check-changed BASE=f243a8c59d6be46baedccdcfcbb81439ad3cca79` passed; `.flow/tmp/checks/20261012-231353/record.md` records three samples passed, mini compile/lift passed, mini step-table equivalence passed, and 139 model + 275 lifter + 141 check tests passed. The record reports checked-in mini IR drift as held, not a failure.

stage: impl-review - skipped(policy: source-batch review deferred)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4c4882ea4d129814adc478fa64841586ee8a4430, fa868908fc1e293fa3aee1165e7dd263fcecf339, 239b9d9949491e565755f2f8400f7ad21d073ecf, 2196a4a843570e473ea8cdad46af44fe1237df39
- Tests: mise exec -- scala-cli test --server=false model/framework/check model/temporal/check --test-only framework.check.SamplesSuite, mise exec -- scala-cli test --server=false model/framework/check --test-only framework.check.lint.SyntaxRuleSuite, mise exec -- scala-cli test --server=false model/framework/check model/temporal/check --test-only temporal.check.CoverageSuite, make model-check-changed BASE=f243a8c59d6be46baedccdcfcbb81439ad3cca79, KNOWN_INHERITED_DEFERRED: FEATURES=activity_mini make model-mini (held IR drift; strict gate deferred to batch end)
- PRs:
