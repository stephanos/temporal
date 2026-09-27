---
satisfies: [R12, R13]
---
# fn-92-compose-entity-machines-into-one-system.6 Docs, GOV-02 drafts, AUTHORING section with multi-file drift markers

## Description
Draft the GOV-02 amendments the spec lists, add the AUTHORING section on the worker entity module and composition with markers in the new and touched modules, extend the drift test to a marker-to-file map, and update the model and plan docs.

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `.plans/UMPIRE4_DSL.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/index.json`, `model/AUTHORING.md`, `tools/umpire/authoring/authoring.go`, `tools/umpire/authoring/drift_test.go`, `model/Temporal/Feature/Worker/Model.lean` and `model/Temporal/Feature/Workflow/Outage/Model.lean` and `model/Temporal/Feature/Nexus/Control/Model.lean` (markers only), `model/Temporal/Feature/Nexus/DESIGN.md`, `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature.lean`, `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [.plans/**, model/AUTHORING.md, tools/umpire/authoring/**, model/Temporal/Feature/Worker/Model.lean, model/Temporal/Feature/Workflow/Outage/Model.lean, model/Temporal/Feature/Nexus/Control/Model.lean, model/Temporal/Feature/Nexus/DESIGN.md, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature.lean, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- GOV-02 drafts with the fn-92 marker: AUT-07a (`compose`, `restrict:`, `extend:`), glossary Machine, Refinement, Table, Party, new Composition entry, AUT-09 note, MOD-11 amendment and a new MOD rule for `feature-entity-uniqueness`; tag new dotted names `*(planned: fn-92-…)*` where needed for `tools/umpire/vocabulary/spec_names_test.go`.
- AUTHORING: new section quoting `Worker/Model.lean`, the Outage composition, and the Control `extend:` under `-- authoring:` markers (markers are lowercase letters, unique per file); `authoring.go`/`drift_test.go` gain a marker-to-file map replacing the single `modelPath`.
- DESIGN.md dated amendments under §2.1, §2.3, §2.5, §4, §6 decision 4, and the `system`-party wording in the reset row; module docs; README; ARCHITECTURE diagrams; ORDER entry marked delivered at close; index flowSpecs entry in sync.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md` glossary lines 262-330 and AUT-07a; MOD-11 and MOD-16 blocks as pattern
- `tools/umpire/authoring/authoring.go:18`, `drift_test.go:16`
- `model/Temporal/Feature/Nexus/DESIGN.md` §2, §4, §6

### Key context
- `.plans/UMPIRE4_DIRECTION.md` had uncommitted edits by another session on 2026-09-26; re-read before editing.

## Acceptance
- [ ] Every listed GOV-02 draft present with the fn-92 marker; MOD-15 spec-names test passes
- [ ] AUTHORING section added; drift test maps markers to files and passes
- [ ] DESIGN, README, ARCHITECTURE, Feature module docs updated; `make umpire-check-plan-index` passes
- [ ] ORDER and index entries reflect the delivered state

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
