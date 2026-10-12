---
satisfies: [R6]
---
# fn-169-a-temporal-free-framework-the-lifter.3 Held sweep: IR schema to an Umpire-owned namespace (umpire.ir.v1)

## Description
TBD


Also consider (2026-10-12, from the `tools/umpire` research): `script.proto` and `realization.proto` hold Temporal-specific oneof arms (Workflow/Activity/Nexus activations, workflow commands, activity settlements). With the namespace move, give them generic activation/settlement shapes with Temporal-specific arms or extensions, so the Go and Scala frameworks need no registered adapter for them. This is a proposal for the sweep; decide with the owner.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
