# fn-180 stop checkpoint

The owner requested that work stop before fn-180 was completed. All five tasks remain `in_progress`; no implementation review, completion review, `flowctl done`, push, PR or CI operation was performed by this worker. This checkpoint preserves partial source integration and does not qualify the current runtime/helper candidate.

Workspace: `/Users/stephan/Workspace/skunkworks/gomad/temporal`, branch `gomad`. Base: `d087db1e48723d1164c19f559f78d75fc5731eb9`. The checkpoint also contains the conductor's already integrated fn-181 and fn-183 changes; their ownership and acceptance remain separate.

Tier: session (jev-unavailable(no_key)); explicit project implementer routing applied.

## Implemented checkpoints and acceptance limits

| Task / requirement | Source checkpoint | Verification and remaining acceptance |
| --- | --- | --- |
| .1 / R1, R4 | Versioned bounded metadata, strict resolver, exact tape/trace mapping, prefix verifier, neutral replay/CLI request plumbing; partial helper producer and dedicated debug ABI | Model/resolver/architecture tests passed on earlier child revisions. Current Rust producer and Go transport edits have not been compiled. Actual authenticated stop execution remains unqualified. |
| .2 / R1, R4 | Runtime precreates a debug goroutine, preserves pending runq/select decisions, freezes application scheduling, emits built-in stack/heap records and compulsory witnesses; host emits standard heap pprof | Focused runtime source fixtures, linked WASM runtime build and real `go tool pprof` passed before producer integration. Live g0/select handoff, frozen application locks/GC and dedicated inspection termination still require real helper execution. |
| .3 / R2, R3, R4 | Public active/no-op `common/gomaddebug`, fixed registry and typed emission buffers, dormant probes, callback isolation, allocation/overflow fixtures | Ordinary/native no-op tests, active archive build, zero-allocation encoder/overflow tests, vet and disabled-call disassembly passed. Actual enabled replay, allocating-probe witness rejection and target-incarnation isolation remain unqualified. |
| .4 / R5 | Bounded scan/conditional bisect core under the existing WASM cluster owner | Algorithm tests passed before final global-deadline refinement. A production evaluator opening, timing out, destroying and reaping a fresh helper/store for every attempt is not integrated or qualified. |
| .5 / R6 | Bounded Temporal shard/mutable-state snapshots at safe mutation seams, dump/probe registration, helper fixture | Active Temporal compilation/execution and retained failure inspection/replay demonstration have not passed. No retained Temporal artifact was produced by this batch. |

Integrated implementation commits include `61a0dc7002`, `909df29250`, `1da428e7d6`, `814cccf732`, `228471daa3`, `057f8198ba` and `f88c016bac`. The unverified producer is preserved by the subsequent stop-checkpoint commit. Child branches and handovers preserve their original identities:

- Runtime: `fn180-runtime`, commits `30cdd5a91a` and `41c8ff3582`; `/tmp/fn180-runtime-summary.md`, `/tmp/fn180-runtime-evidence.json`.
- Library/Temporal source: `fn180-library`, commit `309f1a367f`; `/tmp/fn180-library-summary.md`, `/tmp/fn180-library-evidence.json`.
- Metadata/search/neutral plumbing: `fn180-evidence`; final source-only checkpoint `012d434b06c2c11c69a469313cb3e2a07d464de1` is not integrated into main. It contains queued Runner dispatch/refusal tests, which have not been compiled or run. Its preceding `8d03a6ab2a` checkpoint is already integrated as `f88c016bac`. Handover: `/tmp/fn180-evidence-owner-summary.md` and `/tmp/fn180-evidence-owner-evidence.json`.

## Current partial integration

The dedicated `gomad_debug_v1.exchange` path bypasses ordinary runtime dispatch, model call counters and `close_clock_run`. The new Rust `model/inspection.rs` retains versioned raw streams, registration descriptors, decision boundaries, runtime witnesses and shared cuts under byte limits. Rust state/protocol and Go process transport changes are partial, source-formatted and uncompiled. Ordinary evidence is unchanged when no inspection metadata exists; current instrumented standalone recordings use a versioned modeled-evidence envelope containing the metadata, while debug output is a separate non-evidence stream.

The next implementer must finish and verify, at minimum:

