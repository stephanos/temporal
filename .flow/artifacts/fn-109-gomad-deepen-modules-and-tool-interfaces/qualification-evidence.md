# Current qualification evidence

Task 21 retains a complete finding matrix, an R18 preservation audit and matched
10/100 developmental campaigns. Final acceptance remains incomplete. Neither
qualified native host is available, preservation inventory reconciliation remains
open, and task 19's formal review never dispatched. Root owns formal review,
commits and Flow lifecycle. The task remains `in_progress` for that handover.

## Candidate and baseline

Current committed source is `8604c07def0f97b63cbca3864b4c286d6803c4b1`.
No production implementation changed during task 21. The complete 978 shipped
nested-module paths and their SHA256/mode values are frozen in
[current source inventory](task-21/current-measurement/runs/shipped-source-before-overlay.json).
The fixture scratch adds four paths and reuses the fifth declared supplementary
Linux helper byte-identically, producing 982 paths. Every campaign and final
inventory checked the complete path set, hashes and modes.

The actual first-task baseline is `6782b55f49a0317b230e827ea2a63a37d116d502`
plus retained dirty fn-108.2-.6 work. It is not planning commit `d4d800fb47` or
the bare first-task commit. [Reconstruction](task-21/baseline-reconstruction/reconstruction.md)
retains 670 original paths, source manifest SHA256
`d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`
and input manifest SHA256
`5b1904a15a5e11dd93926465d97c5467ab2a008f48b20610d9b0ffcfa4e8d0e6`.
The later `38957053f1ce342a8797af1803f5f8f6bb53fcad` nested tree is
independent full-blob/mode corroboration, not a whole-repository replacement.
The historical task-description-only metadata drift remains disclosed and the
historical input/source manifests remain immutable.

[Bound baseline](task-21/bound-baseline-measurement/measurement.md) records the
environment before builds. Its immutable 201-entry handoff-output manifest SHA256
is `d1381842f5eb1b8a1a4b1b5b24d0d54544f61ab30cfea311647e65225abc1133`.
The current verifier freshly checked every entry. There is no historical
`driverhandoff.sha256`; the actual manifest name above is authoritative.

The host is Linux aarch64. Patched `tools/gomad3/.toolchain/bin/go` is absent;
the pinned Go1.27.1 source archive exists. Stock Go1.27.1 for Linux/arm64 has
SHA256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.
Stock-host and cross-source checks establish developmental source evidence.
They do not qualify Darwin/arm64 or Linux/amd64, native defaults, process
isolation, patched-runtime consumption, or exact replay.

## Preservation and cross-spec changes

[Preservation audit](task-21/preservation-audit/report.md) retains baseline/current API
inventories, CLI/flag source inventories, deleted-comment spot checks, exact
commands and source hashes. [Intentional fn-109 migrations](go-interface-changes.md)
remain the allowed source changes. The audit compares all eight requested
surfaces and separately records changes integrated from other specs.

R18 remains unproven. Current `runner/campaign_options.go` exports additional
normalization/validation parsers and error types beyond the two parsers recorded
in the interface inventory; its task-2 paragraph still claims no exported
changes. The omitted declarations are `NotSingleBaseSeedError`,
`SemanticCoverageRequiredError`, their `Error` methods, `NormalizeStrategy`,
`NormalizeCoverage`, `ValidateCoverage`, `ValidateChoiceTraceLimit`,
`ValidateChoiceCoverage` and `ParseSingleBaseSeed`. They trace to WIP commit
`a3b9f80efab9356c0be2080779133337e2471ac0`; exact task ownership is unresolved.
The two inventoried task-5 parsers trace separately to `f608449c01`.
Owning-task provenance and each exact declaration are retained in the audit
rather than retroactively editing the inventory to report success.
Additional diagnostics, guidance, target-sharing, select-readiness, maintenance
and soak APIs need their separately owned provenance linked.

Choice trace v2 versus v3 is a recorded protocol change, distinct from changed
toolchain or controller identities. Fn-114.11 explicitly requires a raised wire
version and visible old/new-version rejection; fn-114.12 requires old controller
journal rejection. Their source commits are identified in the audit. Those
independent decisions explain the current behavior but do not silently rewrite
fn-109's recorded-format preservation criterion. The final preservation record
must reconcile this scope before acceptance.

