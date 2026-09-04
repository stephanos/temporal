---
satisfies: [R1, R2, R3, R5]
---
# fn-60-deepen-authored-lean-canonical-json.1 Deepen the canonical JSON construction module

## Description
Extend the existing in-process `Umpire.Json` module into the single typed construction seam required by R1 before any domain migration starts. Keep the interface minimal and prove exact rendering directly, including the existing Core limit formatter.

**Size:** M
**Files:** `model/Umpire/Json.lean`, `model/Umpire/JsonTests.lean`, `model/Umpire/Core.lean`, `model/Umpire/CoreTests.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Json.lean, model/Umpire/JsonTests.lean, model/Umpire/Core.lean, model/Umpire/CoreTests.lean, model/UmpireTests.lean]

### Approach
- Inventory the generic value shapes used by every scoped formatter before changing `CanonicalJson`; add only the missing typed capability, including boolean values.
- Preserve ordered objects, caller-owned array order/duplicates, optional-as-null behavior, Lean JSON string escaping, and all existing compact/pretty/pretty-bytes/semantic-comparison interfaces.
- Preserve the public `canonicalLimitJson` name and type while moving its generic quoting and ordered-object punctuation through a private typed `CanonicalJson` projection; cover its Property, Query, Implementation Link, and Temporal callers.
- Keep domain validation, field selection, sorting, and schema meaning outside the module. Do not add parsing, schema validation, caching, or another traversal.
- Add direct interface tests and import them from the existing `UmpireTests` root. Include a ten-times-size construction/render probe that checks exact output shape without timing-sensitive thresholds.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Json.lean:8-40` — current typed value and compact rendering interface.
- `model/Umpire/Core.lean:509-515` — public limit JSON helper that must retain its interface.
- `model/Umpire/CoreTests.lean` — existing Core test surface for limit compatibility coverage.
- `model/Umpire/Artifact/Codecs.lean:26-114` — established typed-construction consumer pattern.
- `model/Umpire/Artifact/Tests/Codecs.lean:31-40` — existing escaping, option/null, and field-order probe.
- `model/UmpireTests.lean:1-37` — aggregate test import root.

**Optional** (reference as needed):
- `model/Umpire/Planning/Types.lean:151-156` — small typed optional-value consumer.

### Key context
Exact bytes, not merely parsed JSON equality, are the compatibility contract. Include control escapes and U+2028/U+2029. Existing comments and docstrings must be preserved and kept accurate.

### Quick commands
```bash
(cd model && mise exec -- lake build Umpire.JsonTests Umpire.CoreTests UmpireTests)
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```

## Acceptance
- [ ] The typed construction interface covers the required null, string, natural, boolean, array, ordered-object, and optional-as-null shapes without adding parsing, validation, sorting, caching, or another schema/format.
- [ ] `canonicalLimitJson` keeps its public name/type and exact bytes while its generic construction is owned by the typed seam; Property, Query, Implementation Link, and Temporal caller coverage passes.
- [ ] Direct tests pin empty/nested values, caller order and duplicates, naturals, booleans, null, control escaping, U+2028/U+2029, compact/pretty/newline behavior, and exact bytes rather than only semantic equality.
- [ ] A ten-times-size fixture renders through the same interface without an added semantic traversal or timing-sensitive assertion.
- [ ] Existing public formatter names/types, imports, trust dependencies, comments, and current consumers remain compatible.
- [ ] The focused build, regression gate, model lint, and repository lint commands pass or report only a verified inherited baseline.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
