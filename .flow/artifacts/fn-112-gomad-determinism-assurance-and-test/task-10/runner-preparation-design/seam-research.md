# Private Runner preparation seam

Portable orchestration tests can reach the real campaign controller, journals, artifact publication and World assessment by extending the existing per-call `executionDependencies`. The smallest first slice adds two optional functions, one around complete preparation and one around bootstrap construction. Resume and replay need two further functions around adapter verification and toolchain identity discovery. Public operations keep zero dependencies and their existing production behavior.

This is source research at `f0ed4a36d47c12d72383ea5de80370cec8581c9f`, after the task63 decomposition. No Go, build, test, lint, generator, Flow lifecycle or commit command ran. The only written file is this report. The existing 123-failure inventory remains historical evidence at its recorded revision, not a current-candidate run. Its observation and source binding are in `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-10/runner-coverage-inventory/inventory.json:2`; its handover records the unproved assertions and separate native ownership at `runner-coverage-inventory/handover.md:9`.

## Preserved boundary

The primary checkout's owner amendment permits format and identity-value changes but preserves capabilities, defaults, classifications, precedence, transactions, lifetimes and comments (`.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md:3`, especially line 14). This seam requires no format change. Task63 specifically decomposes existing orchestration without changing behavior (`.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.63.md:7`). Its decomposition is integrated source progress, not a completed host gate. R5 explicitly preserves fake failure coverage through private adapters and rejects global mutable hooks (`.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md:862`, primary owner text).

`Preparer` already accepts `target.Spec` and returns `target.Prepared`. `executionDependencies` already carries a private `executionRunner`; public `Explore` supplies its zero value (`tools/gomad3/runner/runner.go:115`, `:123`, `:377`). The README documents both contracts (`tools/gomad3/README.md:1300`). Native qualification remains deferred and unverified under fn149/fn128. Portable orchestration coverage cannot replace native execution (`MILESTONES.md:13`, `:40`).

## What the four operations actually do

| Existing operation | Side effects and validation | Why a scripted orchestration case encounters it |
| --- | --- | --- |
| `preparation.Prepare(context.Context, preparation.Request) (target.Prepared, error)` | With no custom preparer, creates an adapter workspace, discovers/prepares adapters, invokes target preparation, attaches selected adapters, validates the target, and removes the workspace with joined cleanup errors. With a custom preparer, skips adapter discovery but still invokes that preparer, replaces `Adapters` with an empty nonnil slice, and validates the prepared shape/platform/adapters. | A fake `Preparer` alone does not avoid the host-platform gate. The copy and filesystem effects of that preparer are still part of the test. |
| `Spec.BootstrapFrame(target.Prepared, string, uint64) ([]byte, error)` | Checks the default profile and qualified host, then encodes profile, target, Runner, argv and seed identities. No subprocess or filesystem I/O. | Every seed reaches the host gate even with a fake executor. |
| `Spec.VerifyAdapters([]deterministicio.Adapter) error` | Checks the default profile and qualified host, then checks a nonnil, sorted, unique list against embedded exact adapter identities. No installed toolchain lookup, module-cache read or subprocess. | Resume and replay call it independently of preparation. |
| `target.ReadToolchainIdentity(string) (target.ToolchainIdentity, error)` | Validates the installation description, launches its pinned `go env GOVERSION GOOS GOARCH CGO_ENABLED` through the bounded command owner, and checks the reported platform/cgo fields. | Replay preflight always reaches this operation, including verify-only replay and minimization. Resume and shards reach it only with a nil private executor. |

Sources are `internal/preparation/preparation.go:50` through `:105`, `deterministicio/profile.go:278` and `:293`, `deterministicio/bootstrap.go:25`, `deterministicio/adapter_registry.go:121`, and `target/target.go:357`, all below `tools/gomad3/`. The target preparation owner remains the authoritative owner of its stage errors and workspace cleanup. Runner should not reimplement it.

## Options

| Option | Change and coverage | Assessment |
| --- | --- | --- |
| Inject whole local phases or an already initialized `localCampaign` | Replace `prepareTarget`, `validateRequest`, `openCampaign`, or construct the phase object directly. | Useful for existing phase tests, but bypasses journal transitions, progress, failure recording, resume validation and cleanup lifetime. It cannot restore end-to-end orchestration coverage. |
| Export the preparation service bundle or add a general configurable I/O profile | Reuse `preparationServices` across packages and expose pure validation/encoding beneath host checks. | Broadens package contracts and threads configuration through production preparation. The current services are deliberately private. It creates a larger change than the Runner needs and risks making public host refusal configurable. |
| Extend Runner's existing private dependencies at exact calls | Optional function fields with real fallbacks, passed by value through existing private operations. Existing `Preparer` still owns the fixture copy; campaign and storage operations remain real. | Recommended. Two functions restore seed/World orchestration. Two more support later resume/replay work without a new public API or generic framework. |

The function option makes only a constant-size per-call value and does not add selection-sized state. Bootstrap callbacks can run concurrently, so test closures must synchronize mutable observations or be immutable. There is no scheduling change, new goroutine, cache, global state or resource ownership in the dependency owner.

## Concrete shape

Keep `executionDependencies` as the private carrier. The complete eventual shape can be:

```go
type executionDependencies struct {
    executor executionRunner
    prepare func(context.Context, preparation.Request) (target.Prepared, error)
    bootstrap func(deterministicio.Spec, target.Prepared, string, uint64) ([]byte, error)
    adapters func(deterministicio.Spec, []deterministicio.Adapter) error
    toolchain func(string) (target.ToolchainIdentity, error)
}
```

Methods named `prepareTarget`, `bootstrapFrame`, `verifyAdapters` and `readToolchainIdentity` select their optional field or call the exact real operation above. Preserve its argument order and returned error. A private `injected()` predicate checks executor and every optional field; functions cannot be compared by struct equality. Do not eagerly fill defaults. Nil fields preserve both the default meaning and detection of an explicitly injected operation. For the first bounded slice, omit the last two fields and methods until their consumers are admitted.

Replace the `executor executionRunner` member of `campaignRuntime` with an embedded `executionDependencies`. Existing `config.executor` references remain valid through promotion. `campaignRequestFromSpecWith` copies the whole dependency value instead of only its executor. There are only two direct `campaignRuntime` literals, in `campaign_options.go:260` and `coordinator.go:36`, and neither sets `executor`. This avoids parallel copies of dependency state and leaves public `CampaignSpec` and serialized campaign options unchanged (`tools/gomad3/runner/campaign_options.go:90`, `:210`).

Keep a small `preparation_dependencies.go` in `runner` for the dependency type and fallback methods; move the existing two-line type there. Every wrapper resolves at the existing operation's call site. No callback receives a journal, corpus, artifact handle or controller. No callback owns close/cancel behavior.

## Call propagation

| Path | Exact propagation and preserved behavior |
| --- | --- |
| Fresh local campaign | `Explore -> exploreWith -> campaignRequestFromSpecWith -> runLocal -> local.prepareTarget`. Replace only `preparation.Prepare` at `runner_local.go:250`. Retain `BeginPreparation`, `PreparationRoot`, stage classification, guidance, plan recording and `CompletePreparation`. In `runSeed`, replace only `profile.BootstrapFrame` at `runner.go:685`, using the request's dependencies. |
| Choice/simulation exploration | Both already pass the full `campaignRequest` into the common `runSeed` (`choice_exploration_campaign.go:241`, `simulation_exploration_campaign.go:256`). No new strategy-specific hook or round change is needed. |
| Resume | `resumeWith` already forwards dependencies into `exploreWith`. `resumeConfiguration` reads adapter verification through the request. Preserve its existing `request.executor == nil` identity-read condition at `resume.go:83`. Crucially, reconstruct the resumed request with `local.config.executionDependencies` at `runner_local.go:132`; rebuilding an executor-only literal would lose the seam. The resumed path still skips fresh preparation, verifies target bytes, acquires its actual lock and restores real counters. |
| Portable plan | `createCampaignPlanWith` already produces the private request. Replace the complete preparation call at `portable_plan.go:123`. Keep bundle creation, mount capture, target rename, permissions, verification and publication real. Guidance receives the same request. |
| Shard | `runCampaignShardWith` retains the nil-executor identity-read condition at `campaign_shard_execution.go:52`, then forwards all dependencies through its existing request construction at line 104. Its `campaignPlanPreparer` must actually copy, sync, close and verify the immutable target at lines 154-191. The preparation callback invokes that existing preparer rather than substituting an arbitrary prebuilt target. |
| Guidance | At `guidance.go:50`, give fallback `artifactReplayer` the complete request dependency value. An explicit public `Replayer` still wins. Corpus open, snapshot, seed selection, admission and publication remain real. |
| Replay | `replayWith` passes dependencies to the private `preflight`, whose only substituted operation is identity discovery at `replay_operation.go:467`. Replace adapter verification at line 93 and bootstrap construction at line 181. Keep artifact open and payload access, platform matching, compatibility, World validation, build-info/capability validation, choice/simulation records and close precedence real. |
| Minimize | `openMinimizationSession` passes dependencies into the same preflight at `minimize_operation.go:290`. `session.evaluate` uses its dependency bootstrap operation at line 429. The session already stores the complete dependency value and forwards it to default replay at line 533. Keep workspace locks, persisted budget, reductions, accepted artifacts and trial publication real. |
| Isolated execution | `exploreWith` rejects any explicit dependency before `runIsolated`, using the same error string and the current position after `campaignRequestForExplore`. Also retain rejection of public `Preparer` and `Replayer`. The coordinator's decoded `campaignRuntime` has zero dependencies. No function crosses the wire. |

Paths in this table are below `tools/gomad3/runner/`. Current direct `preflight` callers are replay and minimization; no test calls it directly. Changing that private signature to `preflight(config ReplaySpec, dependencies executionDependencies)` is sufficient. Preserve replay's unconditional identity discovery even when an executor is injected. Moving the resume/shard nil-executor behavior into replay would change existing semantics.

## Fixture contract

The initial preparation callback accepts only an explicitly supplied test `Preparer`, invokes it once with the actual request context and spec, retains its copied file/path/digest/size, and supplies the empty nonnil adapter slice that the real complete owner assigns for custom preparers. `newFakePreparer` creates an executable opaque file; `fakePreparer.Prepare` copies it into the actual preparation root (`runner_test.go:1962`, `:1993`). Keep those operations real. Do not pre-populate a published campaign or replace `Prepared.Verify`.

Preparation failures that need `StageTarget` must continue through real `preparation.Prepare` with an error or waiting preparer. That path returns its private stage error before host validation (`internal/preparation/preparation.go:97`). A raw error from the injected complete-preparation function is not a target-stage error. Do not call a stateful preparer twice to reconstruct the stage, parse strings, export `stageError`, or assert a different `HostError` reason. Existing preparation-owner tests retain the real owner and its services.

For seed/World tests whose fake executor never decodes `IO.Config`, a bootstrap callback can return explicit sentinel bytes and record the profile, target digest, Runner identity and seed arguments. Add a request assertion showing those bytes reach the fake executor. Such a fixture tests Runner transport of bootstrap data only. Real bootstrap encoding has existing portable tests at `deterministicio/profile_portable_test.go:121`, and native validation remains separate. Do not manufacture a toolchain-valid frame or claim bootstrap validation coverage from sentinel bytes.

The later adapter callback is a strict test expectation over the fixture's known adapter slice, plus an injected-error case. It does not need a second implementation of the registry. The later toolchain callback returns an explicitly expected fixture identity and checks the requested root; Runner retains the actual recorded-vs-current identity comparison. Genuine missing-root/default behavior still uses the real operation.

Attach dependencies explicitly to the admitted test invocation. Do not make `testConfig`, `newFakePreparer`, all `...With` calls, or all unsupported hosts automatically portable. Several tests deliberately distinguish nil/default/private execution, compiler preparation or actual children. The existing default/private characterizations are in `executor_injection_characterization_test.go:31`, `:55`, `:87`, `:100`, `:124`, `:147`. Their default branches must retain real default operations or remain separately reported as unproved on this host. The primary fixture researcher owns the exact initial test-call selection.

Replay fixtures need an additional caution. Identity substitution alone does not make opaque fake target bytes a replayable binary. Real preflight still reads build information at `replay_operation.go:492`; linked/guarded targets still need their embedded manifest at line 525. Fixture construction also directly calls `BootstrapFrame` and `ReadToolchainIdentity` in `replay_operation_test.go:436` and `:624`. Migrate those explicit fixture dependencies in a separate admitted slice, retaining real payload/build-info validation. This report does not claim that four production hooks alone repair every one of the historical 123 failures.

## Exact implementation surfaces

The first seed/World slice needs these production files only:

- New `tools/gomad3/runner/preparation_dependencies.go` for two optional operations, fallbacks, the existing dependency type and injection detection.
- `tools/gomad3/runner/runner.go` to relocate that type, extend isolated refusal and route bootstrap construction.
- `tools/gomad3/runner/campaign_options.go` to embed/copy dependencies.
- `tools/gomad3/runner/runner_local.go` to route preparation and preserve dependencies across resumed request reconstruction.

Tests need the explicitly admitted calls in `runner_test.go`, one new `preparation_dependencies_test.go` for fallback/refusal/argument/error controls, and additive cases in `executor_injection_characterization_test.go` for isolated refusal of each new field. No existing assertion should weaken. This gives a bounded first implementation with four production paths, one of them new. It does not claim portable resume/guidance/replay coverage.

The complete propagation slice adds `portable_plan.go`, `resume.go`, `campaign_shard_execution.go`, `guidance.go`, `replay_operation.go` and `minimize_operation.go`, plus the last two dependency fields. Its exact production set is therefore ten files including the new owner. Tests then extend the corresponding existing operation test files only for individually admitted private cases. No changes are needed in `internal/preparation`, `deterministicio`, `target`, campaign/corpus/artifact storage, World, strategy engines, CLI wire contracts, generated files or public request types. In particular, fn152/fn153 storage code remains outside this seam.

## Failure and preservation checks for implementation

1. Zero-valued dependencies retain default platform refusal, missing-installation errors and existing public custom-preparer validation. Each explicit function is independently rejected for an isolated campaign with the existing error. Rejected callbacks are never invoked. Preserve resume preflight-before-isolation ordering.
2. Progress failure and parent cancellation before preparation still make zero preparation/execution calls. Keep the existing `TestLocalCampaignPreparationShortCircuits` at `runner_local_test.go:15`. A real error/waiting preparer still produces target-stage failure and cancellation precedence.
3. An injected preparation failure leaves the actual campaign recoverable state and cleanup behavior appropriate to its stage. Default preparation receives the same preparation root, environment and public preparer. No eager preparation happens before its original journal transition.
4. Bootstrap failure occurs after output files open, invokes no executor, retains exit transition/output close handling and the original first error. `runSeed` currently joins those later errors at `runner.go:730`. A successful bootstrap is called once per execution with the seed selected by the real controller.
5. Keep real prepared-file mutation detection, World malformed evidence, watchdog, cancellation, duplicate/distinct failure and publication controls. Existing precedence phase tests are `runner_local_test.go:71` and `:126`; end-to-end cases must still reach their real controller/publication paths. The post-execution `Prepared.Verify` stays at `runner_local.go:527`.
6. Later resume tests prove no fresh preparation, unchanged completed ordinals, dependency survival after request reconstruction, adapter verification before conditional identity discovery, and a real lock. Shard tests prove mount/target verification and actual `campaignPlanPreparer` copy. Nil executor still requests the pinned identity; injected executor still skips it only in these two operations.
7. Later replay tests prove identity mismatch and injected identity errors precede World/build-info work as before, then adapter error wrapping, verify-only's no-execution behavior and bootstrap failure. Minimize default replay and guidance fallback must receive the complete dependency value. Explicit public replayers still take precedence.
8. Preserve cleanup scopes exactly. `runLocal` registers guidance close before overall cancel and journal close, then fail recording; replay uses different close-error precedence for successful operations and failed preflight (`runner_local.go:51`, `replay_operation.go:83`, `:437`). The seam owns no cleanup and moves none of those defers.

Run admitted focused checks with `-tags test_dep`, cheap package-architecture boundaries, and the required scoped/original-base lint on frozen source. The root owns serial gates, review and lifecycle. Full current-candidate native gates remain with their transferred owners. Existing red source acceptance stays open until its own evidence passes; a selected portable subset supplies only its named behavior coverage.

## Authorization

No new user decision is required for the bounded private seam within the existing instruction to finish milestones using the conductor's recommendations. Root must record the concrete source/test admission after task63's reviewed integration and before implementation, because this research dispatch authorizes only the report. Native qualification revival, CI, PR/push, public API expansion, production validation bypass and storage redesign are outside this recommendation. The deferred native hosts do not block this source design. Opaque replay fixture bytes and undispatched fixture migration are concrete implementation prerequisites for the later replay slice, not reasons to broaden validation hooks.
