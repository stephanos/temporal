---
satisfies: [R16]
---
# fn-83-author-a-live-case-from-a-model-file.16 Derive the fixture name from the case name

## Description
Derive a Case's fixture name from its `case` name and remove the `fixture:` slot (R16). `case asyncNexus` names fixture `async-nexus`, so Case ID `temporal.case.async-nexus`, Program and Contract IDs, and the Run scope all follow from the declaration name like every other Definition ID in the file.

**Size:** S
**Files:** the Temporal `case` command (fixture derivation, no `fixture:` key), `model/Temporal/Case/Registry.lean` (duplicate-fixture message names both declarations), `model/Temporal/Feature/Nexus/Success/Model.lean` (`case asyncNexusSuccess` becomes `case asyncNexus`), `Tests.lean` (`case` occurrences and pins), `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md` and `.6.md` (their `case` names become `workerOutage` and `syncNexus`)
**Touches:** [model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/**, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md]

### Approach
- Derivation: lower kebab-case of the declaration's last name component, splitting before each uppercase letter (`asyncNexus` → `async-nexus`, `workerOutage` → `worker-outage`). Decide and pin the rule for digits and runs of capitals (for example `syncNexusV2`, `HTTPRetry`) with `#guard`; the result must be a valid fixture file stem and Definition ID segment, and a name that cannot produce one rejects at the `case` name.
- Renaming `asyncNexusSuccess` to `asyncNexus` keeps every byte of the async-Nexus fixture: the Case, Program and Contract IDs and the Run scope already use `async-nexus`, and the Lean declaration name appears only in the registry. Assert byte-identity with `make umpire-check-case-runtime-conformance`.
- Fixture names are global (one testdata directory) while `case` names are per namespace, so two features may declare the same `case` name. The existing duplicate-fixture rejection stays and names both declarations.
- `register_case … id … fixture …` for the typed examples keeps its explicit fixture; only the `case` command derives.
- A rename of a `case` renames the checked-in file and its IDs; the Go live test loads by fixture name, so the receipt notes that a rename touches the live test.

### Investigation targets
**Required:**
- the Temporal `case` command — `fixture` handling, `caseId` derivation, registry recording
- `model/Temporal/Case/Registry.lean` — `cases`, the duplicate-fixture diagnostic
- `model/Umpire/Case/Producer.lean` — `Identity` defaults
- `tests/testpilot_async_nexus_case_test.go` — `loadTestpilotCase(t, "async-nexus")`

### Key context
- Depends on .15 so it edits the respelled `case` grammar.

## Acceptance
- [ ] The `case` command has no fixture slot; the fixture name is the kebab-case of the `case` name under a rule pinned by `#guard` (including digits and capital runs)
- [ ] `Model.lean` declares `case asyncNexus`; the async-Nexus fixture is byte-identical
- [ ] A `case` name that yields an invalid fixture stem rejects at the name, and two `case` blocks yielding one fixture reject naming both declarations, each pinned by `#guard_msgs`
- [ ] `register_case` for the typed examples is unchanged; `umpire-case --list` output is unchanged
- [ ] `.5` and `.6` wording uses `case workerOutage` and `case syncNexus`
- [ ] `cd model && lake build`, `make umpire-check-case-runtime-conformance` pass


## Done summary
Blocked:
Blocked 2026-09-10 pending a redesign of the `case` abstraction (decided with the user, not yet a spec).

The per-Case `case` block (one Query, one hand-picked realization template, per-Case evidence lines) is being replaced by:

- **Sets per purpose.** A developer declares query sets by kind: functional and canary sets list Queries explicitly; exploratory sets state a coverage goal and a budget over a variation space.
- **One Case per Query.** A set compiles to many Cases run together; "Case = one Program + one Contract" stays.
- **A separate Temporal binding.** Runtime metadata (how an Action is caused, which recorded event confirms a step result, which resources a role needs) lives in a Temporal-owned binding declaration beside the behavioral Model, so Programs and Contracts are assembled from the Model plus its binding rather than from a whole-Program template chosen per Case.

This task builds on the `case` block, whole-Program templates, per-Case evidence, or the Case-registry shape that redesign replaces. Unblock or rewrite it once the redesign spec exists.
## Evidence
- Commits:
- Tests:
- PRs:
