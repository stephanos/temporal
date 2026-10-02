---
satisfies: [R1]
---
# fn-114-gomad-correct-search-path-defects-and.2 Reproduce C2 and E3 with runtime fixtures on the unmodified toolchain

## Description
Second half of R1: fixture programs that reproduce C2 and E3 against the toolchain as it is before any fn-114 runtime edit, with their output retained as files. Tasks 5, 11, and 13 change identities, so this evidence cannot be regenerated later.

**Size:** M
**Files:** new fixture directories under `tools/gomad3/internal/gomadtool/conformance/testdata/` (one for timer-callback identity, one for select readiness), `tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go`, `registry.go`, `runtime_test.go`, `testdata/README.md`, retained output under `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/`
**Touches:** [tools/gomad3/internal/gomadtool/conformance/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/**]

### Approach
- Record the toolchain build key the fixtures ran on.
- C2 fixture: two goroutines each arm a context deadline (or `time.AfterFunc`) due at the same virtual instant, so the firing order follows the schedule. Record the Choice Trace under two schedules that fire them in opposite orders. Show that the callback goroutine identities differ between the two, and that a forced prefix that swaps the order ends in an alternative-set divergence. State in the fixture how each order is produced (seed choice or forced prefix).
- E3 fixture: a finite program covering these select shapes: blocking with zero ready cases, blocking with one ready, blocking with two or more ready, non-blocking with a default, a timer-channel case, a closed-channel case, and a nil-channel case. Record how many select-poll decisions each shape emits and how many of them had fewer than two ready cases.
- E3 baseline for task 12: run an unreduced choice exploration of the fixture to exhaustion and retain its set of outcomes and deadlocks.
- Register the fixtures the way the existing scheduling fixtures are registered. Assertions describe today's behavior; tasks 5 and 12 change them.
- Each E3 shape gets a stable name. Tasks 11 and 12 use the same names for the shape evidence and the eligibility rule.
- If a fixture cannot reproduce its finding, record the finding as refuted or changed per the task 1 closure rule.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go:12` — scheduling behavior checks to extend
- `tools/gomad3/internal/gomadtool/conformance/testdata/select/main.go` — existing select fixture
- `tools/gomad3/internal/gomadtool/conformance/testdata/choice_replay/main.go` — forced-prefix fixture pattern
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:363-371` — timer tie-break draw that decides same-instant order
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:766-770` — the counter the C2 fixture exposes

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/testdata/choice_exploration/main.go` — exhaustive exploration fixture
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:689-721` — select-poll decision and select-result observation
- `tools/gomad3/internal/gomadtool/conformance/registry.go` — fixture registration

### Key context
- fn-112 tasks 2 and 6 add fixtures in the same conformance package. Check their state and rebase onto whichever landed.
- Must finish before task 5 edits the runtime, or the unmodified-toolchain evidence is lost.
## Acceptance
- [ ] The C2 fixture shows different callback goroutine identities under the two firing orders and a divergence on the swapped forced prefix, on the unmodified toolchain
- [ ] The E3 fixture reports select-poll decisions per shape and the count with fewer than two ready cases, for all seven listed shapes
- [ ] The unreduced exploration's outcome and deadlock set for the E3 fixture is retained as a file
- [ ] Fixture output, the toolchain build key, and the commands are retained under `runtime-reproduction/` in the spec's artifacts directory
- [ ] A fixture that fails to reproduce its finding is recorded as refuted or changed and the owning task annotated
- [ ] `make -C tools/gomad3 test-runtime overlay-test` pass on darwin/arm64 with no runtime or overlay file changed; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
