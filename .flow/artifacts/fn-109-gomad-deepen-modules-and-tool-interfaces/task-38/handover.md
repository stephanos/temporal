# Task 38 source progress

Target capability imports now name the existing `compatibility` package explicitly.
Three prepared-cache records use infallible SHA-256 Write with
`fmt.Appendf(nil, unchanged format, unchanged arguments)`. Six new digest tests
retain literal BASE hashes, input immutability and existing read-error wrapping.
The source diff changes exactly five import lines and three hash-write lines;
all other original statements, comments and assertions stay intact.

Status remains `in_progress`. This handover covers SOURCE_PROGRESS_ONLY.
Root owns independent review, Git/index/commit, Flow lifecycle and admission of
the next source writer. The worker ran no review, commit, bridge or lifecycle
mutation. No executed-model metadata was exposed.

Tier: session (jev-unavailable(no_key));explicitAGENTSpinSol retained
stage: impl-review - skipped(policy: conductor-deferred - root owns independent source review)

## Verification

`baseline: red` applies to the actual unfiltered pinned target lint before any
production edit. It reported 17 issues, comprising 12 errcheck and five goimports.
The original focused canonical/projection controls and architecture check passed.
After adding the literal controls, the unchanged production source passed all
10 selected top-level tests, including six new digest tests. The tests preserve
existing behavior; they are not claimed as a failing behavioral reproduction.

The first prescribed `Write([]byte(fmt.Sprintf(...)))` spelling removed the
eight original findings but introduced three staticcheck QF1012 diagnostics.
The actual failure stays in [final-lint.log](final-lint.log). Root approved the
three-statement adjustment to `fmt.Appendf(nil, ...)`, which formats each record
independently, adds no helper or complete-stream accumulator, and cannot add a
new hash-write failure. No suppression, configuration, grant or gate changed.

| Final frozen-source command | Exit | Observation | Receipt |
| --- | --- | --- | --- |
| `go test -v -count=1 -tags test_dep ./target -run 'TestPreparedCacheDigest\|TestCapabilityReviewGoldenCanonicalBytes\|TestCompatibilityPackProjectionPreserves'` | 0 | 10 top-level tests and 20 leaf cases pass; 0.104 package seconds | [controls](final-appendf-controls.receipt.txt) |
| `go test -v -count=1 -tags test_dep . -run '^TestPackageArchitecture$'` | 0 | One architecture test executes and passes; 0.663 package seconds | [architecture](final-appendf-architecture.receipt.txt) |
| `/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target` | 1 | Actual unfiltered result has exactly nine unchanged task39 errcheck findings; all eight assigned findings resolve and none are introduced | [lint](final-appendf-lint.receipt.txt) |
| `go vet -tags test_dep -vettool=/tmp/fn109-lint-tools.ZdNe1t50/errortype ./target` | 0 | No errortype diagnostics | [errortype](final-appendf-errortype.receipt.txt) |
| `gofmt -l` over all seven Touches files, with an empty-output check | 0 | No formatting differences | [formatting](final-appendf-format.receipt.txt) |
| `git diff --check` | 0 | Source/document diff has no whitespace errors | Worker read-only check |

The raw [baseline lint](baseline-lint.log), [unchanged-production literal
controls](baseline-digest-controls.log), [final controls](final-appendf-controls.log)
and [final unfiltered lint](final-appendf-lint.log) retain every diagnostic/result.
[Literal vectors](literal-vectors.md) record independent input streams and hashes.
[Source bindings](source-bindings.md) map every before/after receipt to its unique
manifest, final source hashes and pinned tool/configuration identities. All
commands observed equal source manifests before and after execution. The 1,032
protected entries outside the six changed existing files remain identical;
the only new source is the assigned digest test. Duplicate manifests were moved
to ignored `.flow/tmp/task38-duplicate-manifests/`, where they remain recoverable.

The six new tests cover empty overlay, multiple replacements, cleaned/sorted
original keys, JSON map-order independence, changing temporary replacement
locations, replacement read failure, module files present/absent/empty,
argument order, basename framing, empty file lists and nonregular module input.
The canonical and projection controls preserve ordered complete evidence,
nil/empty storage and detachment. Both successful digest helpers retain input
bytes; absent go.sum stays absent. Replacement errors retain direct PathError
unwrapping and ErrNotExist classification; nonregular module errors retain the
original wrapper and cause text.

Makefile VERSION_INPUTS, BOUNDARY_INPUTS and COMPATIBILITY_INPUTS, protocol
generation's explicit livecap inputs and the version descriptor were inspected.
No Touches file is a generator/schema/template/runtime-overlay input, and no
generated output changed. Generation and `make validate` were not required for
these alias and private host-cache statements.

## Acceptance still open

Actual lint remains red for the nine task39 cleanup obligations. Root's fresh
independent source review and source-progress commit are pending. Every original
R18/R19, task21/predecessor, matched first-baseline, full/default/functional,
affected-consumer/formal and native Darwin gate remains required and open where
unproved. No known unchanged Linux/arm64 patched-runtime failure or unavailable
Darwin/full gate was rerun. Native Linux execution remains owned by fn128.
These developmental stock Go1.27.1 linux/arm64 controls establish only the
source observations listed above.

Defect route:
- prior fixes: committed target history and relevant bug memories checked; root serialized admission after predecessor integration; PR/tracker/other-branch search not done under this bounded conductor-owned source dispatch.
- diagnosis: actual BASE lint confirms five missing explicit compatibility aliases and three unchecked fmt.Fprintf returns; the first attempted spelling caused QF1012 and was replaced by fmt.Appendf; actual final lint confirms those eight findings removed with nine unchanged cleanup residuals.
- introduced by: not bisected; no known lint-green target revision was supplied.
- base: unfiltered lint exit1 with 17 issues at 9f66d05bc8f7f49f18135a53525006198a675036; literal controls pass on unchanged production. Head: uncommitted final source lint exit1 with nine unchanged residuals; controls pass on the final manifest bound by source-bindings.md.
- live: no live app surface; these are private host-cache helpers and import aliases.
