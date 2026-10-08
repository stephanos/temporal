# Frozen lint-preservation sites for owner decision

This manifest identifies **52 unapproved sites** for a possible exact-site preservation decision. They are 42 ST1005 statements, eight invariant panics and two intentional spin loops. It grants no exception, changes no source and selects no lint configuration design. The two already-approved `SeedController.Complete` panics appear separately and are excluded from the request. The sleep at `toolchain/build_test.go:97` remains a separately fixable synchronization problem and is also excluded.

The source anchor is `1deced3efa4e7000163cb269e70e35f0c6b7dbd7`. Every source-file hash below was freshly read and compared with that commit's file bytes; all 24 matched. The retained [265-finding log](../../tmp/next-gate-8486dcb98d/root-full-lint.log) hashes to `c9a8008eba9d4aaa202e4eb44e113ebe99322d65e02dfee9c64da1edc5468bd5`. Its original comparison baseline is `951c5516e9e7b3066e7e069adda9565cfd68844c`, using pinned golangci-lint 2.13.0 and Go 1.27.1. This research ran no analyzer, tests, build or generator. Only this metadata note was written.

Paths and line numbers below are relative to `tools/gomad3/` at the frozen anchor. Each quoted statement includes its exact literal error message or format string. `%s`, `%d`, `%w` and `%q` are format directives; their rendered values remain supplied by the unchanged arguments. Approval of a format string must preserve those arguments, wrapping, evaluation order and return shape as well as its text.

The existing [owner decisions](owner-decisions.md) authorize only the separately listed controller panics and fn-114 migrations. [The bounded-batch assessment](lint-next-batches.md) records causal source owners and remaining ordinary repairs. Host dispatch explicitly requested gpt-6-astra/high; the unavailable routing judge (`no_key`) did not alter that request. Actual model execution metadata is unavailable. This manifest is research, not a formal review verdict.

## Forty-two ST1005 statements

| ID | Frozen path and line | Exact source statement and message |
| --- | --- | --- |
| E01 | `artifact/publication.go:39` | `return Artifact{}, errors.New("World payload paths must use the canonical artifact layout")` |
| E02 | `cmd/gomad/internal/cli/application.go:185` | `return "", fmt.Errorf("Runner is not a regular executable: %s", path)` |
| E03 | `internal/compatibilitypack/schema.go:275` | `return errors.New("Go source inventory is not sorted and unique")` |
| E04 | `internal/compatibilitypack/schema.go:316` | `return errors.New("Go source name is invalid")` |
| E05 | `internal/compatibilitypack/schema.go:319` | `return fmt.Errorf("Go source identity is invalid: %w", err)` |
| E06 | `runner/completion.go:39` | `err = fmt.Errorf("World record seed or schema does not match seed %d", seed)` |
| E07 | `runner/internal/execution/process.go:152` | `return fmt.Errorf("World record and transition limits must be positive")` |
| E08 | `runner/internal/execution/process_unix.go:722` | `return result, fmt.Errorf("World child record exceeded its configured bound")` |
| E09 | `runner/internal/execution/worldrecord.go:45` | `return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("World transition count mismatch")` |
| E10 | `runner/internal/execution/worldrecord.go:56` | `return world.Snapshot{}, world.Snapshot{}, fmt.Errorf("World semantic or transition identity mismatch")` |
| E11 | `runner/internal/execution/worldrecord.go:84` | `return Bundle{}, fmt.Errorf("World transition limit must be positive")` |
| E12 | `runner/internal/execution/worldrecord.go:147` | `return Bundle{}, fmt.Errorf("World transition payload requires %d bytes, limit is %d", len(transitionBytes), transitionLimit)` |
| E13 | `runner/internal/execution/worldrecord.go:188` | `return fmt.Errorf("World quiescence terminal has detail")` |
| E14 | `runner/internal/execution/worldrecord.go:192` | `return fmt.Errorf("World error terminal omitted detail")` |
| E15 | `runner/replay_operation.go:483` | `return nil, fmt.Errorf("World seed does not match target seed")` |
| E16 | `runner/runner.go:1189` | `return SeedSelection{}, nil, fmt.Errorf("Runner build identity is required for campaign resume")` |
| E17 | `target/internal/build/context.go:40` | `return Context{}, fmt.Errorf("Go target package %s is not a directory", source)` |
| E18 | `target/internal/build/context.go:59` | `return Context{}, fmt.Errorf("Go target package %s has no owning go.mod", source)` |
| E19 | `world/mailbox/mailbox.go:115` | `return world.Quiescence{}, fmt.Errorf("World delivered unknown mailbox event %d", delivery.EventID)` |
| E20 | `world/mailbox/mailbox.go:132` | `return Snapshot{}, fmt.Errorf("World contains an incompatible mailbox-owned request %d", request.ID)` |
| E21 | `world/process/config.go:28` | `return nil, fmt.Errorf("World transition limit must be positive")` |
| E22 | `world/process/session.go:28` | `return nil, fmt.Errorf("World child session requires a World")` |
| E23 | `world/process/session.go:40` | `return nil, errors.Join(fmt.Errorf("World child file descriptors are unavailable"), cleanupErr)` |
| E24 | `world/process/session.go:58` | `return nil, errors.Join(fmt.Errorf("World seed does not match the execution configuration"), output.Close())` |
| E25 | `world/process/session.go:108` | `return fmt.Errorf("World terminal error is required")` |
| E26 | `world/process/session.go:115` | `return fmt.Errorf("World child session is invalid")` |
| E27 | `world/process/session.go:146` | `return errors.Join(fmt.Errorf("World recording omitted its header"), session.output.Close())` |
| E28 | `world/recording.go:64` | `return nil, fmt.Errorf("World recording is already active")` |
| E29 | `world/recording.go:80` | `return Recording{}, fmt.Errorf("World terminal error is required")` |
| E30 | `world/recording.go:95` | `return Recording{}, fmt.Errorf("World error terminal kind is required")` |
| E31 | `world/recording.go:102` | `return Recording{}, fmt.Errorf("World recorder is invalid")` |
| E32 | `world/recording.go:107` | `return Recording{}, fmt.Errorf("World recorder is not active")` |
| E33 | `world/recording.go:121` | `return Recording{}, fmt.Errorf("World recording byte accounting changed")` |
| E34 | `world/recording.go:148` | `return fmt.Errorf("World quiescence terminal has error detail")` |
| E35 | `world/recording.go:152` | `return fmt.Errorf("World error terminal omitted detail")` |
| E36 | `world/recording.go:175` | `return fmt.Errorf("World terminal has no matching quiescence transition")` |
| E37 | `world/recording.go:179` | `return fmt.Errorf("World terminal does not match the final quiescence transition")` |
| E38 | `world/recording.go:228` | `return nil, fmt.Errorf("World recording exceeds its size bound")` |
| E39 | `world/replay.go:62` | `return nil, fmt.Errorf("World replay plan exceeds its bound")` |
| E40 | `world/replay.go:69` | `return ReplayPlan{}, fmt.Errorf("World replay plan exceeds its bound")` |
| E41 | `world/replay.go:72` | `return ReplayPlan{}, fmt.Errorf("World replay plan is not valid UTF-8")` |
| E42 | `world/replay.go:94` | `return ReplayPlan{}, fmt.Errorf("World replay plan is not canonical")` |

These statements preserve error bytes, classifications and wrapping at their current boundaries. Replacing no-format `fmt.Errorf` with `errors.New` still leaves the capitalized message subject to ST1005. Moving the literal into a variable, adding a `%s` indirection or moving construction behind a helper solely to evade the analyzer would conceal the same finding. This research found no grounded ordinary correction that both removes these diagnostics and preserves the frozen statements and their behavior under current authority. It does not claim every possible refactor has been disproved. Any genuine existing-owner correction must be evaluated separately and removed from a future exception request before approval.

## Eight unapproved invariant panics

| ID | Frozen path and line | Exact source statement |
| --- | --- | --- |
| P01 | `artifact/manifest_copy.go:59` | `panic(fmt.Sprintf("artifact manifest cannot copy a %s field", source.Kind()))` |
| P02 | `deterministicio/profile.go:211` | `panic(fmt.Errorf("encode deterministic I/O inventory: %w", err))` |
| P03 | `runner/campaign.go:11` | `panic("gomad3: duplicate execution completion")` |
| P04 | `runner/campaign.go:25` | `panic("gomad3: execution completion order has an unresolved gap")` |
| P05 | `runner/inspect.go:638` | `panic(fmt.Sprintf("unknown validated choice kind %d", kind))` |
| P06 | `runner/internal/execution/launch_plan_unix.go:321` | `panic(fmt.Sprintf("unknown descriptor resource %q", resource))` |
| P07 | `world/world.go:202` | `panic("invalid request state")` |
| P08 | `world/world.go:216` | `panic("queued request has no queued event")` |

The panic mechanism, payload value/type, conditions and ordering are part of the frozen behavior. A helper that calls panic, runtime-induced panic, returned error or process exit is outside this preservation request. No demonstrated ordinary correction removes these forbidigo diagnostics while retaining the direct statements. Any future semantic redesign needs its own scope and regression evidence.

## Two intentional spin loops

### L01. CPU-load workers

`internal/gomadtool/conformance/runtime_repeatability.go:310` reports SA5004 at the empty `default`. The containing loop begins at line306 in `startCPULoadWorkers`; the exact loop is below. Each worker has already locked its OS thread and announced startup before entering it.

```go
			for {
				select {
				case <-stop:
					return
				default:
				}
			}
```

The busy loop supplies host CPU load until its stop channel closes. Blocking, sleeping or yielding changes that stimulus. Rewriting the loop only to avoid its diagnostic supplies no correction to its intentional CPU use. Current authority grants no load-mechanism change. This request identifies this loop only, including its cancellation branch; another empty default or spin loop in the same function/file is excluded.

### L02. Unresponsive supervisor helper

`runner/internal/execution/process_test.go:1411` reports SA5002 on the empty loop. The exact containing function begins at line1407.

```go
func TestUnresponsiveSupervisorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_PROCESS_SUPERVISOR") != "1" {
		t.Skip("supervisor subprocess only")
	}
	for {
	}
}
```

This subprocess deliberately never responds. Blocking or yielding could retain nonresponse but changes the watchdog's current execution stimulus. An approved, regression-backed synchronization/stimulus redesign might resolve the diagnostic without an exception; no such design or authorization exists in this bounded research. The owner may decline this site's exception and commission that separate work. No arbitrary test-helper or file-wide SA5002 allowance is requested.

## Already approved and excluded

These two sites are covered by the [prior explicit decision](owner-decisions.md), not by the proposed 52-site request. Root owns their implementation and proof under fn-109.28. Preserve their pre-mutation rejection tests unchanged.

| ID | Frozen path and line | Exact source statement |
| --- | --- | --- |
| A01 | `runner/internal/campaign/controller.go:129` | `panic("gomad3: completed an inactive campaign attempt")` |
| A02 | `runner/internal/campaign/controller.go:132` | `panic("gomad3: completed a campaign attempt without a classification")` |

## Source SHA-256 bindings

These are entire file-byte hashes, not normalized AST or selected-line hashes. The binding includes surrounding conditions and control flow. Any changed file requires source rebinding and review of the affected frozen site before applying a future decision; matching only a familiar message is insufficient.

| Frozen path | SHA-256 |
| --- | --- |
| `artifact/manifest_copy.go` | `f1dad686d9f2fc59ebde1f0f484bafa6dcfad7fb1f84d1d3fe7878eeb44d7dae` |
| `artifact/publication.go` | `44d9d007b0b09397654e62e24596f8321e14366a6797a6f31fc63ec1fb827671` |
| `cmd/gomad/internal/cli/application.go` | `a9a94d5832e597babd74d331ed1a379e8a310c8db38fede6bcda7cc215f57a72` |
| `deterministicio/profile.go` | `cc4661f1a5f06acd3c07105b95eb72c1916a66cc55109d5a09b9fb066e4f03c6` |
| `internal/compatibilitypack/schema.go` | `449fb233c61b6a3691c913dde04e1630cfb1f4e9b2fffea82dacfeb18251f0ed` |
| `internal/gomadtool/conformance/runtime_repeatability.go` | `833989fd248cd1526395e5be90d954aa24fe4108adbcad406a7d6384cc1ab549` |
| `runner/campaign.go` | `f76d2c59da4fc899587c5ef4f048cecdde608589cd332dd85d0ee149a93d83e9` |
| `runner/completion.go` | `c2affbd7be5a16e010a50b09f6bedf3541320aced1be020eb4c42825e88045d5` |
| `runner/inspect.go` | `731851e8e3fda15a39734df3b0470aa3200f565d4743287b2098093ebfc34822` |
| `runner/internal/campaign/controller.go` | `043981c7ac6e97713971c62a308693089ec0f0499e75d52d6c7d9474eec843de` |
| `runner/internal/execution/launch_plan_unix.go` | `52e6c04e83e59be86ca580aa700130108280d65fdcb33da8f47e5082e69bf964` |
| `runner/internal/execution/process.go` | `b86a0493baac035cf932787853899a112469c9f3708746b29d6cc3af68e4e4a0` |
| `runner/internal/execution/process_test.go` | `4103eb4a7436b31da48b8917603c0e33af26915aa79dcef843bc37e6eaa05b4d` |
| `runner/internal/execution/process_unix.go` | `388f3756e72b11c5e52bc7c71911f04816af2a308d9b83966ce1c72138674df6` |
| `runner/internal/execution/worldrecord.go` | `45f18b3c145cb0ce36dd837c3f837b6d3bec26f4c44eb427b5749232538eb882` |
| `runner/replay_operation.go` | `24dbae80d5175671d6b89c163e35df3ab459c17afc5662ef2ad73110538bdb49` |
| `runner/runner.go` | `bdf6d21c8e09713446a803bc112b48db4eb50718827129bd53d4617ce4ff606b` |
| `target/internal/build/context.go` | `c998794068b981697124b63a8a8c544bc678d08bdb94e6873ae6cfc5356febed` |
| `world/mailbox/mailbox.go` | `3bc2d360dab408f4486a5df47b6618057f7aee7b0b3071e3a112cfdc9d896818` |
| `world/process/config.go` | `60310264d60091ae2ee9d6f13521fb976f7ca4fb54a740b78a3bd4584af06454` |
| `world/process/session.go` | `cce3686d15374f697ae063bba9a3c748a888a33ce9a4eae56f33690e1a257ae7` |
| `world/recording.go` | `fc00c9af42ebb2cb74fa593b9c026205a2df19bf4634e99a532f697c7fb4fc27` |
| `world/replay.go` | `3e32d751adb715f3ae1c3d0e66fdd40f05946ca77d251fb0c7ade417cde302ee` |
| `world/world.go` | `5de21e439b3f04aa8bb9dc44b6ce21c95b556952fb4b70d98f403dbb2d4b68c6` |

## Required evidence for any later approved implementation

The owner can approve or reject explicit IDs, with A01/A02 and the sleep excluded. A decision must retain each approved site's statement/snippet, message, path and behavior. This manifest supplies no authority until that decision is recorded. It requests no wildcard for `World`, `Go`, `Runner`, panics, test files, packages or staticcheck rules.

For each approved ID, the implementation owner must retain an actual pinned-tool diagnostic before the policy change and prove that the identical authorized statement is accepted after it. Unrelated diagnostics must remain visible in unfiltered output. Root must preserve source hashes and original behavior controls, including error bytes/wrapping, panic values and rejection order, CPU-load cancellation and the unresponsive supervisor stimulus. The final original-base gate and its reached stages remain separate evidence from these local policy controls.

Refusal controls must use the real analyzer and candidate policy in an isolated fixture or admitted test environment. They must demonstrate that a different capitalized message at an approved path, a different panic statement at an approved path, an otherwise identical statement at another path, and another spin loop at an approved path still produce their original diagnostics. They must also cover a duplicate same-message statement outside the authorized occurrence/context in the same file, so a path-plus-message match cannot silently authorize future sites. Unapproved IDs must remain rejected if only a subset is granted. These are proof requirements; no fixture or analyzer was executed during this research.

If a genuine behavior-equivalent source correction is established before approval, prefer its ordinary regression-backed disposition and remove that ID from the request. This manifest makes no claim that an exception is the only conceivable solution. It freezes the actual findings and current preservation boundary so the owner can choose without granting a broader waiver.
