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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
