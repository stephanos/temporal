# Framework-only source iteration

The owner changed the active scope to the framework only and explicitly requested renaming the Scala sources under `model/temporal` so they cannot be compiled accidentally.

## Requirements

- **R1:** Rename every tracked `model/temporal/**/*.scala` source to the same path with `.disabled` appended, preserving all file bytes. Leave generated caches and historical payloads unchanged.
- **R2:** Record the framework-only scope in MILESTONES.md. Temporal implementation, migration, lifting, regeneration and verification stay held until the owner restores that scope.
- **R3:** Preserve the unfinished fn-174.2 work and its evidence. Do not claim its mini/Temporal gate or closure passed.

## Verification

Check the complete source inventory, destination collisions, before/after content hashes, absence of active authored Scala sources under `model/temporal`, and the final diff. Source disabling intentionally makes prior Temporal build paths unavailable; do not run them as framework checks.
