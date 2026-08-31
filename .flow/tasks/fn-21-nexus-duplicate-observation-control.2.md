---
satisfies: [R2, R7]
---
# fn-21-nexus-duplicate-observation-control.2 Bind the closed faulted runtime configuration and input set

## Description
Add the second exact model-owned participant program, RuntimeConfiguration, and admitted two-member input set for R2/R7. Consume Task `.7`'s already-checked mapping references and preserve the original normal program/configuration/fixture bytes.

**Size:** M
**Files:** `model/Temporal/System/Execution/Nexus.lean`, `model/Temporal/System/Execution/NexusTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/**`, `tools/umpire/runtime/request_test.go`
**Touches:** [model/Temporal/System/Execution/Nexus.lean, model/Temporal/System/Execution/NexusTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/**, tools/umpire/runtime/request_test.go]

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
Bound the exact System-owned duplicate-delivery ExperimentSpec, participant program, checked observation tuple, RuntimeConfiguration, and canonical immutable input set. Strict preflight now accepts only that one-fault closure, rejects every enumerated fault/configuration drift before environment creation, and preserves the normal fixture bytes and no-fault behavior.

Baseline and final parent-spec Quick gates remain inherited red: the named Lean target is obsolete, macOS `/var` resolves through `/private/var`, and the local-run Make commands depend on later-task targets/output. Task-focused aggregate Lean builds, tagged Go runtime/Nexus tests, and `make lint-model` pass; `make lint-code` reports the inherited repository backlog of 1,377 findings with no introduced diff finding.

stage: impl-review - ran [2026-08-31T06:21:22Z..2026-08-31T06:29:05Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3135660356b639c01bd8cc90e1fd04c7dabeb2cd
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.CallerClosureFaultTests failed pre-edit: obsolete target absent), baseline: red (go test -count=1 ./tools/umpire/temporal/nexus/... failed pre-edit: macOS /var symlink containment), baseline: red (go test -count=1 ./tools/umpire/runevaluation/... failed pre-edit: /var vs /private/var path identity), baseline: red (make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=caller-closure-duplicate-delivery failed pre-edit: Make target absent), baseline: red (make umpire-check-local-run-evaluation SET=/tmp/umpire-local-runs/caller-closure-duplicate-delivery OUTPUT_ROOT=/tmp/umpire-local-results failed pre-edit: later-task input set absent), baseline: red (make umpire-check-regression failed pre-edit: macOS /var symlink containment), cd model && mise exec -- lake build Temporal.System.Execution.NexusTests Temporal.NexusExecutionIntegrationTests TemporalExperimentalTests TemporalModelTests, go test -tags test_dep -count=1 ./tools/umpire/runtime, go test -tags test_dep -count=1 ./tools/umpire/temporal/nexus -run 'Test(CheckRequestBindsTheExactCallerClosureProgram|CallerClosureProgramVersionMatchesTheSystemModel|CheckRequestRejectsAnUnsupportedSetBeforeExecution|CallerClosureInputSetIsStrictlyAdmitted|CallerClosureInputSetPassesLocalRuntimePreflight|CallerClosureInputSetRejectsNoncanonicalMemberBytes)$', make lint-model, inherited red: make lint-code (1377 repository findings; no introduced diff finding), verify inherited red: cd model && mise exec -- lake build Temporal.Feature.Nexus.CallerClosureFaultTests (obsolete target absent), verify inherited red: go test -count=1 ./tools/umpire/temporal/nexus/... (macOS /var symlink containment), verify inherited red: go test -count=1 ./tools/umpire/runevaluation/... (/var vs /private/var path identity), verify inherited red: make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=caller-closure-duplicate-delivery (Make target absent), verify inherited red: make umpire-check-local-run-evaluation SET=/tmp/umpire-local-runs/caller-closure-duplicate-delivery OUTPUT_ROOT=/tmp/umpire-local-results (later-task output absent), verify inherited red: make umpire-check-regression (macOS /var symlink containment)
- PRs:
