---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.5 Replace World's generic encoder with a bounded domain-local stdlib codec

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** world/{codec,replay,recording}.go, a World-private stdlib codec and focused tests (3 core callers plus codec/tests).
**Touches:** [tools/gomad3/world/**]

### Approach

- Use the planning probe decision: typed stdlib encode, UseNumber intermediate decode, final Encoder.SetEscapeHTML(false), trim exactly one newline. Restrict the helper to existing World Encode entry points and closed type inventory.
- Add typed original-string checks for terminal detail and every nested replay/transition request/readiness/delivery text. Reuse UTF-8 string checks without imposing limits-dependent model admission on standalone replay plans.
- Replace generic input decoding with strictjson while retaining config-first preflight/allocation/token bounds, semantic Restore/replay checks, terminal unknown-field rejection and exact wire reencode refusal.
- Behavior pin: retain the disposable probe's 5,029-comparison domain matrix as permanent task-scoped wire tests, including every transition body, large integers, all escapes, nil/empty/omitempty, JSONL and complete recording frames. Capture frozen baseline vectors/digests before retiring the legacy comparison dependency; permanent tests must compile after package deletion. Add explicit invalid-string and malformed-frame negatives.
- Read .flow/artifacts/fn-153-gomad-retire-canonical-json-and-private/planning-20261010/world-probe.md and its retained evidence/reproducer before coding. If a real producer/consumer mismatch remains, prove a typed World projection before package removal; no recursive-encoder relocation or wire widening.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/world/codec.go:20`
- `tools/gomad3/world/types.go:24`
- `tools/gomad3/world/snapshot.go:23`
- `tools/gomad3/world/replay.go:16`
- `tools/gomad3/world/recording.go:40`
- `tools/gomad3/world/world.go:308`
- `tools/gomad3/world/codec_test.go:12`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./world/...

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] Permanent equivalence controls preserve config-first snapshots, replay plans, every transition variant, JSONL and complete binary-framed recordings with exact payload bytes.
- [ ] Terminal and nested replay/transition invalid Go strings fail before lossy encoding; standalone plan validation/error order remains its own existing contract.
- [ ] Unknown/duplicate/trailing/raw-invalid-UTF8, alternate wire spelling/surrogates, malformed lengths/magic and configured allocation limits retain refusal.
- [ ] The implementation uses stdlib normalization only inside the closed World domain and contains no copied recursive canonical encoder.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
