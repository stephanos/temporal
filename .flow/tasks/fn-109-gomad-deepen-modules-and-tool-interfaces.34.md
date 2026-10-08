---
satisfies: [R6, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.34 Check CLI private-mode fixture reader cleanup

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Check the actual private-mode characterization fixture's pipe-reader Close return once while retaining stdin restoration before close and every existing command/output/status assertion. Task 33's reviewed source progress is committed at e521cbd2e39e8521bff013de19e569eff8a1392c, with metadata at base 608df98bdbf1e079e6db8a87849330797ba34439. Original predecessor acceptance stays open. This owner advances R6/R18/R19 test-support qualification, not a production output-failure redesign.

**Touches:** tools/gomad3/cmd/gomad/internal/cli/characterization_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-34/**

Use the source-bound plan .flow/tmp/cli-remaining-diagnostics-owner-plan.md (SHA256 cc46d0552568e5438299bd9ca71b1eec393f80984d7a17a185ea2ba21e4641c9). In TestCharacterizeUnknownCommandsAndPrivateModes, retain the defer and os.Stdin restoration; replace only the discarded reader.Close() with a checked return reported through t.Errorf. Preserve primary t.Fatalf failures, descriptor lifetime, one Close, existing assertions and all other test bodies. Use the real pipe/coordinator EOF test. Actual baseline errcheck is the defect RED; disclose that a genuine reader-close failure is not executed rather than inventing an injectable production seam, failure-capture framework or new oracle.

Root owns all Flow, parent/MILESTONES, admissions, reviews, staging and commits. The worker owns only the fixture and its task-unique proof files, excluding root-admission/review/checkpoint files. Preserve all production, other tests, generator inputs, canonical identities, lint rules/config/pins, earlier evidence and unrelated files. No suppression, error discard, redundant production error check, output-status/bytes/error-contract change, schema/API/runtime/pin/dependency/worktree/bridge/download/push/history operation.

**Quick commands:** offline cached stock Go 1.27.1 first on PATH, GOWORK=off GOTOOLCHAIN=local GOPROXY=off, no GOROOT/GOMADSEED/GOMAD3_CHILD_SEED; -count=1 -tags test_dep. Baseline and final private-mode regression plus task-26's portable CLI characterization selection; five actual nested-root architecture/public-signature/external-consumer boundaries; baseline/final pinned unfiltered configured complete CLI lint with --fix=false; errortype; source/gofmt and generator-input inspection (validate when relevant). Checks using shared caches/toolchains run serially. Retain exact commands, start/end/elapsed/exit, source/tool hashes and diagnostic multisets. Do not rerun unchanged unsupported-host, missing patched launcher, whole419/root-fast/native/full failures. No green baseline handoff is implied.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- The existing defer restores os.Stdin before a single reader.Close attempt. A returned close error is reported through t.Errorf without replacing a preceding primary failure. Existing command/output/status assertions and all other test bodies remain byte-for-byte unchanged.
- Retain baseline and final actual private-mode test and portable characterization controls, the actual five nested-root boundaries, errortype and format/source checks on the available pinned stock Go. Inspect generator inputs and validate if affected. Baseline errcheck is RED; the normal real-pipe case is exercised, and real close-error execution remains explicitly unproved.
- Actual unfiltered pinned complete CLI lint runs before and after; resolve exactly the fixture-close diagnostic with no introduced findings. Retain all actual production residual diagnostics without suppressing, filtering, changing error semantics or claiming whole lint clean. Historical whole-Gomad counts remain historical.
- Only characterization_test.go changes in product sources; protected inputs, earlier evidence and original acceptance remain unchanged. Independent source review must authorize only a source-progress checkpoint before another writer. Root commits the implementation, proof and Flow/docs together.
- Original task4/task5/predecessor and task21 acceptance, R6/R18/R19, matched first-baseline fixed identities, full/completion/formal/affected-consumer and native darwin/arm64 qualification remain required. Stock developmental linux/arm64 checks do not fulfill them. Complete this task only when all its corresponding source-owned gates pass; otherwise record reviewed source progress and keep acceptance blocked. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.



## Done summary
Reviewed source progress only; acceptance remains blocked. The existing defer
restores stdin before checking one reader Close through nonfatal testing.T.
Every other fixture byte/assertion and all 1,044 protected inputs stay unchanged.
Baseline/final private-mode 1/1, portable CLI 34/34, five actual boundaries,
errortype and formatting pass. Actual unfiltered CLI lint falls 54 to 53
byte-identical production findings, one fixture diagnostic resolved and none
introduced. Generator inputs are unaffected; no validation rerun is required.

Fresh source review permits SOURCE_PROGRESS_COMMIT_ONLY with no introduced
Critical/Important/Minor issue. [Review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-34/independent-source-review.md),
[checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-34/independent-source-review.json), and
[open acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-34/acceptance-open.md)
retain the evidence and proof limits. Root's read-only receipt audit confirms
all twelve original command/log bindings and serial ordering.
Genuine Close and simultaneous primary/cleanup failure execution remain unproved.

Original R6/R18/R19, task 4/task 5/predecessors/task21, matched first-baseline
identities and complete/full/completion/formal/both-native/affected-consumer
qualification remain required and open. No native or formal SHIP is claimed.
The source/proof checkpoint is the Git commit containing this task's fixture
and reviewed evidence; root owns the commit before the next source writer.

Tier: session (jev-unavailable(no_key))
Requested writer/reviewer gpt-6.1-sol at high, same family; actual models unknown.
stage: impl-review - skipped(policy: conductor-deferred; fresh source review passed, full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

## Evidence
- Commits: source/proof checkpoint retained in this task's Git history.
- Tests: serial baseline/final private1/1, portable34/34, boundaries5/5, errortype/gofmt0; lint exit1 54 to53.
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
