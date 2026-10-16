# Unversioned Gomad naming

The owner requested dropping the 3 from gomad3 everywhere. Use `gomad` and
`GOMAD` throughout maintained code, build targets, environment variables,
workflows, current documentation, package and helper names, and current
persisted contract identifiers. This is a clean namespace rename without
old-name aliases. Older retained artifacts require their matching old build;
historical evidence is not rewritten or relabeled.

## Requirements

- **R1:** Rename maintained identifiers and strings consistently, including
  Make targets, integration/build tags, installation names, environment
  variables, schema/profile names, Rust package/binary names and workflow files.
- **R2:** Preserve behavior apart from the requested namespace change. Historical
  rename-policy fixtures use a distinct neutral old directory so their rename
  tests remain meaningful. Inspect case-sensitive names and collisions.
- **R3:** Regenerate generated projections, checksummed manifests and current
  conformance fixtures through their existing owners. Current producers and
  consumers agree on renamed contracts; stale identities remain explicit
  refusals rather than receiving fallback aliases.
- **R4:** Preserve Git history, `.flow` historical specs/tasks/evidence/source
  snapshots, archived `.plans`/`.turbo` material and dated research/design
  documents under their original identities. Update maintained docs and
  instructions, including AGENTS.md and MILESTONES.md. Account explicitly for
  residual old names in historical material rather than rewriting proof.
- **R5:** Verify the rename with affected source/package, generation, model,
  lint-tool and Rust checks. Capture existing uncompiled fn-180 work and known
  replay-fixture failures before comparing the renamed candidate. Those
  unfinished requirements remain with their original owners; this rename does
  not implement inspection or revive native qualification.
- **R6:** Retain one change/evidence record with exact commands and numeric
  results, updated-name examples, remaining baseline failures and independent
  review. Do not claim WASM, native or performance qualification from source
  checks. No push, PR or CI dispatch is authorized.

## Implementation shape

Use one coherent mechanical source migration, then repair actual naming
collisions and regenerate outputs. Avoid a compatibility shim or parallel
configuration framework. Keep existing semantic `/vN` versions; the project
namespace changes, not every independent schema version. Current binary fixture
formats must be regenerated structurally, not edited by blindly shortening
embedded strings. Keep build outputs and scratch evidence in `/tmp`.

The initial inventory found about 845 maintained files with case-insensitive
references, plus two workflow filenames. Most other references belong to frozen
Flow evidence. The source baseline is `0609c11d80`; fn-180 through fn-185 remain
unfinished independently of this change.
