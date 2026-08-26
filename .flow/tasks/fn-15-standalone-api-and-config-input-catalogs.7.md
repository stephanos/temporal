---
satisfies: [R6, R7, R8]
---
# fn-15-standalone-api-and-config-input-catalogs.7 Wire root Make commands and update input catalog documentation

## Description
Register the executable, expose root-only commands, publish the shared-core contract for fn-5, and reconcile model/roadmap documentation for R6-R8.

**Size:** M
**Files:** `model/lakefile.toml`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_DSL.md`
**Touches:** [model/lakefile.toml, Makefile, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_DSL.md]

### Approach
- Register `temporal-input-catalog` and ensure aggregate model tests include both adapters and CLI cases.
- Add root list/explain/combined-check targets with strict required/default variable handling.
- Document selector grammar, views, canonical JSON, complete API projection, selection policy, six-use scope, and generated-versus-handwritten authority.
- Update C1/C2 implementation status only after every focused/aggregate check passes.
- Preserve the declined no-drift/no-CI boundary. Keep fn-5's semantic adapter distinct and document/retain its Flow dependency on this landed core; actual semantic-adapter imports and consumption are implemented by fn-5, not this task.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:1-20` — executable/test registration.
- `Makefile:988-1032,1254` — root model command conventions.
- `model/README.md:3-66` and `model/ARCHITECTURE.md:105-139` — current generated/model ownership docs.
- `.plans/UMPIRE4_COMPONENTS.md:23-24,141-200,721-728` — C1/C2 status and deferred surface.
- `.plans/UMPIRE4_DSL.md:291-318` — artifact/component authority boundary.
- `.flow/memory/declined/generated-api-drift-verification.md` — binding no-drift/no-CI decision.

### Quick command
`cd model && lake build TemporalModelTests temporal-input-catalog && cd .. && make umpire-check-input-catalogs && make umpire-build-model`

## Acceptance
- [ ] Direct Lake and root list/explain/check commands produce the same bytes and status.
- [ ] Missing/invalid `CATALOG`, `VIEW`, or `SUBJECT` fails before querying with concise error output.
- [ ] Documentation gives copy-paste commands and explains complete generated facts, bounded selection, six-use overlay, and valid unclassified settings.
- [ ] The roadmap marks only tested implementation as built and retains drift verification/CI as absent.
- [ ] The reusable core contract is exported and tested for downstream use, and fn-5's spec/task dependency records that its distinct semantic adapter/executable will consume it; fn-15 does not implement or modify the semantic adapter.
- [ ] No CI file, regeneration/diff recipe, persisted JSON, fourth generated artifact, model-local Makefile, runtime/evidence code, or Umpire3 reference is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
