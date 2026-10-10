# fn-109.73 integrated source and evidence assessment

The integrated Unix mode fixture reaches its seven original real filesystem permission checks through the admitted private preparation assignment. This fresh assessment identifies no introduced code defect. The ordinary Runner and required lint gates remain red, so source acceptance remains open.

## Scope and identities

Assessment date is 2026-10-10. Requested reviewer is `gpt-6.1-sol/high`, in the same GPT family as the requested writer. Tier is session with `jev-unavailable(no_key)`. Actual execution-model telemetry is unavailable and is not inferred. This bounded assessment supplies no formal implementation-review, SHIP, Done or readiness verdict.

PRIMARY P is `/Users/stephan/Workspace/skunkworks/gomad/temporal`. Assessment began at HEAD `887f6978fcc0ba5437326c02feaf4c46b6ad422c`. During assessment root committed only orchestration documentation and the admitted fn-109.23 milestone status row as `671750104b609e4ddba31a096ae0031c4380b5ae`. The product candidate remained unchanged. The current milestone hash below binds that documented advancement; historical milestone hashes remain historical.

Integrated source A is `883eeaaefa0b9a68f59ef0d09796c75845b2a3f1`; integrated handover B is `c0b1fdf0a9a5ed7554242d2506a4427218a8d43b`. The actual immediate predecessor is `2d1806d6c27624dab0b65a217d71b6c5367eccfd`. Candidate C is P's `.worktrees/fn-109-73-unix-mode-candidate`, observed clean at worker B `0efeda7b4f28583e980b477afc959e5e16b1ae37`. Worker A is `0a2bfdeefded0ebc67f5f59dff4c442565708c41`, and its BASE is `ca4d8b88cf95da0b0a911efdc3a18f89bcbb68ae`.

I read PRIMARY's AGENTS.md, full Gomad README, MILESTONES.md, dirty fn-109 owner spec, task73, both admissions, source assessment, root integration, worker summary/evidence and all wrapper and checker sources. The dirty PRIMARY spec remains authoritative. I applied the Flow prose contract when drafting this report.

Paths beginning `task-73/` in the tables resolve under P's `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`. Other relative paths resolve under P.

| Consumed authority or handover input | SHA-256 |
| --- | --- |
| `AGENTS.md` | `8d634df5cbbbffd7dbada06e32b4d20d879707273f8be07bdf253b387211e6f3` |
| `tools/gomad3/README.md` | `fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610` |
| `MILESTONES.md` | `7f8bbd53d0f7de462c49cb354a448a4efaeb3b043d88fd3e6a59f4c7954167d7` |
| `.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md` | `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c` |
| `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md` | `1ccb6e2dfcd264c66af4c53826b2cf2bcd4ba825f65223ea82199f90195de092` |
| `task-73/admission.md` | `9fb8b3ad698b5d355e45229b48c96865d21ee13de1966bd7bd0bb45a464ee44a` |
| `task-73/worker-admission-20261010.md` | `f029f4773c87b0d9756406d9beb403e207ceb6a850eebfe913e2930c181d6c64` |
| `task-73/source-assessment.md` | `e73098ed78be771852bbb23991b5311995849b84cff3745921559babe2c01f00` |
| `task-73/root-integration.md` | `8e5ebab465d98a91c0952fd4131b9a1652b10282c1c4cd452b65077c9aa47e44` |
| `task-73/worker/summary.md` | `58a5cd7f11b2e230da430fb4d8f5da0348f7e2bb91c4d7e67745043c686cba0b` |
| `task-73/worker/evidence.json` | `213d5a62e8e4230650b6a7521faf6200d301a408555534f6a1dc7a11cb3513d4` |
| `task-73/worker/check.py` | `8477225e2f66f655ef90a5caf84b2af89a3f6b5b239dc90aa16c29d2031adf38` |
| `task-73/worker/gates.py` | `95a80869608e9465fb70cbc574c6b4e76ce24efbff7235e53dda6f5523ad49d7` |
| `task-73/worker/run-initial.py` | `e58e761160cf497e8c7f472f576cdecb8ee0ccdbfef71ae837f867ccb1a59a81` |
| `task-73/worker/run-second.py` | `4bb59550e14bb72dc03836cd3cc268e931b86ea04724836d87cd23a1699c5c55` |
| `task-73/worker/run.py` | `deedcbb1f93aab89dd01782bd8b9a0a389d9bc5fc483876e4375c8f206443759` |

## Strengths

The actual A diff changes only `tools/gomad3/runner/runner_mode_unix_test.go`. B adds exactly the seven handover files above. The integrated product diff adds exactly one full assignment line at fixture line 18. Removing it reconstructs the entire original 39-line BASE file, SHA-256 `ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168`. The 40-line integrated file is SHA-256 `dd24ed23e85d82a61ec4526a8cdc3908d6ea1195df9caee32ab5450ca4f862f6`. BASE and the actual integrated predecessor have identical selected files and identical tracked non-Flow product inputs.

The unchanged fixture constructs the preparer, executor and config before `syscall.Umask(0o777)`. It saves and defers restoring the original umask before the assignment, then calls `exploreWith` immediately after it. No parallel test execution was added. All imports, comments, table rows, fatal handling and permission comparisons remain byte-identical.

The assignment uses final `config.Preparer` and the outer `configDependencies.executor`. The unchanged helper at `preparation_fixture_test.go:17` retains that executor and forwards the actual preparation request to that preparer, then calls `prepared.Verify`. Request construction at `campaign_options.go:211` preserves those dependencies; local preparation at `runner_local.go:250` carries the final preparer and target. The outer executor still reaches `Run` at `runner.go:712`.

The filesystem observations are real. `newFakePreparer` creates the executable fixture before the restrictive umask; its `Prepare` reads those bytes, writes the real journal preparation copy and applies mode 0500. `target.Prepared.Verify` opens, stats and hashes the real executable. Campaign publication creates and chmods directories through `campaign_journal.go:190` and `makePrivateDirectoriesContext`. The segmented journal opens and chmods the actual segment at `segmented_journal.go:257`, seals it by rename and writes its actual index through the atomic writer. Campaign publication writes `campaign.json` through that same writer, which explicitly applies 0600. The context mutation hook observes operations and does not replace the filesystem; this fixture uses `context.Background`.

The preserved `os.Stat` loop at fixture line 32 checks `.`/`failures`/`.partial`/`executions` at 0700 and `campaign.json`/`executions/index.json`/`executions/00000000000000000000.jsonl` at 0600. The selected test's retained pass, with this unchanged loop and no skip or early successful return, supports execution of all seven comparisons.

The helper returns the existing explicitly synthetic bootstrap marker. The fake executor returns a scripted result; this fixture proves no decoded bootstrap frame or real process execution. Public `Explore` still supplies empty private dependencies, default preparation/bootstrap still use their real owners, and isolated requests still reject injected preparation/execution. The unchanged public/default/bootstrap and isolated refusal controls pass in both captured phases.

I independently compared all 5,170 tracked non-Flow regular files other than MILESTONES.md between P and C. Every file matched. The sorted path, NUL and binary-file-digest inventory has SHA-256 `6c9a0db97d495f1c60c22a49615e223338e4be9e2a4866c263dded935c9722f2`. The Grafana gitlink matched by Git tree identity. All seven handover files also matched. Product-source changes after the assessed integration are absent. These checks permit bounded reconciliation of the candidate's observations to the integrated source.

I independently checked all 61 sealed raw files and all twenty receipts against worker evidence. Each raw log, binding and receipt hash matched; exact argv/cwd, UTC interval, elapsed time and numeric exit matched. All intervals were nonoverlapping, no exit was 124, and pre/post source/tool manifests matched. Every currently consumed tool file in the 24-entry tool inventory still matched its captured hash. Wrapper bindings resolved to the preserved initial, second or final source. The preservation command had bound its actual checker and Perl before execution.

The initial driver's exit 1 followed a meaningful selected-test preparation refusal on unchanged source. Its raw test command exit was 1, and the diagnostic names the unsupported linux/arm64 host. The later settings reconciliation changes only numeric temporary `go-build` paths in GOGCCFLAGS; raw values remain retained, CGO is disabled and every other setting matches. This explains the capture guard failure without relabeling the test result or inventing a second pre-edit run.

| Retained observation | Independent check |
| --- | --- |
| Selected existing fixture | Preparation RED changes to PASS; exits 1 then 0 |
| Ordinary Runner | Same 673 names; sole change is `TestRunEnforcesBatchModesIndependentOfUmask` FAIL to PASS; 497/164/12 becomes 498/163/12; both exits remain 1 |
| Preparation/injection controls | Same 19 emitted outcomes, all PASS before and after |
| Public deterministic-I/O guards | Same five emitted outcomes, all PASS before and after |
| Preparation failure/cancellation and local phase controls | Original build-failure, cancellation, overall-timeout, short-circuit, completion-precedence and finalization outcomes remain PASS in the ordinary named-set comparison |
| Architecture/private-public checks | Fresh current-source run has 55 PASS outcomes and exit 0 |
| Formatting, affected host vet, standalone errortype | Current-source receipts exit 0; logs are empty |
| Supported source sets | Fresh Runner/conformance/execution vet exits 0 with actual GOOS/GOARCH linux/amd64 and darwin/arm64; static evidence only |
| Generated validation | Canonical `make -C tools/gomad3 validate` exits 0 and actually invokes check-only root test inventory generation for the spec whose package is `./tests` |
| Original-base aggregate lint | Both exits 2; same fifty ordered complete header/source/caret blocks, comprising eight forbidigo and 42 ST1005 findings |
| Configured unfiltered Runner lint | Exit 1 with six findings |
| Original-base fast lint | Exit 2 with three root revive complexity findings before nested scopes |

