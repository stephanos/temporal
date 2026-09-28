---
satisfies: [R12, R13]
---
# fn-92-compose-entity-machines-into-one-system.6 Docs, GOV-02 drafts, AUTHORING section with multi-file drift markers

## Description
Draft the GOV-02 amendments the spec lists, add the AUTHORING section on the worker entity module and composition with markers in the new and touched modules, extend the drift test to a marker-to-file map, and update the model and plan docs (R12, R13).

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_SPEC_MODEL_ARCH.md`, `.plans/UMPIRE4_SPEC_COMPS.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/index.json`, `model/AUTHORING.md`, `tools/umpire/authoring/authoring.go`, `tools/umpire/authoring/drift_test.go`, `model/Temporal/Feature/Worker/Model.lean` and `model/Temporal/Feature/Workflow/Outage/Model.lean` and `model/Temporal/Feature/Nexus/Control/Model.lean` (markers only), `model/Temporal/Feature/Nexus/DESIGN.md`, `model/Temporal/Feature/Nexus.lean`, `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [.plans/**, model/AUTHORING.md, tools/umpire/authoring/**, model/Temporal/Feature/Worker/Model.lean, model/Temporal/Feature/Workflow/Outage/Model.lean, model/Temporal/Feature/Nexus/Control/Model.lean, model/Temporal/Feature/Nexus/DESIGN.md, model/Temporal/Feature/Nexus.lean, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- GOV-02 drafts in the two forms fn-88 uses (`UMPIRE4_SPEC.md:222, 257, 405`): AUT-07a (`:395-419`) amendment for `compose`, `restrict:`, `extend:`; glossary Machine (`:294-298`), Party (`:303-306`), Refinement (`:307-311`), Table (`:312-313`) amendments and a new Composition entry; AUT-09 (`:437-453`) note; MOD-11 (`:218-224`) amendment and a new MOD-18 after MOD-17 (`:257-263`) for `feature-entity-uniqueness`; `UMPIRE4_SPEC_MODEL_ARCH.md:99-110` amended so the import-graph phase is no longer "the single enforcement mechanism"; `UMPIRE4_SPEC_COMPS.md` §7.2 (`:388-411`) notes the entity-module shape. `UMPIRE4_DSL.md` never describes the `machine` command and needs no edit. MOD-15: names this spec's earlier tasks created resolve on their own; tag only a dotted name that does not yet exist with `*(planned: fn-92-compose-entity-machines-into-one-system)*` (`tools/umpire/internal/leannames/spec.go:30`), and leave none behind at close (`spec.go:109` fails a tag on a closed spec).
- AUTHORING: new section quoting `Worker/Model.lean`, the Outage composition, and the Control `extend:` under `-- authoring:` markers (lowercase letters, unique per file, and unique across the whole walkthrough because `Blocks` rejects a duplicate name); `authoring.go`/`drift_test.go` gain a marker-to-file map replacing the single `modelPath` (`drift_test.go:16`), each file's `end` skipped per file, planted-fault tests updated.
- DESIGN.md dated amendments (`> Amended during fn-92 …` as `:50-78` show) under §2.1 (`:50`), §2.3 (`:147`, the guardless `+ workerStop` row at 186-188), §2.5 (`:215`), §4 (`:738`, reset row 752 keeps `system`), §6 decision 4 (`:791-793`); `Nexus.lean` docstring item 3 (Control adds one row) rewritten for `extend:` over Pair; `model/README.md:104-116, 195-220, 490`, `model/ARCHITECTURE.md:46-53, 104-105`, `model/Umpire/ARCHITECTURE.md:43-50, 412-431` gain `compose`, the keys, the Worker module, and the declaration-level lint; the ORDER entry (`UMPIRE4_ORDER.md:34-41`) loses its claim that Workflow and Worker modules are shared by Start and Outage and moves to the delivered list at close; the index `flowSpecs` entry stays in sync.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_SPEC.md:218-263, 292-313, 395-453`
- `tools/umpire/authoring/authoring.go:18-24`, `drift_test.go`; `tools/umpire/internal/leannames/spec.go:30, 109`
- `model/Temporal/Feature/Nexus/DESIGN.md` §2, §4, §6; `.plans/UMPIRE4_SPEC_MODEL_ARCH.md:99-110`

### Key context
- fn-88.7 edits the same model docs and the ORDER and index files; this task lands after fn-88 closes and rebases on it. fn-93 later collapses `model/` Markdown, so these edits are short-lived but `make umpire-check-plan-index` needs them now.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/authoring/... ./tools/umpire/vocabulary/...
make umpire-check-plan-index
```
## Acceptance
- [ ] Every listed GOV-02 draft present in the fn-88 marker forms; MOD-15 spec-names test passes with no `(planned: fn-92-…)` tag left on a name that resolves
- [ ] AUTHORING section added; drift test maps markers to files, block names unique across the walkthrough, and passes
- [ ] DESIGN, README, ARCHITECTURE, Feature module docs updated; `make umpire-check-plan-index` passes
- [ ] ORDER entry reworded to the spec's boundaries and marked delivered; index entry reflects the delivered state
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
