# fn-112.10 source-progress review

Verdict: SOURCE_PROGRESS_PASS for committing the bounded documentation corrections as retained progress. Ready to merge: No. Task acceptance remains open. This review supplies no formal implementation verdict, task completion, native qualification or measured determinism bound.

The six worker-owned changes accurately describe the existing implementation and deferred ownership. I found no introduced defect in those source changes. The root-owned task admission and milestone status changes preserve the acceptance requirements. Required affected adapter lint remains red, so this result cannot authorize completion or merge readiness.

## Scope and fingerprints

The review compares the actual uncommitted checkout with base and HEAD `15f56644664f3d3749bab2387aa97936a1cac6dd`. The commit range is empty. SHA-256 of `git diff 15f56644664f3d3749bab2387aa97936a1cac6dd --binary` is `056333077333d5d9b2cac6738422f8c303dac25588bca64ec1321b6148a897eb`.

Worker scope is `.github/workflows/gomad3.yml`, `tools/gomad3/CLI.md`, `tools/gomad3/README.md`, `tools/gomad3/TUTORIAL.md`, `tools/gomad3integration/README.md` and `tools/gomad3integration/qualification/soak.json`. Root scope is `.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.json`, its task Markdown and `MILESTONES.md`. There are no tracked Go changes. The protected untracked `.turbo` files are outside this review and were not modified.

The current aggregate source fingerprint, independently recomputed using `run_gate.py`'s sorted tracked-file/hash map and JSON encoding, is `5efeac884fbdb101b466760688d6837c7da15c1810985a76efaf47e3dfe8324b`. It matches the sealed source receipt. Sealed worker packet hashes are:

| Artifact | SHA-256 |
| --- | --- |
| evidence.json | faca94c064fbf81aded2cd030656130596e635997f3aa4d1e345eeb9d4bfda2a |
| handover.md | d872f7048f92e6a16cb5825185cc6065fdffa000afeacfc5742613910147ee67 |
| admission.md | d9d6828e578d92c5bf1dad37b0103191d203bc93603e95db79683784e00c7ec4 |

The requested reviewer and writer are both `gpt-6.1-sol` at high, in the same model family. Actual execution-model telemetry is unavailable. The root retained the optional judge's `no_key` result. This independent reviewer used fresh context and read AGENTS.md, the Gomad README, delivery milestones, task 10, its parent, the dated source/native ownership amendments, admission, handover, evidence, and relevant SPEC/ARCHITECTURE contracts. The reviewing, verification and prose skills informed the evidence-first review. The assignment limits this pass to source progress and read-only checks, so it does not dispatch a formal review workflow or rerun Go, build, lint or generation commands.

## Source findings

The CLI correction matches `qualification/soak/soak.go`. A positive `--batches` replaces the maximum and chooses `min(manifest.MinimumBatches, spec.Batches)` for the minimum. It may increase the maximum as well as reduce it.

The Linux smoke description matches the actual workflow predicate. It requires four selected/completed workloads, zero unsupported and infrastructure outcomes, accepted qualified/nondeterministic/replay-divergence classifications and exact replay for every qualified workload. Linux `intermittent` expectations remain intact. The integration guide now distinguishes historical clean runs from current qualification and identifies 149 generated targets as selection rather than execution evidence.

The sizing correction matches `previousRound + previousRound/5`. Both scheduled job matrices, seeds 11/17, the four smoke suites and guarded frontend probe, repeat 32, minimum two/maximum four rounds, two load workers, 120-minute budget, 55-minute qualification timeout and 180-minute job timeout retain their execution policy. Removing comment-only workflow lines produces identical executable workflow source.

Manifest bytes intentionally change from `373bbeba5aee579de261ae02793f20ceb67939649ef3e505c4b3265b6e61262b` to `6399fe11ec2db65a016c5a7b3713da52451e25b9be90fafd4de5148d2f401516`. The complete raw manifest is hashed into report and ledger-run identity at `qualification/soak/soak.go:280` and `:295`. The packet correctly discloses that identity change. Independently comparing parsed policy after excluding sizing prose and informational-reason values confirms identical execution policy. Cohort execution identity is unchanged by these prose fields.

