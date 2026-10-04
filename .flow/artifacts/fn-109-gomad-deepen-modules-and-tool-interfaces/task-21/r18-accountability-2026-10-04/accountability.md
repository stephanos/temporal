# R18 campaign accountability supplement

Task 21 can now assign its eight named campaign seams to the original fn-109
task 5 / R6 operation and reuse task 26's actual CLI migration. All eight
already appear in the current interface inventory. Task 20 also delivered the
nine missing flag descriptions. This checkpoint adds ownership evidence;
original R18/R19/R20 acceptance remains REQUIRED/OPEN.

All current-source citations refer to frozen BASE
`0988ab1041c5580b2d02763088d5048d6ee70586`, branch `gomad`. Historical commits
are named separately. [Root admission](root-admission.json) defines the write
scope. [Evidence](evidence.json) binds the Git blob IDs, complete-file SHA256,
cited line intervals and interval SHA256 observed by the
[read-only verifier](verify_accountability.py). The scratch research note is
an input bound by SHA256
`7ba0029eedfdaf35d93e9765cc0e44a6af5ca8c993a193ddf0782f904afe9410`;
the primary Git objects and retained owner artifacts below supply the evidence.

Bounded verification passed with 159,733 executed assertions, 83 frozen
references and 31,872 protected tracked paths, including one uninitialized
Gitlink. `flowctl validate --spec fn-109-gomad-deepen-modules-and-tool-interfaces
--json` exited 0 for all 36 tasks. Fresh `flowctl show` reads retain task 21
blocked and the parent open with 2/36 accepted. No-index whitespace checks of
the three new files returned 1 for content difference with no whitespace
diagnostic. These are evidence/document checks and supply no source or native
acceptance. Early verifier attempts rejected an incorrectly assumed task-text
token, a Gitlink treated as a file blob, and an unsupported ancestry assumption;
the final verifier checks the actual task text, the Gitlink's recorded identity
and empty checkout, and AdapterRegeneration's separate original Git object.

## Named seams and actual consumers

[Task 5](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.5.md),
lines 4–17, assigns presence-neutral semantics to Runner using
[task 2's options owner](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.2.md),
lines 14–21. [Task 26](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.26.md),
lines 4–28, owns the corrective CLI migration, committed in
`fec3ce56e7ea2f1498617148cb2c6d47f1297bc0`.
[Its independent source review](../../task-26/independent-source-review.md),
lines 9–35, retains behavioral and external compilation checks with open
qualification. Unknown external consumers remain an evidence limit.

Declaration lines below are in
[campaign_options.go](../../../../../tools/gomad3/runner/campaign_options.go).
Each row's original operation owner is task 5 / R6, with task 2's private
options value; each row's first integrated Git introduction is
`a3b9f80efab9356c0be2080779133337e2471ac0` (`wip`).

| Named public seam | BASE declaration and consumer evidence |
| --- | --- |
| `NotSingleBaseSeedError` and `(*NotSingleBaseSeedError).Error() string` | 109, 111; `requireSingleBaseSeed` at 186–191 constructs it. CLI at 838–846 and 862–870 translates cardinality through `errors.As`. |
| `SemanticCoverageRequiredError` and `(*SemanticCoverageRequiredError).Error() string` | 115, 117; `ValidateCoverage` at 145–159 constructs it. CLI at 951–958 translates the probe-facing message. |
| `NormalizeStrategy(Strategy) Strategy` | 121–126; options normalization at 271–273 and `campaign_plan.go:90–97` use the seed default. |
| `NormalizeCoverage(CoverageMode, bool) CoverageMode` | 128–136; CLI at 923 supplies only absent guided coverage. Runner at 1316 and 1330 retains the explicit coverage requirement for direct guided requests. |
| `ValidateCoverage(CoverageMode, []string) error` | 145–159; Runner at 1304 and CLI at 951 consume its errors. |
| `ValidateChoiceTraceLimit(uint64) error` | 161–166; Runner at 1298 and CLI at 971 use it. CLI's enabled-zero rejection still precedes it at 968. |
| `ValidateChoiceCoverage(CoverageMode, uint64) error` | 168–173; Runner at 1307 and CLI at 724 consume it before target parsing. |
| `ParseSingleBaseSeed(string) (SeedSelection, error)` | 175–184; CLI at 838 and 862 consumes it for both exploration strategies. Parse errors precede cardinality errors. |

The verifier binds the exact declarations, methods and calls in
[CLI](../../../../../tools/gomad3/cmd/gomad/internal/cli/cli.go),
[Runner](../../../../../tools/gomad3/runner/runner.go),
[strategy parser](../../../../../tools/gomad3/runner/campaign_plan.go), and
[external consumer declarations](../../../../../tools/gomad3/testdata/runnerconsumer/consumer.go),
lines 36–61. It checks source evidence without executing or claiming fresh Go
compilation. The [current inventory](../../go-interface-changes.md), lines
187–210, contains every member of this named set; this is a bounded inventory
result, not an exhaustive API audit.

## Exact original-owner chain

[Task 5's task-only.patch](../../task-5/task-only.patch), lines 551–633, contains
the complete 83-line helper block. Removing only the leading `+` from every
line yields the exact bytes of WIP `campaign_options.go:109–191` and BASE at
the same interval. All three block hashes equal
`6a393b209153a114ba174e9a9e7212ff6df2f15039cbdcff58d858dc33800de1`.
The complete retained patch hashes to
`a0f0b6c5c3b5963740b0aaab28f66d2f2d97b91ed972d16cfbe20266f670ffda`,
matching [final-source.json](../../task-5/final-source.json), line 50.
The [review-fix patch](../../task-5/review-fix.patch) hashes to
`a6509889521020edf1fe5c3a09f01bb3fc672361a1052a08ee1972145f1cb7f8`,
matching that receipt and
[parent-source-verification.json](../../task-5/parent-source-verification.json).
The parent receipt confirms the final source hashes and
`review_fix_source_bound: true`.

Whole-file bytes differ. WIP `campaign_options.go` SHA256 is
`d44433d2d17754ba846a4c8d71964e44736b1da04ccc8d132d0b2e2851393aed`;
task 5's final receipt records
`0d645d15651f0caad56a48175ce59c72bd0d37bb04431c1415ec1fe69a8020fb`.
The matched block assigns the original operation owner but establishes no
whole-WIP or whole-task equivalence. Task 2's source commit
`7fe6c833c0f03b46d9b2876d9f6a1b59d5f7c2ac` lacks these public declarations.
Task 5's earlier parsing commit `f608449c016b751a8fd4e19561518e979ebfc31e`
introduces `ParseStrategy` and `ParseCoverageMode`, separately from this set.
Historical task 5 acceptance and model/review statements retain their original
source bounds; they do not certify BASE or close current predecessor/native
acceptance.

WIP declaration blame also misleads for `deterministicio.AdapterRegeneration`.
Its primary origin is `076cdcc344ced6e1f6e195df84540c8ca74ca2f1`,
[`adapter_regenerate.go`](../../../../../tools/gomad3/deterministicio/adapter_regenerate.go),
historical lines 76–86. [Fn-113.2](../../../../tasks/fn-113-gomad-reduce-version-pin-maintenance.2.md)
owns that operation. The type was already declared at that earlier commit;
it does not belong to the campaign-helper ownership gap.

## Independent differences and existing owners

Current [guidance.go](../../../../../tools/gomad3/runner/guidance.go), lines
124–152, excludes answered seeds when regression mode is false. The matrix's
guided_selection.go name identifies its historical source, not a current path.

The immutable [audit](../preservation-audit/report.md), lines 24–39, and
[provenance](../preservation-audit/provenance.json) retain the mechanical
declaration differences, blame and histories. The following task records
state the owning operations in their Description/Approach. Evidence JSON
binds each task's lines 4–30, each full primary commit ID and each changed
source path named below. Independent authorization explains differences;
it supplies no fn-109 R18 preservation waiver.

| Difference | Existing owner | Primary Git evidence and source |
| --- | --- | --- |
| Diagnostics, divergence evidence, Runner/qualification fields, `--diagnostics`, `diagnostic-diff` | [fn-112.3](../../../../tasks/fn-112-gomad-determinism-assurance-and-test.3.md), [fn-112.4](../../../../tasks/fn-112-gomad-determinism-assurance-and-test.4.md) | `df2642da26`, `e153d05a76`; choice diagnostic source and `runner/diagnostics.go`. Disabled identities are a separate preservation requirement. |
| Scheduled soak and `set.SoakQualifyArguments` | [fn-112.10](../../../../tasks/fn-112-gomad-determinism-assurance-and-test.10.md) | `bae373d147`; `qualification/soak/soak.go`, `qualification/set/soak.go`. |
| Adapter regeneration graph/errors and target prepared-source digest | [fn-113.2](../../../../tasks/fn-113-gomad-reduce-version-pin-maintenance.2.md) | `076cdcc344`; `deterministicio/adapter_regenerate.go`, `target/adapter_source_set.go`. |
| Pack/profile analysis helpers, refresh and v041 retirement | [fn-113.3](../../../../tasks/fn-113-gomad-reduce-version-pin-maintenance.3.md) | `7fd67d5aae`; compatibility and `qualification/analysis` changes. Pin-impact itself retains fn-113.1 ownership. |
| Choice start ordinal, inspection decisions and stop reason | [fn-114.6](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.6.md) | `80fcf2cb44`; Runner and CLI. Default zero's fixed-identity requirement remains separate. |
| Guidance summary, answered-seed exclusion by default and `--guide-regression` | [fn-114.7](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.7.md) | `2c3de4c982`, `74a53f05b9`; `runner/guided_selection.go` and CLI. The default changed. Opt-in regression mode does not prove the original default unchanged. |
| Minimize resume and per-parent workspace | [fn-114.8](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.8.md), [fn-114.15](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.15.md) | `d00f803843`, `406a354f20`; minimizer source. |
| Target pool/sharing and retained-byte APIs | [fn-114.9](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.9.md), [fn-114.10](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.10.md) | `bc2e970b53`, `6f66f744dd`; Artifact source. Unchanged manifests and accounting remain separate requirements. |
| Readiness/Origin fields, replay readiness and wire/profile v3 | [fn-114.11](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.11.md) | `00633b2b558beb1c762decffe9d55b14631200c1`; choice schema/generated codecs. |
| No-op shape list, omitted-readiness count and controller v3 | [fn-114.12](../../../../tasks/fn-114-gomad-correct-search-path-defects-and.12.md) | `1b970bc144`, `b666b41c09ed152417e44b6c9738e16a58b8b43a`; choice engine/journal. |

Fn-109's own executor, Artifact handle, source-inventory and public
report/pack-directory/World migrations retain
[task 6](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.6.md),
[task 12](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.12.md),
[task 11](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.11.md) and
[task 19](../../../../tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.19.md)
as named in the [interface inventory](../../go-interface-changes.md). Task 19's explicit
World contract deliberately changes arbitrary-error callback admission and
exported-sentinel rebinding; those are actual differences.

## Reused correction and remaining bounds

[Task 20's correction receipt](../../task-20/cli-inventory-correction/acceptance-open.md),
lines 3–35, records the nine existing descriptions and 19 document checks.
Commit `b43aeb5b15d438eebab65f2ce48eecb19e76e55c` delivered that source progress.
The descriptions for `env`, `io-ro-mount`, `max-bytes`, `min-free-bytes`,
`observed`, `prune-qualified-artifacts`, `terminate-grace`, `toolchain-root`
and `world-transition-limit` are present in [CLI.md](../../../../../tools/gomad3/CLI.md),
lines 72, 126, 155–159, 264, 278 and 325–327. Reuse their document evidence;
inherited D5/formal/native acceptance stays open.

The [existing availability disclosure](../preservation-disclosure-2026-10-04.md)
already records three OPEN gaps and their existing owners. Its Choice Trace v2,
controller-v2 journal and selected v041 availability obligations remain distinct.
This supplement repeats no implementation or disclosure scope and authorizes
no removal, restoration, schema change or acceptance waiver.

The original first baseline remains `6782b55f49a0317b230e827ea2a63a37d116d502`
plus dirty fn-108 tasks 2–6, with 670 nested-module inputs. Its
[source manifest](../baseline-reconstruction/source.sha256) SHA256 remains
`d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`.
[Reconstruction](../baseline-reconstruction/reconstruction.md), lines 3–46 and
79–90, is nested-only. Later equivalent tree
`38957053f1ce342a8797af1803f5f8f6bb53fcad` corroborates that input and never
replaces the baseline or identifies the entire historical dirty repository.
Current-only 18-test/55-case projection evidence in the audit, lines 63–69,
does not complete the matched first-baseline fixed-identity comparison.
Error precedence, transactions, affected consumers and native/default behavior
remain unqualified beyond their retained exact inputs.

The historical [qualification-evidence.md](../../qualification-evidence.md)
remains frozen against `8604c07def0f97b63cbca3864b4c286d6803c4b1`. This dated
supplement supersedes ONLY its named-helper ownership/inventory and nine-flag
gap status for this checkpoint. Every other historical measurement, qualification
statement and limitation retains its original source identity. Protected tracked
bytes/modes are compared with admitted BASE, not with the original dirty baseline;
no whole-baseline preservation or source acceptance follows from that check.

Required full/native/formal gates remain red or incomplete. Both native
`darwin/arm64` and `linux/amd64` results, R18/R19/R20, task 21/predecessors,
shared fn-108 and affected-consumer qualification remain REQUIRED/OPEN. The
verifier runs no Go discovery, tests, build, lint, generation, toolchain, cache
or network operation. Task 21 stays blocked; the parent stays open. Root owns
document pointers, independent review, Flow records and the progress commit.
Baseline is inherited red/incomplete as retained in root admission and task 21;
this bounded document scope retries none of those unchanged failures.

Reproduce from the repository root with
`python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/r18-accountability-2026-10-04/verify_accountability.py`.
The command prints JSON and never writes files. Its assertion count and source
bindings are retained in evidence.json. The scratch-note hash is checked when
that non-shipping note exists; its absence after this evidence commit does not
remove the primary Git evidence.

Tier: session (jev-unavailable(no_key))

Requested writer routing is gpt-6.1-sol at high. Executed-model metadata is
unavailable; no historical or current executable-model identity is inferred.

stage: impl-review - skipped(policy: host-deferred; conductor owns source-evidence review, full/native/formal gates remain red)