Boundary-manifest bytes match the reconstructed baseline. Compatibility-pack
differences are inventoried; the scout found no generic import/capability grant.
Comment spot checks separate exact moved text from reflowed/reworded text.
Literal known-error/report projections and source-owned equivalence checks are
retained with their source identities. An unchanged Go signature, constructor
alias or skipped Darwin snapshot test cannot certify all fixed-identity bytes
on both platforms. The fresh pinned-stock preservation command selected 18
top-level tests and 55 cases, skipped none and exited 0 in 0.698637 seconds;
[its receipt](task-21/preservation-audit/focused-preservation-receipt.json)
retains exact argv, output and before/after source bindings. Current native
canonical/replay proof remains open. Nine pre-existing registered flags are
absent from the literal CLI guide inventory, as the audit records.

## Bounded developmental comparison

[Current campaigns](task-21/current-measurement/measurement.md) completed four
serialized cases and companion controls with 98 successful commands. The driver
hash was frozen before launch and stayed unchanged. All 14 explicit environment
bindings and binary build-setting records match the reviewed baseline; additional
compiler controls were not assumed to have historical prebindings.

Both-role named logical storage is 4120 baseline versus 4472 current bytes at
both 10 and 100 jobs. The 352-byte increase belongs to four completion slots.
At each paired checkpoint both sides retain exactly four 1MiB streams and two
1MiB encoded transcripts, with zero at completion. Per-execution producer
allocations match, with site-specific allocation/category differences retained.
Publication now allocates nine versus eight 64KiB streaming buffers per one
novel artifact. Raw stacks assign the additional buffer to `verifySharedPayload`
hashing the fixture's 20-byte shared target, introduced by fn-114.9 commit
`bc2e970b5306aa09f594657a8d42c159cf4a1270`. Six inline writes, manifest write
and source-target copy remain the other eight buffers on both sides. This
extra filesystem verification read is neither an observed 1MiB heap clone
nor a measured performance gain or universal absence-of-copy proof.
The measurement report records the private-novelty, attribution, allocator,
transport-guard and real-process limits. R19 still requires native execution.

## Required native gates

[native-command-ledger.json](task-21/native-command-ledger.json) retains 89
workflow/required/Make-recipe command rows with exact command strings, cwd, source line,
step, workflow hashes, platform and explicit incomplete dispositions. Every
native result below is incomplete on both platforms. No command that did not
run carries a synthetic exit code or a pass. Execute the native gates serially
within each platform against one frozen source and pinned patched build, since
they share toolchain/cache and qualification directories.

| Required command from repository root | Darwin/arm64 | Linux/amd64 |
| --- | --- | --- |
| `make -C tools/gomad3 validate` | incomplete | incomplete |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test` | incomplete | incomplete |
| `go test -count=1 -tags test_dep ./tools/gomad3sim/...` | incomplete | incomplete |
| `make gomad3-runner` | incomplete | incomplete |
| `make gomad3-integration-test` | incomplete | incomplete |
| `make gomad3-smoke-qualification` | incomplete | incomplete |
| `make -C tools/gomad3 compatibility-pack-qualification core-qualification-set` | incomplete | incomplete |
| `make gomad3-qualification` for affected Temporal workloads | incomplete | incomplete |
| `tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim` | incomplete | incomplete |
| `make lint-code-fast` on the qualified host | incomplete | incomplete |

Full `test` covers eleven targets. The ledger lists each individually so the
Linux CI group and Darwin dossier cannot hide an unexecuted tier.

```text
test-harness test-toolchain intercept-test test-host overlay-test
test-simulation world-test test-builder test-live-capability
test-runtime test-upstream
```

The Make-defined overlay suite includes `internal/gomadfs`, `internal/gomadwire`,
`internal/gomadchoicewire`, `internal/gomadmodelwire`, `internal/gomadsim`,
`internal/gomadio`, `internal/gomadio/mount`, `os`, `cmd/internal/gomadcap`.
The canonical simulation target uses its seeded `-exec` wrapper and runs the
Runner transport selection for network/filesystem/process cases, then the
separate forward node-clock regression. The bare task Quick command above
cannot replace that canonical target. The original strict-delay watchdog
finding remains a separate open source-owned finding.

Both platforms must retain functional closure analysis using
`tools/gomad3/.bin/gomad analyze --capability-mode=closure --format=json
--timeout=15m --build-tag disable_grpc_modules --build-tag gomad --build-tag
test_dep go-test ./tests`, with zero blockers and the platform's exact functional
pack. Darwin additionally requires `make -C tools/gomad3 upgrade-dossier`
under the workflow's bound `GOMAD3_BASELINE_REF` and any explicit approved
boundary digest. The dossier executes the privileged
`make -C tools/gomad3 clock-audit`; full `test` alone does not cover it.
Linux has no dynamic DTrace gate and does not reopen deferred D11.

The exact workflow jq report assertions are stored as command strings in the
ledger. Core requires seven qualified workloads at seed 17 with exact replay.
Darwin smoke requires four qualified workloads at seed 11 with zero divergence
and exact replay; Linux preserves its named D12 intermittent dispositions.
Representative Darwin requires all 28 supported/qualified on seeds 11/17;
Linux preserves ten named unsupported boundaries and eighteen supported-or-failed
workloads under unchanged classifications. Darwin's explicit smoke tracing
manifest assertion is a separate row.

Scheduled/dispatched soak commands and their report/ledger administration are
recorded separately as fn-112 obligations. Linux soak remains informational
under D12. Full generated `./tests` qualification remains on demand under the
milestone contract; smoke plus affected suites are required here. A stand-in
soak, cross-build, stock source test or historical report supplies no native
qualification. Missing native report execution is not an observed D12/D14
failure. Preserve the existing owner dispositions; attribute an actual future
divergence with retained evidence rather than weaken an expectation.

The native ledger also hashes six current qualification/disposition JSON sources
and notes whether a core report exists. Historical Git-base blobs are identified
as projections only, since the dirty full repository was not reconstructed.
No current source-bound core report was admitted as qualification evidence.

## Feasible checks and gate honesty

Before edits, `make -C tools/gomad3 validate` under pinned stock Go1.27.1 exited
0 and Flow validation exited 0 (22 tasks). Logs are retained from
`.flow/tmp/task21-baseline-*`. These are developmental generator/administrative
results and are different from both native full-suite Quick commands. No unlike
baseline handoff or green receipt was applied.

Pre-edit `make lint-code-fast GOLANGCI_LINT_FIX=false` exited 2. The existing
`.bin/golangci-lint-v2.13.0` has Mach-O arm64 magic and cannot execute on this
Linux host; the original log remains unchanged. Root installed the same pinned
Linux tool versions into an isolated LOCALBIN and ran the unchanged target with
fixing disabled. That rerun also exited 2 (golangci exit 7) after 147.354 seconds;
the root loader rejects nested-module/runtime-overlay package inputs. The
[tooling diagnosis](task-21/lint-tooling-diagnostic.md) and raw logs retain both
failures and exact tool identities. The pre-edit lint baseline is red; no later
narrow check can convert it into a full-target pass. No formatter/fixer changed
production source.

The matched current driver exited 0; `verify_current.py` exited 0 after recording
the actual 352-byte delta. Original preflight/initial-verifier failures remain
retained. The independent measurement scout recalculated all 32 raw profile
metric/site/size-bucket sets without rerunning campaigns. Source and output
hash verification covers local bulk artifacts; a new checkout must reproduce
those local outputs. Administrative Flow validation supplies no runtime result.

## Acceptance still owed

The [completion matrix](completion-matrix.md) maps exactly F1-F11 and S1-S5,
reuses verified fn-108.5/.6/.7 for R2/R3 and assigns D1-D5 exactly once. Fn-108.8
retains Linux qualification; fn-105.3/.4/.5 remain blocked in current Flow state.
Task 20 has formal SHIP for guidance, while inherited D5 native acceptance
remains open. Task 19 has a committed reviewed source candidate and no formal
verdict after its predispatch failure.

Before task/spec acceptance, the conductor must reconcile each R18 API/CLI/
format provenance gap with its actual owner, obtain task-19 and task-21 formal
review, record both native integrated command/report results with unchanged
dispositions, and complete the owning transferred acceptance. The current
matched fixture supplies bounded developmental numbers and explicitly scoped
copy-site evidence. It does not waive any original native or preservation
requirement. Implementation gaps return to their owning task.
