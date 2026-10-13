---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R9, R10]
---
# fn-152-gomad-runner-storage-on-one-append-only.9 Close integrated crash evidence and update current storage contracts

## Description
Close the final source acceptance matrix and current contract documents for R1-R10.

**Revival trigger (2026-10-13):** The shared-owner cleanup was moved into the WASM queue, and fn-152.8 reached verified done/SHIP on the integrated gomad candidate, making this final source-and-contract closeout ready. On linux/arm64, the supported-host crash proof is the owner-approved WASM built-CLI gate recorded in the parent spec; the older native TestCLIKillResume remains source-only under fn-128/fn-149.

**Size:** M
**Files:** Core files (5): tools/gomad3/{README,ARCHITECTURE,SPEC,CLI}.md and cmd/gomad/integration_test.go. Wider touches: only residual integrated crash fixtures/architecture inventories identified by the final matrix; earlier owner tasks supply their own tests and deletion evidence. Combining this task with Inspect would couple 9-10 production/doc files, broad fixtures and evidence review, producing an L-sized task.
**Touches:** [tools/gomad3/README.md, tools/gomad3/ARCHITECTURE.md, tools/gomad3/SPEC.md, tools/gomad3/CLI.md, tools/gomad3/cmd/gomad/integration_test.go, tools/gomad3/internal/archcheck/architecture_test.go, tools/gomad3/backends/wasm/command/**, .flow/artifacts/fn-152-gomad-runner-storage-on-one-append-only/**]

### Approach

Reconcile owner tests and retain a complete R1-R10 evidence matrix for the final source candidate. Keep scripted seed/round/corpus/merge failpoint tests portable; update built-CLI SIGKILL/resume readiness and retain supported-host execution of these new storage cases under fn-152; only inherited transferred qualification stays with its native owners. Recheck complete active-consumer and legacy-decoder inventory, both-source-set architecture/static coverage, stdlib-only dependency diff, generated validation and scoped preservation of unaffected first-baseline behavior. Record all red obligations without excusing them through native deferral. Rewrite current README, ARCHITECTURE, SPEC and CLI storage sections for protected append framing, read-only replay versus exclusive repair, uncertain commits, transaction/reference ownership, new lineage refusal, unchanged CLI grammar and corpus-history growth. Keep canonical-JSON/atomic-write consolidation with fn-153.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/README.md:456`
- `tools/gomad3/ARCHITECTURE.md:541`
- `tools/gomad3/SPEC.md:434`
- `tools/gomad3/CLI.md:249`
- `tools/gomad3/cmd/gomad/integration_test.go:379`
- `tools/gomad3/Makefile:125`
- `.flow/artifacts/native-scope-transfer-2026-10-07.md`

### Verification

Final commands must be freshly retained with their actual scopes: portable owner/Runner/CLI tests using -tags test_dep; make lint-code-fast; make lint-code-gomad3; make -C tools/gomad3 validate; affected generation/check commands if declarations changed; architecture tests with -tags test_dep. Only explicitly inherited transferred native acceptance stays with its native owner. New storage and built-CLI crash cases remain this spec's required gates; a filtered stock-Go run cannot satisfy supported-host execution.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.
## Acceptance
- [ ] R1-R10: final candidate has a requirement-to-test/source-review matrix, exhaustive owner append/sync controls, uninterrupted/resumed semantic comparisons and explicit current source/lint/static/generated/preservation outcomes. Include guarded writable-open/repair replacement controls and durable admission/receipt/atomic-stop-drain recovery, with buffered real outcomes and committed cancellations.
- [ ] R2/R4/R8: final inventory finds no active old campaign segment/staging/archive/corpus/strategy journal implementation or legacy decoder; storage-only golden checks are replaced by retained behavioral coverage.
- [ ] R9/R10: current docs and CLI examples state implemented read-only corruption/tail behavior, lineage errors, artifact ownership, finite frame/live-corpus limits and separately growing history bytes; dependency and architecture reviews confirm in-tree storage.
- [ ] R3: built-CLI kill/resume uses committed log readiness and retains its behavioral gate; portable scripted success never substitutes for supported-host execution of this new storage gate; the older native transfer does not waive it.
- [ ] R1-R10: fresh independent review and full required source gates retain existing red obligations, fn-155 priority and fn-128/fn-149 deferrals; parked corpus measurement reports actual source-bound inputs or absence without manufacturing a result.

## Done summary
Closed the integrated fn-152 final candidate with a retained R1-R10 source/test matrix, current append-only storage documentation, deletion of the stale merged-storage golden, and a Linux/arm64 built-WASM-CLI `explore -> SIGSTOP/stable inspect -> SIGKILL -> inspect --json -> resume` gate. The gate fixes the durable prefix before process discovery, proves exact post-kill state, and resumes to the uninterrupted 12-outcome semantics.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-13T07:25:35Z..2026-10-13T07:47:52Z] (SHIP after one NEEDS_WORK fix loop)

### Result

- Retained `.flow/artifacts/fn-152-gomad-runner-storage-on-one-append-only/final-candidate-matrix.md` maps R1-R10 to current controls, tests, source review, owner append/sync boundaries, and final verification outcomes.
- Removed the tracked `pre-target-evidence-merged` storage golden and its redundant storage-only assertion; dynamic behavioral lineage refusal remains.
- Updated README, ARCHITECTURE, SPEC, and CLI for protected `GMLG` framing, lock-free read-only inspection, exclusive writer repair, uncertain commits, lineage refusal, payload ownership, live corpus limits, and separately growing history bytes.
- Added the supported-host built-CLI WASM crash/resume gate. Resume preparation now treats nil and empty ordered argv/tag slices as semantically identical, with a focused regression test.
- Source inventory found no active old storage decoder or tracked stale merged golden. Identity-bound paths, generated paths, `go.mod`, and `go.sum` have empty diffs. No source-bound corpus input exists on this host, so no measurement or estimate is claimed.

### Verification

- PASS: `go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus`; committed-head receipt `9f7a325b-unittest.json` was honored in Phase 5.
- PASS: stock-Go owner packages `./runner/internal/logstore ./runner/internal/campaign ./runner/internal/corpus ./artifact` with `-tags test_dep -count=1`.
- PASS: exact storage Runner/CLI/backend selector retained in the final-candidate matrix, including append-fault resume, repeated resume, inspect tail/corruption/lineage behavior, CLI rendering, and `TestValidatePreparedAcceptsCanonicalEmptySequences`.
- PASS: stock-Go `go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/...`.
- PASS: focused stock-Go `TestValidatePreparedAcceptsCanonicalEmptySequences|TestInstalledOperatorKillResume`.
- PASS five of five: stock-Go `go test -v -tags test_dep -count=5 -timeout=10m -run '^TestInstalledOperatorKillResume$' ./backends/wasm/command`; every run killed after one committed outcome and resumed to the 12-outcome baseline.
- PASS: `make -C tools/gomad3 test-wasm GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`; the required built-CLI gate executed and passed. Optional prototype gates kept their explicit missing-root skips.
- PASS: stock-Go `make -C tools/gomad3 validate`.
- PASS: `GOMAD3_ARCH_ALL_PLATFORMS=1` architecture and purity packages with `-tags test_dep -count=1`.
- PASS, 0 issues: task-base `make ... lint-code-fast` and `make ... lint-code-gomad3` using Linux-native golangci-lint/errortype and `GOLANGCI_LINT_FIX=false`, both before and after SHIP.
- Expected unrelated RED diagnostic: default-main broad `make lint-code-fast` reports untouched `cmd/tools/lintcode/main.go:412` cyclomatic complexity 28 (>25). Its initial wrong-host cached-binary launch was inconclusive; the single blank-line `--fix` side effect was inspected and removed.

### Review and obligations

- Review receipt: `/tmp/impl-review-receipt-657da2bc4466-fn-152-gomad-runner-storage-on-one-append-only.9.json`; terminal verdict SHIP, R1-R10 met, no surviving findings.
- The older native `tools/gomad3/cmd/gomad.TestCLIKillResume` remains source-only under fn-128/fn-149 and is not claimed as executed here.
- fn-155 priority and the fn-128/fn-149 native qualification/rotation deferrals remain unchanged; `MILESTONES.md` was not edited.
- Full rotation/requalification remains with the recorded owners; portable tests were not substituted for it.

### Commits and changed files

- `4137ce1a04` — implementation, docs, fixture retirement, matrix, WASM gate, and prepared-identity fix.
- `9f7a325bdc` — freeze-before-discovery review fix and completed gate ledger.
- Changed task files: the final-candidate matrix; README/ARCHITECTURE/SPEC/CLI; WASM backend provider/test and command kill-resume test; campaign merge test and retired fixture; flow review ledger. The required post-review memory entry is committed with the task receipt.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4137ce1a04aecfc9284615603e20dbcd1acfb3b7, 9f7a325bdcce098883e8ebc2ee899ecfaeb38e7a
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/corpus, env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/logstore ./runner/internal/campaign ./runner/internal/corpus ./artifact, env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -race -count=1 ./runner/internal/..., env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -v -tags test_dep -count=1 -timeout=10m -run '^(TestValidatePreparedAcceptsCanonicalEmptySequences|TestInstalledOperatorKillResume)$' ./backends/wasm/backend ./backends/wasm/command (from tools/gomad3), env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go test -v -tags test_dep -count=5 -timeout=10m -run '^TestInstalledOperatorKillResume$' ./backends/wasm/command (from tools/gomad3), make -C tools/gomad3 test-wasm GOMAD3_STOCK_GO=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go, env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make -C tools/gomad3 validate, env -u GOROOT GOTOOLCHAIN=local GOWORK=off GOFLAGS= GOMAD3_ARCH_ALL_PLATFORMS=1 PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck ./internal/puritycheck, PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make GOLANGCI_LINT_BASE_REV=35fecd5fc974b4d294456b98635dce480747addc GOLANGCI_LINT_FIX=false GOLANGCI_LINT=.flow/tmp/fn152-8-linux-tools/golangci-lint ERRORTYPE=.flow/tmp/fn152-8-linux-tools/errortype lint-code-fast, PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH make GOLANGCI_LINT_BASE_REV=35fecd5fc974b4d294456b98635dce480747addc GOLANGCI_LINT_FIX=false GOLANGCI_LINT=.flow/tmp/fn152-8-linux-tools/golangci-lint ERRORTYPE=.flow/tmp/fn152-8-linux-tools/errortype lint-code-gomad3, GATE_SKIPPED:unittest:green-receipt 9f7a325b - baseline reused from prior post-gate pass
- PRs: