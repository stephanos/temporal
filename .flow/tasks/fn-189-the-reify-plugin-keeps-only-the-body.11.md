---
satisfies: [R10]
---
# fn-189-the-reify-plugin-keeps-only-the-body.11 After .8 and fn-174.8: remove the old lifter tables, bindings and seams from reify/body (R10)

## Description
**Recut 2026-10-18 (one-language series):** `framework/export` is `framework/lift` after fn-201; the `export/project.scala` check names the new module.

Touches: model/framework/reify/body/BodyContext.scala, BodyEngine.scala, BodySyntax.scala, Expansion.scala or its successor

### Description
After fn-174.8 retires `framework/lift` as a reader, `reify/body` has one host, the plugin. Remove what exists only for the old lifter: the declaration-level tables the plugin never fills (`actions`, `inputTokens`, `inputDefaults`, `machines`, `machineStates`, `channels`, `monitors`, `assumptions`, `holes`, `compositions`, `properties`, `scenarios`, `queries`, `progress`, `realizations`, `capabilityKinds`, `capabilitySections`, `capabilityWaivers`, `capabilityWaiverGaps`), the `generatedName`/`generating`/`takeGenerated` plumbing, the declaration-fold bindings (`boundFunctions`, `boundTypes`, `boundValues`, `readingDeclaration`, `declarationCallback`, `functionArgument`, `suppliedAt`) where the plugin's lifter no longer binds a declaring function's parameters, and the override seams the plugin supplies with constant answers (`storedFunction`, `storedSectionFunction`, `storedStatus`, `contextualPhaseRead`, `contextualPhaseWrite`, `expansionAllowed`, `machineNamed`). Keep `functions`, `types`, `channelFields`, `opaqueRanges`, `renamed`, `lifting` and whatever fn-190.9's var lifting adds. `export/project.scala` stops compiling `../reify/body`.

Cross-spec prerequisite, stated here because Flow deps are spec-local: fn-174.8 (the old reader retired). Also after .8 of this spec, which is sequenced on the same prerequisite.

**Size:** M

### Verification
- The plugin builds and the six samples lift to the same IR; `reify/body` has no member that only `export` read (checked by removing the `using file ../reify/body` directive from `export/project.scala` and compiling).
- The counts receipt (.7) is re-recorded with the body-lifter line count after this task.
## Acceptance
- [ ] `reify/body` declares no declaration-level IR tables, no declaration-fold bindings and no old-lifter override seams.
- [ ] `framework/export` does not compile `reify/body`.
- [ ] The six samples lift to the same IR; the body-lifter line count is recorded.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: