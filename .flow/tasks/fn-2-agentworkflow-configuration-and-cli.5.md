---
satisfies: [R6, R7]
---
# fn-2-agentworkflow-configuration-and-cli.5 Update Agentworkflow documentation and run final verification

## Description
Update current product documentation and perform the final cleanup/verification pass for R6 and R7. Keep historical design records historical and link the approved replacement design where useful.

**Size:** M
**Files:** `tools/agentworkflow/README.md`, `docs/research/agentworkflow-oss-landscape.md`, `docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md`, any current references found by the final scan
**Touches:** [tools/agentworkflow/README.md, docs/research/agentworkflow-oss-landscape.md, docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md]

### Approach
- Rewrite the tutorial, configuration reference, provider selection, protection/snapshot, resume, and extensibility sections around the new path, model mapping, Cobra CLI, and internal-only implementation.
- Remove current migration guidance and former project-config detection/flag/path prose; retain runtime JSON/JSONL documentation.
- Update current implementation links in the OSS landscape document.
- Scan the active tool and current docs for obsolete `.spec`, former config, public root API, and backendtest references.
- Run focused tests first, then full module tests/build, import formatting, diff checks, and repository lint.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/README.md:52-179` — tutorial, tasks, and resume
- `tools/agentworkflow/README.md:179-389` — YAML contract, protection, providers, and migration
- `tools/agentworkflow/README.md:442-463` — snapshot safety and public extension prose
- `docs/research/agentworkflow-oss-landscape.md:19-71` — current implementation references
- `docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md` — approved contract

**Optional** (reference as needed):
- `docs/superpowers/specs/2026-08-23-agentworkflow-yaml-workflows-design.md` — historical superseded design

### Acceptance
- [ ] Current user documentation consistently uses `.agentworkflow/config.yml` and explains stage models plus `--model` precedence.
- [ ] Current docs no longer promise a public Go API or backend test helper.
- [ ] No current tool code/test/doc contains former project-config migration behavior or obsolete `.spec` examples.
- [ ] Runtime JSON/JSONL and `--json` documentation remains accurate.
- [ ] Full tests, build, formatting/import checks, diff checks, and repository lint have fresh recorded results.

## Acceptance
- [ ] R6 current documentation and reference cleanup is complete.
- [ ] R7 verification evidence is recorded.

## Verification record

Fresh verification completed at 2026-08-24T19:32:53Z against task code commit
`cb732439615db48b4ad27ffd642aef6606e96706`:

- `cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./...` — pass.
- `cd tools/agentworkflow && GOWORK=off go build -o /tmp/agentworkflow-fn2-task5 ./cmd/agentworkflow` — pass.
- `cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./...` — pass.
- `cd tools/agentworkflow && GOWORK=off go test -count=1 -race -tags test_dep ./...` — pass.
- `gofmt -d` over every Agentworkflow Go file and
  `.bin/gci-v0.13.6 diff --skip-generated -s standard -s default tools/agentworkflow` — no diff.
- `git diff --check` plus active code/current-document scans for obsolete `.spec`, former project
  configuration, public root API, and backend test-helper promises — pass; retained runtime
  JSON/JSONL/`--json` references remain present.
- `GOLANGCI_LINT_BASE_REV=36797c82174dacce262e12a38f49e9c758272a75 GOLANGCI_LINT_FIX=false make LOCALBIN=/tmp/fn2-5-lint.VBkjDB/tools lint-code`
  in a disposable case-sensitive clone at the task commit — all 13 configured analyzers ran with
  zero issues. The first attempt exhausted generated Go cache space; after clearing only generated
  build/analysis caches, the identical command passed.

The user-approved inherited exception remains unchanged: default `make lint-code` compares this
branch with a six-month-old `main` baseline and reports 1,811 unrelated findings, so the task-scoped
full analyzer result is the completion gate.


## Done summary
Updated current Agentworkflow documentation for `.agentworkflow/config.yml`, strict per-stage Codex/Claude models, whole-run override and resume behavior, Cobra semantics, and the internal-only executable surface. Removed obsolete current migration/API promises, repaired implementation links, and retained accurate runtime JSON/JSONL documentation.

Fresh tagged module tests, build, vet, race, formatting/import, reference, and task-scoped 13-analyzer lint gates pass at the final task head. The approved inherited branch-wide lint baseline exception remains unchanged.

stage: impl-review - ran | verdict: SHIP | session: 01a03538-d9c7-7d11-8d80-c039d1339671
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cb732439615db48b4ad27ffd642aef6606e96706, c3c31b8e868d8328a30b2b2cdac52d8101bea1d1
- Tests: baseline: green (cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./...; cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow; focused gci diff), cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/cli ./internal/project ./internal/agentworkflow, cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build -o /tmp/agentworkflow-fn2-task5-verify ./cmd/agentworkflow, cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go test -count=1 -race -tags test_dep ./..., find tools/agentworkflow -type f -name '*.go' -print0 | xargs -0 gofmt -d, .bin/gci-v0.13.6 diff --skip-generated -s standard -s default tools/agentworkflow, git diff --check plus active-code/current-doc obsolete-reference and implementation-link scans, GOLANGCI_LINT_BASE_REV=36797c82174dacce262e12a38f49e9c758272a75 GOLANGCI_LINT_FIX=false make LOCALBIN=/tmp/fn2-5-lint.VBkjDB/tools lint-code (disposable case-sensitive clone at c3c31b8e868d8328a30b2b2cdac52d8101bea1d1; 13 analyzers; 0 issues), INHERITED_RED: default make lint-code reports 1811 unrelated findings against the user-approved six-month-old main baseline, NO_RECEIPT: gate classification was forced full by unrelated user-owned .plans files; all full task gates were rerun at final HEAD
- PRs:
