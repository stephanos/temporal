---
satisfies: [R5]
---
# fn-96-gomad-f2-close-the-go127-port-on.3 Reproduce the Temporal corpus result on darwin/arm64

## Description
Run `make gomad3-qualification` on darwin; compare supported/unsupported counts and blocker paths against manifest expectations; fix or correct with evidence.

## Acceptance
- report matches manifest expectations on darwin/arm64 with 0 infrastructure errors

## Done summary
`make gomad3-qualification` on darwin/arm64 now meets every manifest expectation. The final run reported `expectations-met=true supported=5 unsupported=11 failed=2 infrastructure-errors=0 completed=18/18`. The 5 qualified and 11 `unsupported_target` tier 2 workloads match the manifest, including all 11 first-blocker paths. The darwin blocker for temporal-cache-concurrent is `xxhash_arm64.s`. The 2 failed workloads are the tier 3 ones, which ran `nondeterministic`, and both match their `intermittent` expectations.

Three defects blocked the darwin result, and all are fixed in commit a8777f5d73:

1. **Stale darwin compute pack.** `temporal-functional-compute-darwin-arm64` pinned golang.org/x/crypto v0.54.0. The module is now at v0.55.0, so the pack never activated and both tier 3 workloads were `unsupported_target` on arm64 assembly. I re-ran discover, review and generate at review digest `sha256:f17760eeb6fa02c315586cf99923c0f55aa3bf92f75f8508728e61fd7b32614f`. The only change is the module pin; the chacha20 sources are identical.
2. **No darwin pack for `./tests`.** I added the new pack `temporal-functional-tests-darwin-arm64` at review digest `sha256:b009df6c9380eeb9766b3bf7144df7d6477018ee879ae8586ed2a2742b841ce6`. It admits the `syscall` and `golang.org/x/sys/unix` imports of the Prometheus client's darwin process collector (`process_collector_darwin.go`), which is the darwin counterpart of the linux procfs admission. Both darwin packs are now in the darwin `COMPATIBILITY_PACK_QUALIFICATIONS` list and pass `compatibility-pack qualify`. `gomad analyze --capability-mode=closure` with the `gomad` tag on `go-test ./tests` now reports `supported` with 0 blockers on darwin. This overlaps F4 task .2 (the darwin `./tests` closure); F4 can take this as done. The overlap with F4 task .1 is only the compute-pack refresh. The fx, SDK and otel adapter pins did not break darwin analysis.
3. **Valid replay rejected as a runner failure.** The runner reports a choice replay as `exact` whenever the choice tape replays, even when a stream digest diverges. `qualification.cloneReplay` rejected that combination ("exact choice replay status requires a match"), so the child `gomad qualify` exited 3 and the set counted an infrastructure error. The validator now accepts that result as `replay_divergence`. `set.projectSeedReport` no longer sets `choice_replay_exact` on a seed whose replay did not match, which `validateSetReport` requires. Tests:
   - `TestBuildReportClassifiesExactChoiceReplayWithEvidenceDivergence` (qualification_test.go)
   - `TestProjectSeedReportDoesNotClaimExactReplayForDivergedReplay` (set_test.go)

   Both failed before the fix.

**Changed expectation:** I changed the darwin/arm64 expectation of `frontend-system-info` from `qualified` to `intermittent`, based on these darwin runs:
- An isolated set run: seed 17 qualified with exact choice replay, and seed 11 was a runner failure (the replay bug above).
- Full run 3: both seeds `nondeterministic`, one with a `choice_profile` divergence at ordinal 1778 and one on stderr.
- The final run: both seeds `nondeterministic` on `stderr.full_sha256`.
- Direct `gomad qualify` runs on seed 11: `replay_divergence` in one run and `nondeterministic` in another.

`unrepeatable` would reject a qualified seed, so it does not fit. The linux expectation (`unrepeatable`) and the goal default (`qualified`) are unchanged. `user-timers-workflow` also ran `nondeterministic` on both seeds on darwin (stderr divergence), which its existing `intermittent` expectation accepts.

**CI and docs:** the darwin `temporal-integration` assertion now also pins `.platform` to darwin/arm64, `.unsupported == 11` and `.supported + .failed == 7`, and its comment explains the intermittent split. The predicate passes on the final report. I updated `tools/gomad3/README.md` and `tools/gomad3integration/README.md`.

**Files and cleanup:**
- The final report is saved as `temporal-qualification-set-darwin-arm64.json` in the scratchpad. The run 3 report is saved as `temporal-qualification-set-run3.json`.
- The retained artifacts were deleted.

stage: impl-review - ran [codex fan-out rid 9f8763a51d20461886dd2150ec1a711e, 3/3 draws SHIP] SHIP
## Evidence
- Commits: a8777f5d7333e6ff09829441adc6c2ecae1a0f82
- Tests: baseline: none (spec defines no Quick commands), make gomad3-qualification (darwin/arm64): expectations-met=true supported=5 unsupported=11 failed=2 infrastructure-errors=0 completed=18/18, make -C tools/gomad3 validate, go test -count=1 ./internal/compatibilitypack/... ./qualification/... ./cmd/gomadtool/... (tools/gomad3), go test -count=1 ./qualification/... ./cmd/gomad/... ./runner (tools/gomad3), make gomad3-integration-test, jq -e <darwin temporal-integration CI predicate> temporal-qualification-set.json
- PRs: