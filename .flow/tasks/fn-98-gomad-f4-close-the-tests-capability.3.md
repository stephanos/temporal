---
satisfies: [R4]
---
# fn-98-gomad-f4-close-the-tests-capability.3 Re-evaluate the eleven unsupported leaf cases on darwin

## Description
Record darwin first-blocker evidence; flip to qualified where packs allow or ensure blocker is not a forbidden import; update temporal.json and CI assertion.

## Acceptance
- each case qualified or non-forbidden-import blocker

## Done summary
All eleven formerly unsupported Temporal leaf cases now run on darwin/arm64 without a forbidden-import blocker. Ten of them qualify with exact replay. The eleventh, activity batch cancel, became a tier 3 `./tests` suite. It runs to success, but its same-seed evidence is intermittent (`replay_divergence` in the latest run). Two new exact darwin packs make this possible: `temporal-leaf-xsys-darwin-arm64` covers the golang.org/x/sys/unix facts that the Prometheus client reaches, and `temporal-leaf-xxhash-darwin-arm64` covers the arm64 xxhash and klauspost assembly. `!gomad` build-tag seams keep the cassandra, mysql, and postgresql files out of the gomad build of `common/persistence/tests`. `temporal.json`, the darwin CI assertion, the Makefile pack list, and the READMEs are updated to match. Commit 5285983bb9 fixed set-report validation for a seed whose repetitions recorded different choice tapes: that case now withholds choice-replay exactness instead of failing the run as unusable.

Darwin outcome per case (latest full `make gomad3-qualification`: expectations-met, 16 supported, 0 unsupported, 2 failed as intermittent tier 3, 18/18 completed):
- activity-batch-cancel-boundary: moved to tier 3, `./tests` closure; intermittent (replay_divergence this run); analysis pack set: modernc-libc-xsys-v047, modernc-libc-xsys-v047-isatty-v021, reflect2-go126, temporal-functional-{compute,tests}-darwin-arm64, both temporal-leaf packs
- sqlite-persistence-boundary: qualified (gomad tag); analysis pack set: modernc-libc-xsys-v047 (+isatty-v021), temporal-functional-{compute,tests}-darwin-arm64, both temporal-leaf packs
- temporal-cache-concurrent: qualified; new pack temporal-leaf-xxhash-darwin-arm64 names it (analysis pack set also carries temporal-functional-tests-darwin-arm64 and temporal-leaf-xsys-darwin-arm64)
- temporal-dither-pass: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-poller-history: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-queue-key: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-sqlite-schema-rewrite: qualified (gomad tag); analysis pack set: modernc-libc-xsys-v047 (+isatty-v021), temporal-functional-{compute,tests}-darwin-arm64, both temporal-leaf packs
- temporal-transition-history: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-update-abort-matrix: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-version-set-merge: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it
- temporal-workflow-backoff: qualified (gomad tag); new pack temporal-leaf-xsys-darwin-arm64 names it

On linux/amd64, `platform_expectations` now expects the ten leaf cases to stop at `foreign:assembly:xxhash_amd64.s`, which the darwin-scoped packs do not admit. It is not a forbidden import.

stage: impl-review - ran [2026-09-27] codex fan-out (3 draws SHIP, 0 findings)
## Evidence
- Commits: 5285983bb97903d53e3ddcd863d5fc939afc37ca, cf6525009330cdcd791e76cb71d6739488f98cce
- Tests: baseline: green (re-verified post-restart; spec defines no Quick commands), CGO_ENABLED=0 go build ./..., CGO_ENABLED=0 go build -tags gomad,test_dep,disable_grpc_modules ./..., go vet -tags test_dep ./common/persistence/tests/, go vet -tags gomad,test_dep ./common/persistence/tests/, make -C tools/gomad3 validate, go test -count=1 ./qualification/set ./internal/compatibilitypack (tools/gomad3), make gomad3-qualification (darwin/arm64): expectations-met=true supported=16 unsupported=0 failed=2 infrastructure-errors=0 completed=18/18, darwin CI jq assertion from .github/workflows/gomad3.yml against the report: true
- PRs: