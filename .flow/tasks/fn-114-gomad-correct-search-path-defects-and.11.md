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
The runtime now records each completed `select`'s readiness and the replay plan carries it onto that select's poll decisions. Choice wire v3 (`gomad3-choice-trace/v3`): a record's two reserved bytes become a readiness word (flags known/default/nil/timer/closed/repeated, ten-bit ready count, bounded by the 256-alternative poll cap) and an observation record carries, where a decision holds its selected identity, the origin ordinal at which its select began recording. `selectgo` reads the origin before its poll loop and counts readiness after `sellock` and before the first pass with a peek that mirrors `dequeue` (no allocation, no draw, no dequeue), then records both with the select result alongside the polled-case count. `ProjectReplayPlan` matches a result to its decisions by origin and polled count, requires them to be that select's poll steps (same site, alternatives 2..n), fails closed otherwise, encodes the readiness into the tape records so it survives `ValidateReplayPlan`, and exposes it as `ReplayPlan.Readiness`, one `SelectReadiness` per decision; a select that recorded no result leaves its decisions unknown. It sits beside `Decisions`, not inside `Decision`, because the exploration engine compares a prefix's decisions to the child's by struct equality and a select that never resumes in the child must not read as a prefix divergence. Readers refuse v2 traces by profile name and v2 or later-version headers, tapes, and terminal frames by version; the runtime refuses a v2 tape header. The fixture asserts the readiness on every select-poll decision of every execution for the seven task-2 shapes plus `timer-channel-due` (the poll loop runs the due timer, so its send counts) and `repeated-channel`, and shows through the diagnostic trace that the pass-1 shapes allocate nothing and draw nothing between their last poll decision and their result. The seven reference shapes still exhaust at 196/68/68/68/196/68/68 executions, and the timer sections' transcripts and prefix outcomes match task 5's evidence.

Toolchain build key 245141dc -> 2008ea81. Outside the declared Touches, as mechanical consequences of the identity change and listed here: `record/validation.go` (the pinned profile name and trace schema, two sites), `runner/runner.go` (the trace schema string the manifest writes), the test pins of those strings in `record/record_test.go`, `runner/inspect_test.go`, `runner/replay_operation_test.go`, `runner/runner_test.go`, `artifact/publication_test.go`, `qualification/qualification_test.go`, `runner/completion_characterization_test.go` (`choice.Version2` -> `Version3` and the pinned `v3 evidence` error text) and `runner/internal/execution/process_test.go` (`Version3`), the runner tests' shared record normalizer giving a fixture select result a known readiness, the retained root segment `runner/internal/campaign/testdata/pre-start-ordinal-journal/.../segment.json` re-encoded with the v3 tape header through `CommitRound` and its repinned after-state identity in `choice_exploration_start_test.go`, the refreshed golden `runner/testdata/diagnostic-identity-choices.json`, and the regenerated live-capability identities. No hunk touches a prohibited collector file.

Not carried per decision: the case count. The tape decision has sixteen free bits; the result record keeps the case count in `Alternatives` (plus one when a default is present), and a select's polled-case count is the alternatives of its last poll decision. Follow-ups: for task 14, `tools/gomad3/README.md` still says "v2 choice recording" and ARCHITECTURE's choice-trace paragraph does not yet mention readiness; from the review (P3), fold `runChoiceAccepting` callers onto `runChoiceSpec` and drop the duplicated `ready_at_poll` evidence field.

Gates on darwin/arm64, run after the final edit as separate commands: `make -C tools/gomad3 validate test-toolchain overlay-test` exit 0 (2:21), `make -C tools/gomad3 test-runtime` exit 0 (16:13 under a load average of 13 to 20), `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` exit 0 (3:16) on its third run: the first found three faults of this task, fixed before the second (the alias, the hanging inspect fixture, the v2 retained segment), and the second failed only on vanished host go-build cache entries, confirmed environmental by a focused rerun. Baseline: green via handoff (host tier at 25330890, runtime tier at b98c5018). linux/amd64 was not run. Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-11/` (gates.md, red-first.txt, search-reproduction.json, toolchain-build-key.txt).

stage: impl-review - ran (backend claude, model claude-fable-5-1 at high, same family as the writer; SHIP on the first round with two P3 findings left as follow-ups: runChoiceAccepting is now a pass-through to runChoiceSpec, and the retained evidence carries ready_at_poll beside readiness.ready)
## Evidence
- Commits: 7c93c66538f4d8ee43c5e7462a806bb33106b998, 95ddc41278976f3bcfced0727b893a7c4fd27c88
- Tests: make -C tools/gomad3 validate test-toolchain overlay-test, make -C tools/gomad3 test-runtime, GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host, go test -tags test_dep -count=1 -run TestRuntimeSearchFixtures ./internal/gomadtool/conformance, go test -tags test_dep -count=1 ./choice/... ./record/... ./artifact/... ./internal/gomadtool/... ./cmd/gomadtool ./qualification/... ./target/internal/livecap/..., baseline: green via handoff (host tier at 25330890 by fn-112.15; runtime tier at b98c5018 by fn-114.5 on build key 245141dc)
- PRs: