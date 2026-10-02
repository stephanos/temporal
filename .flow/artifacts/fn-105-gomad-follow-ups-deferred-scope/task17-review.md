# fn-105.17 review (raw codex bridge, gpt-5.6-sol at high, read-only, working-tree diff; commits forbidden)

Three rounds. Rounds 1 and 2 returned NEEDS_WORK with blockers, all applied. Round 3 returned NEEDS_WORK with no blocker and one should-fix, which was applied and is recorded as applied-unreviewed: the strict-failure wording in fn105-d17-evidence.json now matches the report's qualified sentence.

## Round 1

## Findings

- **BLOCKER** — [GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:19](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:19) claims “No runtime choice diverges,” and [line 179](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:179) claims no trace overflow. Neither is observable: watchdog runs skip trace validation ([runner.go:1515](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/runner/runner.go:1515)) and retain no choice profile ([runner.go:1653](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/runner/runner.go:1653)); the campaigns were deleted ([evidence.json:672](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json:672)). Correct to: no runtime-choice divergence or overflow was reported, but neither can be excluded because no complete trace exists; the first retained evidence divergence is virtual time.

- **BLOCKER** — The categorical “unrelated”/“no connection” conclusion at [report:264](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:264), [evidence.json:635](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json:635), and [GOMAD_MILESTONES.md:202](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:202) exceeds the evidence. The fixed D14 toolchain shows that D14’s correction did not resolve D17 and the diagnosed pool wait has a distinct demonstrated cause. There is no unfixed-toolchain comparison, and no Linux run bears on D12. Replace with “no evidence of a shared cause; D14’s fix is insufficient; D12 remains unassessed.”

- **BLOCKER** — “Differ only in `virtual_time_elapsed_nanos`” at [report:14](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:14) and the milestone row is not retained evidence. `first_divergence` stops at the first mismatch ([qualification.go:394](/Users/stephan/Workspace/temporal/gomad/tools/gomad3/qualification/qualification.go:394)); retained reports contain baseline evidence plus per-execution digests, while the full reports/artifacts were deleted. Say “the first divergent evidence field is…” unless a complete normalized field comparison is retained.

- **BLOCKER** — The strict-tick diagnosis at [report:251](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:251) incorrectly generalizes the frontend-first span list to both failures. The retained evidence shows `TestOperation` has zero spans, while the frontend-first comparison belongs to `TestWorkerOperation` ([strict-failures-seed11.txt:8](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-strict-failures-seed11.txt:8)). Record the two observed failures separately and remove the unproven sort-based causal attribution.

- **SHOULD-FIX** — [Report:176](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:176) says every tracked-tree execution ends under the watchdog, contradicting the virtual-timeout target failures at [line 67](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:67). Scope this to the unbounded qualification executions.

- **SHOULD-FIX** — The one-slot regression criterion at [report:318](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:318) is not executable under A2+B, where environment overrides are rejected and the Gomad build-tag path supplies two slots. Specify a test seam, temporary one-slot variant, or native one-slot control.

Acceptance: criteria 1, 2, and 5 are met; criterion 3 is incomplete because choice divergence/overflow is asserted despite unavailable trace evidence; criterion 4 is not met because D12/D14 is concluded categorically rather than hedged. Seeds, run counts, filenames, throwaway attribution, owners, open-work status, tracing assertions, and the isolated D17 diff scope otherwise agree.

VERDICT: NEEDS_WORK

## Round 2

## Findings

- **BLOCKER** — [GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:361](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:361) still claims that “no runtime choice diverges” is inferred. Equal output and successful two-slot controls do not establish choice equality in watchdog-killed runs with no Choice Trace. Replace with: “No diverging runtime choice was observed; whether choices diverged in the killed runs is unknown.”

- **SHOULD-FIX** — [GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:230](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:230) says seed 17 contains no corresponding `%p` line, but retained evidence only establishes that its replay matched. Remove that statement or retain the seed-17 stderr/search evidence.

- **SHOULD-FIX** — [GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:262](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:262) says every expected span was present in another order, while [fn105-d17-strict-failures.txt:1](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-strict-failures.txt:1) retains only a reduced five-field projection, omitting kind, URL path, and Nexus attributes used by the actual equality assertion. Retain the full projected lists or narrow the report’s claim to the retained fields.

- **NIT** — [GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:119](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:119) dates the environment hunk to 2026-08-23; blame attributes its introduction to 2026-08-21. Correct or remove the date.

Round-1 findings 2, 3, 5, and 6 are resolved. Finding 4 is appropriately non-causal now, subject to the retained-projection issue above. Finding 1 remains unresolved at line 361.

Acceptance: reproductions and identities, native and slot controls, termination classification, D12/D14 hedging, ownership, next actions, qualification criteria, preserved tracing assertions, and skip retention are present. The first-divergent-event requirement is not cleanly met while the unsupported choice-equality inference remains. Excluding the declared D19 changes, the diff stays within the allowed D17 files and lines.

VERDICT: NEEDS_WORK

## Round 3

No BLOCKER findings.

- SHOULD-FIX — [fn105-d17-evidence.json:530](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json:530) still says “every expected span [is] present in another order.” The raw structs differ in order-assigned trace/span/parent IDs; multiset equality holds only after removing those IDs. Replace this with the qualified wording used in the [report:262](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md:262).

All five acceptance criteria are otherwise met. The cited source supports the causal chain; the round-2 limits, seed-17 scan, raw strict-tick evidence, and commit-message qualification are present. D12/D14 remains appropriately hedged, the throwaway variant is clearly identified, the skip remains, and D17 changes stay within the allowed files.

VERDICT: NEEDS_WORK
