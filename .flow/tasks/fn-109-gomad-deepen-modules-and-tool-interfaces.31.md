---
satisfies: [R13, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.31 Preserve artifact directory and shared-verifier cleanup

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Bounded R13/R18/R19 source repair after public-copy checkpoint 2f75b26addcbc9b1948437407ff52f98da798917 and evidence follow-up 39fc19c4618322b6a939f5b603d9ba69aab00b9b. Preserve original task 12/predecessors and task 21 acceptance. Root owns Flow, parent, MILESTONES, review and Git/index/commit.

**Size:** M
**Touches:** [tools/gomad3/artifact/store.go, tools/gomad3/artifact/target_pool.go, tools/gomad3/artifact/store_test.go, tools/gomad3/artifact/target_pool_test.go, tools/gomad3/artifact/publication_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-31/**]

Reuse .flow/tmp/artifact-directory-verifier-cleanup-owner-plan.md and the prior sixteen-site source map. Read AGENTS.md, Gomad README/MILESTONES, original fn109 spec/task12 and artifact interface inventory. This owner resolves three production errcheck sites at baseline store.go:490 and target_pool.go:216/221 plus three closely related test-handle cleanup sites at publication_test.go:42/132 and target_pool_test.go:262. Public CopyPayload, private payload helpers, clone/reflection/invariants, publication.go, callers and all other source remain unchanged.

Baseline whole ordinary artifact tests and actual unfiltered pinned lint/errortype before production edits. Follow test-driven-development, writing-good-tests, code-style and verification-before-completion. The actual six unchecked cleanup findings are source-policy RED. Add real-file preservation controls before production edits; passing characterizations are not first-Close OS-fault regression proof.

Keep syncDirectory's os.Open, directory.Sync and one deferred Close. Its private helper may return separate operation and cleanup errors; check artifact-package call sites before changing its signature. Preserve syncDirectoryContext's exact pre-context error and acquisition boundary. The inner Close runs before the post-context check. Only when Open/Sync succeeded, call the existing post ctx.Err() and select its exact result as primary; skip that call after an Open/Sync error as before. Then conditionally merge Close. Nil Close leaves the primary object untouched, nil primary adopts exact Close error, and dual failure joins primary first then cleanup. Do not let a new cleanup error hide the existing post-cancellation or add directory/mode validation.

Name verifySharedPayload's existing File/error results without changing its function type. Install root observing defer after OpenRoot acquisition, then file observing defer after openSharedFile acquisition. LIFO file-before-root release attempts each Close once and never retries. Nil Close leaves metadata/error unchanged. Real Close failure clears metadata to record.File{} and adopts raw cleanup if primary nil, otherwise joins primary first. Preserve existing metadata mode/size checks, copyWithContext, hash/count, returned literal metadata and caller wrapping/zero-sharing result.

Directory cleanup before publication prevents further publication through existing staging ownership; after successful no-replace rename, published artifact/pool winners remain independently owned. Shared verification runs after the pool link enters staging; failure removes staging through the existing owner but retains the winning pool entry. Preserve no-replace, manifest-last, validated reuse, retry/hard-link fallback, pruning, fixed-input canonical bytes/identities and resource costs. No new public/helper framework, fake Close/syscall seam, descriptor theft, retry or race.

Use standard-context real-file controls. Directory precancellation on a missing path returns exact context.Canceled before opening; live missing-path returns raw *os.PathError with open Op/path and os.ErrNotExist; real temporary directory sync succeeds. Verifier valid regular literal target with an already-cancelled context returns exact context.Canceled and zero File, retaining bytes/mode; missing target under an existing parent returns raw *os.PathError and zero File. Include literal successful verifier metadata/bytes/mode if existing direct coverage is absent. Extend the existing poisoned target-pool matrix only with missing transaction assertions after failed second publication: no new artifact or .publish-* under store, original artifact and winning pool entry remain. Do not duplicate its damage matrix or weaken existing assertions. Observe the three existing opened-test Close defers at their same lifetime. Genuine first-Close, simultaneous operation/Close, verifier cleanup-to-zero metadata failure and post-Sync cancellation timing proof remain unexecuted without legitimate deterministic seams; source inspection is not runtime proof.

Use cached stock Go1.27.1 linux/arm64, GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS='' cleared GOMADSEED/GOMAD3_CHILD_SEED, tests -count=1 -tags test_dep. Final whole artifact, focused directory/verifier/Store/publication/pool/retained/private/public-copy tests, actual architecture/public-alias/signature/external Runner boundaries, unfiltered pinned lint and errortype. Inspect generator inputs and run make validate against protected current inputs without regeneration. Reuse task30 run-gate pattern and compact protected aggregates. Root source-admission binds all tracked Gomad modules/root go.mod/go.sum/lint config excluding only five admitted source paths. Retain complete23-file artifact-source hashes per gate, raw logs/exact commands/env/cwd/start/end/elapsed/child exit/tool/config/timeout and stable sources. Include exact in-memory recipes/hashes for any meaningful bounded mutations and restored receipt.

Pinned Go /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go; analyzers /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 and errortype. Unchanged .github/.golangci.yml with --build-tags test_dep --fix=false ./artifact; errortype -tags test_dep ./artifact. Actual baseline binds task30's10 residual findings; record actual count, six-site mapped delta and every residual/introduced diagnostic. No filtering/suppression/config/rule/golden/pin/runtime/dependency/download changes. Historical whole-scope419 is not a fresh count. Do not rerun unchanged broad rootfast/full/native environment failures.

Keep all bytes outside admitted helper/cleanup/test additions unchanged. Original R13/R18/R19, task12/predecessors/task21, matched original first-baseline fixed identities, complete/full/formal/both patched-native and affected consumer qualification stay open wherever unproved. Worker owns only five source/test files and task31 artifacts; root alone reviews, stages/commits and handles Flow. No worktrees/stash/bridge/push/history rewrite or unrelated writes; no flowctl done or formal SHIP. Return with lean task31 handover.md/evidence.json only when all owned commands/delegates terminal. Root verifies/reviews/commits verified progress before another source writer.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Three production and three related test cleanup findings are resolved at their existing lifetimes with one Close attempt per descriptor, file-before-root verifier release and directory Close before post-context.
- [ ] Directory pre/operation/post-context precedence, exact nil-close primary identity, raw sole cleanup errors and verifier zero metadata on cleanup failure are preserved or explicitly source-inspected where genuine fault execution is absent.
- [ ] Original helper operation bytes, publication/pool transaction owners, fixed-input bytes/identities/costs, public/private payload APIs and all original assertions retain their strength.
- [ ] Actual baseline/characterization/final package/real-file/boundary/lint/errortype/generator evidence binds stable source/tool/config/log/exits and six-site delta with every residual/introduced finding retained.
- [ ] Fresh independent source review has no actionable introduced defect; root commits progress while original R13/R18/R19/task12/predecessors/task21/fixed-identity/full/native/formal acceptance stays open.


## Done summary
SOURCE_PROGRESS_ONLY. Three production and three related test cleanup checks
are repaired at their existing lifetimes. Directory Close precedes the existing
post-context selection and cleanup stays secondary to the original primary.
Verifier file-before-root cleanup clears metadata on genuine cleanup failure.

Worker and seven fresh independent review gates pass for the whole artifact,
focused/retained/private/public controls, five actual boundaries, errortype and
static checks. Protected generator evidence passes without regeneration.
Actual unfiltered artifact lint remains red at four unchanged findings, down
from ten with six repairs and no introduced diagnostics. Original helper/test
bytes recover exactly and the protected 1,039-input aggregate is unchanged.

Fresh independent source review returned SOURCE_PROGRESS_COMMIT_ONLY with no
actionable introduced defect. Genuine first-Close and post-Sync timing proof,
original R13/R18/R19/task12/predecessors/task21, matched first-baseline fixed
identities, complete/full/formal/both patched-native and affected consumer gates
remain open. Task is blocked, not done. Root commits this verified progress
before another source writer.

Tier: session (jev-unavailable(no_key))
Requested writer/reviewer gpt-6.1-sol at high, same configured Codex family.
Actual model metadata remains unknown.
stage: impl-review - skipped(policy: conductor-deferred; fresh independent source review passed, but full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

Blocked: original qualification and genuine fault/timing proof remain open. See
[acceptance limits](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-31/acceptance-open.md).

## Evidence
- Commits: c0e21c9cbb54a9d2909d0d47da081fdd44c94d6c
- Tests: task-31/evidence.json and independent-source-review-checks.json retain exact commands, exits and source-bound logs
- PRs: none

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
