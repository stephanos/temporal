# fn-228.6 review: retire legacy Testpilot, verify replacement coverage and size (R6)

Commits reviewed: 96d635bc17, 7a2eb2458b, 166d864a6f, e689481277, da7c84d64d (branch `umpire`;
base be5103b6bc). HEAD 9c7c78be93 differs from da7c84d64d outside `.flow` only in MILESTONES.md.

## Verdict: NEEDS_WORK

The retirement itself is sound. The coverage map is complete and honest, the line counts are
correct, and nothing was moved out of the counted subtree to meet them. Two canary safety issues
need fixing before this ships. The fence can be bypassed through the read-instruction envelope,
and nothing checks the policy's `modelHash` pin against the committed guest. The rest is minor or
nit.

## What I verified (no finding)

- **The seven legacy failures.** All seven are gone: no `func TestTestpilot{AssessRecordedRuns,...}`
  remains. Their names match `.flow/tmp/fn-216-4/legacy-comparison.md`, and the four pre-existing
  live failures from fn-228.3 are among them.
- **The coverage map** (`.flow/artifacts/fn-228.6/coverage-map.md`, tracked, identical to the tmp
  copy). Every retired live test function I listed from 96d635bc17^ appears in it: 25 functions
  across 13 files. Each gap is stated explicitly: Nexus, Workflow, workflow-scheduled activities,
  heartbeats, attempt timeout and IR-bridge minimization.
- **Cited test names.** Every one of the 30 replacement tests the map cites exists, along with the
  slice and canary subtests.
- **Line counts.** I recomputed them from git at base and head.
  - Testpilot: 32,240 before, 1,706 after (21,506 + 10,734, then 36 + 1,670).
  - Runner: 1,611 before and after.
  - Canary: 3,848 before, 3,010 after.
  - tools/umpire: 29,726 before, 17,088 after.
  - The excluded generated `v1` code totals 5,734, as claimed.
  - The script (`count.sh`) excludes only `_test.go` and `testdata`; the v1 exclusion is applied by
    hand in the totals and is correct.
  - The only new non-test Go file is `tools/canary/cmd/umpire-canary/path.go`: 43 lines moved
    verbatim from `testpilot/publish/path.go` and counted in the canary total.
- **Targets.** Generic is 1,647 against at most 5,000. The total is 3,317, or 6,387 with the
  canary and cmd/umpire, against at most 11,000. Both are met.
- **Dependencies.** cel-go and antlr are gone from go.mod and nothing was added. `cel.dev/expr`
  stays for `api/testpilot/v1`.
- **model/umpire.** Nothing under model/umpire changed, so nothing Temporal-specific leaked there.
- **Imports and build.**
  - No importer of a deleted package remains in Go code.
  - Make targets and the CI workflow name no deleted target or path.
  - The `umpire` Make target exists.
- **Tests and vet.**
  - `go test -tags test_dep ./common/testing/testpilot/... ./tools/canary/...` passes, with
    CGO_ENABLED=0.
  - `go vet` is clean under `test_dep`, `test_dep canary_harness` and `test_dep integration`
    (./tests).
  - The two ir tests the handoff fixed pass. The ir `Layout` failures (TestFrameworkLayout,
    TestNexusFormsLayout, TestFoundationsAndActorsLayout) are red at base too
    (`baseline-unit.log`).
- **The canary rewrite.**
  - Publication goes through `runner.Publish` (controller.go:294-299).
  - Preflight hashes the same bytes the Runner executes (`Scope.Guest`), so there is no TOCTOU.
  - The modelHash in production-canary.json equals the sha256 of the committed
    `umpire_temporal_guest.wasm` today (757fd2e8...).
  - Lease, fence signal, stale-fence and cleanup semantics carry over from the workflow design and
    are unit-tested, including the stale fence and unverified-close cases.
  - The Transport holds credentials only in memory, and I found no printing of Options.
- **Retiring tools/umpire/lint is justified.**
  - Its only caller, `umpire-lint`, imported `lower`.
  - Its IR input (`model/ir`, `model/temporal/features`) no longer exists.
  - The package was red at base.

## Findings

1. **major: the fence is bypassable through the read-instruction envelope, and through starts not
   named `Start*`.**
   - **Where:** tools/canary/controller/fenced.go:84-99 (claim at :42-44 and :17).
   - **The cause:** `within` inspects only the outer `payload.MessageName()`. The Temporal Driver
     also accepts `umpire.temporal.v1.ObserveActivity`, whose inner `Request` can be any
     WorkflowService request (act/exchange.go:289-306, sent repeatedly by `read`, exchange.go:199+).
   - **Proof:** an overlay probe test (no source change) gave `fenced(...).Start` two payloads. Both
     reached the wrapped Driver with a nil error:
     - an `ObserveActivity{Request: StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"}}`;
     - a bare `SignalWithStartWorkflowExecutionRequest`.
   - **Also unchecked:** `ExecuteMultiOperation` and `CreateSchedule`.
   - **Failure scenario:** a Model or Realization bug proposes a read whose request starts a
     workflow, or another activity ID. The canary starts an execution that is in no `run-opened`
     signal. `act.Driver.started` does not record reads either, so cleanup, reconcile and Driver
     Close all miss it. The lease is still released as "clean", against the fence's whole purpose.
   - **A regression:** the old fence checked both `InvokeRPC` and `PollRPC`.
   - **Fix:** fence by allowlist.
     - Unwrap `ObserveActivity` and check its inner request.
     - Admit only the activity-family RPCs the canary World needs.
     - Admit `StartActivityExecution` only with the fenced ID, and only as a unary call.
     - Refuse everything else, including SignalWithStart, ExecuteMultiOperation, CreateSchedule
       and any `Start*` or `Create*`.
   - **Test:** add the envelope and SignalWithStart cases to `TestTheFenceAdmitsOnlyItsOwnActivity`
     and to `TestARunCrossingItsFenceFailsAtItsBoundary`.

2. **major: no check ties the policy's `modelHash` to the committed guest.**
   - **Where:** tools/canary/policy/production-canary.json:3, policy_test.go:10, and
     .github/workflows/umpire.yml (`canary-check-case` removed in 7a2eb2458b).
   - **The gap:**
     - The old CI kept the canary's pinned Case in sync with the generated one
       (`make canary-check-case`, asserted by workflow_test.go).
     - The replacement pin is `modelHash`. No test compares it with `sha256(temporal.Guest())`.
     - controller_test.go:68 and the live harness both overwrite the hash with the computed one, so
       they cannot detect drift.
   - **Failure scenario:** any `make umpire-guest` rebuild merged to main leaves every test green.
     The next production dispatch then refuses with `model-mismatch`. Model work rebuilds the guest
     routinely, so this is likely, not hypothetical. The README's "update modelHash" note
     (README.md:90) is not enforced.
   - **Fix:** add `TestTheCommittedPolicyPinsTheCommittedGuest` in tools/canary/policy or
     controller, asserting `policy.Embedded().ModelHash == hex(sha256(temporal.Guest()))`. Then a
     guest rebuild fails CI until the reviewed policy update lands with it.

3. **minor: published Runs now carry raw coordinates; a stale doc comment and a missing test go with
   that.**
   - **Where:** tools/canary/authority/authority.go:45-46; tests/testpilot_canary_test.go:139-141;
     tools/canary/README.md:46-48, 265-266.
   - **What changed:** the earlier artifact carried "no credential, raw coordinate, payload or
     recorded Run". A published Run now holds the World and the exact request bytes, so the raw
     namespace and task queue reach the Actions artifact. This follows from the carried item
     "publish Runs with runner.Publish", and the README states it.
   - **Still to fix:**
     - authority.go:45 still says the coordinates "are never written".
     - The README's claim that a Run never holds the gRPC target or the credential is untested: the
       live harness scans only stdout and stderr, never the output directory.
   - **Failure scenario:** a later Driver change records the dial target, or a status detail
     carrying the host, into evidence. The production artifact then leaks it, and no test fails.
   - **Fix:**
     - Correct the comment.
     - In `invoke`, also scan every file under `job.output` for `h.coordinates.GRPC` and `host`.
     - Record the owner's acknowledgment that namespace and task queue are now public in artifacts.
       Both are GitHub environment secrets today.

4. **minor: some module-map and README text still describes deleted packages as live.**
   - **Where:**
     - .plans/UMPIRE_MODULES.md:77: the `tools/umpire/lint` row was edited in e689481277 but kept
       as a live module.
     - .plans/UMPIRE_MODULES.md:87-95: "Testpilot format 4.0 ownership", in the present tense,
       about cel-go, `internal/ir`, `cel` and `predicate`.
     - .plans/UMPIRE_MODULES.md:141, 257, 311, 412, 454: present-tense references to `lower`,
       `recordedrun`, `campaign` and `explore`.
     - tools/umpire/README.md:19 and :24: lint listed as a package, and "export and lint depend
       only on...".
     - model/README.md:510-517: "the unchanged-scope umbrella ... retains all legacy checks. Its
       seven inherited failures ... This umbrella is red". This contradicts the updated lines 76-78.
     - AGENTS.md:45, 57, 58: Testpilot "prepares, runs and evaluates a Case", the canary "runs one
       pinned Case", and tools/umpire does "lowering to Cases, conformance... exploration". The
       handoff flags this one.
   - **Failure scenario:** AGENTS.md:9 makes the module map binding for every Umpire task. An agent
     reading row 77 or the format 4.0 section treats lint, cel-go or `internal/ir` as permitted,
     existing dependencies, and may re-add them.
   - **Fix:**
     - Delete the Lint row and fold lint into the "Retired in fn-228.6" row.
     - Mark the format 4.0 section and the later references historical, or delete them.
     - Fix tools/umpire/README.md and model/README.md:510-517.
     - The owner or conductor updates AGENTS.md.

5. **minor: the canary's symlink containment lost its tests in the move.**
   - **Where:** tools/canary/cmd/umpire-canary/path.go (from `testpilot/publish/path.go`) and
     run_test.go:59.
   - **What is uncovered:** `resolve` and `within` exist to resolve symlinks before refusing an
     output under the model or a recovery under the output. `TestPublishResolvesTheRoot` and
     `TestProposalRootsResolveSymlinks` covered this and went with `publish`.
     `TestMainRefusesAnOutputUnderTheModel` uses a plain path. The coverage map's row
     "publish (10) → runner Publish tests" does not cover this canary-side check.
   - **Failure scenario:** a regression in `resolve`, such as not walking up to the existing parent,
     lets `--output` be a symlink into `model/`. Runs are then written into the model tree, and no
     test fails.
   - **Fix:** add run_test cases where the output is a symlink into `model/` and the recovery path
     is a symlink under the output. Or list this as a gap in the coverage map.

6. **minor: two passing live canary tests were retired rather than rewritten, and the pause hook is
   now dead.**
   - **Where:** tools/canary/testharness/names.go:12-13 and testharness.go:76.
   - **What is lost:**
     - `TestTestpilotCanaryHarnessLeavesALiveInvocationAlone` and `...RecoversATimedOutLease` were
       green at base. The map does state this (section 2) and points to fake-server unit tests.
     - The real-server behaviour those tests pinned is no longer exercised live: a lease
       `WorkflowRunTimeout` actually firing, and the close-event read classifying it
       `closed-otherwise`.
     - `VariablePause`, the hook only the first test used, is now used only by its own unit test.
   - **Failure scenario:** a server-side change to how a timed-out run's close event appears, or a
     describe/close race on the real frontend, breaks reconcile in production. The fake passes.
   - **Fix:** rewrite both tests over the Runner, since the harness already supports pause and
     crash. Otherwise remove the pause hook and keep the gap explicit.

7. **nit: the act Transport has no act-level test, and the authority test stopped asserting the
   namespace header.**
   - **Where:** common/testing/testpilot/temporal/act/act.go:68-79 and 214-220;
     tools/canary/authority/authority_test.go (the 7a2eb2458b diff).
   - **What changed:** the old test asserted
     `metadata.Pairs("temporal-namespace", namespace)`. The new one only counts `DialOptions`
     (2 or 3).
     - The live harness exercises the `DialOptions` path only with insecure credentials.
     - Nothing exercises TLS, the API key, or the header value.
   - **Failure scenario:** the interceptor sends the wrong namespace header. A namespace-scoped API
     key then fails in production with PermissionDenied, and no test caught it.
   - **Fix:** add a bufconn fake WorkflowService test that dials through `Transport.Driver(ns)` and
     asserts the header plus the per-RPC credential. Add an act test that a non-empty
     `DialOptions` replaces the insecure default.

8. **nit: the size report leaves out the Host the Runner imports.**
   - **Where:** handoff.md "Line counts". The Runner imports `umpire.local/model`, which is
     `model/umpire/hosts/go/{host,annotations,probes}.go` at 624 lines.
   - **Scope:** this is not gaming. The Host predates fn-228 and is unchanged, and with it the
     totals are still well under target: generic 2,271, total 3,941.
   - **Fix:** list it in the R7 DESIGN.md table (fn-228.7) for a complete picture of the host loop.

9. **nit: the slice gate does not require the subtests the map relies on.**
   - **Where:** Makefile:480.
   - **The gap:** the coverage map calls `TestTestpilotModelSlice`'s CancelAfterCompletion,
     CancelBeforeWorkerStart and Annotations the only live coverage of those behaviours. The
     required-pass list names only Complete, Fail, HostRefusal and Cancellation for that test. A
     failure still fails `go test`, but a future `t.Skip` would pass the gate silently.
   - **Fix:** add `TestTestpilotModelSlice/{CancelAfterCompletion,CancelBeforeWorkerStart,RetryThenComplete,Annotations}`
     to the required list.

## Accepted owner-behalf decisions

These are reasonable as recorded, but the owner should confirm them.

- `internal/ir` and `internal/execution` were deleted rather than shrunk. Act-and-observe lives in
  `temporal/act` (fn-228.4), and instruction decoding has no consumer.
- `TestTestpilotModelSlice` is kept, and `act` is not folded into its parent.
- `tools/umpire/{check,export,interp,ir}` stay red: the failures pre-exist and depend on fixtures
  the Scala era removed. CI's `go test ./tools/umpire/...` therefore stays red. A follow-up spec
  should retire or re-home the 17k-line reader.
