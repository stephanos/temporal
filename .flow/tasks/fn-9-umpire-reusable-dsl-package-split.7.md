---
satisfies: [R2, R3, R4, R6, R7]
---
# fn-9-umpire-reusable-dsl-package-split.7 Cut over builds docs and remove the old interface

## Description
Complete the clean replacement and make the new targets the only live build/documentation surface (R2-R7). This task owns deletion and final regression enforcement after every consumer has moved.

**Size:** M
**Files:** `model/Temporal/Experiment/**` (remove), `model/Umpire/Examples/testdata/switch-experiment-spec.json`, `model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json`, `model/lakefile.toml`, `Makefile`, `model/README.md`, `.plans/UMPIRE_DSL.md`
**Touches:** [model/Temporal/Experiment/**, model/Umpire/Examples/testdata/switch-experiment-spec.json, model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json, model/lakefile.toml, Makefile, model/README.md, .plans/UMPIRE_DSL.md]

### Approach
- Switch `make umpire-check-regression` to `UmpireTests`, `TemporalUmpireTests`, and `temporal-umpire-inspect`; change no model-local Makefile.
- Remove old test/executable targets, the complete old module tree, and the import-only DSL/Compiler/Json facades without aliases or re-exports.
- Update current model documentation and the design record's realized-code inventory to the final ownership/module names.
- Add final live-source checks: no old namespace/import in Lean, root Make recipe, or current model docs; no Umpire import of Temporal/Nexus. Exclude historical Flow/design prose from stale-name enforcement.
- Compare both final inspector outputs byte-for-byte with the checked-in target-state fixtures; retain literal suite assertions for identities, digests, format version, planner outcomes, and portable fields so deletion of old targets cannot erase the oracle.

### Investigation targets
**Required** (read before coding):
- `Makefile:999-1010` — stable regression recipe and deterministic checks
- `model/README.md:38-70` — semantic authoring, source-path, and inspector documentation
- `model/lakefile.toml:1-16` — old targets to remove after new targets exist
- `model/Temporal/Experiment/DSL.lean:1-4` — import-only facade to delete
- `model/Temporal/Experiment/Compiler.lean:1-2` — import-only facade to delete
- `.plans/UMPIRE_DSL.md:920-1063` — current inventory and accepted architecture

**Optional** (reference as needed):
- `model/Temporal/Experiment/Json.lean:1` — final import-only facade

### Key context
Do not erase historical records that explain the migration. The stale-interface check targets live code/build/user documentation only. Preserve unrelated generated-API and dynamic-config work sharing the root integration files. Confine investigation and implementation to the task-listed model/build/docs files; do not conduct repository-wide implementation searches.
## Acceptance
- [ ] No old module, namespace, import, alias, re-export, Lake target, or facade remains in live model/build surfaces.
- [ ] Root `Makefile` is the only Makefile changed and `make umpire-check-regression` builds/runs only the new targets.
- [ ] `model/README.md` documents the new ownership, paths, inspector name, and stable Make command.
- [ ] The design record's codebase inventory describes the realized layout while historical reasoning remains intact.
- [ ] Live-source scans reject old imports and reverse Umpire-to-Temporal/Nexus dependencies.
- [ ] Generic and Temporal suites plus both deterministic inspector scenarios and unknown-scenario failure pass.
- [ ] Final inspector output matches both checked-in target-state fixtures byte-for-byte, and retained literal assertions cover semantic identities, digests, format version, planner outcomes, and portable fields.
- [ ] Fixture derivation records exactly two source-path substitutions and no other pre/post migration delta.
- [ ] No Observation, `SemanticValue` redesign, procedural Drive DSL, out-of-allowlist implementation use, or model-local Makefile is introduced.
- [ ] Existing comments are preserved except for truthful path/namespace wording updates.
- [ ] `git diff --check` passes for the scoped change.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
