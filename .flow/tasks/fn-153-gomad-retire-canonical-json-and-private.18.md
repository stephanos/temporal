---
satisfies: [R1, R2]
---
# fn-153-gomad-retire-canonical-json-and-private.18 Reconcile contracts and verify the integrated cleanup

## Description
Implements R1/R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** README.md, SPEC.md, ARCHITECTURE.md, CLI.md and bounded verification/review artifacts. Product corrections route back to their worker owner.
**Touches:** [tools/gomad3/README.md, tools/gomad3/SPEC.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/CLI.md]

### Approach

- Update ordinary JSON/strictness and shared file-publication contracts after fn152's storage docs, preserving semantic canonical vocabulary and wire/approval contracts.
- Audit the final caller/publisher inventory, public signatures, identity input coverage and all first-baseline pins. Measure removed/added production and test code separately.
- Run focused negatives, make lint-code-fast, nested lint-code, generated validation, both-source-set static architecture checks and one frozen supported-host gate per MILESTONES. Reuse exact same-revision overlapping results; keep existing RED obligations open under their owners.
- Coordinate one independent integrated source review; route actionable findings to the responsible workers and re-review fixes.
- Preserve the fn155 implementation priority, fn128/fn149 deferrals and no-CI/PR/push boundary. Report unavailable new acceptance explicitly; old native transfer does not automatically cover these tests.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/README.md:508`
- `tools/gomad3/SPEC.md:553`
- `tools/gomad3/ARCHITECTURE.md:424`
- `tools/gomad3/CLI.md:305`
- `tools/gomad3/Makefile:132`
- `tools/gomad3/architecture_test.go:225`

### Verification

Focused command: make lint-code-fast; make -C tools/gomad3 lint-code; make -C tools/gomad3 validate

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Current docs describe stdlib ordinary encoding, exact wire carve-outs, actual strict decode guarantees and shared staged publication without older-version support.
- [ ] Final requirements audit covers every surviving caller, all five private publishers, complete identity/projection and publication error controls, architecture fixtures and current generated inputs.
- [ ] Owned source gates and independent review pass on the integrated candidate; unavailable supported-host acceptance remains open and is never inferred from the planning probe.
- [ ] Scope/preservation report binds actual source/commands/results, remaining obligations and net code deltas; no deferred qualification or external action is silently revived.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
