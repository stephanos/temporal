# Original caller reconstruction

The coherent dependency anchor is `a3b9f80efab9356c0be2080779133337e2471ac0`. Export its complete `tools/gomad3` module into a fresh scratch directory, then substitute the four exact source preimages listed in `task-7/preimages.json`. Keep the anchor's remaining production files and tests together. This reconstructs the original Explore and CreateCampaignPlan implementations without translating their protocols to current dependencies. No builds, tests, lint, gates, worktrees, or source edits were performed for this research.

The dispatch candidate was `b4602685b3184387cf2d713178095247f0c11d8f`. The actual host reports `Linux aarch64`. Neither `qemu-x86_64` nor `qemu-x86_64-static` is on PATH. Successful preparation through the unmodified public callers is unavailable on this host because both original and current deterministic-I/O validation reject linux/arm64. The matched controls below can retain actual rejection and preparer-failure behavior; they cannot establish successful prepared-target byte equivalence here.

## Dependency evidence

The anchor's three Runner postimages exactly match the original task's `final-source.json` SHA-256 values.

| File | Anchor SHA-256 |
| --- | --- |
| `runner/runner.go` | `c257c5ee5f299e7b2dc1fec54a42918ec535291c8de67912e949bc6a0c94d2cf` |
| `runner/portable_plan.go` | `95112a7c797ef8592c7af6b19fd187e3d55f4daef21cbb4b9720ab770d8bd3ee` |
| `runner/deterministicio.go` | `f55f752c2611ae6da7519e25223a2c11144cd3495aea2d0fc9abe00e07d54425` |

Full `git show a3b9f80efa:tools/gomad3/<file>` comparisons with the saved preimages show only the task-7 migration for these three files. The anchor already carries compatible `campaignRequest`, private execution dependencies, the original completion representation, and the contemporaneous Artifact API. Its parent `d635e23f00d926a43b942f25a9d05bd0ccb72025` lacks earlier task-2/task-6 caller changes and is therefore the wrong complete baseline for the saved preimages. HEAD has later completion constructors, artifact APIs, and dependency-bearing `artifactReplayer`; directly overlaying the original Runner onto HEAD would require unrelated adaptation.

The four source substitutions are the exact `preimages/tools/gomad3/runner/{runner.go,portable_plan.go,deterministicio.go}` files and `preimages/tools/gomad3/architecture_test.go`. Verify their bytes against `preimages.json` before any compile. The architecture preimage is a preservation input, not a claim that the complete later anchor satisfies the old package-owner inventory. Restrict any original-caller execution to `./runner`; an old architecture test against later packages is a different experiment.

Keep the anchor's `runner/*_test.go` dependencies rather than copying current tests into the old tree. Both anchor and current `go.mod`/`go.sum` are unchanged and pin Go 1.27.1 and `golang.org/x/mod v0.37.0`. Existing original helpers `testConfig`, `fakePreparer`, `fakeExecutor`, `toolchainRoot`, and private operation entrypoints exist together at the anchor. `preparation_owner_test.go` there hashes to the original `e8d0309b038a77217d0d4e011a9430838563468b087be105d6a93ad191a7b66c`.

## Probe provenance

Restore the saved `runner/preparation_equivalence_legacy_test.go` byte-for-byte for the original probe. Its SHA-256 is `328bfdec937d16d1771e746818231b33522c58ed9f19214d5978ea99ec61b17e`. It calls `executionAdapters` from the restored original `deterministicio.go`, has no owner dispatch, and does not create the fresh/cache directories. Create required directories as fixture setup outside its function body, if the probe is eventually executed.

The anchor's later version hashes to `0db44bbf2f4c84617d75e8b77ad52df735a6834a02064a4dfc9896400b098404`. It is the full task-patch addition, with `preparationForEquivalence`, owner selection via environment, directory creation, and the adapter probe. It must not be relabelled as the original saved simple-probe body. The supplied artifacts do not separately retain a pre-edit body for the historical adapter probe. Its historical command and snapshot are retained in `handover.json` and `equivalence-adapter/before/`; the later body is recoverable from the task-only patch/anchor. Preserve that provenance distinction.

The saved `internal/preparation/preparation_test.go` is the initial RED probe, SHA-256 `34aecf8ee75b2d42886c1ba8feb0c875f23ad059c82ab8ad9f978b119669aaa2`, not original production source. Retain its exact body separately; replacing the current owner tests with it would not reconstruct the original failure context because the anchor already contains the owner.

Historical `equivalence/{before,after}/snapshot.json` are byte-equal at `29195da5e9abda68ae04dc414b0c500d0be353c6da9343328668f2039d51b34f`; the adapter pair is equal at `f393401b100b6c311cf962f5dd7b3ffcda0355ce5d91050cd788b9c15a7c5aed`. They normalize only Prepared.Path and exercise a spelled-out protocol. Neither invokes the actual original Explore caller. Directory labels `fresh` and `cache` alone establish no observed cache miss or hit.

## Minimal matched caller inputs

Use one identical new test-only harness alongside each source set, invoking public `Explore` and `CreateCampaignPlan` directly. It should construct request values explicitly rather than depend on current helper implementations. Use these fixed values in both runs:

- Seeds `1`, Parallel `1`, OnFailure `PolicyAll`, FailureBudget `1`, ExecutionTimeout `1s`, OverallTimeout `10s`, TerminateGrace `100ms`, OutputLimit `64`, WorldTransitionLimit `64`.
- Environment `MODE=test`, RunnerBuild `sha256:` followed by 64 zeroes, SupervisorCommand `unused`, no coordinator, guide, choice trace, simulation, coverage, or mounts.
- Target Kind `go-run`, Source `.`, fixed fixture module bytes `module example.com/fn1097-preservation\n\ngo 1.27.1\n`, and `package main\n\nfunc main() {}\n`; record actual fixture and tool paths rather than pretending they are historic paths.
- A custom Preparer that records the received Target spec, writes identical fixed target bytes below its supplied PreparationRoot, and returns a detached Prepared with deterministic digest/size, argv `gomad3-target`, empty build tags/compatibility, GoVersion `1.27.1`, fixed BuildKey `cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e`, and actual GOOS/GOARCH `linux/arm64`. Supply the same deliberately nonempty adapter slice to prove caller clearing if the observable boundary permits it. No fake platform or toolchain claim.
- Separate clean artifact/output roots for each operation, or reuse one exact scratch pathname sequentially after retaining its prior result. Normalize only explicitly named destination paths; never normalize identities, error reasons, bytes, or adapter ordering.

Run a small table for both public callers with (1) a fixed sentinel preparer error, (2) that custom successful Prepared followed by actual platform rejection, and (3) nil Preparer with the same simple fixture followed by adapter-stage platform rejection. Record original/current error text, `errors.Is` for supplied sentinels, `HostError.Reason`, preparer call count, received root, output/bundle cleanup, and retained journal state. The preparation-stage wrapper is private and may differ in concrete type; retain its unwrap chain rather than assert false raw-type identity. In Explore, target-preparer failure should retain `HostError.Reason == target_preparation`; plan returns the preparer error without that Runner wrapping. A canceled-input row may be added with an already canceled context to avoid timing dependence.

These rows execute the actual caller bodies on stock linux/arm64 and require no patched runtime. The successful-Prepared row still ends at platform rejection, so the evidence must say that post-validation campaign/plan construction did not run. Do not substitute passing lower-level target cache tests for successful actual-caller equivalence.

Use the installed stock Go `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go` with `GOTOOLCHAIN=local`, `GOWORK=off`, `-tags test_dep`, `-count=1`, and an exact new control selector, after checking the executable prerequisite. Unset inherited GOROOT, GOBIN, GOMADSEED and GOMAD3_CHILD_SEED. Proposed execution is one bounded captured `go test ./runner -run '^TestOriginalCallerSourceControls$'` for each dependency tree, preserving each exit and collected test count. This command was not run by this researcher.

## Cache and successful-control limits

For the proposed custom-preparer rows record cache state as `not applicable; custom preparer skips build and adapter selection`. For nil-preparer rows record `not reached; profile rejects actual host before toolchain/cache access`. The fixture root and runner output roots must begin empty. The Go test build cache is distinct from target.Prepared caching and must never supply a target-cache-hit claim.

If a later authorized supported execution becomes available, run the untouched original and current callers with a common fixed toolchain identity, target fixture bytes, module sums and adapter inventories. Separate per-side empty prepared-target stores establish the first miss, and a second preparation on each same store establishes reuse only with retained entry identity and build-command evidence. `target/prepared_cache.go` derives the production store from the toolchain build and does not use the per-call PreparationRoot as the cache root. A Progress callback returning a sentinel at `ProgressRunning` can stop public Explore after its recorded preparation, before target execution, while retaining the real campaign plan. Public CreateCampaignPlan needs no target execution.

The original plan performs adapter selection before assigning `config.Target.PreparationRoot = bundle` (`preimages/.../portable_plan.go:122-134`). Its known adapter-backed empty-root input fails with `deterministic I/O build adapter requires module cache and preparation root`, as retained by `legacy-portable-overlay.log`. Current plan intentionally assigns the bundle root first. Preserve an empty-root negative control and its deliberate before/after behavior; for a matched successful adapter case supply the same existing, private nonempty PreparationRoot to both callers. That distinction must not be hidden by silently creating or assigning a root inside the old caller.

The original probe calls target.Prepare and its pinned driver; supplying stock Go in place of the missing patched driver cannot produce honest original probe success. A platform-input overlay or emulated supported source set would be a separate explicitly authorized source experiment, not the actual native host or existing original probe execution. No such overlay or emulator was introduced here.
