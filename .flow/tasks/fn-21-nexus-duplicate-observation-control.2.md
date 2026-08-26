---
satisfies: [R2, R7]
---
# fn-21-nexus-duplicate-observation-control.2 Bind the closed faulted runtime configuration and input set

## Description
Add the second exact model-owned participant program, RuntimeConfiguration, and admitted two-member input set for R2/R7. Consume Task `.7`'s already-checked mapping references and preserve the original normal program/configuration/fixture bytes.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/**`, `tools/umpire/runtime/request_test.go`
**Touches:** [model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/**, tools/umpire/runtime/request_test.go]

### Approach
- Extend fn-19's model-owned execution composition with one second closed program/configuration identity rather than widening the normal program.
- Bind exactly the Task `.1` fault-bearing ExperimentSpec, Task `.7` checked profile/program/mapping references and digests, existing local profile/protocol/budgets, one participant, exact target/action/occurrence, and cancellation capability.
- Extend preflight by a closed exact-match capability: the new pair requires one matching requested fault; the normal pair still requires none. Perform every check before the environment factory.
- Generate and strictly fn-18-admit the canonical two-member faulted input set; keep normal/faulted semantic, artifact, set, and fixture identities distinct.
- Mutate fault count/ID/occurrence, checked mapping/program/config crossing, profile/protocol/capabilities/budgets/seed/attempt and assert the spec mutation table's preflight status-1/no-IO result.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-21-nexus-duplicate-observation-control.7.md` — final checked evidence-profile/program/mapping references
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.5.md:13-31` — normal configuration/program fixture pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.2.md:13-30` — domain-neutral checked request/preflight seam
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:83-84` — phase/control attempt wire contract
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:134-180` — runtime/set admission and closure

### Acceptance
- [ ] The second input set is canonical, complete, immutable, and strictly admitted through fn-18 using Task `.7`'s checked mapping references.
- [ ] Only the exact one-fault ExperimentSpec/configuration/program/mapping closure produces a checked run request.
- [ ] Every crossing/drift mutation returns the exact preflight status-1/no-execution result.
- [ ] Existing normal configuration/program/input bytes remain identical and still reject every fault.
- [ ] No hard-coded future digest, new artifact family, authority material, arbitrary fault value, or reusable Temporal vocabulary is introduced.
## Acceptance
- [ ] R2 closed faulted binding and no-IO preflight contract are complete.
- [ ] R7 existing authority/format/user-surface boundaries remain intact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
