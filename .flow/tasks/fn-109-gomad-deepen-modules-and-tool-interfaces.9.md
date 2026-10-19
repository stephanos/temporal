---
satisfies: [R10]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.9 Give target Go commands one private host-command seam with bounded output

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 3, R10 (F9). Target compilation, Go identity queries and package listing each start processes their own way; the build captures unbounded combined output. Route them through one private Go-command adapter over the existing `hostexec` primitives while keeping the two output contracts distinct.

**External ordering:** after `fn-108-gomad-reduce-code-size-without-removing.2` (local cleanup; it removes `validateGoCapabilityClosure`, `target/capability.go:184`). Re-anchor `capability.go` line references after it lands.

**Size:** M
**Files:** `tools/gomad3/target/target.go`, `target/capability.go`, `target/internal/capabilityreview/list.go`, a new private adapter under `target/internal/`, tests.
**Touches:** [tools/gomad3/target/**, tools/gomad3/ARCHITECTURE.md]

Current retained source admission (2026-10-08): [conductor admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/conductor-source-acceptance-20261008/admission.md). fn109.8 is verified Done. Task40's reviewed integrated mechanism remains byte-identical to its source-progress candidate; the milestone source-admission rule permits task9 implementation and proof while preserving the task40 dependency and all retained source acceptance. Task40 remains Todo, and task9 cannot complete until its predecessor acceptance is reconciled. Root owns commits, independent review and Flow lifecycle; the worker owns the exclusive shared execution lane and per-task code/tests/evidence. Native transfers remain deferred/unverified.

### Approach
- Call sites: `target.go:379` (`go env GOVERSION GOOS GOARCH CGO_ENABLED`, no context), `:405` (`go env GOMODCACHE`), `:442` (`go mod download -json`), `:736-739` (build, `exec.CommandContext` + unbounded `CombinedOutput`, then `cacheUse.Release()` `:740`), `capability.go:881` (`go list std`), `capabilityreview/list.go:104` with its own `runBounded` (`:150`).
- Reuse `hostexec.Run` (`internal/hostexec/command.go:17-58`): it needs a working directory, a positive timeout and an output limit, and gives process-group termination and head/tail capture with full hashes (`output.go`). Owner `target` may already import `hostexec` (`architecture_test.go` `ownerMayImport`). Follow the private `dependencies{run: hostexec.Run}` shape at `toolchain/build.go:63-82`.
- Two contracts, one mechanism: structured output (`go list -json`, `go mod download -json`, `go env`) requires complete bounded data and rejects overflow instead of decoding a truncated prefix; diagnostics (compiler output) may keep bounded head/tail plus hashes. Do not merge them into one "output" type that makes truncated structured data acceptable.
- Timeouts: derive from the caller context deadline; `hostexec` rejects a non-positive timeout, so decide the bound used when the context has no deadline (`ReadToolchainIdentity` currently has no context) and keep cancellation returning `ctx.Err()` as `list.go:108-110` does.
- Error text and wrapping are behaviour: `"prepare %s target: %w: %s"` and `linkedCapabilityBuildError` (`target.go:744-748`), `CommandError{Stderr, InvalidInput}` (`list.go:111-114`), `"release target build cache: %w"` precedence over the build error (`:740-742`), `"query pinned Go command"` (`:383`). Characterize before changing.
- The fake adapter should let fresh/cache preparation tests run without real compilation.

### Investigation targets
**Required:**
- `tools/gomad3/target/target.go:350-460,640-760`
- `tools/gomad3/target/internal/capabilityreview/list.go` and `list_test.go`
- `tools/gomad3/target/capability.go:870-905`
- `tools/gomad3/internal/hostexec/{command.go,command_unix.go,output.go}`
- `tools/gomad3/toolchain/build.go:59-82,200-215`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./target/... ./internal/hostexec
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./target -run 'Prepare|Cache|Capability'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/analysis/...
make test-live-capability
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.

### Current command inventory gap, 2026-10-05

AdapterPreparedSourceSetSHA256 in tools/gomad3/target/adapter_source_set.go also runs go list -e -find -json . directly through exec.CommandContext with unbounded stdout/stderr buffers. This path was omitted from the inventory above. Task39 owns only its deferred GOPATH removal and does not repair this R10 command omission. The prior checked acceptance and Done/Evidence below are historical; they do not establish complete R10 coverage on the current source.

Task9 owns the missing transport integration and task21 consumes its source-bound preservation evidence. Serialize overlapping adapter_source_set.go work after task39's reviewed progress commit; neither task39's full acceptance nor task21 completion is a source-admission prerequisite. Before changing transport, characterize BASE literal digest/projection and ordinary exit, signal, startup, empty/relative-directory, cancellation/deadline and stderr/error text/type/unwrap/precedence. A simple Structured swap loses raw exec errors and stderr and changes cancellation outcomes. Preserve R18 rather than silently accepting that migration; any shared hostexec write surface needs bounded admission and independent existing-consumer review. No new public seam, command framework, pin, grant or requirement waiver is authorized by this inventory amendment.

Retained decision and source research are .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/adapter-command-gap-2026-10-05/. Static diagnosis is not execution evidence. All original R10/R18/R19, predecessor/full/default/functional/affected/formal/native Darwin requirements remain open where unproved; Linux execution remains with fn128 and nonblocking.
### Reviewed compatibility admission and empirical BASE, 2026-10-05

Task40 owns the narrowly opted-in shared hostexec/gocommand compatibility mechanism. This task retains AdapterPreparedSourceSetSHA256 integration, decode/projection and R10 qualification. Its original task8 dependency remains; task40 is an additional dependency. A reviewed source-progress commit supplies evidence, not readiness or permission to start a dependency-blocked task.

The actual public-helper BASE probe ran 29 cases twice on stock Go1.27.1, developmental linux/arm64, at 0c2e091b73325368b22ec142e76da5ed5ab4cad2. After acknowledged startup and stderr, both cancellation and deadline return wrapped exec.ExitError, SIGKILL/9 and ExitCode -1 without context-sentinel identity. Before-start sentinels and bare-PATH lookup precedence differ from missing absolute commands. Concrete startup PathError/errno, ordinary exit/signal, malformed/trailing JSON, empty/relative directories and ./ command resolution are retained in base-process-probe. Both runs observed 18 reaped children and removed their 18 observed GOPATHs. This probe's empty-source JSON establishes no actual platform-source selection, listing-size justification, pin, overflow, descendant or native qualification.

After dependencies are ready, route only the command block through the admitted compatibility operation with gocommand.New injection. Preserve argv, caller context, supplied Dir, environment override order, task39's checked cleanup and the existing decode/projection body. Retain BASE/final literal nonempty Go/foreign source-set controls for both supported source sets, import override and exact quoted-comment suffix handling, source-read/malformed/trailing/missing-field failures, and execution-before-decode precedence. Keep failed results' digest empty. Measure admitted listing sizes before choosing the proposed 4 MiB per-stream capacity; disclose the finite 15-minute watchdog and capacity refusal as R10 bounds. Verify both streams and dual overflow, plausible JSON prefixes, nonzero/context plus overflow and cleanup precedence through actual or lawful existing injected execution. Real capture/descendant tests must establish physical behavior. Preserve default Structured/Diagnostic contracts and inspect their affected consumers.

Update the target command-contract paragraph in ARCHITECTURE.md when integration lands. Fn113 retains exact pins, approval, regeneration and publication; a skipped adapter-pin test supplies no execution proof. All original acceptance and historical Done/Evidence below remain unchanged. Root commits reviewed source progress separately; full/default/functional/affected/formal/native Darwin/static both-source-set and matched-first-baseline acceptance stays open wherever unproved. Linux stays under fn128 and nonblocking.
### Bounded adapter integration source admission, 2026-10-05

MILESTONES item 3 permits source implementation after its predecessor candidate is integrated and reviewed. For this checkpoint only, the reviewed task40 compatibility mechanism and cleanup at `d2e0e035519f1385b9acf630a70152655b113f61` replace the operational waits above: both "not readiness or permission to start a dependency-blocked task" and "After dependencies are ready" are superseded for this source admission. Retain task8/task40 dependency edges and every original completion requirement. Root records this exception with `flowctl start --force --note` before dispatching the single writer; it does not mark either predecessor done or claim task9 completion. Root owns precise staging and separate verified-progress commits, superseding the historical user-owned-commit constraint for this checkpoint.

Fresh unchanged-source evidence belongs under `task-9/adapter-integration-20261005/`. Re-run the retained public-helper process probe on this BASE; bind the literal nonempty Go/foreign source inventories and digests to both supported source sets using the pinned stock Go release. Measure real module-off listings before selecting a per-stream capacity, retaining the sample scope and headroom rather than claiming all adapter pins qualified. Historical empty-source probes and skipped exact-pin tests are not substitutes.

Modify only `target/adapter_source_set.go`, a focused same-package test file and the owning `ARCHITECTURE.md` command paragraph. Follow the private `*With(..., gocommand.Runner)` pattern and existing `gocommand.New` test seam. Route the command through `Compatibility`; preserve the public signature/comment, argv, caller context, verbatim empty/relative directory, environment override order, checked GOPATH lifecycle, outer error wrapping and complete decode/source projection. Shared hostexec/gocommand production code, adapter pins, approval, regeneration/publication and default Structured/Diagnostic contracts stay unchanged. Fn113 retains pin ownership.

Retain a behavioral RED for the missing public-helper output bound before transport edits. Final controls cover exact-capacity and limit+1 on both streams, dual overflow, plausible JSON prefixes, execution failure or cancellation plus overflow, infrastructure and cleanup precedence, import override, exact quoted import-comment suffix, malformed/trailing JSON, missing fields and unreadable sources. Verify default finite watchdog and earlier caller deadline forwarding; distinguish injected propagation from actual process termination. The unchanged public process probe compares literal error type, unwrap order, errno, ProcessState/exit/signal, context identity, stderr and digest; normalize only explicitly identified volatile paths/PIDs. Re-run actual shared-mechanism capture, watchdog and descendant controls and inspect affected consumers without duplicating their owners.

Before a source-progress commit retain focused tests, check-only generator validation, architecture, formatting, standalone errortype, unfiltered scoped lint, the actual integrated lint result and fresh independent source/evidence review. Complete raw command records retain source/tool identities, argv, cwd, exit, elapsed time, stdout/stderr and skips. Original first-baseline, predecessor, full/default/functional/affected, formal/native Darwin and static both-source-set acceptance remains open wherever unproved; Linux stays nonblocking under fn128. A source-progress review is not formal SHIP or DONE.

### Reviewed bounded adapter source progress, 2026-10-05

The public prepared-source helper now uses the admitted Compatibility runner with 4 MiB per stream and its 15-minute watchdog. It preserves the public comment/signature, caller context, argv, raw directory, ordered environment, checked GOPATH cleanup, outer wrapping and complete decode/source projection. Shared command implementations, existing tests, generated inputs and pins retain their BASE bytes. Only the helper, new focused tests and owning architecture paragraph changed.

Evidence belongs to `task-9/adapter-integration-20261005/`. The unchanged-production behavioral RED accepted all three overflowing valid-JSON prefixes; exact-capacity controls passed. The recovered initial test and runner match their recorded hashes. Stock gofmt makes the entire original public-boundary and child functions byte-identical to the final functions, preserving assertions and payloads. Final helper results are 35 passing tests, shared command packages 57, bounded target controls 43 and architecture/public-signature/purity controls 3, with zero failures/skips. Check-only generator validation, standalone errortype and formatting pass. The final 29-case public probe matches fresh BASE, with all 18 observed children reaped and GOPATHs removed. Both literal nonempty platform inventories and source digests match; exact prepared-pin execution stays behind its existing gate.

Root independently reran the final helper suite with 35 passes in 2.557 seconds and executed the actual integrated lint gate against the original `951c5516e9e7b3066e7e069adda9565cfd68844c` base. Make exited 2 in 192.966 seconds across 55 host packages. Its complete stdout is byte-identical to task44's retained observation, with 319 unchanged findings (252 errcheck, 3 exhaustive, 11 forbidigo and 53 staticcheck). Integrated errortype is UNREACHED. Unfiltered scoped lint retains two inherited ST1005 findings in unchanged build context. Root verified all 988 final worker input hashes, raw command records and the same 990-input root command manifest.

Fresh independent source and evidence progress reviews permit this checkpoint. They establish no formal SHIP or DONE. Root corrected two evidence wording errors and retained exact historical runner/test preimages; no raw capture changed. The stock Go1.27.1 linux/arm64 executions are developmental. Source binding covers selected nested inputs, not a full repository/toolchain closure. Original predecessor, matched-first-baseline, preservation, full/default/functional/affected, formal/native Darwin and remaining static acceptance stays open wherever unproved. Linux remains deferred and nonblocking under fn128.

Pointers are `implementation/handover.md`, `implementation/evidence.json`, `implementation/verification.json`, `implementation/runner-history.md`, `implementation/red-test-history.md`, `root-verification.json`, `root-integrated-lint/`, `root-adapter-focused/`, `source-progress-review.md` and `evidence-progress-review.md`. Root commits this verified progress separately before further source work.

stage: wave-dispatch - ran (single source/cache writer; root-owned lifecycle and commit)
stage: impl-review - skipped(policy: original required gates red/unavailable; bounded progress reviews retained)
stage: completion-review - skipped(policy: task and spec acceptance incomplete)
stage: plan-sync - skipped(config: planSync.enabled=false; no task reached done)
stage: QA - skipped(policy: no separate live application surface in this source checkpoint)
Tier: session (jev-unavailable(no_key)). Writer and fresh reviewers were dispatched to the Sol/high tier; execution metadata was not exposed.
Tracker sync: n/a (bridge inactive).

### Reviewed public overflow preservation progress, 2026-10-08

The three-line public collection-boundary projection restores first-baseline generic overflow text/type while preserving the private typed refusal and context/error ordering. Additive real-process tests retain behavioral RED then 10 GREEN passes. All 11 public and 18 ordinary preservation outcomes match the first baseline literally; 286 affected tests pass with 10 original skips. Architecture, check-only generation, standalone errortype, fast Make lint, formatting and both supported source inventories pass. Fresh independent SOURCE-PROGRESS review has no actionable findings. [Root verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/resumption-20261008/root-verification.md) binds the reviewed evidence and audits.

This is verified progress only. Scoped lint retains two findings; integrated lint retains 24 with integrated errortype unreached. Default command RED and prior upgrade publication RED remain unresolved; two fresh-root upgrade passes do not diagnose the old failure. Task40 is now in_progress for separately admitted test-readiness repair, with its original dependency and genuine cleanup-fault acceptance unchanged. Task9 stays in_progress; formal completion and every other unproved source obligation remain open. Native fn128/fn149 stay deferred/unverified. Root commits only this reviewed progress; no push, PR or CI follows.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [x] Target compilation, Go identity queries and listing run through one private Go-command adapter built on `hostexec`; `target` no longer calls `exec.Command` directly for them.
- [x] Structured-output overflow is rejected and never decoded; compiler diagnostics are bounded head/tail with full hashes.
- [x] Tests through the adapter cover cancellation and process termination, long diagnostics, structured overflow, malformed listing, build failure and build-cache lock release on every path.
- [x] Timeout, unsupported capability, invalid-input diagnostics and cleanup failure remain distinct errors with their existing wrapping and precedence.
- [x] Fresh and cached preparation produce the same `Prepared` value and provenance as before for a fixed fixture.

Current adapter integration gaps (historical checked criteria above are unchanged):

- [x] The public adapter listing uses Compatibility with the measured finite per-stream capacity and watchdog, preserving its public API, command request, error and cleanup contracts.
- [x] Fresh BASE/final literal nonempty inventories/digests for both source sets and the public process probe match under only recorded volatile normalization; exact-pin qualification stays behind its existing gate.
- [x] Focused boundary/projection/error-priority tests and actual mechanism process controls establish overflow refusal, bounded capture and physical cleanup without decoding partial structured output.
- [x] Source-bound focused/generator/architecture/format/errortype/unfiltered lint and independent progress reviews are retained; failures and unreached stages remain explicit.
- [ ] Original predecessor, preservation, first-baseline, full/default/functional/affected/formal/native Darwin and static both-source-set requirements are satisfied before task9 DONE; the source-only start exception waives none of them.
## Done summary
Won't do: native backend retired by the 2026-10-19 owner decision; closed without implementation.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
## Original-base lint clarification (2026-10-09)

The resumption packet's 24 integrated findings use the later `d635e23f00d926a43b942f25a9d05bd0ccb72025` revision filter. Root now records the original `951c5516e9e7b3066e7e069adda9565cfd68844c` comparison as authoritative for retained source acceptance. Task19's fresh before/after receipts report 215 to 213 findings across 55 host packages; integrated errortype remains unreached. [Baseline accounting](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/digest-lint-20261009/lint-baseline-accounting.md) retains both scopes and exact ownership counts. Historical raw packets remain unchanged. The two scoped target-context findings remain open and are included in that original inventory. Task40's separately reviewed readiness repair resolves its admitted default fixture failures; genuine cleanup faults and the prior upgrade publication RED remain unproved. Task9 stays in_progress with its original predecessor, preservation, full/affected/functional and formal source requirements intact. Native fn128/fn149 remain deferred/unverified.
