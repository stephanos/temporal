# fn-222.2 Go support dispositions, 2026-10-20

Disposition: **pending; zero qualified rows**. Every test is unsupported on the
current candidate. This report is not a `gomad.wasm-support-list/v1` publication
and contains no qualified Result references. The producer refuses empty lists.
No new guest preparation, execution, measured cell or qualification ladder was
run for this report. Ordinary producer unit tests supply source coverage only.

The inspected source checkpoint is `1e94b19ce25b5f41701fdecb1eda26388be5922f`.
No integrated qualification source manifest, prepared module or matching final
success/canary Result and PublicationReceipt pair is available for that
checkpoint. Historical identities below remain historical; they do not identify
the new fn-228 helper or the eventual Rust binary.

## Required selections and current gates

| Registered cell | Package | Named test | Disposition |
| --- | --- | --- | --- |
| `onebox/unchanged-workflow-type-smoke` | `./tests` | `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` | Pending all matching-candidate gates |
| `onebox/frontend-system-info` | `./tests/gomadfunctional` | `TestFrontendSystemInfo` | Pending all matching-candidate gates |

For each row, `unchanged-source-preparation`, `measured-pass-replay-success`,
`measured-pass-replay-canary` and `storage-transport-inspection` have **no
current-candidate exit**. Missing exits are not zero. Neither row has final
success/canary Results or authenticated named-test PASS witnesses. A sweep pass
cannot supply any missing input.

