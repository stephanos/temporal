# fn-105-gomad-follow-ups-deferred-scope.26 Rebind stale compatibility packs to the HEAD deterministic-I/O profile digest

## Description
Prerequisite repair found by D22 on 2026-09-30. HEAD d4d800fb47 changed the darwin/arm64 deterministic-I/O profile implementation digest to 9cd0cff9…, while packs modernc-libc-xsys-v041, -v047 and -v047-isatty-v021 still bind 80351583…; pack selection requires equality (tools/gomad3/internal/compatibilitypack/v2_selection.go:222), so every ./tests target (and control TestUserTimersTestSuite) analyzes as unsupported on darwin/arm64 and no Gomad verification can run. `make -C tools/gomad3 validate-compatibility` still reports packs current, so the staleness check misses this drift.

Re-review and regenerate every stale pack through the governed discover, review, generate --approve-review, check, qualify flow (README 'Compatibility-pack development'); do not widen policy, add facts beyond the fresh review, or bypass digest equality. Audit all packs/requests for the same stale binding on both platforms; regenerate darwin/arm64 ones on this host and record linux/amd64 ones that cannot be regenerated locally as explicit open work. Make validate-compatibility (or its test) detect a pack whose bound profile digest differs from the current profile, with a negative test.

**Touches:** tools/gomad3/internal/compatibilitypack/** (requests, reports, packs, generation.json, validation), tools/gomad3/Makefile only if the validate target needs wiring, .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/.

Quick: make -C tools/gomad3 validate compatibility-pack-qualification; tools/gomad3/.bin/gomad analyze --format=json go-test ./tests (closure, test_dep/gomad tags as the root wrapper selects) reports supported; qualify TestUserTimersTestSuite on seed 11 as a control. Build with PATH=$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin:$PATH because host go is 1.26.5.

## Acceptance
- Every darwin/arm64 pack binds the current profile digest via a fresh governed review; closure analysis of go-test ./tests reports supported with zero blockers on darwin/arm64.
- compatibility-pack-qualification and validate pass on darwin/arm64; a control functional suite qualifies on seed 11.
- Validation fails on a pack bound to a non-current profile digest (negative test retained).
- linux/amd64 packs with the same drift are listed with exact identities as open work; nothing is claimed for linux.
- No policy widening; no commits/staging/worktrees.

## Done summary
Rebound the three stale darwin/arm64 compatibility packs (`modernc-libc-xsys-v041`, `-v047`, `-v047-isatty-v021`) to the current deterministic profile digest `sha256:9cd0cff9…` through the governed discover, review, `generate --approve-review` flow. Each fresh review changed only `profile_implementation_sha256` on the adapter bindings (plus the approval and request digests); no module, source, fact, or disposition changed. The drift was intended: HEAD registered seven adapters in the profile and updated the profile goldens, but left the libc-bound packs behind.

`make -C tools/gomad3 validate-compatibility` now also runs `TestHostPacksBindCurrentProfile` (`internal/compatibilitypack/profile_binding_test.go`), which fails on any host-platform pack whose adapter binding differs from the current profile; `TestStaleProfileBindingsAreReported` is the retained negative test. On darwin/arm64 `validate` and `compatibility-pack-qualification` pass (9 requests), closure analysis of `go-test ./tests` reports supported with zero blockers, and `TestUserTimersTestSuite` qualifies on seed 11 with exact replay.

Open work, nothing claimed for linux: `modernc-libc-xsys-v047-linux-amd64` (pack `sha256:f5d91a03…`, request `sha256:f835de48…`, approval `sha256:9bdb99de…`) still binds `sha256:96487435…` while the linux profile golden is `sha256:84b27e62…`. It needs rediscovery on a linux/amd64 host; until then `make validate` is expected to fail there in the new test. Identities and results are retained in `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-task26-pack-rebind-evidence.json`.

No commits: the user owns commits for this run, so the evidence records `"commits": []` and all changes are uncommitted and unstaged in the working tree. Follow-up outside this task's Touches: README "Compatibility-pack development" does not yet mention the profile-binding check.

baseline: red (make -C tools/gomad3 validate compatibility-pack-qualification failed pre-edit: stale packs, the defect this task fixes)

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) [round 1 NEEDS_WORK (map-keyed bindings could mask a stale entry) .. round 2 SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: red (make -C tools/gomad3 validate compatibility-pack-qualification exited 2 pre-edit: validate green, modernc-libc-xsys-v041 'no longer matches the current target review'), env -u GOROOT make -C tools/gomad3 validate compatibility-pack-qualification (darwin/arm64, go1.27.1; exit 0, 9 requests qualified, TestHostPacksBindCurrentProfile ok), env -u GOROOT make -C tools/gomad3 test-host (exit 0, 45 packages ok; run before the round-1 review fix), .toolchain/bin/go test -tags test_dep -count=1 . ./internal/compatibilitypack/... (exit 0, after the review fix; includes the architecture test), negative: make -C tools/gomad3 validate-compatibility with the previous isatty request/report/pack restored and generation.json regenerated (check printed 'compatibility packs are current', target exited 2 in TestHostPacksBindCurrentProfile); files restored, red-first: TestHostPacksBindCurrentProfile failed on the restored stale v047 pack; TestStaleProfileBindingsAreReported/repeated_keys failed on the map implementation, tools/gomad3/.bin/gomad analyze --format=json --capability-mode=closure --build-tag=disable_grpc_modules --build-tag=gomad --build-tag=test_dep go-test ./tests (exit 0, supported, 0 blockers, 1043 packages), tools/gomad3/.bin/gomad qualify-set on the TestUserTimersTestSuite manifest entry, seed 11 (exit 0, qualified, replay_match and choice_replay_exact true), gofmt -l and go vet -tags test_dep ./internal/compatibilitypack/... (clean), gate receipt not written: NO_RECEIPT worktree dirty outside the ignore set (.plans/GOMAD_MILESTONES.md, another worker's file)
- PRs: