# Framework-only source iteration

## DSL-first execution hold (owner override)

The source iteration checkpoint is committed. Broad export verification in .2 remains incomplete and held while native framework DSL work runs. It is not a prerequisite for fn-164.2/.10, fn-165.7-.9, fn-173.10-.12, fn-123.9/.10 or fn-153.7. Native tasks run isolated native compiler/runtime and relevant lint tests; do not spend them repairing exporter/IR drivers. Resume broad framework export verification after the final native API, preserving its failed/interrupted evidence. Temporal sources remain disabled and excluded.

The owner changed the active scope to the framework only and explicitly requested renaming the Scala sources under `model/temporal` so they cannot be compiled accidentally.

## Acceptance

- **R1:** Rename every tracked `model/temporal/**/*.scala` source to the same path with `.disabled` appended, preserving all file bytes. Leave generated caches and historical payloads unchanged.
- **R2:** Record the framework-only scope in MILESTONES.md. Temporal implementation, migration, lifting, regeneration and verification stay held until the owner restores that scope.
- **R3:** Preserve the unfinished fn-174.2 work and its evidence. Do not claim its mini/Temporal gate or closure passed.
- **R4:** Provide a repeatable framework-only compile/test entrypoint using the existing framework modules and tool runner. It must not read or compile `model/temporal`, use `model/project.scala`, or depend on a cached Temporal Model jar or API jar. Build the generic framework and compiler plugin from current source with the existing IR jar.
- **R5:** Discover and account for framework suites through the existing registry vocabulary. Run supported generic suites and explicitly identify integration checks held by the owner override. Preserve assertions, fixtures and evidence; a held test earns no passing credit. Verify the boundary and command construction with focused regressions and run the resulting framework check to completion.

## Verification

Check the complete source inventory, destination collisions, before/after content hashes, absence of active authored Scala sources under `model/temporal`, and the final diff. Source disabling intentionally makes prior Temporal build paths unavailable; do not run them as framework checks.
