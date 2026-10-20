# Gomad integration checkpoint — 2026-10-20

This is source-integration evidence, not candidate qualification or milestone completion.
Source candidate: `57d8b5f1ad9c7c597421d85a75524d42815a68bf`.

## Integrated source

- Sweep source `cc84570e2838dda3ec185764b5a099b284796ec9`: method/registration and nested-module discovery, selected-method verdicts, authenticated compile-only cache, bounded preparation/execution, clean-source admission and torn-journal repair. The independent sweep reviewer reported SHIP for the repaired patch; the compile-cache reviewer separately reported SHIP. Earlier wrapper-only v1 proof remains historical evidence and does not establish method coverage. Corrected live proof uses the isolated frozen sweep checkout, not this later integrated candidate.
- Determinism source `161c61798ad705e8942200f7b84a5fc064cb3e80`: genuine Runner stress retention/replay adapter, Go event-order parser/comparator, actual trace-to-Decision mapping, full-cohort perturbations and shared executable CAS. Exact Rust event diagnostics, capture, real CLI prefix replay and full stress execution are not established by this integration.
- fn-222 source mapping correction `5bc5a16988`: mandatory smoke is `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds`, not `TestFrontendSystemInfo`. The latter remains required as a separate supplementary cell for the first completed support list. No gate, capacity or original minimum frontend requirement is removed.

## Focused verification

Compiler: `tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go` (Go 1.27.1); commands run from `tools/gomad`, with `GOMAXPROCS=2`, `-p=1` and `-tags test_dep`.

| Command after the compiler path | Exit | Scope |
| --- | --- | --- |
| `test -p=1 -tags test_dep ./internal/backends/wasm/cmd/wasmsweep ./internal/backends/wasm/command` | 0 | Sweep command and installation command contracts. |
| `test -p=1 -tags test_dep -run '^Test(Stress.*\|Compilation.*\|CachePublication.*\|OptionsObservationIsNotSerialized)$' ./internal/backends/wasm/backend` | 0 | Stress admission, compile-only cache and publication source regressions; no live guest gate. |
| `test -p=1 -tags test_dep ./internal/backends/wasm/internal/sweep ./internal/backends/wasm/determinism/... ./internal/backends/wasm/installation ./internal/backends/wasm/command ./cmd/gomad/internal/cli` | 0 | Source compatibility of the integrated batches and existing complete setup policies. |
| `test -p=1 -tags test_dep -run 'Stress\|CompilationCache\|CompilationIdentity\|Retention\|TemporalConfig' ./internal/backends/wasm/backend ./internal/backends/wasm/cluster ./internal/backends/wasm/internal/measure ./internal/backends/wasm/wasi` | 1 | Backend selection also matched live integration tests without `GOMAD_STOCK_GO`; those refused setup. Cluster, measure and wasi package selections exited 0. No guest failure or integration pass is inferred. |

Two earlier command-selection mistakes are retained here rather than hidden: the first source command named nonexistent packages `internal/backends/wasm/cmd/gomadsweep` and `internal/command` (exit 1; corrected above). A broader backend selection matched `TestProvenanceRejectsChangedIdentityBeforeExecution` and `TestProviderRejectsEmptyModelIdentity`, which require explicit integration inputs (exit 1); the source-only selection above then exited 0. These are not substitutes for the required integration gates.

`make lint-code-gomad GOLANGCI_LINT=/home/agent/wt/fn222/.bin/golangci-lint-v2.13.0 ERRORTYPE=/home/agent/wt/fn222/.bin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=9cb5bc3149` exited 2. Log: `/home/agent/wt/evidence/integration-2026-10-20/lint-57d8b5f1ad.log`. Findings: two gci import-format issues (`determinism/event_order_test.go`, `determinism/observed/inspection_test.go`) and one testifylint assertion-style issue (`backend/determinism_integration_test.go`). The determinism owner is repairing them; this candidate is not lint-green.

`go test -p=1 -tags test_dep ./internal/archcheck` exited 0 in 58.469 s on the same source candidate. Log: `/home/agent/wt/evidence/integration-2026-10-20/archcheck-57d8b5f1ad.log`. `flowctl validate --all --json` exited 0: 59 specs, 420 tasks, zero errors and 36 warnings (including historical evidence/coverage warnings and fn-227's new heading/R-ID formatting warnings). Structural validity does not establish acceptance for any open task.

## Resource preservation

Removed unused, clean, merged checkouts `/home/agent/wt/cand2-frozen` and `/home/agent/wt/qualdriver` after checking their commits are ancestors of the integrated branch and no live process used them. The source commits remain in Git, the qualification-driver branch remains, and external evidence is retained. Only reproducible checkout/generated preparation copies were removed. Available disk rose to approximately 15 GiB; subsequent builds can change that figure.

## Subsequent integrated checks

At `28d2c881475249902cc4627bff975133ed7c2e45`, attribution source and all four feature-mode sidecar updates through `f172c17499` were integrated with determinism through `c60ae60207`. The independent attribution source/fixture reviewer reported SHIP. The registry merge retains both stress and cluster-owner producers. The broker merge retains both diagnostic World spans and supplemental event contexts. An actual compile check caught duplicate cluster-stderr accessors; integration retains one detached accessor, one completed-result clone and a regression that directly mutates the detached result to verify session isolation.

- Integrated `make lint-code-gomad` with the exact overrides above exited 0, zero issues. Log: `/home/agent/wt/evidence/integration-2026-10-20/lint-28d2c88147.log`.
- `cargo check --locked --all-targets --features diagnostic-profile` exited 0 (6.44 s), and the separate `--features stress-jitter` check exited 0 (5.90 s). Logs: `/home/agent/wt/evidence/integration-2026-10-20/rust-{diagnostic,stress}-28d2c88147.log`.
- An earlier combined `diagnostic-profile,stress-jitter` check exited 101 because the source explicitly forbids that feature combination. That guard remains intact; it was not a supported-role build failure. Log: `/home/agent/wt/evidence/integration-2026-10-20/rust-combined-28d2c88147.log`.
- After integrating context fixes through `791a61ed2d`, at root `dc5af510c610d7c577c22f6c9928fbe0e3fb9502`, fresh focused determinism/observed, sweep, measurement/measure and CLI package tests exited 0. `go vet -p=1 -tags test_dep` over backend, cluster, wasi, measure and CLI also exited 0. Exact supplemental state-commit diagnostics and live prefix probes still have their own fn-188 acceptance gates.

The corrected fn-221 two-package proof and same-candidate resume are retained in `fn221-method-proof-2026-10-20.md`. Root reran both committed v2 sidecar verifiers: exit 0 each. Independent final evidence review returned SHIP; `flowctl done` and a subsequent `show` verified fn-221.1 done before commit `c3ec0975e6`. Full survey and reason ranking remain open. The actual fn-185 early activity/runtime-cost proofs are separately retained in `fn185-attribution-port-2026-10-20.md`; they do not replace integrated qualification or full matched measurements.

The unused clean merged `/home/agent/wt/cand2` checkout was subsequently removed with its branch preserved. The adapter owner separately removed its own unused clean `/home/agent/wt/fn188-runner-adapter`, preserving all original source commits on its branch and all required evidence externally. No live checkout or raw execution evidence was removed.

At `1739cad84f0f4ed180275dcc2009f9adc2eb6e2f`, supplemental namespace/recovery commit diagnostics through `20a81a8191` are integrated. `cargo fmt --all -- --check` and separate locked all-target checks for empty features, `diagnostic-profile`, `stress-jitter` and `determinism-fixture` all exited 0. Logs are `/home/agent/wt/evidence/integration-2026-10-20/rust-{fmt,release,diagnostic,stress,fixture}-1739cad84f.log`. Live handoff/prefix acceptance remains separate from compilation.

Qualification preflight's static equivalence check found 101 reference stages minus five retired native stages, exactly 96 required DAG stages, identical commands and no problems. Initially suspicious tool paths were verified, not changed: stock trees `2198cf19890232fc` and `878c9b2935e2c9e6` have identical relevant source/compiler inventories and compiler SHA-256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. The retained Linux/AArch64 lint binaries execute, have the same mandated versions/build inventories as fn-222's copies, and verify against their original manifest. Original driver bindings remain intact. No full candidate qualification has started at this checkpoint.

After checking no process depended on them, root removed reproducible Rust incremental and generator compilation caches. The unused root-owned `q1` frozen checkout was also removed after preserving its exact current helper at `/home/agent/wt/evidence/q1/retained-worktree-gomad-wasmhost` (SHA-256 `ed6cbaeafab2fb5bf9631100d943276da9015525a1648aee744dc6906209d189`) and both staged source fixtures under `/home/agent/wt/evidence/q1/retained-staged-fixtures/`. Copies were compared byte-for-byte before removal. Historical qualified helper/module/receipt evidence under `evidence/q1` remains untouched; the saved worktree helper is not relabeled as the qualified ordinary helper. The checkout's source is recoverable at ancestor `e50c6476fba40d3b31ddb20d1030abf08807412c`. An old matched-pair driver naming `q1` must recreate that checkout and its staged files; future measurements may bind a new frozen candidate. The transferred matched-release baseline checkout `fn189-cluster-baseline` remains. Free disk after removal was 14,332,912 KiB; this is preflight headroom, not a promise about later resource use.

## Integrated validation regression

`make -C tools/gomad validate` at `ef48c836b026fd7615e7da6a9018781677f46309` exited 2. Protocol generation check, Rust inventory/helper corpus/engine refusal tests, Go determinism packages and archcheck passed. The separate WASM root architecture gate failed on both darwin/arm64 and linux/amd64: pure `wasi/temporal.go` imports `path/filepath` and `time`. The same function performs host schema capture, so merely replacing the imports would not repair ownership. Log: `/home/agent/wt/evidence/integration-2026-10-20/validate-ef48c836b0.log`.

Root reopened fn-220 through new R3 regression task fn-220.4 (the tracker refuses restarting a done task directly), reset completion review to unknown, and restored its unfinished milestone row. The repair separates shared host capture from pure complete-policy assembly, preserving all configuration bytes and caller error behavior. No architecture allowlist or exemption is relaxed. Full qualification, stress references and full survey remain unstarted pending the integrated fix and passing validation. Prior fn-220 execution and review evidence remains historical, not proof that this newly exercised architecture gate passed.

The repair `f108a425bf4dfee0c616ae17812c4afd0032f9f8` is integrated at `a8597f7451a5e365126bd5b186016e373fb8e282`. Root repeated focused caller/policy packages: exit 0. Its full-manifest before/after comparison is byte-identical, SHA-256 `fbbf5cab6fb43c2253a67453415576aec4b6b6c195d0c87217d3c132098db332`; actual-schema configurations for all policies plus installation are byte-identical, SHA-256 `8c24859bdf22d0c7228e2a966aa31b3de21f3c9013aac410b3b5ae6207f6164c`. The existing model identity rotation predates the repair; it adds none. Worker commands/raw evidence are retained at `/home/agent/wt/evidence/integration-2026-10-20/boundary-worker.md`.

The next integrated default validation passed both-platform architecture, then exposed a distinct fn-188 target-wiring bug: Rust tested the current helper under inherited `CARGO_TARGET_DIR`, while Go tests selected an older checkout-local helper. Actual unknown-field/schema results were `invalid`, contradicting the new `unsupported` contract. Log: `/home/agent/wt/evidence/integration-2026-10-20/validate-fn220-boundary.log` (exit 2). Root retained the old checkout helper at `pre-determinism-publication-gomad-wasmhost` in that evidence directory, SHA-256 `bbbb48fb4dc899393f117ef6c8446aa3c587ad7043f9b301b65df2ca7b4210c0`, before normal publication replaced it.

`test-determinism` now depends on the existing `helper` target, matching other runtime-consuming targets and its atomic build/copy/compare publication. No assertion, import allowance or refusal expectation was changed. A fresh `make -C tools/gomad validate` with the integrated repair and this prerequisite exited 0: protocol check, Rust inventory/helper corpus/engine refusals, Go determinism/archcheck/both-platform WASM architecture and transport-boundary gates. Log: `/home/agent/wt/evidence/integration-2026-10-20/validate-published-helper.log`. The published helper SHA-256 at completion was `cd9bfdb53cac6447cc0f88895c768993278f339176be420ba82da74858b79316`. Cargo's later test builds may replace its shared-cache executable, so the published copy is independently retained for consumers. Independent source review and the new current-helper named setup proof remain separate from this validation pass.

The fresh setup proof under `/home/agent/wt/evidence/fn220/current-setup.JLtluL` subsequently recorded install, discover, explore and replay exits all 0. Explore retained one success with a complete 3943-record/2943-decision trace; replay reported `reproduced=true diagnostic=false result=success choice-replay=exact`. Its full identity and selected-test evidence are retained separately by the setup owner; this is not full candidate qualification.

After confirming no Cargo or rustc process was live and notifying the determinism owner, root removed only `/home/agent/.cache/codex-build/cargo-target/debug/incremental` (15 GiB of reproducible compiler cache). Executables, dependency outputs, release builds, modules, artifacts, receipts and raw logs remain intact. Available disk rose from approximately 9 GiB to 25 GiB. The attempted force-removal command was rejected before execution; plain recursive removal succeeded. This cache is recoverable by rebuilding.

The official three-draw fn-220.4 implementation review finalized SHIP with zero findings, all three draws successful, on `3cbd8b3cf7..c20b280e2b`. RID `a63de3f014fc407ba3fdc5b1533da87f` retains the draw outputs; the receipt is `/tmp/impl-review-receipt-657da2bc4466-fn-220-gomad-one-command-temporal-test-setup.4.json`. Reviewer and writer share the model family. Root then completed fn-220.4 via `flowctl done`, verified `show` reports done, validated all four tasks structurally (zero warnings/errors), and committed the task/proof/milestone updates at `08cb4d2f5c`.

Spec completion review from the original base `9cb5bc3149` refused before dispatch (exit 2): its 69,522,212-byte integrated branch diff exceeds the tool's 67,108,864-byte identity limit. No content was truncated or review verdict claimed. The prior fully reviewed fn-220 completion head `c302ae18464aefcab1e05a99d69701524f6bb13b` is an ancestor of the repaired candidate; its delta is 14,483,818 bytes. The same completion receipt/session is resumed from that last reviewed head, retaining all R1–R4/current four-task compliance scope rather than re-reviewing unrelated earlier branch history. No force/reset or manual terminal verdict write is used.

That actual resumed review returned SHIP (exit 0), all R1–R4 met with zero surviving findings, at 2026-10-20T17:15:18Z. Root read the receipt/output and verified official `completion_review_status: ship`, then closed fn-220 and verified its status is done with all four tasks done. Structural validation reports zero errors/warnings. No tracker is active. Fresh Go tests were unavailable inside the review's read-only sandbox; root/worker executed tests remain the gate evidence, not inferred reviewer execution. The stable current stress-jitter helper was separately built successfully with Rust 1.99.1, two jobs, incremental disabled; source preparation is not stress acceptance.

## Reference preflight and remaining inventory corrections

Clean candidate `f046982309e12d2c419f8941c6bdae2ce4e27a61` was frozen at `/home/agent/wt/qualification-next-frozen`. Existing freeze and strict verification exited 0 over 8,888 regular inputs and two symlinks; source-manifest SHA-256 is `6ce6d518ff531ec056b0e02f29729de5047fbbe93a67ce72b02640fbd70f394b`. The external config is `/home/agent/wt/evidence/qualification-next/candidate.env`. This preparation launched no qualification stage and proves no ladder acceptance.

The four-reference preflight waited for actual unrelated build handles to exit, retaining load snapshots. Its first runtime CLI attempt then stopped before any guest: generic measurement retention created the authenticated run directory, but the stress producer still required that directory not to exist. `/home/agent/wt/evidence/fn188/reference-preflight-f046982309/runtime-0.log` records the staged path, refusal, Go child exit 2, outer exit 1 and 3.450 seconds. The failed root is preserved. No reference, stress cell or qualification pass is inferred. Diagnosis also found child executions attempted nested retention and reruns published staged paths. The existing fn-188.5 owner is repairing those linked assumptions against the real public Run boundary, with red/green regressions and unchanged central retention guards.

A parallel independent source-acceptance audit found R1 gaps: runtime per-site rows exist in Go shared-grammar tables but are not represented/linked individually in the JSON inventory; its aggregate runtime rows can disappear without failing runtime completeness. Current duplicate-ID checks also do not enforce the historical ID-to-site nonreuse contract. A disjoint worker owns per-site JSON linkage and append-only identity-history enforcement under existing fn-188.1/.2. Those tasks remain incomplete, without weakening the specification. The frozen candidate above is retained as pre-fix evidence; the eventual integrated corrections require a new freeze before reference/stress/qualification claims.

## Map-order repair and qualification continuation

Source candidate `04e06bc95f958e1957a8de04f6bef680b6515121` integrates fn-227's
stable shutdown and storage-validation ordering. Its source verification is in
`fn227-source-acceptance-2026-10-20.md`; R1–R3 source checks passed, while R4
qualification and independent acceptance review remain open. The compiled model
identity rotates to
`sha256:73c1a4f3f64dedd4c18dad47849f6dcc8bc04d08e27067dedd096acde05e2e60`.
Thirteen regenerated corpus files change implementation pins only.

Historical candidate `ea45f94e59` completed unchanged one-box preparation with
exit 0 in 100.382 seconds after an initial hard-linked-helper admission refusal
(exit 1). Its receipt, exact commands and identities remain under
`/home/agent/wt/evidence/qualification-final/early-onebox-ea45f94e59/`.
This is preparation evidence, not execution or qualification of the repaired model.

The repaired candidate is frozen at
`/home/agent/wt/qualification-final-frozen`, with source manifest digest
`536e7864ebc8e2bc07a0e3946a4d4272ae700123329a6586180173c348ded87f`.
The qualification worker owns execution session `77189`, script
`/home/agent/wt/evidence/qualification-final/run-04e06bc95f.sh` and adjacent log.
It runs prepare-only first, retains that receipt before measured cells overwrite
it, then runs the unchanged 96-stage ladder with its 3600-second limit.
Check the live handle and actual stage results before any resume or restart.
No completed ladder is claimed here. External build/scan processes are observed
for quiet-host admission and are never interrupted.

Full stress must follow the repaired source: 41 references and 501 perturbed
fresh executions (K=3, concurrent fanout=2), with inapplicable handoff axes
reported explicitly. The full fn-221 survey remains unstarted. Its recorded
budgets are 3 minutes per execution, 8 minutes preparation and 24 hours overall,
seed 0, all discovered packages and methods, `--max-tests 0`, and
`--config-source temporal-install`. Build fresh launcher/sweep executables and
bind the qualified helper. The old external full-sweep wrapper both names a stale
candidate and compares bare hashes against `sha256:`-prefixed sidecar values;
invoke the existing harness directly with stable resume paths. A verifier pass
does not imply every row ran; retain not-run and skipped counts separately.

Remaining worktrees: the main `gomad-milestones` integration checkout;
`/home/agent/wt/qualification-final-frozen` for the active candidate;
`/home/agent/wt/fn227-map-order` for review fixes while fn-227 stays open;
`/home/agent/wt/fn188` for the pending stress run; and
`/home/agent/wt/fn221` for the pending full survey. The latter two are still at
`04e06bc95f`; they advanced cleanly before execution. No milestone task is
completed by this checkpoint.

The repaired candidate's prepare-only gate passed (exit 0, 242.311 seconds),
retaining the same guest module digest as the earlier preparation. The first
driver entry-point attempt then exited 127 before any stage: inherited
`BASH_ENV=/etc/sandbox-persistent.sh` changes nested-shell working directories.
The corrected absolute-path launch removed that cwd-only startup hook and began
all 96 scheduled stages in session `86946` at 19:21:53 UTC.

Official three-draw review `4f1c248253d54205a452842b8010c8f1` finalized
`NEEDS_WORK` on the integrated changes. Correctness and contracts found no sorting
defect; integration found 62 network-adapter fixtures, 51 network-cluster
fixtures and the Rust cluster WASI pin still carrying the previous model digest.
The merged receipt retains this defect and unfinished R4 qualification as two
open findings. Model-dependent corpus checking must cover these consumers too.
Review model was `gpt-6.1-sol` at high effort, from the writer's model family.

Because that candidate was demonstrably stale, the conductor stopped its ladder
through supported SIGTERM propagation. Session `86946` ended with exit 130;
the scheduler records the interrupted validate stage at 19:23:01 UTC, and
`driver-exit.txt` plus `wall-time.json` retain both exit codes and 75 seconds
elapsed. No stage completion or qualification pass is claimed. The qualification
worker verified no live process remained in the frozen checkout or scratch root.
The fn-227 worker owns the missing-pin repair through the existing generators;
a new clean freeze is required after integration. Preserve both previous
prepare-only receipts and the interrupted run as historical evidence.

The missing network pins are integrated at `e3c98d4256` (worker checkpoint
`1ae035f31f`): all 113 JSON payloads are unchanged except implementation identities,
and the existing generator also refreshes the Rust WASI pin. Two additional
acceptance-check defects were repaired before another freeze. Commit
`50951abe80` makes the Go helper-corpus checker require `unsupported` for the
unknown storage-field case, matching the existing Rust literal and corpus;
the new regression first demonstrated rejection of `unsupported` and acceptance
of wrong `invalid`, then passed with both wrong classes refused. Its focused
Go tests exited 0 (0.006 seconds), and `make lint-code-fast` against `fd74838967`
with the retained Linux lint tools exited 0 with zero issues. These commands'
tool outputs are the evidence; no separate raw lint log is claimed.

Commit `429cd93008` integrates the Rust feature-gating repair from `7c1d7ab371`.
Its separate `fn188-rust-cluster-test-feature-gating-2026-10-20.md` records all
seven lint modes passing and default cluster-test preservation. The integrated
WASI preflight then ran pinned Go 1.27.1 with `-p=1 -tags test_dep -count=1
-timeout=10m -run 'Corpus|ImplementationIdentity'`: exit 0, 2.756 seconds.
Raw verbose output is
`/home/agent/wt/evidence/fn188/gate-preflight.XgjpFv/go-corpora.log`.
It used a private single-link helper copy with SHA-256
`ae4c7f9e2b7cc07d8d2490d3da755436a53f5b39f7d7213cee7cd6665695462f`
in that directory and explicit stock compiler/test-helper environment variables.
The run covers the regenerated network consumers and the corrected helper refusal
checker together; it is not full candidate qualification or stress acceptance.

The next clean source candidate is `8cd478404b2db7d2fb2c528d210bc7cfa29b8be8`.
Frozen manifest digest:
`4fdcc6b2d1dcde0499c6b53d60ebdc3506ed33902ce86bb5d73a9c7f7061fa79`.
Qualification orchestration session `79596` is live: root observed its
prepare-only test and compiler subprocesses, after the fresh helper build.
Its log is `/home/agent/wt/evidence/qualification-final/run-8cd478404b.log`;
the adjacent explicit candidate config and `run-current.sh` bind its inputs.
The qualification worker owns this handle through terminal completion. Do not
restart it because an observation times out. The `fn188`, `fn221` and frozen
qualification worktrees are all at this candidate; the fn-227 repair worktree
stays at `7c1d7ab371` for possible review fixes. Acceptance review must resume
the existing merged receipt after the missing full qualification is supplied.

Session `79596` subsequently ended with driver/scheduler exit 1 after 276
seconds: 15 stages passed, root lint and the test harness failed, and all other
stages stayed unrun. Root lint misclassified the two tagged debug fixture roots;
the launcher routing test inherited the driver's measurement guard and reached
the intentionally strict environment refusal. Required Rust lint passed.
The failing logs, receipts and preparation remain in `qual-8cd478404b`.

Worker repair `579d1b2bbe` is integrated at `c94a4d65d8`. It classifies only the
two exact fixture roots and isolates the launcher test environment while
preserving missing-package and nonempty/empty guard-refusal assertions.
The original broad-base lint command against `65a5f64ffb`, the root router
suite and full test harness under the driver environment all passed.
`fn188-validation-repairs-2026-10-20.md` records the RED/GREEN commands and exits.
No helper, runtime, model or production launcher behavior changed.

Read-only native-deletion planning also disproved fn-223's identity-neutrality
assumption: the shared cluster digest embeds the glue being removed, and the
WASM cluster digest and scenario bindings compose it. The existing spec/tasks
now record a conductor decision allowing only proven source-driven rotations,
preserving complete hash coverage and unchanged remaining implementation
identities, and requiring full changed-candidate qualification. The deletion
goal, live WASM instrumentation and fn-219-closed start gate remain binding.
fn-223 is still unstarted; structural validation reports zero errors/warnings.

The next qualification freeze is `3f532910be73f622e923273bff86c401645fe7ba`,
with source manifest
`39182df082494be677778bdbd642b4812defef546b81189bf7a2de36d52a2fba`.
Session `78369` remains live under the qualification worker. Prepare-only
passed in 90.925 seconds with the same guest module digest. The full ladder
started at 19:47:50 UTC, with a 20:47:47 deadline; repaired root lint and harness
stages both exited 0. This is ongoing evidence, not a qualification pass.
Its logs and explicit candidate configuration remain under
`/home/agent/wt/evidence/qualification-final/`, in `qual-3f532910be` and
`run-3f532910be.log`. Subsequent plan-only edits do not change this frozen source.

fn-223 plan review resumed session `01a15ada-1927-7042-9467-d055d92ef0ac`
through the same receipt and returned **SHIP** on round 2. All four findings
are fixed: obsolete fixture consumers are removed with their fixtures,
reference-bearing docs precede the tracked-input scan, native smoke jobs and
routing are explicitly retired, and task write scopes are declared. The
reviewer was `gpt-6.1-sol` at high effort, the writer's model family; no new
blocking or maintainability findings remain. The receipt is
`.flow/tmp/plan-review-receipt-fn-223-gomad-delete-the-native-backend.json`;
raw round-2 output is `.flow/tmp/fn223-plan-review-r2.log`. This approves the
plan only. fn-223 stays unstarted until fn-219 closes, and its implementation,
identity accounting and changed-candidate qualification remain required.

Qualification session `78369` ended with exit 1 on frozen `3f532910be`.
`qual-3f532910be/wall-time.json` records 2,417 seconds against the 3,600-second
limit: 31 stages passed, `ordinary` failed, and 64 stages did not run.
Network and retained Temporal inspection passed while the fail-fast scheduler
drained; runtime and mandatory workload cells were never admitted. The literal
`make test-wasm` command exited 2. Installation command tests inherited the
driver's `GOMAD_STOCK_GO` and reached its intentional refusal before their
own assertions. Separately, the WASI package exhausted its unchanged 900-second
budget: the 100-fresh-instance test passed in 606.05 seconds and
`TestStorageOwnerParity` had run only nine seconds when the package alarm fired.
This is not evidence that the latter test itself regressed. External Rust and
Go build/test load is retained in `external-load.log`; its causal contribution
is unproven. The terminal summary and full stage logs remain with the run.

Test-only repair `0e9fa028de` and evidence `d7555b695a` are integrated as
`b2edb69131` and `2ce6d4a2c5`. The worker demonstrated RED exit 1 followed by
GREEN exit 0 with both populated and empty ambient variables, retained explicit
empty/nonempty refusal tests, and passed required focused lint with zero issues.
The conductor independently reran `TestInstallTemporal` under the driver
environment on the integrated tree: exit 0, 0.007 seconds. No production
installation, helper, model, runtime, timeout or assertion was relaxed.

The next diagnostic is exactly one isolated ordinary-stage run on the repaired
freeze, preserving the original command, package deadline, 100 fresh instances
and scratch/cache rules. Its actual quiet-host timing must inform scheduling
before another full ladder is attempted; persistent cost justifies targeted
fn-185 profiling, not weakened acceptance. fn-221's full sweep remains an
observation and does not require fn-219 closure or a full qualification pass.
It may proceed on an explicitly bound frozen candidate once the host is free,
without supplying a support claim or completing the failed qualification.

After verifying no live processes, the qualification worker removed the empty
early scratch directory and moved the 6.4 GiB compiler/test scratch recoverably
to `/home/agent/wt/.trash/qualification-scratch-3f532910be`. Evidence, prepared
module and helper inputs remain. The new fn-219 ledger reconciliation worktree
is retained at `/home/agent/wt/fn219-ledger-reconcile` for final failure records.
Untracked `.flow/specs/fn-228*` through `fn-231*` and `tools/gomad/.turbo/` in
the integration checkout are unrelated work and remain untouched.

At the next checkpoint, local `gomad` fast-forwarded from `ed2c85cf73` to
`d1696078e7` and became the integration checkout's current branch. Ancestry and
clean tracked/index state were verified first; fetching from the local `.`
remote performed a fast-forward without changing source, then `git switch
gomad` preserved the unrelated untracked files. Nothing was pushed. The
`gomad-milestones` ref remains at that checkpoint; subsequent integration work
continues on `gomad`. This branch move supplies no qualification acceptance and
does not change the frozen `2ce6d4a2c5` isolated-diagnostic candidate.

## Rust-first continuation

The latest merged order in MILESTONES.md supersedes the isolated Go diagnostic
and the older whole-spec ordering above. The isolated ordinary run on
`2ce6d4a2c5` ended with exit 130 after 501 seconds when the Rust-first decision
was applied; its `interruption.json` is under
`/home/agent/wt/evidence/qualification-final/ordinary-isolated-2ce6d4a2c5`.
No further full Go qualification or Go-only profiling is scheduled.

Owner-authored fn-228 through fn-232 plans are committed in `c36dddcb3e`.
The interleaved gates and explicit pending Rust qualification transfers are
reconciled in `0b55f8bf7a`; no transfer is an execution pass. fn-228's amended
plan passed independent Codex review in round 2 (`gpt-6.1-sol`, high, same
family), session `01a15b20-888a-7f30-8471-1aeaba46cc01`. The receipt is
`.flow/tmp/plan-review-receipt-fn-228-gomad-helper-owned-replay-tape.json`.

Rust replay-plan projection checkpoint `1430d50b01` is integrated as
`8d37d31bc9`. The worker's 44-fixture Go/Rust corpus, two-pass regeneration,
tagged Replay selection and Rust/Go lint passed; exact commands and logs are
indexed by `/home/agent/wt/evidence/fn228/handover/task1-summary.md` and its
evidence JSON. The conductor reran tagged Go choice tests on the integrated
tree: exit 0, 0.025 seconds. All four fn-228 tasks remain in progress until the
combined transport/Runner/oracle batch passes review and acceptance.

The preserved Go behavioral baseline now belongs to open fn-232 at
`/home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a/README.md`. Both hash
manifests pass independent verification; the bundle includes executable/helper
copies and a tracked-source archive, not a qualification or migration claim.

Current command handles: sweep `97951` (owned PID `879280`, resumed journal,
latest reported 70/1488 classifications: 69 pass and one unresolved capacity
observation); fn-232 plan review `23300` (still running, no verdict). Poll these
same handles rather than restarting them. The sweep is shared-host triage,
not quiet-host performance evidence or a support claim.

Open implementation ownership: `replay_tape_impl` owns Rust/WASI transport in
`/home/agent/wt/fn228`, and its `runner_bridge` child owns Go producers/adapters
in `/home/agent/wt/fn228-runner`. Their interface is
`/home/agent/wt/evidence/fn228/handover/batch-interface.md`. `promotion_closeout`
owns the fn-219 ledger/audit continuation in `/home/agent/wt/fn219-closeout`;
the conductor owns review, integration and Flow completion. None may push or
dispatch CI.

Retained worktrees: `fn228` and `fn228-runner` for the active replay batch;
`fn219-closeout` for the ledger continuation; `fn221` for the live sweep;
`qualification-final-frozen` for still-referenced failed/interrupted evidence;
`fn219-ledger-reconcile`, `fn188` and `fn227-map-order` for their still-open
owners' evidence and commits. Remove only after ownership transfer and the
closed-spec checks required by AGENTS.md. Unrelated `tools/gomad/.turbo/`
remains untouched. Nothing has been pushed.
