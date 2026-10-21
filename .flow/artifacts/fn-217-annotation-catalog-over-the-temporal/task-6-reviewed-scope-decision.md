# Review of the first complete reverse-probe findings

Reviewed by the root conductor on 2026-10-21 against committed source
`edd4829162610c9d840dd50e4cf5ef56b746208c`, the complete actual live run,
and independent nested-context and opaque-context investigations. This records
the decision and its conditions, not task completion or final live success.

The immutable original report is `fn2176-original-live-gap-report.txt`, SHA-256
`3b1f9b6860115a87aef24d8ff760d24394e26ca3b36209018a612faf493f94a3`.
Root verified it byte-identical by SHA-256 to the conventional report before any
subsequent run. It contains 249 Unannotated rows, all actual rejection code 3.
All 63 pre-existing strict claims confirmed. No claim is waived by this decision.

## Missing representable rules: explain, do not disposition

Thirty-two observations require sixteen real declarations: namespace maximum
byte length on all eight requests, namespace presence on five tokenless requests,
and token presence on three worker replies. Namespace remains Text, worker tokens
remain Secret, and worker namespace omission remains valid. Native RED/GREEN
controls must include the 1000/1001-byte boundary and opaque nonempty-token
preservation; a complete actual live rerun must verify their expected errors.

## Reviewed scope limitations

The task explicitly preserves seven Start ignored-field explanations, opaque
Secret handling, and the five-operator presence algebra. Do not pretend these
already express arbitrary callback/link, serialized-token, or value-ordering
validation. Do not invent a maximum token length or a counter-valued proxy merely
to fit observed errors.

- The 203 observed Start cases under completion_callbacks, links, user_metadata,
  search_attributes and priority retain their existing explicit Ignored scope.
  The two worker-version build-ID cases are also under an explicit Ignored field.
  Approve only those exact 205 reviewed identities with dated, group-specific
  scope reasons. New fields and mutations do not inherit acceptance.
- Eight observed malformed-token cases concern opaque decoding/binding, not a
  proven byte-length threshold. Their explicit disposition must state that
  limitation and retain Secret privacy and required-token checks.
- Two normal_name cases require a real NORMAL-queue empty/one-character pair
  before disposition as an unsupported value-dependent queue relationship.
- Two retry initial-interval cases require real initial1001/max10 rejection and
  initial1001/max2002 success controls, plus attempts1/max0 bypass checks, before
  disposition as unsupported cross-field ordering. No initial upper bound is
  inferred.

Nested raw mutation paths are provenance, not proof of the rejecting constraint.
Source inspection confirms creation of default missing ancestors/oneof variants;
the live report also contains value-dependent distinctions within those contexts.
Do not dismiss every finding as a construction artifact, suppress nested fields,
or claim live-valid parent context for every synthesized descendant.

## Admission of the reviewed ledger

Use a fixed exact-identity ledger dated 2026-10-21, never an outcome-driven
automatic acceptance list or a field-prefix waiver. The new entries apply only
to their reviewed Unannotated rejection class and code 3. The `Accepted` and
`Mismatch` outcome variants, different-code unexplained rejections and new-identity
unexplained rejections must still fail. An actual success with no violated catalog
claim remains `Confirmed` under the independent generic classifier, receives no
disposition, and changes the golden; this ledger does not create a new Rejects
claim for an unsupported constraint. Retain the existing
general dated Accepted/Mismatch disposition behavior separately; do not replace
that positive control to force the new scoped policy globally.

Keep Unannotated rows, dates and reasons visible in the report. Verify unknown,
duplicate and malformed entries; preserve the complete mutation inventory,
30-second operation bounds, privacy, unchanged frame caps, all Model/Realization
semantics, and the original report. Review the final exact ledger/report diff
before updating the golden and rerunning the actual complete corpus.

## Paired-control evidence

The worker's session 31149 completed the real
`TestTestpilotModelGapScopeBoundaries` test with exit 0. Root inspected
`fn2176-scope-boundaries-live.log`: all nine subtests passed (0.235 s package time).
They distinguish inverted/ordered retry intervals, attempts1 bypass and max0
defaults; NORMAL queue empty/one-character normal_name; and empty/malformed/issued
worker tokens with omitted namespace. Thus the conditions above for the four
retry/queue relational identities are met. This does not establish an opaque
token size limit, validate every ignored descendant, or replace the complete
post-change live run.

## Exact ledger review

Root read the complete 432-line ledger and its twelve group-specific reasons.
A source-only diagnostic comparison (including concatenated literal suffixes)
against the immutable original report found exactly 217 entries, 217 unique IDs,
zero missing and zero extra IDs relative to the original 249 minus the 32
namespace/token-presence cases. This is a static identity check, not compiled or
live-result evidence. The production report supplies the pre-existing general
disposition list separately from the new known-Unannotated list; native controls
retain the general Accepted positive case and check code/outcome/identity binding.
Root approves that fixed ledger source. The final report diff and golden update
remain separate review steps after actual execution.