The fifty complete aggregate blocks retain SHA-256 `034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea`; none was introduced or removed. Both aggregate commands leave errortype unreached. Standalone errortype's success closes only its separately executed check.

## Critical findings

None identified in the admitted integrated source change or its bounded evidence reconciliation.

## Important findings

No introduced code defect is identified. Required source acceptance remains incomplete. Ordinary Runner still has 163 FAIL outcomes and exits 1. Aggregate lint remains RED50, configured Runner lint remains RED6, and original-base fast lint remains RED3. Their preserved failures do not become successful gates through the selected fixture's pass or this assessment.

The fast-lint findings belong to unchanged `cmd/tools/lintcode/main.go`, at loadOwnership, ownership.classify and coveredPackages. Its hash is `f7ecb5ea8c1686defa55141fdfbcec7d56212ed6af22972a1802966c5dc6f761`. Git history confirms fn-155's `d5374728a752f98e23023d2ae79123e9ade36ba8` introduced the relevant routing additions. They are inherited from task73's BASE, with no proof here that a pre-fn-155 original-base run had those failures. Root integration states that distinction and task23 owns their correction.

The raw receipts remain local to C's `.flow/tmp/fn10973-evidence`; B commits their seals, wrappers and compact handover rather than their bytes. The review successfully consumed those current files. Retaining that workspace is necessary to keep these independently verifiable receipts available.

## Minor findings

None requiring a corrective product edit. The worker evidence's word "inherited" for fast lint should continue to be read with root integration's explicit task73-BASE limitation.

## Recommendations

Keep task73 in progress and retain its raw workspace. Preserve the independent failure owners and carry task23's correction through its own admission and current-source verification. Reconcile still-owned ordinary and lint requirements before formal completion. This report completes only the requested bounded integrated assessment.

Do not broaden the single fixture assignment to default preparation, bootstrap, shared helpers or another consumer. Native fn-128/fn-149 remains deferred and unverified. No native full-host pass, replay proof, soak bound, revival, CI, PR or push authority follows.

## Code-only assessment and verification limits

The exact admitted assignment preserves the original fixture and exercises real campaign modes with the outer executor. No additional product change is indicated by this assessment. Passing focused/static/generated observations support this source checkpoint; remaining red gates keep original acceptance open.

I ran only read-only shell, Git, hashes and an independent inline Python reconciliation checker. No Go/build/lint/vet/generator, bridge, native, Git mutation or Flow mutation ran. Every shell used `login:false`, `env -u BASH_ENV bash -c` and explicit inner `cd P`. The historical worker checker expects an uncommitted diff and root's historical checker expects the pre-integration PRIMARY file, so neither was blindly rerun against the new state.

The executed inline checker was bound before invocation with SHA-256 `2b544c94d75d9839bca45576e10fcc2d77156aaf786b449ffc540d546ca875fe`, then executed as `/usr/bin/python3 -B -c <bound inline source>` and reached its final JSON success output. `/usr/bin/python3` SHA-256 `fd82483aa72498a59e905cdba1ad7a5a2aea2592e7ebc6398cebdaf1dee4adac`, `/usr/bin/git` SHA-256 `89fbabe92d1e190036a9b15e8e30efc16643beb7562329592855e622c59c3e00` and `/usr/bin/sha256sum` SHA-256 `956b53a693650d61b2b4d3f035237de469b28522df27c58f7628d7d731e7f1e1` matched before and after that invocation. Root's read-only-inspected historical checker has SHA-256 `b8c2810bba32f4c70664541b92f17b44f8c7ac28851c2a7a6f981438ff7856ac`. The inspected worker Perl identity is `/usr/bin/perl`, SHA-256 `0953404d494ccb2618aaf418313376fc217a243ec21574c7f2a0dfa005e0acc3`.

Those current input checks corroborate retained pre-execution bindings; they do not retroactively capture omitted inputs. Shared Go/build/module caches, selected environment capture and executable-file hashing establish no hermetic execution, exhaustive inherited-environment capture, shared-library stability or cache-content stability. No timeout occurred. The historical wrapper's unexercised timeout path supplies no future process-tree termination proof.

I independently rehashed the fourteen source inputs in [source-assessment.md](source-assessment.md)'s consumed-source table and confirmed every hash against current P. The full current-source inventory reconciliation also binds those files. Additional inspected source inputs are below.

| Additional consumed source input | SHA-256 |
| --- | --- |
| `tools/gomad3/runner/runner_local_test.go` | `6a8db7341cc00b1ca39e595bb0d4fe087d6e84099844f48154bb111f440ac320` |
| `tools/gomad3integration/qualification/tests.generator.json` | `6c74bf98230a02785162991509552ad54c83b29ffd2220b6fcc54277d41dff31` |
| `cmd/tools/lintcode/main.go` | `f7ecb5ea8c1686defa55141fdfbcec7d56212ed6af22972a1802966c5dc6f761` |

The report's only write is this assessment file. Root retains integration, lifecycle and final acceptance ownership.
