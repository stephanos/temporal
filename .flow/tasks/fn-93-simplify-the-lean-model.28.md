---
satisfies: [R7]
---
# fn-93-simplify-the-lean-model.28 One JSON helper module and CanonicalJson.object writers (A5)

## Description
Lane A5, part two (fn-60's re-scoped successor). The surviving private `quote`/`array`/`sourceJson`/`quoteJson` copies (22 after lane B: `Model/Canonical` 3, `Property/Check` 3, `Scenario/Check` 3, `Query` 2, `Query/Check` 2, `Case/Projection.lean` 2, `Promotion` 2, `ImplementationLink/Language` 2, `Core` 1, `Artifact/Codecs` 1, `Temporal/Tool/Inspect` 1) become one module; hand-escaped writers that survive move to `CanonicalJson.object` (insertion order preserved). Lookalikes (`jsonString` in `ModelLint/ModuleIndex`, `Testpilot/Authoring`; `object` in `Testpilot/ProtoJSON`) join only if byte-identical.

**Size:** M
**Files:** the modules above; `model/Umpire/Json.lean` (home of the helper set, or a new `model/Umpire/Json/Helpers.lean`)
**Touches:** [model/Umpire/Json.lean, model/Umpire/Json/**, model/Umpire/Model/Canonical.lean, model/Umpire/Property/Check.lean, model/Umpire/Scenario/Check.lean, model/Umpire/Query.lean, model/Umpire/Query/Check.lean, model/Umpire/Case/Projection.lean, model/Umpire/Promotion.lean, model/Umpire/ImplementationLink/Language.lean, model/Umpire/Core.lean, model/Umpire/Artifact/Codecs.lean, model/Temporal/Tool/Inspect.lean]

### Approach
- Re-count copies first (`grep -rnE 'private def (quote|array|sourceJson|quoteJson)'`).
- Escaping must match byte for byte: compare each copy's escape table before merging; a copy with different escaping stays separate and is listed.
- Goldens are the oracle; add a `#guard` over a string with every escaped character class.

### Investigation targets
**Required:**
- `model/Umpire/Json.lean`
- `model/Umpire/Property/Check.lean` JSON writers
- `model/Umpire/Scenario/Check.lean:1-70`

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens umpire-check-regression
```

## Acceptance
- [ ] No private `quote`/`array`/`sourceJson` copy remains (or each kept one is listed with its escaping difference)
- [ ] Every golden, Case fixture and Fingerprint byte-identical


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
