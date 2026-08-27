---
satisfies: [R1]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.2 Hard-cut core Definition and Model Trace vocabulary

## Description
Apply R1's foundational source rename across Core and every compiling Umpire/Temporal call site. This task owns names for definitions, source locations, model values, and model traces; later tasks own fingerprints, limits, Observation, and wire fields.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/**/*.lean`, `model/Temporal/**/*.lean`, aggregate import/test roots
**Touches:** [model/Umpire/**/*.lean, model/Temporal/**/*.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/TemporalExperimentalTests.lean]

### Approach
- Rename the `Declaration*` family to the `Definition*` family, including fields, errors, diagnostics, helper namespaces, and tests.
- Rename `SemanticSource` to `SourceLocation`, `SemanticValue` to `ModelValue`, and the trace/step family to `ModelTrace` and `ModelTraceStep`.
- Update imports, declarations, examples, fixtures expressed in Lean, proof signatures, diagnostics, and comments in one compiling change.
- Choose context-specific field names such as `definitionId` instead of retaining generic `identity` when the value is a Definition ID.
- Preserve existing comments and their intent while replacing obsolete wording.
- Add no type synonyms, deprecated names, forwarding modules, or compatibility notation.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:8-230` — definitions and pure model data at the center of the rename.
- `model/Umpire/Target/Language.lean` — largest checked-model consumer.
- `model/Umpire/Examples/Switch.lean:1-220` — domain-neutral teaching example.
- `model/Temporal/Feature/Nexus/Operations.lean:1-80` — representative Temporal model authoring.
- `model/UmpireTests.lean` — aggregate Umpire test root.

### Key context
Do not globally replace the word identity. Artifact identity and ordering fields have different meanings and are handled by Tasks `.3` and `.5`. The compilation sweep must follow types rather than spelling.

## Acceptance
- [ ] Umpire and Temporal model libraries compile using Definition, Source Location, Model Value, and Model Trace source names.
- [ ] Existing comments remain present with human-readable updated wording.
- [ ] Focused tests cover Definition ID validation and Model Trace behavior under the new names.
- [ ] Old public names and imports do not resolve.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
