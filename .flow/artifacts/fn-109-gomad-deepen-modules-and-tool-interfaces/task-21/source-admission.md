# Task 21 evidence/source admission

The conductor admits task 21's verification and evidence work under
MILESTONES immediate-delivery item 4. The existing `gomad` branch is retained;
the source checkpoint is `8604c07def0f97b63cbca3864b4c286d6803c4b1`.
This admission overrides the task-20 dependency for source progress only.
Task 20 and transferred D5 acceptance remain blocked. No required native gate,
task completion, formal task-19 verdict or source qualification is waived.

Task 20's implementation is committed in `2e96c6e17927985f9f72e79c91014d0d32f48850`
and its paired metadata in `7cf8855c5e12280b4ff132e96e43fca9ac6b58c7`.
Commit `8604c07def0f97b63cbca3864b4c286d6803c4b1` retains its actual three-draw
SHIP receipt and open acceptance. Before admission the conductor freshly ran
`task-20/post-review-status-check.py`: receipt and metadata match their original
bytes, all three draws report SHIP with no findings, and current guide/evidence
hashes, links and fences verify. All 973 non-guide entries of task 19's
`round4-final-source.sha256` still match. The tracked tree and index were clean.

The conductor read the reconstruction, bound baseline report, independent
checkpoint source review and conductor verification. A fresh
`bound-baseline-measurement/verify_bound_evidence.py` run exited 0: complete
675-file scratch inventories, original 670 files, historical 337 artifacts and
675 scratch files, 98 successful commands and 171 completion output hashes
verify. A subsequent quiet check of all 201 `handoff-output.sha256` entries
exited 0. Reconstruction inputs verify except exactly the already disclosed
task-description metadata path; historical manifests remain unchanged.

Task 21 implements no production change. Its owner prepares the matched
current-tree 10/100 controls, complete F1–F11/S1–S5 matrix, R18 preservation
audit and exact per-platform qualification command/status ledger. Preserve the
baseline's fourteen explicit environment bindings and fixture semantics;
declare minimal current private-seam adaptations. If additional compiler
controls are needed, establish them on both sides before comparison. Keep
profiles and binaries local with retained hashes. Any production gap returns
to its implementation owner rather than being fixed under task 21.

The actual host is Linux aarch64 and the patched Go executable is absent.
It qualifies neither native platform. Root owns Flow lifecycle, independent
review, precise staging and progress commits; no push, worktree, stash or
history rewrite is authorized. The worker may delegate independent read-only
investigations, with one evidence writer in this checkout.
