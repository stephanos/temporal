# Task 43 source-gate handover

The source candidate enforces R21 through the shared five-import admission list. Production changes are limited to its two new entries and owning comment. `ValidatePack`, selection, evaluation, external loading and authoring algorithms retain their bytes. Source gates are terminal; Flow, review and commits remain root-owned. Required lint is red, so this is source progress, not completed acceptance or formal SHIP.

Source base: `43d264e24cae659ed57b81b9028e14fccb5a1fa3`; admission: `6cd5df49bdaafc43a0585bc3a251caa7081817ac`. The baseline and admission have identical scoped source. Actual recorded Go is stock go1.27.1 on developmental linux/arm64 ([pins](pins.json)); no native Darwin/Linux qualification or plugin/cgo execution is claimed.

The independently enumerated controls cover all five imports through pack validation, decoding/loading, unselected-token selection/identity revalidation, external directory/environment loading with a valid earlier file, allowed request validation/decoding/review, generation refusal before publication, and denied-request generation with an exact syscall grant. Literal controls retain within-pack structural priority, sorted rule/capability priority, earlier cross-pack digest/duplicate errors and request fact-validation priority. Generated positive controls retain denial evidence and grant only the exact syscall capability. Existing policy/digest tests preserve the other fixed-input decisions and grants.

The initial [RED](red.json) exposed plugin/cgo admission but its generation calls reached approval mismatch. [Approval discovery](red-approval-discovery.json) then retained successful unchanged-production approval hashes. The frozen literal-approval [final RED](red-final.json) demonstrated both prohibited grants publishing into populated temporary roots. The same fixture bytes pass after the source correction ([focused final](final-focused.json), [RED source hashes](red-sources.json)). All earlier attempts remain retained.

| Capture | Exit | Seconds | Result |
| --- | --- | --- | --- |
| [Baseline packages](baseline-packages.json) | 0 | 0.697 | 43 top-level + 288 subtest pass events; 1 skip |
| [Baseline scoped lint](baseline-lint.json) | 1 | 0.931 | 7 inherited findings |
| [Final RED](red-final.json) | 1 | 0.463 | 38 pass, 25 fail events; expected plugin/cgo failures |
| [Final check-only validation](final-validate.json) | 0 | 8.255 | Actual `make -C tools/gomad3 validate`; no regeneration |
| [Final packages](final-packages.json) | 0 | 1.693 | 137 top-level + 461 subtest pass events; 1 skip |
| [Target consumers](target-consumers.json) | 0 | 0.871 | 12 top-level + 25 subtest pass events |
| [Architecture/purity](architecture-purity.json) | 0 | 36.578 | 6 top-level tests pass |
| [Focused final](final-focused.json) | 0 | 0.560 | 11 top-level + 52 subtest pass events |
| [Formatting](formatting.json) | 0 | 0.026 | `gofmt -l` output empty for all six candidate files |
| [Standalone errortype](errortype.json) | 0 | 0.759 | Both affected packages, including tests |
| [Final scoped lint](final-lint.json) | 1 | 1.874 | Same 7 inherited findings; none introduced/resolved |
| [Source diff check](source-diff-check.json) | 0 | 0.016 | Tracked candidate source whitespace clean |
| [Preservation](preservation-final.json) | 0 | 0.185 | All 12 preservation checks pass |

Each capture retains the actual argv/cwd, UTC start/end, elapsed time, exit code, complete stdout/stderr, JSON test counts and actual skips. Go/cache commands ran serially under the offline environment in [baseline](baseline.json) and [final](final.json). The sole package skip in both ordinary runs is `TestHostPacksBindCurrentProfile`: `no deterministic profile for linux/arm64`. Check-only validation also ran against the final comment spelling before broader consumers; the preceding [validation](validate.json) remains as an earlier successful attempt.

Final scoped lint retains two S1016 findings in `mutation_test.go`, three ST1005 and one S1016 in `schema.go`, and one gci finding in `schema_timezone_test.go`. The schema locations move by one line from the owning-comment expansion; messages/source and dispositions are unchanged. Authoring and added tests introduce no scoped findings. Root owns the actual original integrated gate, which must run after this terminal handover: `make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false`. Its fresh count and stage reachability are pending here; no previous integrated count is presented as current evidence.

[Preservation proof](preservation-proof.json) carries one final six-file hash set and the exact historical diffs. Both named manifests cover 1,219 tracked scoped files and four executable hashes. Exactly five existing candidate files changed; 1,214 protected files, including 51 explicitly identified generated/pin files, retain exact hashes. The new schema test is separately hashed. Reconstructed existing external/request/generation test bytes match admission exactly after removing only added tests/imports. `policy_exhaustive_test.go` changes only its two authorized expected loader errors. Task42 evidence, all other source, pins, grants, generated bytes and CI remain unchanged. The first preservation command used the same filename for its proof and command capture, so its capture overwrote the proof; separate proof/capture paths corrected that artifact collision, with both commands retained. No Go source changed after freeze.

Review and acceptance stay with root. [Task43](../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.43.md) defines the bounded source admission; [task42's discovery](../task-42/external-pack-policy-gap.md) remains immutable. Task11 retains R17, and task21 consumes R21 directly. Original matched-first-baseline/predecessor/preservation/full/default/functional/affected-consumer/formal/native-Darwin/static-both-source-set acceptance stays open with those owners. Task21's original commands include `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, the complete `make -C tools/gomad3 test` gate and `make gomad3-tests-qualification` for functional/affected qualification. Remaining native linux/amd64 proof belongs to fn128 and is nonblocking for source progress. Native toolchain/host gates were unavailable on this stock linux/arm64 host and were not bypassed.

All six returned execution session handles (53027, 44176, 97164, 8801, 72854, 38829) were polled through terminal results; no Go/cache command remains live. No Flow state, commits, reviews, pushes or unrelated files were authored by this worker.
