# Public reports, pack-directory and UTC source review

The independent public_utc_repairs_review found no demonstrated production
regression, no Critical issue, two Important preservation-evidence gaps and one
Minor formatting issue. The conductor forwarded all findings to the sole writer.
This is a source-only early assessment, not formal SHIP, merge readiness or
native acceptance. In-progress external-consumer/analyzer/World work was excluded.

## Strengths

Runner maps every journal/artifact field in declaration/tag order, preserving
Uint64String, string outcomes and optional pointers. Target retains every nested
evidence field, nil/empty distinctions and old outer nil-to-nonnil-empty behavior;
mutable slices/pointers are cloned. Pinimpact keeps the delayed load after
baseline/candidate validation and adapter evaluation; explicit directories use
LoadPackDirectory and defaults use LoadPacks. Missing-directory semantics,
unknown-pin reporting, path masking and refresh authoring-root intent remain.

All four timestamp corrections preserve layouts and validation positions;
record retains original strings, governance retains its trailing-Z requirement.
Record preservation tests compare complete ParseError values/messages and fixed
record/failure/canonical-byte hashes. The populated target fixture compares all
nested fields against independent literal JSON.

## Important: complete the target projection matrix

In the inspected target/capability_projection_test.go:23–32, Activation and Rules
do not exercise nonnil-empty slices; Linknames does not exercise nil/empty
containers. Absent adapters on populated modules, multiple-element ordering and
mutation isolation are also missing. Existing upstream compatibility-policy
tests do not exercise this new target projection.

Extend the matrix, comparing complete public values and independent literal
JSON. Mutate source and projected nested slices and adapter/governance pointers
independently to demonstrate the full detached graph. This is a bounded evidence
gap, not a source-inspected mapping defect.

## Important: freeze governance canonical bytes and precedence

In the inspected internal/compatibilitypack/schema_timezone_test.go:8–15,
grammar and owner precedence are covered, but no independent complete canonical
pack/digest fixture is retained. Zero-valued governance and justification/time/
workload/platform/approval ordering are not covered. Existing successful decode,
rejection and identity-availability checks do not freeze these bytes/order.

Add a valid complete pack with literal canonical bytes/digest and original
ReviewedAt spelling. Simultaneous-invalid cases must assert the exact existing
message at each validation stage. Run preservation controls against captured
old source as well as the corrected source; new computed expectations alone
cannot establish preservation.

## Minor: use normal compound-statement formatting

target/capability_evaluation.go:373–399 compresses nested loops, allocation and
assignments onto single lines; inspect_capacity.go:32–36 and new tests follow
the same pattern. Expand compound statements and run the repository formatter.
The spec explicitly preserves readability and prohibits compressed formatting.

## Snapshot and evidence limits

The reviewer reported all 13 assigned repair-file hashes unchanged before/after.
Two full returned identities were:

```text
target/capability_evaluation.go
64f62707c55ce0841367497495c83f88ae0aee24ef789863ae60a88e09ab3de9
internal/compatibilitypack/schema_timezone_test.go
b059d85be4c205a849060ada1c77bf41d7613e7dcaeaff6a16730b7f26f82e08
```

Other full hashes were not returned and are not inferred. The assessment
compared relevant working-tree changes and old source at
0dd05b313acd0986312da7fd3159520e6a21f1bf, not an empty HEAD..HEAD range.
Source may change after this review; reconcile every finding before freeze.

Tier: session (jev-unavailable(no_key)), judged once for this assignment.
Requested gpt-6.1-sol/high, same family as the writer; actual execution metadata
unobservable. Reviewer performed reads/searches/hash comparisons only, no
tests, builds, package loading, generation, writes, Flow/Git mutations, bridges
or additional agents. Complete both evidence gaps before claiming full repair
preservation; native and formal acceptance remain separate.
