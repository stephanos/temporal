# Task 35 source checkpoint and outstanding acceptance

Corpus snapshot/entry readers now check two original cleanup attempts, and four
fixtures check their original Close at the same defer positions. Nil Close
preserves result and concrete primary error. A genuine cleanup failure returns
zero result with a raw sole error or primary-first join. Validation still
precedes publication; committed true,error after cleanupCases remains intact.

The [worker handover](handover.md), [source review](independent-source-review.md)
and [root reaudit](root-reaudit.json) bind the frozen candidate to BASE
`a80ad9b9d1a4195c4aeb2fe135557f71e6e6552a`. Complete original source/test
reconstruction, all 1,043 protected inputs, 39 worker freeze entries, 16 worker
receipts and seven reviewer receipts verify. Original operations, comments and
fixture assertions remain intact outside the admitted signatures/wrappers.
Four appended real-file controls retain error order, zero failed values,
snapshot/publication state and one nonempty literal BASE canonical snapshot.

Fresh independent serial checks pass corpus 24/24, focused 14/14, five actual
nested-root boundaries, errortype and formatting. Actual unfiltered configured
corpus lint falls from six diagnostics to zero, with none introduced. These
are stock Go1.27.1 developmental linux/arm64 results. Generator inputs are
unaffected; validate was not required for this change. The historical 419
whole-Gomad count remains historical; no current whole count is inferred.

The reviewer permits a source-progress checkpoint with no findings. Requested
writer/reviewer models are gpt-6.1-sol/high, the same requested family. Actual
executing-model metadata is unavailable. Formal impl-review and SHIP remain
open. Root commits source, tests, proof and Flow/docs before another writer.
The source-progress commit is the Git commit containing this checkpoint.

## Required open gates

- Genuine first-Close failure and simultaneous operation/Close failure execution
  remain unproved. The new cleanup branches have source evidence only. Normal
  real-file preservation tests do not close this gap.
- Original R3/R13/R18/R19, shared fn108 assessment/retention,
  task12/relevant predecessors/task21 and matched original first-baseline
  fixed identities remain required and open.
- Complete/full/completion/formal/affected-consumer and native darwin/arm64 plus
  linux/amd64 qualification remain required and open. Source checks fulfill
  none of those platform or full-scope requirements.

Task 35 and task 21 remain blocked; fn-109 remains open with 2/35 done.
Acceptance criteria and parent requirements are unchanged. The independent
source review does not authorize task completion or waive unavailable gates.

Product/document and all other scoped diff checks pass. Full staged diff-check
exits 2 only for the immutable audit-environment.log's final empty-GOFLAGS line.
The raw log, receipt and worker freeze retain their bytes; the archive warning
is disclosed rather than rewritten.

The root audit's first combined receipt pass failed on its own environment
bookkeeping after appending timing-only entries. A minimal reproduction and
corrected read-only pass are recorded in root-reaudit.json. Product source and
archived worker/reviewer evidence remained unchanged.
