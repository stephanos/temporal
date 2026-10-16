# Framework-only source iteration

> HTML render lens (local): open `.flow/artifacts/fn-188-framework-only-source-iteration/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## DSL-first execution hold (owner override)

The native implementation checkpoints are committed: fn-164.2/.11/.10, fn-165.7-.9, fn-173.10-.17, fn-123.9/.10 and fn-153.7. Task .3 is complete with formal native-batch review SHIP. The framework DSL remains priority 0: any demonstrated unmet author-facing DSL requirement is scheduled before build/export, IR, schema, Go or backend work. Do not invent new DSL features or reopen completed checkpoints to fill the queue. Existing final-source native evidence is retained; only missing, stale or invalidated checks need rerunning.

Task .2 remains incomplete and owner-held, with its .3 dependency satisfied. Completion of .3 does not automatically release .2 or any IR, capture/exporter, schema, Go or backend work: that later lane requires an explicit scheduling decision after DSL requirement triage. Native work uses isolated native compiler/runtime and relevant lint checks; do not spend it repairing exporter/IR drivers. Before admitting .2, reconcile its suite accounting and no-Go execution boundary: the current broad runner includes a real Go build test and omits native sample suites, so it is not a DSL-only gate. Preserve those checks and all failed/interrupted evidence without pass credit. Temporal sources remain disabled and excluded entirely, including their contents, and restoring that scope requires owner authority.

The owner changed the active scope to the framework only and explicitly requested renaming the Scala sources under `model/temporal` so they cannot be compiled accidentally.

Native Duration follow-up fn-147.5 is complete (SHIP): 22 focused construction tests and 29 lint tests pass, with runtime source unchanged. It activates neither export nor checked field admission. An audit of all 41 open specs and remaining milestone sections demonstrated no additional currently authorized native DSL requirement. The next decision is framework-only build/artifact/suite-boundary admission, not a new native feature; .2 and other integration tasks remain held until that deliberate re-scoping/admission is recorded.

## Acceptance

- **R1:** Rename every tracked `model/temporal/**/*.scala` source to the same path with `.disabled` appended, preserving all file bytes. Leave generated caches and historical payloads unchanged.
- **R2:** Record the framework-only scope in MILESTONES.md. Temporal implementation, migration, lifting, regeneration and verification stay held until the owner restores that scope.
- **R3:** Preserve the unfinished fn-174.2 work and its evidence. Do not claim its mini/Temporal gate or closure passed.
- **R4:** Provide a repeatable framework-only compile/test entrypoint using the existing framework modules and tool runner. It must not read or compile `model/temporal`, use `model/project.scala`, or depend on a cached Temporal Model jar or API jar. Build the generic framework and compiler plugin from current source with the existing IR jar.
- **R5:** Discover and account for framework suites through the existing registry vocabulary. Run supported generic suites and explicitly identify integration checks held by the owner override. Preserve assertions, fixtures and evidence; a held test earns no passing credit. Verify the boundary and command construction with focused regressions and run the resulting framework check to completion.
- **R6:** Finish the framework-native DSL batch before admitting deferred IR/exporter/schema/Go/backend work: retain final-source native compile/test/lint/boundary/format evidence, verify independent public clients and intended runtime/compiler refusals, and obtain a formal batch implementation review with any native findings resolved. Missing, failed, stale or partial evidence is not a pass; no native gate closes deferred integration requirements or permits Temporal source consumption.

## Verification

Check the complete source inventory, destination collisions, before/after content hashes, absence of active authored Scala sources under `model/temporal`, and the final diff. Source disabling intentionally makes prior Temporal build paths unavailable; do not run them as framework checks.

## Quick commands

```bash
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests
```

This native-only command is not the broad exporter gate. Existing valid results may be retained when native source has not changed.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Rename every tracked `model/temporal/**/*.scala` source to the same path with `.disabled` appended, preserving all file bytes. Leave generated caches and historical payloads unchanged. | fn-188-framework-only-source-iteration.1 | — |
| R2 | Record the framework-only scope in MILESTONES.md. Temporal implementation, migration, lifting, regeneration and verification stay held until the owner restores that scope. | fn-188-framework-only-source-iteration.1 | — |
| R3 | Preserve the unfinished fn-174.2 work and its evidence. Do not claim its mini/Temporal gate or closure passed. | fn-188-framework-only-source-iteration.1 | — |
| R4 | Provide a repeatable framework-only compile/test entrypoint using the existing framework modules and tool runner. It must not read or compile `model/temporal`, use `model/project.scala`, or depend on a cached Temporal Model jar or API jar. Build the generic framework and compiler plugin from current source with the existing IR jar. | fn-188-framework-only-source-iteration.2 | — |
| R5 | Discover and account for framework suites through the existing registry vocabulary. Run supported generic suites and explicitly identify integration checks held by the owner override. Preserve assertions, fixtures and evidence; a held test earns no passing credit. Verify the boundary and command construction with focused regressions and run the resulting framework check to completion. | fn-188-framework-only-source-iteration.2 | — |
| R6 | Finish the framework-native DSL batch before admitting deferred IR/exporter/schema/Go/backend work: retain final-source native compile/test/lint/boundary/format evidence, verify independent public clients and intended runtime/compiler refusals, and obtain a formal batch implementation review with any native findings resolved. Missing, failed, stale or partial evidence is not a pass; no native gate closes deferred integration requirements or permits Temporal source consumption. | fn-188-framework-only-source-iteration.3 | — |
