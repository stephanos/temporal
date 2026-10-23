# Task3 correction audit addenda

Pinned correction: 3d0acb886866abfd04a2d2c34e8e87368cef2d5c. Original audit reports remain unchanged in conductor-audits.md.

### Correctness axis

## Correctness addendum — `8f1b4e7` → `3d0acb8`

- **Original finding resolved — walked:** `model/umpire/backends/apalache/src/replay.rs:171` rejects generated safety monitors without native refusal; line 178 selects invariant and negative registrations separately. Collision regression at `tests/replay.rs:405` covers all reported cases.
- **Start corrections resolved — walked:** `model/umpire/dsl/macros/src/state.rs:137` preserves field-width inference; line 139 validates the selected prefix through domain membership without enumerating combinations. Original domain evaluation order, empty-domain suppression and panic payload capture remain intact.
- **New substantiated correctness findings:** none. No unresolved finding from this audit.

Inspected regression assertions and retained passing output for category collisions, billion-wide prefixes, wide integers and actual start/vector controls. No tests executed during this addendum. Fresh current-source guest/slice/gaps proof and final required gate remain pending.

### Standards axis

## Standards correction addendum — `8f1b4e7201b6..3d0acb886866`

- **Resolved:** Fixture size finding. Executed comparison confirms the test JSON is one line, parses identically, and retained gzip decompresses byte-for-byte to the original witness.
- **Resolved:** Vector documentation finding. Cited README/DESIGN changes accurately describe ordered native projection and its Wasm boundary.
- **Unresolved original findings:** None.
- **New standards findings:** None. Walked corrections retain one start-domain generator, select prefixes without domain enumeration, and replace the old projection method. The three expansion mirrors change only the projection interface; handlers and laws remain unchanged.
- **Limits:** Read-only review of 27 changed files; no builds or runtime tests executed. Fresh guest/live/full-gate evidence remains pending.

Correctness: 0 unresolved, 0 new findings.
Standards: 0 unresolved, 0 new findings.

