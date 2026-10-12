---
satisfies: [R8]
---
# fn-164-gomad-backend-layout-shared-core-with.8 Keep pre-move artifacts inspectable and make identity mismatch explicit

Touches: tools/gomad3/artifact, tools/gomad3/runner, tools/gomad3/backends

## Description
Collect one retained native artifact and one retained wasm artifact produced before the move (from the qualification corpus or a fresh pre-move build on the parent commit). Verify `inspect` output is unchanged. Where the move changes a model or runtime identity, replay must fail with a message naming the recorded identity and the current one, not a generic decode or mismatch error. Add a fixture for each backend.

## Acceptance
- `gomad inspect` on both pre-move artifacts produces output identical to the pre-move binary's output.
- Replay of a pre-move artifact whose identity changed exits non-zero with both identities in the message; replay of one whose identity did not change succeeds.
- The two fixtures are checked in and run in the ordinary test suite.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
