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
Blocked 2026-09-10; superseded by fn-85 ("Model side effects as typed interfaces and run query sets").

The per-Case `case` block (one Query, one hand-picked whole-Program template, per-Case evidence lines) is replaced by:

- **Side effects in the Model.** Entities with structured state, and interfaces with a kind (`call`, `command`, `reply`, `observation`), a party, input classes with representatives, and result classes. Request fields that decide the outcome are Model behavior, not binding detail.
- **A Temporal Realization** that binds interfaces, result classes, observations, setup parameters and parties to RPCs, workflow commands, handler replies, history events and dynamic config. The Producer assembles Program and Contract from the witness; whole-Program templates and the `case` command are removed.
- **Query sets per purpose.** A set binds each party to test or environment; a functional set compiles to one Case per Query; canary and exploratory sets are admitted for fn-70/fn-29 and fn-33.

fn-85's final task closes this task as superseded and names where its concern went. Design record: `model/Temporal/Feature/Nexus/DESIGN.md`.
## Evidence
- Commits:
- Tests:
- PRs:
