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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
