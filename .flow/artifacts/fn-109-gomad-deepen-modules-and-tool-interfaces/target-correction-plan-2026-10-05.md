# Target lint corrective ownership

Developers can repair the remaining target lint under two bounded owners while
the original qualification contracts stay open. fn-109.38 owns five compatibility
aliases and three prepared-cache hash writes. fn-109.39 owns nine cleanup returns.
Task21 consumes both owners through direct dependencies. Source work runs serially;
task38's independently reviewed progress commit precedes task39's writer, without
requiring full task38 acceptance or task21 completion for source admission.

The actual retained target command at 8a8b57e8dc42202e8b5bb3dd974d6f306a913ea2
returned exit1 with 17 findings. HEAD 7f68a6e52d0083b6d83c3e442876ad218bf0cfba
changes authoring imports only and leaves the surveyed target inputs unchanged.
Each worker must freshly reproduce lint before edits rather than infer a count.

| Sites | Count | Owner |
| --- | --- | --- |
| target/capability.go, capability_collection.go, capability_evaluation.go, capability_golden_test.go, capability_review_test.go imports | 5 | fn-109.38 |
| target/prepared_cache.go hash writes at 245, 257, 264 | 3 | fn-109.38 |
| target/adapter_source_set.go deferred RemoveAll at 36 | 1 | fn-109.39 |
| target/target.go Close at 890, 913, 922, 926, 941, 945, 949 | 7 | fn-109.39 |
| target/target_test.go mutation fixture Close at 518 | 1 | fn-109.39 |

Paths above are relative to tools/gomad3. Target input SHA-256 values are
adapter_source_set.go b74e2cfeb046f9dcae21a32112657c26ff93d88240227f5504734bde6568d3ec,
prepared_cache.go 5f1fd3e5a47e25e154314dceae384ec3107ceb3785094c5a8b2568f96189388f,
target.go 2b5557b4f74bd63865914f4313949977a52e229913850f94ccce40fe0398ca3f,
target_test.go 6f6287653f0530023c37dd51f53b6ef3119cd3fa2f0b070eee3e8ef1a56891d3.
The pinned golangci v2.13.0 binary is /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0,
SHA-256 acacd04faf1d17489a890a4589ae36a62484c1076cf5fbcb7a33274cbc6a6bdc.
The unchanged .github/.golangci.yml hash is
2abff492b6a1aeaaded801bccc2d8ab85ed366608862b0b9a5dd5311ae0fed43.

SHORT Route A planning ran the repo/docs-gap and spec research on requested
Astra/high, mechanical memory inventory on Luna/low and gap decisions on
Sol6.1/high. Actual execution model metadata is not exposed. Memory used the
retained bm25 result after jev-unavailable(no_key), with no rerank retry.
The four relevant memories pin full source/build cache identity, module-sum
immutability, target build tags and profile-bound adapter packs.

Literal BASE/final vectors cover overlay sorting and replacement content,
absent/empty/present module files, argument/basename framing and read failures.
Cleanup controls retain real-file bytes/modes, validation/error precedence,
exclusive creation and partial destinations. Genuine first-Close, simultaneous,
post-open Chmod/Write/Sync and RemoveAll faults remain unexecuted unless a lawful
reproducer supplies actual proof. No public seam or general filesystem framework
is admitted. Newly checked cleanup failures are disclosed behavior, with exact
nil-cleanup primary identity retained.

No new forward or reverse spec dependency is needed. Native Linux remains with
fn128. Original R18/R19, predecessor, static both-source-set, matched fixed-identity,
full/default/functional/affected-consumer/formal/native Darwin requirements remain
required. Existing task states, immutable evidence and prior acceptance are unchanged.
The plan adds no product/API migration and requires only the owned Flow pointers
and MILESTONES rows. A plan-review verdict qualifies this decomposition, not the
implementation, native runtime or source-spec completion.

The fresh Codex backend review returned SHIP on gpt-6.1-sol/high. Its executed
command and receipt establish that model; author and reviewer share the Codex
model family. The linked target-correction-plan-review-2026-10-05.json retains
the complete result. Two P2 findings are advisory. Overlapping owners require
the already declared serial source admission. Task9's R10 command inventory
must include AdapterPreparedSourceSetSHA256, which currently runs go list with
unbounded buffers outside the shared seam. Task39 owns cleanup only and cannot
establish R10 command/output guarantees. This pre-existing verification gap
remains open under task9; the plan verdict supplies no implementation pass.
