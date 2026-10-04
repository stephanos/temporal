---
satisfies: [R1]
---
# fn-124-shrink-and-simplify-the-umpire-go.1 Delete the umpire0 and model0 archives and the empty umpire-gen-lean-api command

## Description
Implements R1 of the spec. Use `git rm -r` for tools/umpire0, model0 and tools/umpire/cmd/umpire-gen-lean-api (git rm is a git command, not a raw rm). Then fix every reference: common/testing/testpilot/dependency_boundary_test.go, tools/umpire/model/{ownership,isolation,nexus_close_baseline}_test.go, tools/umpire/lower/lower_test.go, Lean/model0 comments in lower/internal/producer/{producer,build,localize}.go and model/internal/checker/canonical.go (a comment that explains a rule keeps the rule and loses the citation), the producer's 'must match the Lean producer' constraint wording, and indexes or live docs that point at the archives (dated research may keep historical mentions). The untracked tools/gomad/ and tools/umpire3/ are owner files: leave them. Gates: module build and vet, full Go tooling suite, Testpilot tests, lint-code-fast, model gate.
## Acceptance
- [ ] TBD

## Done summary
Deleted the tools/umpire0 archive (frozen Lean-era Go tooling, its own module) and the model0 archive (Lean/Quint, 13 MB) with `git rm -r`: 830 tracked files, about 220k lines. git history keeps them, and origin/umpire 0b31bab957 still has them. tools/umpire/cmd/umpire-gen-lean-api had no tracked files, only an empty untracked directory tree, which I removed with rmdir.

Reference fixes:
- dependency_boundary_test.go: dropped the archive prefixes, and replaced the two archive fixture rows with one `model/go/umpire` row so the `model` prefix keeps a test.
- ownership_test.go: removed the archive import rule, the archive FileExists checks and five crossed-owner rows; reworded the liveGoFiles comment.
- isolation_test.go: the escaping case now uses `model/../MILESTONES.md`, an existing file, so it is rejected for escaping `model/` and not for being missing.
- nexus_close_baseline_test.go and lower_test.go: comments reworded.
- producer.go, build.go, localize.go, canonical.go, and the two canary identity comments: each comment keeps its rule and drops the Lean/model0 citation.
- Docs: removed the archive line from .gitignore; removed the archive bullet from UMPIRE4_SPEC.md; added a deletion note to the UMPIRE_MODULES.md archive rules and to the .plans/archive/README.md index.

Decisions:
- Links in dated research (UMPIRE4_INSPIRE.md, UMPIRE4_TLA_COMPAT.md) now point at GitHub permalinks at the pushed commit 0b31bab957 instead of being deleted. The .plans/umpire-migration-*.json files, docs/superpowers/specs and .plans/archive/* are historical records and stay unchanged.

Review: claude-opus-5-5 at high, the same model family (Opus) as the writer.
- Round 1: SHIP, with one P3 (comments citing "retired Lean" sources). I applied it in 58a8105dcd and 53a93346fd.
- Round 2: SHIP.

Gates:
- The first baseline run, outside the lock, was OOM-killed under shared load. The re-run under the lock passed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c95fb97ad4, 58a8105dcd, 53a93346fd
- Tests: go build -tags test_dep ./... && go vet -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), MODEL_GATE_ARGS=--skip-go-checks make umpire-check-model (exit 0), make umpire-check-cases (exit 0), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 2 SHIP)
- PRs: