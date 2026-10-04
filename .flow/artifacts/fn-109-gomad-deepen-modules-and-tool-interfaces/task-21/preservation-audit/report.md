# R18 preservation source audit

R18 remains open against the actual first-task baseline. The exported surface and CLI inventory include separately owned changes, and fn-109's interface record omits additional public campaign helpers. This audit identifies those differences and retains current source checks. It grants no native qualification or review verdict.

## Inputs and receipts

`inventory.json` retains complete pre/post hashes for the reconstructed 670-file baseline at `/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3` and the current 978-file nested module. Both inventories remained unchanged throughout the inventory run. The baseline `source.sha256` check exited 0. Its manifest hash is `d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`.

Every Go command uses `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`. The execution host is Linux aarch64. `go doc -all` ran for 17 public packages across the eight requested surfaces with each of `GOOS=darwin GOARCH=arm64` and `GOOS=linux GOARCH=amd64`. These cross-platform declaration reads are source evidence. They do not run the native platforms. The 69 command receipts retain argv, cwd, environment overrides, UTC start/end, monotonic duration, exit code, output path and SHA-256. All commands exited 0.

`public-api.diff` is the full baseline/current diff for both platform selections, including comments and declaration order. Every raw document is retained as `<platform>-<role>-<package>.godoc.txt`. `provenance.json` contains 202 declaration differences, current source references, declaration-line Git blame, added/changed field blame, introduction histories for added declarations and full commit bodies. A declaration-name mention in the interface record is a retrieval signal only. It cannot approve a migration. Named result renames, declaration order and constant initializer rewrites can appear in these mechanical differences without changing the public contract.

## Public Go migration

The inventoried executor removals, owned Artifact handle migration, public detached Runner/Target inspection types, source-inventory removal and World detached terminal boundary are present. Their source boundaries remain the ones `go-interface-changes.md` describes. World `Recorder.FinishError` rejects custom or external-wrapper inputs without callbacks; callers normalize at `world/process.Session.FinishError` or submit `FinishTerminal`. That intentional admission change must remain explicit. It is not evidence of preserving arbitrary-error behavior.

The record's task-2 statement of no exported changes and task-5 list of two additions do not account for these public declarations in `runner/campaign_options.go:109-175`:

- `NotSingleBaseSeedError`, `SemanticCoverageRequiredError` and their `Error` methods.
- `NormalizeStrategy`, `NormalizeCoverage`, `ValidateCoverage`, `ValidateChoiceTraceLimit`, `ValidateChoiceCoverage` and `ParseSingleBaseSeed`.

Current Git blame and `git log --all -S 'func NormalizeStrategy('` place these helpers in `a3b9f80efab9356c0be2080779133337e2471ac0`, whose subject is `wip`. Its campaign_options.go addition contains the public helpers. The task-2 source commit `7fe6c833c0` does not independently establish their introduction, and the two task-5 parsing additions trace to `f608449c01`. Therefore this report records the helpers as an uninventoried public addition with a WIP-history ownership gap. It does not manufacture a task-2/task-5 acceptance result.

Other changes have independently identifiable owners and remain differences against the actual baseline:

| Changed public area | Source/Git evidence | Owning work |
| --- | --- | --- |
| Diagnostics, divergence evidence and qualification diagnostic fields | `df2642da26`, `e153d05a76`; choice/diagnostic*.go, runner/diagnostics.go, qualification/qualification.go | fn-112.3/.4 and its determinism follow-ups |
| Scheduled soak package and `set.SoakQualifyArguments` | `bae373d147`; qualification/soak, qualification/set/soak.go | fn-112.10 |
| Adapter regeneration graph, errors and Target prepared-source digest | `076cdcc344`; deterministicio/adapter_regenerate.go, adapter_rewrite.go, target/adapter_source_set.go | fn-113.2 |
| Analysis compatibility/profile helpers and v041 pack retirement | `7fd67d5aae`; qualification/analysis, removed v041 pack/request | fn-113.3 |
| Choice start ordinal, inspection decisions and stop reason | `80fcf2cb44`; runner and CLI start-ordinal fields | fn-114.6 |
| Guidance summary and regression mode | `2c3de4c982`, `74a53f05b9`; runner/guided_selection.go, CLI guidance | fn-114.7 |
| Minimize resume request flag and workspace scoping | `d00f803843`, `406a354f20`; runner/minimize_operation.go, CLI minimize | fn-114.8/.15 |
| Artifact target pool, sharing and retained-byte APIs | `bc2e970b53`, `6f66f744dd`; artifact/target_pool.go, retained_bytes.go | fn-114.9/.10 |
| Readiness types, `Record.Readiness/Origin`, `ReplayPlan.Readiness`, wire version 3 | `00633b2b55`; choice and generated wire | fn-114.11 |
| No-op shape list and omitted-readiness summary | `1b970bc144`, controller change `b666b41c09`; choice/no_op_select.go, Runner exploration summary | fn-114.12 |

The exhaustive symbol-level history is in `provenance.json` and `history-*.log`; the table groups it for review. A WIP or merge commit can be the last declaration-line editor while the changed field comes from one of the named semantic commits. The field blame and introduction history expose that distinction. These independently delivered changes need reconciliation with R18's first-baseline preservation condition and the intentional migration record. Their presence alone neither proves accidental breakage nor permits an unchanged-inventory claim.

## Recorded identity migrations

The baseline's generated choice profile is `gomad3-choice-trace/v2`, with wire version 2. Current `choice/internal/wire/wire_generated.go:12,21` uses profile v3 and wire version 3. `choice/trace.go:140-144` explicitly refuses v2. fn-114.11's acceptance explicitly requires raising the wire version and visibly rejecting the previous version. Commit `00633b2b55` records the migration and the repinning of diagnostics-off goldens. Version 1 remains the separately supported legacy path. A current passing golden that was repinned to v3 cannot establish byte equivalence with the first-task v2 baseline.

The exploration controller identity is a separate migration. `runner/internal/exploration/choice/engine.go:25` uses `deterministic-rounds/breadth-first-rank-prefix/v3`, introduced by `b666b41c09`. fn-114.12 records the intentional refusal of controller-v2 journals on resume. It is not the Choice Trace wire profile, and changing either identity cannot be used as evidence that the other stayed stable.

These are known, deliberate independently owned migrations. The fn-109 aggregate record currently omits them. Exact fixed-identity canonical equivalence across the first-task baseline and current integrated tree remains incomplete outside the retained specific projection vectors.

## CLI inventory

All 15 public `gomad` commands remain in the dispatch source. The only removed registered flag is private `--__plan`, which fn-109.5 intentionally replaces with direct plan dispatch. Private child mode dispatch moved into the application owner.

The baseline/current inventory adds public `--choice-start-ordinal` (fn-114.6), `--diagnostics` (fn-112.4), `--guide-regression` (fn-114.7) and minimizer `--resume` (fn-114.8/.15). The guidance default now excludes answered corpus seeds as fn-114.7 requires. New maintainer commands include adapter regeneration, diagnostic-diff, pin-impact and soak, with the owners above. `cli-inventory.json` retains function, FlagSet, file, line and registration expression, including literal defaults; `cli-inventory.diff` retains every registration/dispatch difference.

The current guide mentions the newly added public flags, but nine existing registered user flags have no literal mention in `CLI.md`: `env`, `io-ro-mount`, `max-bytes`, `min-free-bytes`, `observed`, `prune-qualified-artifacts`, `terminate-grace`, `toolchain-root` and `world-transition-limit`. These are documentation inventory gaps rather than demonstrated parser removals. Their registrations were already present in the baseline. This static registration inventory does not independently prove every validation precedence and default path.

## Comments and policy boundary

The owning-file source comparison found 33 deleted standalone comment lines. Eighteen still have exact occurrences elsewhere in the current module. Fifteen have changed spelling or wrapping. `deleted-comments.json` retains every removed line, its original file/line and exact current occurrences. Spot-checks found preserved meaning in the expanded cancellation characterization, adapter rewrittenModule explanation, shared-target prune explanation and the MILESTONES filename references. The runtime forward-clock explanation and clock inventory descriptions changed with independently owned clock semantics. No moved-code comment loss was established by these checks. This is a source spot-check, not an assertion that every comment is byte-identical.

`deterministicio/boundary/manifest.json` is byte-identical to the baseline. Current forbidden-import evaluation in `target/internal/capabilitypolicy/policy.go:131-133` retains syscall, os/exec, os/signal and golang.org/x/sys rejection except exact pack selection. The changed Linux v047 pack/request alter only profile implementation pins and approval/request digests, with unchanged capability/rule/source sets. The v041 pack/request removal belongs to fn-113.3's documented retirement of the unselected variant. This bounded audit found no new generic host-I/O grant. The removed workload/variant remains an intentional independently owned baseline difference that must be disclosed.

## Current fixed-projection checks and reusable evidence

`focused-preservation-receipt.json` records one pinned-stock run against the current source. It exited 0 in approximately 0.699 seconds, selected 18 top-level tests and 55 named cases, and skipped none. `focused-preservation.log` retains the real execution output. Pre/post hashes match the full current module inventory. Tests cover terminal bytes and all categories, callback rejection, sentinel rebinding, process classification precedence, composed record/failure/encoded hashes, timestamp spelling, complete capability evidence/nil/empty/order/detachment, literal capability canonical vectors, Runner report projections, Artifact manifest copying and diagnostics-off report field absence.

This command did not execute a matched first-task baseline, patched process/runtime, or either native platform. `TestDiagnosticsOffPreservesExistingCanonicalIdentities` additionally requires darwin/arm64 and skips on this host, so it was excluded from the selection. Its repinned v3 snapshot cannot close first-task v2 equivalence.

The task-19 `round4-final-source.sha256` still matches all production and test files. Exactly five documents differ after task 20: ARCHITECTURE.md, CLI.md, README.md, SPEC.md and TUTORIAL.md. `inventory.json` lists expected/current hashes. Independently checking `round4-command-logs.sha256` found all 138 retained logs unchanged; the log manifest hash is `f432cbf3c4afd4afa95494fcb312ef22afd4acbba853a2f461f6a88de449f87d`. Those retained source checks and projection vectors may be reused with these explicit source bounds. They do not become a fresh full-tree qualification result.

Constructor aliases and detached-value equality do not exclude transient full-payload copies. R19's separately assigned bounded memory/copy measurements remain necessary.

## Open preservation obligations

- Reconcile the uninventoried public campaign helpers with their WIP-history introduction and owning fn-109 operation contract.
- Link each independently delivered API/CLI/format/variant migration into the aggregate first-baseline preservation audit. In particular, disclose Choice Trace v2 refusal and controller-v2 journal refusal as separate changes.
- Complete the CLI documentation inventory or explicitly retain its nine existing gaps under the documentation owner.
- Retain full matched fixed-identity comparison evidence where applicable, separating deliberately versioned changes from preserved projection vectors.
- Run required native gates on darwin/arm64 and linux/amd64. Current stock-host checks and reused historical evidence close neither platform.