The [qualification contract](../../tools/gomad/internal/backends/wasm/QUALIFICATION.md#one-box-cell-gates)
defines the exact Go semantic-reference commands and environments. From
`tools/gomad/internal/backends/wasm`, their test command suffixes are:

```sh
# Definitions only; none was executed for the inspected checkpoint.
go test -v -tags test_dep -count=1 -timeout=90m -run '^TestActualOneboxQualification$' ./qualification
go test -v -tags test_dep -count=1 -timeout=90m -run '^TestActualOneboxFrontendSystemInfoQualification$' ./qualification
go test -v -tags test_dep -count=1 -timeout=100m -run '^TestTemporalInspectionRetainedFailure$/^onebox$' ./backend
```

Preparation sets `GOMAD_WASM_QUALIFICATION_PREPARE_ONLY=1` and
`GOMAD_WASM_QUALIFICATION_CELL=success`. Measured modes omit prepare-only and
select success then canary in separate fresh processes. Only measured smoke
sets `GOMAD_WASM_QUALIFY_WORKLOADS=1`; frontend refuses it and uses its own root.
Each row's Results and terminal receipts must validate independently. Preserve
each preparation receipt before later modes rewrite it and compare its module
with both final Results. The shared inspection gate does not establish either
row's own measured test execution.

The canonical guest argv is exactly `gomad-target -test.v -test.count=1
-test.timeout=72h -test.run=<selection> -persistenceType=sql
-persistenceDriver=sqlite`. Smoke's success selection is
`^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`; frontend's is
`^TestFrontendSystemInfo$`; both canaries use `[`. No extra flag is accepted.
Every success execution must authenticate complete retained stdout containing
exactly one named PASS and no named SKIP. Wrapper PASS, no-tests, listing,
zero-count and skip suppression establish no support.

## Retained historical observations

All relative evidence paths in this section resolve under the open fn-232 owner:
`/home/agent/wt/evidence/fn232/fn219-go-attempts-2026-10-20/`.
Its `inputs.sha256` authenticated all 172 retained inputs with exit 0 during this
preparatory phase. Original paths embedded in receipts are provenance; use the
independent retained copies. Nothing below is relabeled as a matching-current
gate.

| Attempt | Recorded numeric exits | Retained reference / disposition |
| --- | --- | --- |
| `ea45f94e5913d27a9e03c8312a86614e807eb67c` | Helper build 0; initial preparation 1; private-copy preparation retry 0 | `early-onebox-ea45f94e59/attempt.json`; helper link-count admission failure repaired for preparation only; full ladder not started |
| `04e06bc95f958e1957a8de04f6bef680b6515121` | Preparation 0; initial entrypoint 127; corrected driver/scheduler 130 | `qual-04e06bc95f/interruption.json`; stale source pins, zero completed stages |
| `8cd478404b` | Driver/scheduler 1; root lint 2; test-harness 2 | `qual-8cd478404b/wall-time.json` and stage receipts; 15 passed stages, no qualification claim |
| `3f532910be73f622e923273bff86c401645fe7ba` | Preparation 0; inspection 0; ordinary command 2; driver/scheduler 1 | `qual-3f532910be/terminal-summary.json`; 31 passed stages, 64 unrun, no measured workload or runtime qualification cells |
| `2ce6d4a2c5d1a18a6ed22439eb751fda6fd4d4f9` | Initial isolated-stage setup 2; interrupted terminal 130; partial ordinary receipt 2 | `ordinary-isolated-2ce6d4a2c5/interruption.json`; owner stopped obsolete Go-host diagnosis for Rust-first ordering, no full suite pass |

The 3f53 source manifest digest is
`sha256:39182df082494be677778bdbd642b4812defef546b81189bf7a2de36d52a2fba`
on Linux/aarch64. Its `early-onebox-3f532910be/prepare-only-producer.json` records:

| Identity | Historical value |
| --- | --- |
| Compiler | `sha256:1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64` (Go 1.27.1) |
| Prepared module | `sha256:4e19470a4f6f68083f6bea28d3ff61d7ae7317d2799fdde7185e37e42506da78` |
| Preparation helper | `sha256:8956496039ed213d2c1d0e88266bb2f7a9f2c9916e299ae8b734387f2a6dbe5f` |
| Admitted ladder helper | `sha256:b25439140f0c850a82f99471c9fd48ac864569a75e407a76850873f6ece055a1` |
| Model | `sha256:73c1a4f3f64dedd4c18dad47849f6dcc8bc04d08e27067dedd096acde05e2e60` |
| Runtime overlay | `sha256:6453d2a998662888534e8641ff68e6411e6adc14d5315ca7ee873b8a7ce8e506` |
| Preparation BuildKey | `e6f427eda85c548890f7b084fadb53b33ef8fa92b7e49cf6ece42347de4bae81` |

Preparation and admitted helpers are distinct identities. The preparation
command/environment is retained in `run-current.sh` and `run-3f532910be.log`.
The actual inspection command is retained in
`qual-3f532910be/temporal-inspection.command`: stock Go `test -v -tags test_dep
-count=1 -timeout=90m -run '^TestTemporalInspectionRetainedFailure$' ./backend`
from the historical WASM directory. Its receipt records exit 0 and the same
3f53 source manifest. This historical full inspection gate includes onebox;
neither its exit nor preparation can fill the missing measured Results.

The independent behavioral baseline at
`/home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a/README.md` binds Go binary
source `1d7b37f00a74a85399993f1edcad6fe742e778e3`, launcher hash
`407263f9be007312375e77462cc38d92eac333f28d72d79312caf7d2a78c5239`
and the admitted `b2543914…` helper. It is a fresh-comparison input, not a
qualified support list. Old Go artifacts are not Rust inputs.

Content hashes (SHA-256) for independently checking this report's inputs:

| Retained path | SHA-256 |
| --- | --- |
| `inputs.sha256` | `4d93f3020112b4e0541bf468601ec53db5df8a8019168e81d1b684ca666529f7` |
| `qual-3f532910be/terminal-summary.json` | `9a996b98000e9f9c2402d57854a7d0b265243af4f68fe159e43c8529eaa383a2` |
| `early-onebox-3f532910be/prepare-only-producer.json` | `bf25a8666c016e62a5c35ba55d05f762219d608559d2a3cb7772e9b715269edc` |

## Remaining final Rust acceptance

Task fn-222.1 retains R1 and final R2 after fn-232.10. Freeze the integrated Rust
candidate after identity-changing work lands, qualify both registered selections
with all four gates, authenticate actual named PASS and replayable canary
evidence, then retain the producer's JSON/Markdown unchanged beside the final
checkpoint. A smoke-only interim list cannot complete R2. Every changed
candidate starts with no earned rows; every unlisted test remains unsupported.