1. Compile the Rust producer and Go transport against the combined source, resolving any type/borrow/API errors. Formatting is not compilation evidence.
2. Finish `ClusterSession.collect`/`finalizeEvidence` handling of dedicated inspection terminals and global `inspection` bytes. The partial code receives the stream but does not yet bind it into the final `ClusterResult`. Normal full-tape and final World checks must remain unchanged for ordinary replay.
3. Complete cluster shutdown semantics: `inspected_prefix` must abort and destroy other fresh stores without being reclassified as infrastructure failure or demanding ordinary lifecycle/full-tape completion. Preserve delayed/never-started stores, holds, streamed terminals and Finish.
4. Implement the actual optional WASM `InspectionProvider.Inspect` and retained standalone/cluster replay bridge. CLI/Runner options currently reach an optional provider interface; the provider implementation and concrete control adapter were discussed but not written. Preserve the queued child tests before continuing. Clone the newly added diagnostic `Result.DebugOutput` bytes in `Provider.CompletedResults` as well.
5. Validate helper control construction from authenticated retained raw bytes, including all stream-prefix hashes, exact available registration/closure prefixes, all-incarnation witnesses, and clock-run cuts. No final-only hash or replay-claimed prefix may authorize inspection.
6. Update neighbor network cut material immediately before broker handoff after serving `net_stream`; a pre-call snapshot alone does not retain that operation's local network evidence. Finish lifecycle/order coverage and final-cut accounting, including unexecuted incarnations and capacity/incomplete metadata.
7. Finish fuel-pressure classification and bound/cancel/error handling for nonreturning callbacks and invariant checks; no timeout/panic/divergence may become a predicate result. Whole-search deadline exhaustion must report incomplete, distinct from a predicate-local timeout.
8. Run the live helper fixtures: every admitted decision kind; select observations versus tape ordinals; diagnostics-off allocating-probe divergence; altered nonselected guest, World, stdout/stderr and storage/recovery prefixes; stops inside folded clock runs; restart/never-started and terminated-incarnation cases; dump panic/truncation/timeout; frozen-lock dumps and GC.
9. Run final host-source/lint/architecture, `make -C tools/gomad validate`, Rust/helper/model suites, full `test-wasm`, current-candidate cooperative runtime qualification and the retained Temporal failure demonstration, followed by an independent integrated review. Native execution remains deferred.

## Commands and evidence

Baseline: none; fn-180 has no explicit Quick commands and the milestone batch policy owns integrated final gates.

| Command / observation | Exit | Scope |
| --- | --- | --- |
| Pinned Go `test -tags test_dep ./internal/backends/wasm/toolchain ./internal/backends/wasm/wasi -run '^TestDebug\|TestOverlayBuildsLinkedWASMRuntime' -count=1` | 0 | Child runtime/decoder revision `41c8ff3582`, before final target-field assignment and current producer integration. Includes a linked runtime and actual pprof consumption. |
| Pinned Go `test -tags test_dep -count=1 ./common/gomaddebug` | 0 | Library child, including primitive and overflow zero-allocation checks. New package test first failed with exit 1. |
| Pinned Go `test -tags 'gomad test_dep' -count=1 ./common/gomaddebug` | 0 | Native no-op source set. |
| `GOOS=wasip1 GOARCH=wasm` pinned Go `build -tags 'gomad test_dep' ./common/gomaddebug` | 0 | Active archive only; not runtime execution. |
| Pinned Go `vet -tags test_dep ./common/gomaddebug` | 0 | Library child. |
| Pinned Go `test -tags test_dep ./internal/model/cluster ./internal/archcheck -count=1` | 1, then 0 | Initial forbidden owner edges were refactored; successful architecture run took 74.649s. Before final guards, CLI plumbing and producer/control edits. Logs `/tmp/fn180-evidence-model-architecture.log` and `/tmp/fn180-evidence-model-architecture-final.log`. |
| `cargo test --locked --release debug_exchange_preserves_open_clock_run_and_canonical_state` with shared target, first invocation | 0, inconclusive | Collected zero tests because Cargo reused another worktree's crate binary. This is not a pass. |
| Same Cargo test after touching current `src/main.rs` to force the crate rebuild | 101, intended red | Missing `Model.debug_exchange`, log `/tmp/fn180-debug-red.log`. No post-implementation Cargo test was run before stop. |
| Temporal history package test started by library child | 130, inconclusive | Cancelled to honor shared build ownership; `/tmp/fn180-temporal-baseline.log`. No active Temporal test pass. |
| `gofmt`, `rustfmt --edition 2024`, `git diff --check` at stop | 0 | Source formatting and whitespace only. No tests/builds were started during the conductor's fn-182 timing window. |

The shared Cargo target is `/tmp/gomad-fn178-batch-target-20261015`. A worktree switch can incorrectly reuse the root crate binary when timestamps are older than the last build. Force the current root crate to rebuild and check the selected test count and source provenance before accepting a gate; dependencies can remain shared. Keep future outputs in `/tmp`; the workspace filesystem had approximately 13 GiB free at the last conductor observation.

Final handover pointers: `/tmp/fn180-summary.md` and `/tmp/fn180-evidence.json`. No completion or review verdict is claimed.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate; owner requested stop before review)
