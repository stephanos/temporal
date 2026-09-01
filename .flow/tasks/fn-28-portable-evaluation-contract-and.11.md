---
satisfies: [R7, R8]
---

# fn-28-portable-evaluation-contract-and.11 Document portable canary decisions and deferred fleet boundaries
## Description

Synchronize the Umpire architecture, generated contract workflow, resident executor interface, disposable-cluster test, eventual Evidence closure, local decision semantics, and explicitly deferred whole-world/fleet work.

**Size:** M
**Files:** `tools/umpire/portableevaluation/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `docs/**`
**Touches:** [`tools/umpire/portableevaluation/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`, `docs/**`]

### Approach
- Explain that Lean compiles contracts ahead of time while the canary independently executes and evaluates one exact contract without Lean.
- Keep detailed statuses separate and document the conservative `pass`/`fail`/`inconclusive` mapping, explicit Evidence closure, and stable-vs-dynamic comparison fields.
- State that fleet scheduling, leases, persistence, crash recovery, production deployment, release eligibility, Claim Assessment, and whole-world claims remain deferred.

### Investigation targets

**Required** (read before coding):
- Parent spec and completed implementation behavior.
- Existing Umpire architecture, glossary, component map, roadmap, and Run Evaluation documentation.
- Exact test commands and generated fixture/drift workflow.

## Acceptance
- [ ] Documentation matches the shipped protobuf schema, compiler, executor interface, HTTP transport, tagged test, statuses, Limits, and failure behavior exactly.
- [ ] The roadmap no longer names the obsolete external staging blocker and accurately gates P3 on the completed self-hosted portability proof.
- [ ] Docs explicitly exclude whole-world and deferred production/fleet claims; documentation tests, links, plan index, and global Flow validation pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
