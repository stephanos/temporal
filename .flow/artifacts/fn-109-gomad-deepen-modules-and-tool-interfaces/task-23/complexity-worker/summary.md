# Task 23 complexity correction

The admitted two-file correction removes the lint router's three current complexity findings. Task 23 remains `in_progress`; the original-base fast gate still fails on 50 unchanged Gomad findings, so the conductor retains acceptance and review.

Tier: session (jev-unavailable(no_key)), explicit project route authoritative.
Requested implementer `gpt-6.1-sol/high`; executed child-model metadata is unavailable. No child agents or implementation bridge ran.
stage: impl-review - skipped(policy: conductor owns review and lifecycle under MILESTONES)

## Source checkpoint

Checkpoint A is `7dc3849ee300b31c419b7e23a56c36ff1f962894`, based on `671750104b609e4ddba31a096ae0031c4380b5ae`. It changes only `cmd/tools/lintcode/main.go` and `main_test.go`, with 147 insertions and 33 deletions. Checkpoint B carries this compact packet; its ID is supplied in the worker return because a commit cannot embed its own ID.

The private `registerHostPackages`, `validateGomadHostSource`, `packageSourceCoverage`, and `classificationPaths` phases move existing logic. Reconstruction checks recover every original production byte by reinlining those phases, and recover every original test byte by removing the new controls and `io` import. The checks preserve `regularHostDirectory`, exact-directory admission, fixture/overlay precedence, incremental registration, AST error order, metadata error order, all five source categories, sorted/deduplicated package arguments, and exact invocation behavior. All 5,167 other bound product inputs remain unchanged, including the sole classification literal in `architecture.go`.

The original admission, authoritative primary spec and task were read before edits. Primary spec SHA is `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`; task SHA is `81bf456605ad8f5b6332aba5667a522b3c4f0dccd6526061b0f7be179b7b54ec`. The intermediate cognitive-26 finding justified the fourth extraction, admitted by the conductor in primary commit `c2b92ec179115228bc47ad439d23bd3c63bff908`. The original admission remains unchanged.

Final source SHAs are `cb96fa8a2a8b73ed1d50c8ee201a5048fe20c349c9aebe76342ed3737365c986` for `main.go` and `f7ad16372cae34ee431d0f0b07dd9d84cf54cd1d2b2e02e2fb2a4030a7738627` for `main_test.go`.

## Observed gates

Receipts and raw logs stay in `.flow/tmp/fn10923-complexity/`. Each `<name>.json` points to its immutable input binding and raw log, with exact argv/CWD, actual tool paths and hashes, selected environment, raw Go settings, UTC times, numeric exit, elapsed time, and before/after source hashes. `evidence.json` lists the names and commands.

| Receipt | Exit | Observation |
| --- | ---: | --- |
| before-helper-lint | 1 | Actual unfiltered RED3, cognitive 35/28/27 |
| before-helper-tests | 0 | 85 emitted PASS outcomes, one intentional SKIP |
| before-characterization | 0 | 13 emitted PASS outcomes against unchanged production |
| after-helper-lint | 1 | Intermediate RED1, loadOwnership cognitive 26 |
| final-helper-tests | 0 | 98 emitted PASS outcomes, same SKIP; every original outcome preserved |
| final-helper-lint | 0 | Unfiltered pinned tool, zero issues |
| final-helper-vet / final-helper-errortype / final-gofmt | 0 | Helper vet and standalone errortype pass; gofmt produces no diff |
| final-ownership / final-architecture | 0 | Each focused test executes; architecture discovery covers both supported source sets |
| final-validate | 0 | Canonical nested Make validation, including actual root `./tests` inventory |
| final-fast-lint | 2 | Original base `951c5516e9e7b3066e7e069adda9565cfd68844c`, 159.201685618 seconds; root and integration lint/errortype advance, nested lint RED50 |
| final-gomad-lint | 2 | Explicit canonical nested gate, same RED50 |
| final-mixedbrain-lint | 0 | Separate actual mixedbrain lint and errortype complete |
| final-preservation-check | 0 | Exact reconstruction, test outcome preservation, protected-input equality and RED50 block equality |

`TestLintRuntimeHostRegistration` covers valid absent directories, an absent leaf below a symlink ancestor, and partial registration before the first invalid entry. `TestCoveredPackagesMetadata` covers EOF, malformed/truncated trailing JSON, package-before-dependency errors, the first dependency error, all five source categories, deduplication/sorting, and uncovered source rejection. These characterization controls passed before extraction; the pinned standards analyzer supplies the failing regression contract. The retained original-base task-73 baseline separately reports loadOwnership cyclomatic 30, classify cognitive 28, and coveredPackages cognitive 27. Its cyclomatic metric must remain distinct from this worker's actual cognitive-35 diagnostic.

`TestLintPolicyRealGolangci` intentionally skipped because `LINT_POLICY_GOLANGCI` was unset. The suite supplies no fresh actual-tool policy-fixture execution. Policy, config and fixture inputs remain unchanged; this correction uses bounded retained policy coverage plus fresh unfiltered actual-tool helper lint. No full suite was rerun to hide that skip.

Both final Gomad logs retain exactly the same 50 three-line diagnostic blocks as task 73's `after-aggregate-lint.log`. Their ordered block SHA is `034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea`, comprising 42 staticcheck ST1005 findings and eight forbidigo findings. No admission authorizes changing those error strings or panics. Nested errortype is unreached in both red routes; mixedbrain is unreached in the fast route and passes only in its separately captured invocation. The sequential Make recipe supports stage reachability; hidden `@` recipe commands are inferred from that bound recipe rather than separately traced child argv.

## Evidence limits and ownership

All gate handles exited, every owned process group was inspected empty, and no timeout occurred. A final process-table inspection found no live Go, lint, Make, generator, runner or check command. No child agents require reconciliation. The worker explicitly releases the shared Go lane in its return to the conductor.

The captures use pre-existing shared Go/module/lint caches and inherited nonselected environment; they are nonhermetic. The initial runner broadly normalized numeric `go-build` strings in `GOGCCFLAGS`. The final preservation check independently reconciles retained raw settings with the narrower temporary-path rule and retains both executed runner preimages under `.flow/tmp/fn10923-complexity/run-preimage-*.py`; this later check does not retroactively improve the original capture. Tool bindings omit `/usr/bin/ps`, which the process-group observer used. That historical observer-tool capture gap remains explicit.

Source history identifies the regression in primary `d5374728` following original task-23 routing `ce80d242`. No competing source repair was found. External prior-fix lookup remained unchecked after invalid GitHub authentication; no network writes occurred. No bisect worktree was created under the conductor's workspace prohibition. No live-app surface applies.

The worker used systematic debugging, TDD characterization, code-style and verification skills for the source correction, and the Flow prose contract for this packet. No lint policy, gate, test assertion, baseline, exclusion, Make recipe, module file or classification inventory changed. No ordinary Runner-wide rerun or unsupported native toolchain attempt ran. Native Linux/Darwin qualification remains deferred to fn-128/fn-149; these linux/arm64 host-source results supply no full native host pass. The conductor owns integration, independent review, the remaining RED50 assessment, and every Flow lifecycle transition.
