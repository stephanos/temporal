---
satisfies: [R3, R4]
---
# fn-218-test-space-patterns-cover-and-the-full.3 Kettle and activity cover tests; README grammar table

## Description
Demonstrate the complete authoring surface on the real Models at
`model/umpire/tests/fixtures/sample_kettle` and
`model/temporal/features/activity`, using .2's actual API handoff.
Kettle cover has Cases((80+1)*2)=162 alternatives: room-temperature start,
PressButton, [Heat;0..=80], explicit gap, then DepressButton|Failover.
The gap is essential because lower-count Failover alternatives need additional
heats; show every extra action in expansion. Preserve the literal adjacent form
as an unwitnessed-alternative refusal, never change enabledness to make it pass.
Activity cover has four closings times zero-or-one explicit Poll (eight choices),
with an explicit gap before closing for Complete/Fail's zero-prefix choices and
the declared `Deadline::StartToClose` timeout. The four closing actions are
`Client::Cancel`, `Worker::RespondCompleted`, `Worker::RespondFailed`, and
`Deadline::StartToClose`. Do not add ScheduleToStart as a fifth alternative or
claim zero explicit Poll means no actual Poll: the gap may supply it for completion,
failure, and StartToClose. Preserve closed Cancel rejection loops.

Express the archived Kettle enabledness table as one dense cover demonstration
for enabled cases, retaining every disabled assertion as a refusal/disabled control
in the same test. Do not introduce negative-pattern syntax or discard inconvenient
table rows. Preserve exact destinations and boundary cases, including above-boiling
Failover, which actually refutes fails_over_at_boiling. Prefer typed compact test
authoring over hand-enumerating 162/8 concrete cases.

Every cover/sample demonstration emits reviewed expansion and coverage goldens;
include at least one genuine seeded sample with its finite experiment report.
Preserve Kettle's top-level `fixture!` and its inline negative claim; compose suite
outputs through `fixture!(world, budget, negative, extra = suite_files()?)`.
The named assignment keeps source lifting from treating an arbitrary helper call
as another negative claim. Preserve native checker/negative outputs and reject
duplicate output keys. The actual source-lift proof is recorded in
`.flow/tmp/fn218-3-hook-proof-named-evidence.md`. Enable SUITE in both registries
and checked-in consumer manifests. README explains full grammar, alternatives/Cases, input declarations,
catalog coverage limits, gaps, independent budgets, shortest cover versus finite
sampling, and concise runnable examples. Do not revise the independent Models.

Activity edits are tests/expected only; wider test-crate edits are consumer
registries/manifests only as needed to enable the existing SUITE consumer.

**Touches:** `model/umpire/tests/fixtures/sample_kettle/**`, `model/temporal/features/activity/**`, `model/README.md`, `model/umpire/tests/**`, `model/temporal/testing/**`
**Quick:** `cd model && mise exec -- cargo test -p umpire-tests -p umpire-temporal-tests`.
Full gates: `make model-check`, `make model-check UMPIRE_BACKENDS=require`, `make model-time`.

## Acceptance
- [ ] Real Kettle cover admits exactly162 authored choices, and real activity admits exactly8, with exact choices/actions/outcomes/final states in goldens.
- [ ] Explicit-gap rationale and literal impossible-form refusal are pinned; neither Model enabledness nor claims are changed to fit tests.
- [ ] Archived enabledness positive and disabled rows remain covered with exact boundary outcomes; no negative-pattern language is added.
- [ ] Every cover/sample example has expansion and honest typed coverage goldens, including reproducible real finite sampling and untapped/unknown metadata where applicable.
- [ ] README matches the complete implemented grammar, budgets and coverage/sampling guarantees; focused suites, actual default/required gates and timing pass.

## Done summary
Kettle and activity authors now have reviewed expansions for 162 and eight closing choices. Kettle also compresses the archived enabledness table into nine exact cover alternatives with all seven disabled controls, retains the adjacent impossible-form refusal, and records four reproducible seeded walks with finite coverage.

The independent Models and claims are unchanged. The named fixture extra argument preserves native and negative outputs; its independent source-lift proof remains at .flow/tmp/fn218-3-hook-proof-named-evidence.md. Both consumer registries enable SUITE. The activity test manifest and generated Cargo.lock dependency entry add only the existing umpire-suite dependency. README documents the full grammar, finite input declarations, catalog coverage limits, explicit gaps, independent budgets, shortest cover and finite sampling.

baseline: green (cd model && mise exec -- cargo test -p umpire-tests -p umpire-temporal-tests), captured before edits in .flow/tmp/fn218-3/baseline-quick.log. Both new golden consumers first failed on missing snapshot metadata in kettle-golden-red.log and golden-red.log; update.log records initial generation. Expansion checks pin every actual step, outcome and final state, including gap-supplied Heat/Poll actions. The above-boiling postcondition-refutation exploration and closed Cancel rejection-loop tests remain active.

Review fanout e847e46426c34d5196d5889b059c23a6 returned SHIP from correctness, contracts and integration with zero findings. The reviewers performed static/golden verification; their Cargo attempts were read-only sandbox blocked. Worker Quick, strict all-target Clippy and full gates supply executed test evidence. The live review receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-218-test-space-patterns-cover-and-the-full.3.json.

Default make model-check and separately required make model-check UMPIRE_BACKENDS=require both passed with exit zero. make model-time passed with gate total 11.921s against 30s and inner loop 1.661s against 10s. The final focused suites passed with exit zero (11 shared-fixture tests, seven activity tests and the boundary suite). Strict all-target Clippy passed. Logs are in .flow/tmp/fn218-3/. Gate receipts could not be minted because preserved user MILESTONES.md edits make the checkout dirty outside the receipt ignore set. No green receipt or gate skip is claimed. All full gates actually ran.

Unrelated fn211/fn227/fn228/fn229 planning and MILESTONES.md edits remain uncommitted. Staging used git add -A followed by unstaging those user paths. The conductor owns feature-map and milestone updates and the spec-completion review. Changed authoring routes are cover/sample examples in the shared Kettle and system activity fixtures, plus the README grammar guide.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran (codex; gpt-6.1-sol at high, same-family reviewer; first-round SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c368e5a526fa17d3129732ae41ddcc7d9accebf6
- Tests: cd model && mise exec -- cargo test -p umpire-tests -p umpire-temporal-tests, cd model && mise exec -- cargo clippy -p umpire-tests -p umpire-temporal-tests --all-targets -- -D warnings, make model-check, make model-check UMPIRE_BACKENDS=require, make model-time
- PRs: