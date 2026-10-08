# Deterministic builder lock-wait observation

Replace the 25 ms sleep in `TestBuildSerializesConcurrentSameKey` with a test-only context `Done()` observer passed exclusively to the second build. Wait for that observer before releasing the first build. In the inspected fixture, `Done()` is reached only after a real nonblocking lock attempt reports contention and `acquireBuildLock` sets `waited = true`. This supplies the required causal ordering without a new production seam, public API, synthetic lock result or production behavior change.

This is a source-grounded recommendation for root's admission decision. No implementation, test, build, lint, cache, Flow, Git, review or CI action was performed. The requested research routing remains gpt-6-astra/high; the unavailable judge (`no_key`) did not select a fallback. Actual execution-model metadata is unavailable.

## Why the observation is precise

The [current test](../../../tools/gomad3/toolchain/build_test.go) at lines72-111 starts a first build and waits for its fake compiler's `started` notification. That callback blocks on `release` while the first build owns its genuine build-key lock. The test then launches a second build, sleeps 25 ms and releases the first. Scheduling or input-snapshot work can consume more than 25 ms, so the sleep does not establish that the second build ever saw contention.

The [builder](../../../tools/gomad3/toolchain/build.go) acquires the real lock at line136, stores `Waited` at line140 and defers release through the remainder of compilation and publication. The fake compiler callback executes through `runCommand` much later in the first build. The lock remains held while that callback waits on `release`.

[acquireBuildLock](../../../tools/gomad3/toolchain/lock.go) has the decisive order at lines20-35.

```go
lock, err := hostfs.Try(path)
// The switch returns for success and every non-contention error.
waited = true
select {
case <-ctx.Done():
	return nil, waited, ctx.Err()
case <-time.After(10 * time.Millisecond):
}
```

This excerpt omits the switch only to show the sequence; the complete source remains authoritative. `ctx.Done()` is evaluated when entering the select, after the contention switch and `waited = true`. The observation proves a failed contended acquisition and entry into the retry path. It need not prove that the goroutine has parked in the select or that a timer has fired. Those stronger timing claims are unnecessary for `Waited` and serialization.

[hostfs.Try](../../../tools/gomad3/internal/hostfs/lock_unix.go) requests `LOCK_EX|LOCK_NB` at line17. Its `lock` function maps real `EWOULDBLOCK`/`EAGAIN` to `ErrContended` at lines44-45 and closes the unsuccessful descriptor before returning. No lock is fabricated by the proposed observer. [Existing hostfs tests](../../../tools/gomad3/internal/hostfs/lock_unix_test.go) already check that a second real Try is contended until the first holder releases it.

For this exact build fixture, every earlier context-bearing operation is observable in source. [inspectBootstrap](../../../tools/gomad3/toolchain/build.go) delegates its commands through `runCommand`; [runCommand](../../../tools/gomad3/toolchain/build.go) at line609 invokes the supplied runner directly. [fakeRunner](../../../tools/gomad3/toolchain/build_test.go) at line223 never reads `ctx.Done()` or `ctx.Err()`. The fake validation callbacks take no context, and the real snapshot/key/layout operations preceding lock acquisition take none. The fake archive, extraction and materialization callbacks occur only after lock acquisition. Consequently the second build's first `Done()` observation belongs precisely to the contended lock path in this fixture. Production `hostexec.Run` may inspect context earlier, so this claim must not be generalized to a real-command integration test.

`Err()` is unsuitable for a passive notification here. The lock code calls it only after the cancellation channel is selected. Forcing that path would turn the successful waiter into a canceled build.

## Minimal test-only shape

The repository already has the exact pattern in [simulationProgressContext](../../../tools/gomad3/runner/internal/execution/simulation_progress_fixture_test.go) at lines17-27. It embeds a context, closes an observation channel through `sync.Once` in `Done()`, and returns the underlying context's channel unchanged. Apply that private pattern locally to the builder test rather than importing another package's test helper or adding a production dependency hook.

Suggested local structure, subject to root's implementation choice.

```go
type buildLockWaitContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (ctx *buildLockWaitContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.entered) })
	return ctx.Context.Done()
}
```

Use an underlying `context.Background()` for the normal-success fixture, preserving its nil Done channel and nil Err. Construct the observer before starting the second build, pass it only to that build, and replace only `time.Sleep(25 * time.Millisecond)` with waiting on `entered`. Keep `close(release)` after that receive. `sync.Once` handles repeated lock retries and concurrent observations safely. It does not pretend to set `BuildResult.Waited`, alter lock state or cancel either build. The existing `time` import remains needed by `testConfig`'s `BuildTimeout`.

The resulting event order is first build holding lock and blocked in fake compilation, then second build receiving real contention, then observer notification, then release of first compilation, then successful publication/unlock, then second acquisition and reuse. The existing 10 ms production retry interval remains unchanged; the test no longer uses elapsed time to infer contention.

Preserve every current assertion. Both returned errors must be nil, the build keys must match, the build counter must remain one and at least one result must report `Waited`. Preserve the existing fake runner, build inputs and publication checks. The receiver's first and second results are completion order, not stable worker identities. Any optional assertion specifically naming the second build must capture its identity rather than assume channel order.

Failure handling may use an ordinary test/harness deadline solely to report a missing handshake and terminate a mutant or regression. Deadline expiry must fail the test. It must never release the first build and count the test as a success, nor serve as the evidence that contention happened. If root adds local failure cleanup, close `release` once on every failure exit and join launched workers before fixture teardown; keep that cleanup separate from the success handshake. Avoid indefinite cleanup waits for intentionally broken lock mutants by bounding their isolated test process. No polling interval, `Eventually` delay, runtime stack inspection, `/proc`, descriptor theft or scheduling race is needed.

## Owner and Touches

[Fn-109.10](../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.10.md) owns installation/build layout and already includes `tools/gomad3/toolchain/*.go` in Touches. Root can admit this narrowly described test correction there, or use a bounded R18/R19 correction linked to that owner. [Tasks47](../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.47.md) and [48](../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.48.md) own different exact archive/patch corrections; their scopes do not implicitly expand. Fn-109.21 remains the implementation-free consumer of final gate evidence. Fn-110.2's temporary ownership of the single product/cache writer lane does not assign it this source contract.

Proposed product Touches contain only `tools/gomad3/toolchain/build_test.go`, including the private observer and the changed handshake. Task evidence may use its admitted artifact directory. Leave `build.go`, `lock.go`, `hostfs`, exported types, lock retry timing, fixture descriptor, patches, overlays and generated files untouched. No missing private production seam was found. No new lint exception or owner policy waiver is needed for the behavior-preserving test correction.

## Baseline, regression and mutation proof

The retained original-base lint receipt identifies the forbidden sleep at `build_test.go:97`. Preserve that real diagnostic as policy RED. The original test can already pass, so runtime failure against its old version is neither guaranteed nor required. Record any original scheduling failure honestly; do not manufacture one by adding load or shortening delays.

After root admits implementation, execute the unchanged ordinary builder preservation controls and the revised same-key test with stock Go 1.27.1 and `-tags test_dep`. The fake compiler's darwin/arm64 response is part of its existing portable unit fixture; it establishes no native toolchain qualification. Repetition and the race detector can supplement the causal proof. A large repeat count cannot replace the proof that `Done()` is reached only at contention. Applicable controls include `TestBuildPublishesAndReusesImmutableToolchain`, `TestBuildInjectedFailuresLeaveNoTemporaryState`, `TestBuildRejectsUnsupportedHostBeforePreparingInputs`, and the real hostfs Try lifecycle test.

Use isolated mutations to demonstrate that the new test still checks the contract. Changing the contended path's `waited = true` to false should reach the notification but fail the existing Waited assertion. Bypassing lock acquisition or letting the second build return before reaching contention must fail through missing notification or an explicitly rejected early result; it must never satisfy the test using a synthetic Waited value. Keep mutants outside the product tree, bind their exact patches and distinguish a bounded missing-handshake timeout from a normal assertion failure.

Do not overstate the existing `builds == 1` assertion. [fakeRunner](../../../tools/gomad3/toolchain/build_test.go) shares a `sync.Once` around the build counter and callback, so the counter alone cannot prove absence of every duplicate compiler invocation. Retaining the real contention observation, actual lock and Waited assertion supplies the relevant protection for this bounded repair. Redesigning the fake runner to count additional behavior would be separate test scope, not a prerequisite to removing the sleep.

GREEN requires the actual pinned analyzer to remove this exact forbidigo diagnostic without an exception or new finding, with the preserved runtime controls passing. Keep the full unfiltered scoped output and parent original-base gate accounting separate. This note supplies no new gate result and no task completion claim.

## Read-only source bindings

The inspected files had these complete SHA-256 values. Rebind before implementation if another authorized writer changes any dependency.

| Source | SHA-256 |
| --- | --- |
| `toolchain/build_test.go` | `4d7063425b908409511276c5fed0688b591ce595f0dfcc503ad434507e2aa573` |
| `toolchain/build.go` | `3702837c2459d3e14f05f1f2c6684a00c0c29ee8b2373d43b49914bf06aab9af` |
| `toolchain/lock.go` | `24ff7ee94613156b4f01264e546b082058844503ba7f603f5288bae9a2678c6c` |
| `internal/hostfs/lock_unix.go` | `80b21577f4389edfeb99f36c482ce5b4f113f3e614a74ee1bbfd98604e6539d6` |
| `runner/internal/execution/simulation_progress_fixture_test.go` | `12cbd8911c03b5ed4e0717397fec2742508daab3e28b8e91a6ca376671db8add` |
