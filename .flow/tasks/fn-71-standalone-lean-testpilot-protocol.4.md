---
satisfies: [R6]
---
# fn-71-standalone-lean-testpilot-protocol.4 Move Umpire provenance and compiler onto Testpilot

## Description
Make Umpire a producer of generated Testpilot Cases while retaining sole ownership of semantic lowering and its provenance payload. Preserve checked-input and unsupported-lowering behavior as compiler output moves through the neutral authoring facade.

**Size:** M
**Files:** `model/Umpire/Case/Compiler.lean`, `model/Umpire/Case/CompilerTests.lean`, `model/Umpire/Case/Provenance.lean`, `model/Umpire/Case.lean`, current Umpire Case producers and tests
**Touches:** [`model/Umpire/Case/Compiler*.lean`, `model/Umpire/Case/Provenance.lean`, `model/Umpire/Case.lean`, affected `model/Umpire/**/*Test*.lean`, affected `model/Umpire/Examples/**`]

### Approach
- Freeze the logical and byte representation of current Umpire `producerData`, including definitions, fingerprints, sources, Known Gaps, ordering, and optional subject/detail fields.
- Give Umpire sole responsibility for encoding that payload and construct generated provenance and Cases through `Testpilot.Authoring`.
- Migrate Umpire compiler and producers without exposing unchecked semantic evaluators or changing typed `LoweringError` behavior.
- Keep Testpilot and Go admission opaque to Umpire payload contents; malformed producer-owned JSON remains an Umpire concern rather than a generic protocol rejection.
- Retain temporary old-module compatibility only where required to keep the staged consumer migration buildable.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Case/Compiler.lean`
- `model/Umpire/Case/CompilerTests.lean`
- `model/Umpire/KnownGap.lean`
- current provenance encoding in `model/Temporal/Testpilot/TestpilotProtoJSON.lean`
- `.flow/memory/bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05.md`
## Acceptance
- [ ] Compiler and Umpire Producers return generated Testpilot Cases through the public authoring API.
- [ ] Exact Umpire producer bytes preserve definitions, fingerprints, sources, Known Gaps, list order, and optional fields.
- [ ] Checked-input rules and all existing unsupported-lowering diagnostics remain typed and unchanged.
- [ ] Generic Testpilot and Go paths neither require nor interpret the Umpire payload schema.
- [ ] Intermediate compatibility access creates no second protocol type or serializer implementation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

