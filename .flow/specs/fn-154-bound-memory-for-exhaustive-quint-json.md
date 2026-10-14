# Bound memory for exhaustive Quint JSON agreement

## Goal & Context

Deferred by the owner on 2026-10-09 while closing fn-145. Record the heavyweight
whole-Model Quint JSON roundtrip and its concurrent memory-fit work as follow-up,
not as a passing gate. The schema and artifact equivalence proofs and compact
JSON compatibility checks remain required. This deferral does not waive the
separate Activity conformance assertions.

Quint produces ITF JSON. The faithful test currently synthesizes a complete
1,894,733,345-byte Activity dump in memory before comparison. A measured
141,982,619-byte section expanded into 2,387,566,128 bytes of generic decoding
allocations. The row-wise reader removes the full generic intermediate but
retains the whole input buffer. The latest concurrent run killed conformance
while the export process continued; four faithful fixtures subsequently passed.
Neither that partial result nor focused preservation establishes concurrent fit.

## Proposed approach

Generate and consume the JSON incrementally instead of materializing the entire
document. Keep JSON at the Quint protocol boundary and native Go tables or
existing protobuf data internally. Replacing the boundary test with protobuf
would bypass the reader it must verify. Assess streaming generation and reading
together, including the real RunQuint file path, before selecting an API change.
Temporary disk spooling is an option to assess, not a required design. Do not
check in multi-hundred-megabyte dumps.

## Acceptance Criteria

- **R1:** Preserve complete states, classes, results, claims, products, ordered
  receipts and replay witnesses, plus syntax, number, duplicate and refusal
  behavior at the actual JSON boundary.
- **R2:** Bound transient JSON memory through incremental generation and
  consumption, with an explicit bound and independently preserved comparison
  evidence. Do not substitute sampling or a protobuf-only roundtrip.
- **R3:** Verify the complete deferred gate under the required concurrent p2
  configuration and report actual resource evidence before restoring it as a
  required gate. Attribute Activity semantic failures separately.
- **R4:** Keep large generated dumps temporary and out of version control;
  document the streaming path, compatibility and any API tradeoffs.

## Scheduling

Deferred and not ready for execution. Do not activate this follow-up as part of
fn-145 or start another memory optimization now. Revisit the design and existing
evidence when the owner schedules it.

## Evidence

Existing worker evidence lives in /tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454/.
Use raw-reader-affected.*, raw-reader-style-affected.*, lint-raw-reader-style-readonly.*
and the terminal go-full-raw-reader-canonical.* result when available. These
paths are local evidence, not durable checked-in dump artifacts.
