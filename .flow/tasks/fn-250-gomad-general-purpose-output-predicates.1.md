---
satisfies: [R1, R2]
---
# fn-250-gomad-general-purpose-output-predicates.1 Predicate schema, validation and evaluator replace campaign/soft_assertions.rs

## Description
Define the typed output-predicate schema (name, streams, format json-line|text, match conditions field_equals/field_prefix/field_regex/line_contains/line_regex, identity field|capture|literal, severity fail|report) with deny_unknown_fields. Compile regexes once at preparation with the regex crate; refuse unknown fields, duplicate names, uncompilable or over-size patterns with status 2 before execution. Rewrite the scanner so it evaluates declared predicates over the existing record detector and strict field parser, keeping dedup, the 4096-byte message bound and the inconclusive rule for truncated or non-UTF-8 streams. Emit the typed OutputPredicateReport in place of the soft_assertions Report. Decide and record whether the preset uses several same-name predicates or a one-level any_of. Ordering: after fn-240 and fn-245 complete (spec dependencies).

## Acceptance
No string literal failed assertion, failed-assertion, FailedAssertion or soft_assertion remains in tools/gomad/rust outside the launcher preset and tests. Invalid predicate files are refused with status 2 naming the predicate. Regexes are applied only to parsed field values or raw text lines. Hits dedup sorted messages; truncated or undecodable streams set inconclusive and never pass. cargo test --workspace and the lint-rust/test-harness make targets green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
