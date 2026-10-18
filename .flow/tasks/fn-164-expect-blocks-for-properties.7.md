---
satisfies: [R2, R4, R5, R6]
---
# fn-164-expect-blocks-for-properties.7 Migrate framework Property callers and predicate fixtures to expect

## Description
**Superseded by fn-190 (owner phasing 2026-10-17).** fn-190.4/.7 rewrite every framework Property caller and fixture to `check` blocks, so migrating them to `expect` first is not done; close this task when fn-190.7 and fn-190.9 land.

Framework-only source task. Touches model/framework/dsl, samples, export/testdata, associated generic export/reify tests and framework migration helpers only if needed. Inventory every executable framework Property caller, generic capability constructor, composition, outer claim/monitor, generated witness declaration, example and template. Freeze complete predecessor declaration/function/type inventories and source positions, then migrate callers to expect with exact contextual parameter names, Boolean association, eager lets and helper references. Keep independent ordinary-core twins using direct Property/PropertyDecl construction, explicitly inventoried as core oracle fixtures rather than a legacy authoring surface. Never read or restore model/temporal or depend on its migration. Use the committed fn188 framework-only loop and focused generic suites. Preserve negative fixtures and existing witness identities. Public lambda finishers remain only until this migrated framework inventory is complete, then .8 removes them. No Go/schema/regeneration or full Temporal R6 credit.
## Acceptance
Complete framework caller inventory and migration; exact full declaration/function/type IR parity with explicit source-position mappings; generic/runtime regressions pass via framework-only commands; intentionally invalid and direct-core proof fixtures are accounted for; no Temporal sources are consumed and no full-spec R6 completion is claimed.

## Done summary
Already superseded by fn-190; status hygiene 2026-10-18

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: