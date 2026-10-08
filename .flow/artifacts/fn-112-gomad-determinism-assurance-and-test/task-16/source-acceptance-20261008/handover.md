# Task 16 source handover

Source-ready for conductor acceptance review; task remains `in_progress`. The historical collision fix is preserved, and a portable campaign regression now checks both publication-key policies end to end through real artifact and journal storage. No production code, native guard, identity projection, schema, lint configuration or dependency changed.

Tier: session (jev-unavailable(no_key)); explicit project implementer gpt-6.1-sol/high remains selected. The judge call occurred just after dispatch, not before. Executed model metadata is unavailable; no actual-model inference is made.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Scope and source proof

Worker base and unchanged HEAD: `4a64eb2cd68cc4040045d9d09bbd8bfc82b81819`, branch `gomad`. Worker made no commit, staging, Flow lifecycle, MILESTONES, PR, push or CI change. Conductor owns independent reruns, fresh source implementation review, Git and completion.

The sole product edit is `tools/gomad3/runner/internal/campaign/retained_evidence_test.go` (+140 lines). `input-proof.json` binds the exact current 1,280-file bounded source/configuration set, including the new fixture and actual `.github/.golangci.yml`, and separately identifies 22 implementation/review source bodies. Its source-map digest is `1121a50e9906298d7059ed3510f50d7e729c09f408b0eadf8017151cdea02770`. Every captured command with before/after snapshots has an empty source-change list; final acceptance commands bind this current source map. Earlier pre-edit captures are retained as baseline observations, not substituted for final proof.

The worker read actual AGENTS.md, Gomad README, MILESTONES, Flow usage/anchor, full task and owning spec, trusted blocked-reevaluation assurance, and relevant implementation bodies. The applicable work/defect route, TDD, writing-good-tests, code-style and prose instructions informed the bounded test and evidence. Existing artifact helpers and campaign lifecycle were reused; no new production abstraction was needed.

## Fix provenance and preservation

| Source lineage | Exact fix | Exact prechange | Meaning |
| --- | --- | --- | --- |
| Actual Darwin CLI correction | `a3b9f80efab9356c0be2080779133337e2471ac0` | `d635e23f00d926a43b942f25a9d05bd0ccb72025` | Default signature-key success collision falls back to full record identity; original native RED and real CLI seeds 1–2 belong to this lineage. |
| Earlier experimental harness | `6be0755fef727146f86bac6658436f6874680cc8` | `bce0407286458f995f640fba97b4708a5d05b412` | Execution-key policy adds execution-identity fallback; developmental linux/arm64 evidence is not qualified native proof. |
| Integration | `ca3345469a2541b7686606268b0f19cde030a8fe` | Both lineages | Current store retains both policies; `7f2bd1ed8d1e567072441e82c22c6dd431cc07cd` additionally pins first-success reuse. |

`input-proof.json` records subsequent source history and SHA-bound original task-16 receipts plus shared fn-114/task-13 integrated evidence, losslessly by reference. The current store, publication/open/copy paths, completion/retention, campaign validator, identity projections, Runner and both exploration call sites are in the bounded review surface; this is not an evidence-only empty review. Later integrated sources are compared individually, never credited by a whole historical checkout snapshot. Historical `upgrade/pin_impact.go` and its test no longer exist at those paths after the later upgrade/pinimpact extraction; absent historical entries are explicitly classified, not assigned invented current hashes.

Original noncollision literal assertions remain unchanged:

- Directory: `sha256-8804bc935588b0e0ac9fd7f890e4da67`
- Record hash: `sha256:27c9b74965e1b7cb30ef6f914b8f028eda0072216f5df3794bd84f8536db5f9d`
- Outcome signature: `sha256:8804bc935588b0e0ac9fd7f890e4da6718d567133466c52672ce1b11e9b454be`

No runtime/toolchain source changed. The exact 87-input runtime binding from fn-110/task-2 and fn-112/task-5 was verified against current bytes; 72 retained raw files (521,317 decoded bytes) were verified without copying their payloads. The actual 35,109,201-byte Go archive remains SHA256 `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`. Both supported-platform materialized source inventories are static input proof only. This narrow reuse asserts no equality of all files in an old 5,089-file checkout.

## Acceptance matrix

| Requirement/observable | Current source evidence | Limitation |
| --- | --- | --- |
| R9 distinct replayable retained successes despite equal outcome signature | Existing `TestPublishKeepsSuccessesWithOneSignatureAsDistinctReplayArtifacts` and `TestPublishKeepsEachExecutionOfOneOutcomeSignature`; new `TestOpenCampaignKeepsSameSignatureSuccessesDistinct` over default and execution key policies | Pure source coverage, not native execution. |
| R9 campaign acceptance, execution count, disk count, retained bytes and identities | New fixture seeds 7 and 8 produce equal signatures, distinct paths/record hashes; first signature directory preserved; journal publication and OpenCampaign accept exactly two executions/artifacts with matching counts/bytes | Fixture produces real storage/journal observations without launching a target. |
| R9 refusal of cross-wired retained execution evidence | Seed, ordinal and artifact-reference negative controls under both policies; nine passing test/subtest events total | Existing native execution/CLI assertions remain unchanged. |
| Exact-repeat idempotence and failure signature dedup remain unchanged | Original artifact regression bodies run in final portable suite; literal independent noncollision expectations retained | No alternate identity/hash schema introduced. |
| Test sensitivity to the actual collision behavior | Two separately retained current-source policy mutants each fail because published success artifact 2 does not match its campaign execution | Mutants are sensitivity controls, NOT exact historical portable RED. Historical native RED is retained separately. |
| Ordinary current-source regression and generated validation | Final artifact/record/campaign run: 155 top-level pass, 0 fail, 0 skip; four pure Runner retention controls pass; final `make -C tools/gomad3 validate` passes | No full native test-host aggregate claim. |
| Task-caused configured standards | Changed campaign package configured lint passes with 0 issues; six-package configured errortype vet passes; gofmt and diff-check pass | Broad aggregate lint remains red as detailed below, not waived. |
| Native CLI/Runner and qualification | Original real CLI reachability retained as historical provenance; current native obligations belong to fn-149/fn-128 | Current Runner assertion fails unsupported linux/arm64 guard; current CLI setup fails missing `.toolchain/bin/go` before collecting tests. Neither is a current pass. |

## Commands and standards

`evidence.json` indexes every recorded acceptance command with its exact argv, environment observation, exit, duration/counts and SHA-bound raw outputs. Per-command JSON retains timestamps and source/Git proof. The earliest artifact baseline passed 49 tests before any source edit; its inherited environment and full command duration were not snapshotted, so those fields remain explicitly unknown. Initial portable source coverage passed 154 tests; final coverage passes 155 after the new table-driven test. The initial single-policy new-test receipt is superseded by the final two-policy suite.

All Go tests used `test_dep`; no integration tag was used. Captured commands use stock Go 1.27.1 on actual Linux/aarch64, UID 1000, without changing host/build identity. No supported patched runtime is available here. Guard failures were observed once and not rerun to manufacture credit.

Configured `make lint-code-fast` against original actual-CLI implementation base `d635e23f00d926a43b942f25a9d05bd0ccb72025`, with `GOLANGCI_LINT_FIX=false`, exits 2 with 68 findings (64 errcheck, one forbidigo, three staticcheck). Explicit six-package configured source lint exits 2 with 12 findings. The immutable config, lint routing and exact lint/errortype binaries are bound in `input-proof.json`. `standards-attribution.json` binds every printed finding to source text, frozen-base equality and blame, and distinguishes original task-owned paths from later integrated owners. No finding is introduced by this worker or appears in the native task's four original source/test implementation paths; final changed-package lint is genuinely green. This is causal attribution, not a global-green claim or waiver.

The 12 six-package findings are the preserved production panic in `artifact/manifest_copy.go:59` (fn-109.12/19/21; fn-109.33 explicitly says not waived), writer/error handling and uppercase diagnostics in CLI construction/semantic owners (fn-109.4/26/21), and uppercase World replay error in `runner/replay_operation.go:483` (fn-109.12/21). Broader findings map individually to existing fn-109 correction/aggregate owners and fn-112 soak / fn-113 maintainer owners in the attribution file. Broad runner Touches grants no authority to repair unrelated lint or suppress anything; no suppression was added. Root must adjudicate the attribution under scoped source acceptance and retains the aggregate owners.

## Evidence integrity defect

The first `source-binding` capture failed (actual exit 1, 0.739 seconds, source_changes `[]`) when a historical moved path was read. Its original raw `source-binding.stdout` and `.stderr` remain byte-exact. After correcting absent-path classification, the proof emitter accidentally reused `source-binding.json` and overwrote that FAILED command's full JSON metadata. Root had retained key names only, so full failed timestamps/environment/source hashes cannot be restored losslessly. `failed-binding-observation.json` is explicitly a partial observation, not a reconstructed receipt or green result.

The successful proof bytes were mechanically moved with `apply_patch` to `input-proof.json`; `source-proof.json` is the distinct successful command receipt. The emitter now uses `input-proof.json`; capture rejects existing receipt labels. This bookkeeping loss is disclosed for source review and is neither production RED nor native credit. No earlier command JSON was fabricated to fill the loss.

## Preservation and conductor handoff

The two unrelated user documents are unchanged at their supplied SHA256 values: `.turbo/plans/gomad3-glossary-update.md` (`97868a86c0a263fbea61c336bd9557e4d71cf43ae4390e9a2449e6f7cd815188`) and `.turbo/technical-debt.md` (`c219247c01fb305592f0314ec46971cee30f00e5985dbd280c1c9e3aafe60287`). They are preservation checks only and excluded from every source-scope claim.

Reusable `capture.mjs` provides guarded receipt creation; `conductor-verify.mjs` is read-only and its capture import cannot execute a gate. The source binder refuses to overwrite an existing input proof before doing any work; the helper guard check observes refusal with protected evidence unchanged. Root may run `node .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-16/source-acceptance-20261008/conductor-verify.mjs`, then independently rerun final portable source selection and generated validation with fresh capture labels. `bundle-manifest.json` binds a worker-owned evidence snapshot, excluding conductor command receipts. An initial concurrent snapshot included conductor receipts; the worker regenerated only its own evidence/manifest with explicit exclusions, leaving conductor receipts untouched. Source-ready here is an implementation handover, not a review verdict. Fresh real three-axis source review and Flow completion remain root-owned.

All worker-started commands are closed. The Go/generator/cache lane is released to the conductor. No inherited-green receipt or native transfer was counted as current native proof.
