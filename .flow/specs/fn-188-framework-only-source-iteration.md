# Framework-only source iteration

The owner changed the active scope to the framework only and explicitly requested renaming the Scala sources under `model/temporal` so they cannot be compiled accidentally.

## Acceptance

- **R1:** Rename every tracked `model/temporal/**/*.scala` source to the same path with `.disabled` appended, preserving all file bytes. Leave generated caches and historical payloads unchanged.
- **R2:** Record the framework-only scope in MILESTONES.md. Temporal implementation, migration, lifting, regeneration and verification stay held until the owner restores that scope.
- **R3:** Preserve the unfinished fn-174.2 work and its evidence. Do not claim its mini/Temporal gate or closure passed.
- **R4:** Provide a repeatable framework-only compile/test entrypoint using the existing framework modules and tool runner. It must not read or compile `model/temporal`, use `model/project.scala`, or depend on a cached Temporal Model jar or API jar. Build the generic framework and compiler plugin from current source with the existing IR jar.
- **R5:** Discover and account for framework suites through the existing registry vocabulary. Run supported generic suites and explicitly identify integration checks held by the owner override. Preserve assertions, fixtures and evidence; a held test earns no passing credit. Verify the boundary and command construction with focused regressions and run the resulting framework check to completion.

## Verification

Check the complete source inventory, destination collisions, before/after content hashes, absence of active authored Scala sources under `model/temporal`, and the final diff. Source disabling intentionally makes prior Temporal build paths unavailable; do not run them as framework checks.
