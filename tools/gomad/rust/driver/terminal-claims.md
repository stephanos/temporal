# Native terminal claims

Current standalone native execution snapshots the Store's attempted output counter
before returning its mutable `Outcome`. A compile or initial-memory admission
refusal has no Store and snapshots zero attempts. Cancellation changes terminal
classification while preserving physical counters and observations.

Current standalone publication emits `gomad.backend-outcome/v2` and binds the
canonical `gomad.native-output-accounting/v1` payload by SHA-256 and by a Record
stream descriptor. Publication and public artifact intake require matching
counters, attempted bytes within the admitted output limit, retained stdout plus
stderr no greater than attempted bytes, remaining fuel within its admitted limit,
and terminal classification and exit status matching the Record. Attempts against
closed descriptors and evidence-capacity failures can make attempted bytes greater
than retained bytes. Fixed host accounting is retained outside the guest evidence
budget; guest stream retention bounds do not increase.

Existing `gomad.backend-outcome/v1` payloads remain readable without canonical
Record or payload rewriting. Their terminal and fuel claims are checked, but they
have no independent output snapshot. Historical cancellation could reset the v1
output counter after retaining output, so v2's retained-output comparison does not
apply to v1. Synthetic and early-cancelled executions without a native snapshot
continue to use v1.

Observed replay's `backend-evidence.bin` retains its v1 format. The final retained
reader authenticates a v2 artifact and derives that v1 observation by removing
the output snapshot digest and restoring the v1 schema before comparing raw
observations. Original v1 artifact observations remain byte-for-byte comparisons.

This checks the trusted native producer's Store-to-Outcome publication boundary
and artifact consistency. It does not attest execution origin: an attacker able
to rewrite and reseal all artifact inputs can change both counters, or construct
a compatible v1 artifact. The former untrusted helper transport is retired; these
checks do not recreate its wire parser or claim its original test passed. Cluster
publication has its own aggregate outcome and is outside this v2 receiving scope.

The receiving controls live in `evidence::store::record_intake_tests`: coherently
resealed fuel/output claims, mutable native outcomes, staged publication,
legacy v1 exit/cancellation, closed descriptors, capacity, cancellation after
output, initial-memory admission refusal, and contradictory terminal claims.
Inventory migration remains gate-refusing until actual receiving tests and the
remaining lifecycle assertions are accepted by the qualification owner.
