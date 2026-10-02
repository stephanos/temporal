---
satisfies: [R8]
---
# fn-114-gomad-correct-search-path-defects-and.11 Record select readiness in the runtime and the Choice Trace

## Description
E3 (R8), runtime half: the runtime records how many cases of a `select` were ready and which shape it had, and the replay plan exposes both for each select-poll decision. The reduction rule and its soundness comparison are task 12. Recording alone changes no schedule.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/go1.27.1.patch` (the `selectgo` hunk), `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/choice/schema/choicewire.json` and generated wire files, `tools/gomad3/choice/trace.go`, `tape.go`, `wire.go` and tests, the E3 fixture from task 2
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/**, tools/gomad3/choice/**, tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/internal/gomadtool/conformance/**, tools/gomad3/toolchain/version/version.json]

### Approach
- Poll order is drawn before the channels are locked, so readiness is unknown when the select-poll decisions are recorded. Count ready cases in the locked first pass and record the count with the select-result observation the patch already emits.
- A ready count alone cannot tell a proven shape from an unproven one. Record shape evidence with it: the case count, whether the `select` has a default, and one flag each for a nil-channel case, a timer-channel case, a closed channel among the ready cases, and one channel appearing in more than one case. These are the properties that distinguish the task 2 shapes; add a flag if a shape there needs one.
- The count covers every case of the select, including the ones after the first ready case in poll order. Counting must not allocate, draw from the seeded stream, or change which case is chosen.
- The replay-plan projection drops observations today. Carry the ready count and the shape evidence onto the select-poll decisions of the same `select` in the projected plan, so the explorer reads it from the decision. State how the decisions and the result of one `select` are matched, and what happens when a result is missing (a `select` that blocks and never resumes): the decisions keep an unknown count and no shape.
- Define the field in the wire schema and regenerate with `make -C tools/gomad3 generate`. Never hand-edit generated files. Raise the wire version; readers reject the older and any newer version visibly.
- Extend the task 2 fixture: every shape reports the expected ready count and shape evidence, including nil-channel cases that shorten the poll order.
- No record is dropped and no capacity bound changes in this task.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:680-725` — `selectgo` hunks: site, poll decision, result observation
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:804-822` — select-poll decision recording
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:313-347` — record struct and append path
- `tools/gomad3/choice/tape.go:145` — projection that drops observations
- `tools/gomad3/choice/wire.go:25-27` — record kinds
- `tools/gomad3/choice/schema/choicewire.json` — wire schema
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:344-357` — codec generator

**Optional** (reference as needed):
- `tools/gomad3/choice/legacy_v1.go` — how an older wire version is handled
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go:6-16` — generated version and kinds

### Key context
- fn-112 task 3 adds a diagnostic record to the same wire schema and overlay file, and fn-110 tasks 2 to 4 re-emit the patch. Check their state first, rebase onto whichever landed, and bump the wire version once for both if they land together.
- A timer-channel case can run its timer inside the poll-order loop, before the lock. Record what that does to the count in the fixture.
- Whether a decision and its result are always adjacent in the trace is not established. Do not rely on adjacency without a test.
## Acceptance
- [ ] Each completed `select` records its ready-case count and shape evidence, and both are correct for all seven fixture shapes
- [ ] Two fixture shapes with the same ready count are distinguishable from their recorded evidence
- [ ] The projected replay plan exposes the count and shape evidence on each select-poll decision; a `select` with no result yields an unknown count
- [ ] Recording performs no Go-heap allocation and no seeded draw, shown by a test
- [ ] For a fixed seed, the decision content of the core set's traces is unchanged apart from the new field
- [ ] A trace with the previous wire version and one with an unknown later version are each rejected visibly
- [ ] Generated wire files match the schema (`make -C tools/gomad3 generate` leaves no diff)
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
