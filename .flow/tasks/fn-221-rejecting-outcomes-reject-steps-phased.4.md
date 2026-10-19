---
satisfies: [R4]
---
# fn-221-rejecting-outcomes-reject-steps-phased.4 Live cancel scenarios, gates and records

## Description
Add two real RPC scenarios to the existing in-process server/SDK slice: cancel after completion (actual NotFound error judged as `Client::Cancel` / `Rejected(NotFound)`, unchanged phase) and cancel before the worker starts (actual successful response judged as `Client::Cancel` / `Accepted`, then actual Canceled Describe as a zero-step current match). Mutate a copy of the actual observed error code and require Unexplained with exactly `Client::Cancel` expected, followed by frozen-run evidence. Run slice, model and boundary gates. Record the first evidence-driven DSL expansion and measured latency in DESIGN and update the realization README with cancel projection, local Error descriptor choice, and the Started restriction.

**Size:** S
**Touches:** [tests/testpilot_model_slice_test.go, model/DESIGN.md, model/temporal/realization/README.md]

### Approach
- Read `.flow/tmp/fn2214-preflight.md` and revalidate current code. Existing context-`Cancellation` scenario releases a blocked callback; retain it and add distinct RPC scenarios. Preserve Complete, Fail and HostRefusal, the real SDK worker, the generic transport-only host, independent hosts, callback cleanup, freeze/reset and no-duplicate-body checks.
- Judge the actual RPC response or actual error code/message, not a response/status synthesized from Model expectations. Use fresh request IDs so this does not test excluded deduplication semantics. No Model or realization changes to make the live run agree; a real conformance disagreement must remain evidence and be reported to the conductor.
- Submit the before-start cancel while the worker is still stopped and prove zero activity-body calls; don't exercise Started cancellation or introduce CancelRequested into the six-phase Model. Submit the after-completion cancel before any existing impossible-observation control freezes the host.
- Assert parallel step/outcome vectors: Accepted and Rejected(NotFound) each carry one Client::Cancel step; zero-step Describe carries neither. After rejected cancel, a genuine Describe must still match the same terminal phase. A mutated recognized status must be Unexplained and freeze on subsequent genuine observations, not a malformed/unsupported refusal.
- Keep numeric RPC mapping owned by the Temporal binding. Tests may use the RPC library's status codes to inspect actual errors; generic host implementation must not import Temporal types. Use existing in-process environment and SDK patterns, not mocks or a second host loop.
- Record actual observed commands, toolchain/CGO configuration and measured Answer/Judge/live-slice latencies without inventing a scaling or general conformance claim. Preserve prior deferred R33–R40/Case obligations and documented six-phase limitations.
- Verify `make model-check`, `make umpire-check-model-slice`, configured parent direct Go selector, and nonfixing exact-baseline Go lint. Existing Make gate includes boundary controls; retain direct boundary evidence as appropriate. Only root updates MILESTONES at handoff; it is not a worker write surface.
## Acceptance
- [ ] Both live scenarios and the mutation control pass under `make umpire-check-model-slice`.
- [ ] `make model-check` and the boundary gate pass; DESIGN and README updated.
- [ ] No Model change made to fit runtime evidence.

## Done summary
The live SDK/server slice now judges cancellation after completion as one Client::Cancel with Rejected(NotFound), and cancellation before worker startup as one Client::Cancel with Accepted followed by a zero-step Canceled Describe. A copied real error mutated to FailedPrecondition is Unexplained with exactly Client::Cancel expected and freezes subsequent genuine observations; the independent Model, realization and generic host were unchanged.

Complete, Fail, HostRefusal and context Cancellation remain present, with callback cleanup, reset, host isolation and no-duplicate-body controls. Before-start cancellation keeps the worker stopped and activity-body calls at zero. Both new RPCs use fresh request IDs and carry actual returned response/error values. DESIGN records the first evidence-driven rejecting-outcome expansion, Phase/Phased laws, local Error descriptor choice and measured Answer/Judge/live timings. The realization README documents cancellation projection, serviceerror conversion and the live Started restriction. Deferred R33-R40 and Case obligations remain open; no general conformance claim was added. No mapped user route changed; the feature map is absent.

baseline: green. All Quick commands ran before edits without a handoff or honored receipt. Logs are baseline-model.log, baseline-slice.log, baseline-live.log and baseline-lint.log under .flow/tmp/fn2214. Both newly registered scenario names first failed through the old unsupported callback-policy path before the scenario glue was added (red-scenarios.log). The initial adapter run failed because status.FromError did not recognize the frontend client's serviceerror (green-slice.log, exit 2); replacing that assumption with the repository's serviceerror.ToStatus conversion preserves the actual RPC code/message. This was an adapter correction, with no Model or realization adjustment.

Final verification passed make model-check, make umpire-check-model-slice, the configured direct Go selector and GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=dc9fe22cc4 make lint-code-fast. The direct boundary command also passed both boundary_controls and generic_source_boundary (2 tests). Evidence logs are verify-model.log, green-slice-status.log, verify-live.log, verify-lint.log and verify-boundary.log under .flow/tmp/fn2214. Nonfixing lint reported zero issues. The live gate reports all six scenarios passed. The first successful expanded live run took 339.680626 ms including cleanup, excluding compilation, on linux/arm64 with Go 1.27.0, Rust 1.99.1, SDK 1.48.0 and CGO_ENABLED=0, using /home/agent/.cache/codex-build/cargo-target. DESIGN contains individual Answer/Judge samples; these observations make no scaling or percentile claim.

The first formal review returned three SHIP draws with zero findings, explicitly executing codex:gpt-6.1-sol:high for correctness, contracts and integration, from the writer's model family. Reviewer execution was blocked by the read-only filesystem; worker gate logs supply execution evidence. The finalized receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-221-rejecting-outcomes-reject-steps-phased.4.json and draw evidence is .flow/review-fanout/af7ec212973548fc9ab50ba1f5da8199/. No fix loop or memory capture was needed. Final cumulative classification required full gates; their verified post-change receipts were honored at 846b26bd.

GATE_SKIPPED:unittest:green-receipt 846b26bd - verified post-change model gate reused at final verification
GATE_SKIPPED:smoke:green-receipt 846b26bd - verified post-change slice gate reused at final verification
GATE_SKIPPED:live:green-receipt 846b26bd - verified configured direct Go selector reused at final verification

stage: impl-review - ran [2026-10-19T21:45:32Z..2026-10-19T21:46:54Z] (three SHIP draws; model: gpt-6.1-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 846b26bd5e5881238d9cbdc99077e3d60f3bd4d1
- Tests: make model-check, make umpire-check-model-slice, UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=dc9fe22cc4 make lint-code-fast, cd model && mise exec -- cargo test -p umpire-temporal-tests --test boundary, GATE_SKIPPED:unittest:green-receipt 846b26bd - verified post-change model gate reused at final verification, GATE_SKIPPED:smoke:green-receipt 846b26bd - verified post-change slice gate reused at final verification, GATE_SKIPPED:live:green-receipt 846b26bd - verified configured direct Go selector reused at final verification
- PRs: