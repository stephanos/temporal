---
satisfies: [R2, R3]
---
# fn-250-gomad-general-purpose-output-predicates.2 Bind the predicate list into recipe and plan identity; thread the report through receipts, ledger and inspect

## Description
Replace soft_assertion_report_only in the prepared-target recipe, campaign plan, explore options and run-guest admission with output_predicates. Keep the list outside the compilation key and inside the full provenance digest (extend the compilation-key test that today flips soft-assertion). Resume, execute-shard and retained replay refuse a plan whose predicate list differs. Rename the soft_assertions field on execution receipts, ledger rows, inspect --json and progress lines to output_predicates carrying the typed report; the failure signature of a fail hit covers predicate name plus sorted messages, and a report hit leaves the outcome class unchanged. Regenerate goldens once per the compatibility policy and update CLI.md.

## Acceptance
Changing, adding or removing a predicate leaves the compilation key unchanged and changes the full provenance digest. Resume, execute-shard and retained replay refuse a differing predicate list with an explicit message. Same predicate name plus same sorted messages share a failure signature across seeds; different predicate names do not. report-severity hits keep outcome class success. Retained scenario fixtures and the TestSoftAssertion* controls in cycle/plan/declarations.rs produce the same hit sets, inconclusive verdicts and grouping as before (field name aside). CLI.md documents the schema and the report.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
