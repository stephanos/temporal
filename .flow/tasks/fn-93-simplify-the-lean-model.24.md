---
satisfies: [R6]
---
# fn-93-simplify-the-lean-model.24 One registry_entry macro for the command and Case registries (A4)

## Description
Lane A4, part one. One `registry_entry` macro replaces the 13 extension-plus-accessor blocks in `Umpire/Command/Registry.lean` (~197-276, accessors ~278-401) and the 2 in `Temporal/Case/Registry.lean` (~32, 43); `ModelEntry` (~28) and `MachineEntry` (~127) merge.

**Size:** M
**Files:** `model/Umpire/Command/Registry.lean`, `model/Temporal/Case/Registry.lean`, a macro module (e.g. `model/Umpire/Command/RegistryEntry.lean`), readers of `ModelEntry`/`MachineEntry` in `model/Umpire/Command/Syntax.lean` and `model/Umpire/Command/Authoring.lean`
**Touches:** [model/Umpire/Command/Registry.lean, model/Umpire/Command/RegistryEntry.lean, model/Temporal/Case/Registry.lean, model/Umpire/Command/Syntax.lean, model/Umpire/Command/Authoring.lean, model/Temporal/Case/Syntax.lean]
**Depends on other specs:** fn-92.2 edits `Registry.lean`.

### Approach
- Model the macro on core's `register_label_attr` (expands to `initialize ext : … ← …`); take the extension ident from the caller so accessors can name it; keep each extension's persisted `name` unchanged.
- The macro module declares, other modules populate (an extension's `initialize` runs only for importers).
- Merging the entry types invalidates oleans; expect a full rebuild.

### Investigation targets
**Required:**
- `model/Umpire/Command/Registry.lean` (whole)
- `model/Temporal/Case/Registry.lean:25-50`
**Optional:**
- Lean core `src/Lean/LabelAttribute.lean:84` (`register_label_attr`)

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] Every registry extension is one macro use; `ModelEntry`/`MachineEntry` merged
- [ ] Every command test `#guard_msgs` unchanged; goldens and Case fixtures byte-identical


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
