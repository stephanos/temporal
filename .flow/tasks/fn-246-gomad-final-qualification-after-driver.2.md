---
satisfies: [R1, R3]
---
# fn-246-gomad-final-qualification-after-driver.2 Audit and unify every retained payload caller bound after crate extraction

Touches: extracted replay/qualify/stress and other retained-payload consumers, their boundary fixtures, .flow/artifacts/fn246-payload-bounds-*

## Description
Size M. Receives fn-232.32/R5/R14. Depends on .1 because retained reach ports change replay/preflight and inspection surfaces that this audit also owns. Re-anchor all extracted paths and caller bounds after .1 merges before editing; overlapping writes are serialized. Enumerate every opened.payload and descriptor.read caller with a literal bound across the complete extracted host, not just the old replay/qualify/stress paths. Anchor from old replay/cluster.rs:350, qualify/workloads/mod.rs:196/:276-278/:313-314, qualify/workloads/retry.rs:163-164 and stress/report.rs:247-248, then resolve current paths. For each literal derive admission from the authenticated descriptor, owning incarnation evidence limit or configured output limit, or document the actual fixed contract such as the512MiB module bound. Preserve pinned-handle, digest, identity, module and generic descriptor checks. This is the inherited caller-bound repair; fn-245's broad shared-read/type conversion is separate work. Inventory first and split through Flow before implementation if distinct surfaces exceed one coherent M change; do not narrow the audit. Quick verification runs a boundary/one-byte-over fixture per unified site plus existing replay/open refusals and unchanged identity goldens.

## Acceptance
- [ ] Inventory every fixed retained-payload caller bound in the extracted host and retain a derived/documented disposition with its authenticated owner.
- [ ] Every consumer of the same payload uses one admission contract; valid declared-limit payloads pass and one-byte-over inputs refuse with the existing classification/message.
- [ ] No unbounded reads, weakened descriptor/pinned-handle/identity/module guards, behavior-format changes or regenerated goldens.
- [ ] Focused caller-bound and replay/open regressions pass with exact commands/exits; source232.32/R5/R14 mapping is retained.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