README and TUTORIAL agree with the current per-platform, per-cohort cumulative clean-batch bound and diagnostics requirement. They retain closure-mode guard limits, uncontrolled-channel exclusions and the absence of any current native bound. Native execution and reports remain with fn-149.4 and fn-128.5/.7; D12 allowance removal remains with fn-128.2. No native owner is revived. The tracked task JSON's `todo` field is a lifecycle materialization snapshot; the task body already identifies flowctl as current status authority, and root confirms the live state is in_progress in `.git/flow-state`.

## Receipt review

Read-only checks independently verified all 26 gate raw-log hashes and receipt exit codes, eight sealed helper/tool/config hashes, all six worker source before/after hashes against the base/current files, and all 15 sealed audit input hashes. None mismatched. `git diff --check` returned zero. No command receipt was treated as a newly executed test in this review.

The actual JSON test events confirm the following retained observations:

| Run | Passed tests/subtests | Passed top-level tests | Failed leaves | Skips |
| --- | --- | --- | --- | --- |
| Initial wrong-cwd baseline | 0 | 0 | No product tests selected | 0 |
| Corrected original-filesystem baseline | 102 | 57 | 2 | 0 |
| Pre-edit local-temp baseline | 104 | 59 | 0 | 0 |
| Final local-temp soak/set | 104 | 59 | 0 | 0 |
| Final architecture/source sets | 9 | 6 | 0 | 0 |
| Draw-inventory negative controls | 2 | 2 | 0 | 0 |
| Integration manifest controls | 2 | 2 | 0 | 0 |
| Soak invalid-input controls | 5 | 1 | 0 | 0 |

Named soak controls actually executed, including AA-then-BB divergence, toolchain identity rollover, cumulative counts, earlier-run baseline comparison, trace/differ retention for cross-batch divergence, overflow exclusion and infrastructure classification. The trace-retention test uses injected qualification batches of three equal repetitions followed by three different equal repetitions. Together with the exact AA/BB ledger test, it supplies the stated portable controls without native execution.

The original failed leaves are `TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact` and `TestRunCountsRetainedRunnerFailureAsInfrastructure`. The first wrong-cwd receipt remains inconclusive. Original FUSE and local overlay runs preserve identical pre-edit source fingerprints while explicitly changing TMPDIR/GOTMPDIR. The raw hardlink controls record Nlink 3→3→3 on FUSE and 3→2→1 on overlay; read-only stat now confirms the retained shared files still have Nlink 3 and 1 respectively. This supports the pruning input diagnosis. It does not establish the separate safefile-sync failure's cause or a universal filesystem fix.

Current architecture and check-only validate receipts are green. The earlier architecture receipt crosses a documentation fingerprint change and is superseded by the frozen final run. A subsequent integration README clarification changes no Go bytes and is bound by the sealed source audit. The configured qualification-only lint receipt is green. Unfiltered qualification plus command-adapter lint remains red with 63 errcheck diagnostics. Fast lint's zero exit explicitly selects no changed Go packages and supplies zero adapter coverage. Historical global 208 diagnostics and the full command package's Git-fixture failure remain unresolved evidence; focused controls do not replace those gates.

The historical fn-111 procedure remains red before and after with 54 diagnostics. Its retained results were not rewritten. The independent 39 help entries, 37 local links and manifest/workflow checks are separate source checks. Their narrower green result cannot manufacture acceptance of the old procedure or native clock probe. The current helper hashes bind sealed helper versions; they should not be read as per-invocation hashes of earlier helper versions before the disclosed cwd/parser corrections.

## Actionable corrections and remaining requirements

One packet locator correction is required for accurate future routing. `evidence.json` and `handover.md` cite the first two soak-adapter diagnostics at lines 36 and 46. The actual retained configured-lint log reports `tools/gomad3/cmd/gomadtool/soak.go:37`, `:47`, `:55` and `:62`. This review is the append-only correction; preserve sealed worker artifacts.

Root must route the four task-related adapter error checks and the other 59 adapter diagnostics through their authorized source owner before declaring required affected lint green. The historical guide procedure also remains a separately unresolved red observation pending an evidence-backed disposition. These existing failures do not invalidate committing the bounded documentation corrections as progress, but they prevent formal acceptance, Done and merge readiness.

The review created only this artifact. All reviewer shell commands completed without live handles. It performed no Go execution, cache mutation, lifecycle change, stage, commit, push, PR, CI action or native qualification.
