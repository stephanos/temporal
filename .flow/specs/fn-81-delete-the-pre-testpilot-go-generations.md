## Goal & Context
<!-- scope: business -->

Status: implementation specification for a mechanical deletion sweep of the pre-Testpilot Go
generations. [UMPIRE4_SPEC](../../.plans/UMPIRE4_SPEC.md) remains normative; nothing here changes
modeled behavior.

The repository still carries every earlier generation of this project as live, compiled, and
partly CI-wired code. Nothing in the current Umpire, Testpilot, or Temporal model imports it. The
2026-09-08 assessment measured it:

| Tree | Go lines | Still wired into |
| --- | --- | --- |
| gomad, gomad1, gomad2, gomad3, gomad3sim, gomad3integration | 209,000 | gomad3 workflow, Makefile targets, four nested go.mod files, a root go.mod `replace` |
| umpire1, umpire2, umpire3 | 92,700 | umpire3 and umpire-model-verification workflows, two Makefile variable blocks and about 70 targets, a second Lake project under umpire3 (677 MB on disk) |
| common/testing/umpire and the testcore monitor | 26,400 | history-service instrumentation in seven files, the functional test harness monitor and gRPC interceptor |
| agentworkflow | 11,200 | Makefile, its own go.mod |
| legacy tests under tests | 6,200 | the live-test gate that pins nine expected failures by name |
| cmd/umpire-genmodels | — | imports umpire2 and common/testing/umpire/verify; mise tasks and an install script |

Live Umpire plus Testpilot is about 42,000 lines. Two costs are paid today. The gomad3 workflow
triggers on any change to go.mod or the Makefile, so it runs on unrelated pull requests. The
working disk is at 97 percent and umpire3's Lake build alone holds 677 MB.

Three things the first draft got wrong and this revision fixes. `tools/fairsim` and
`cmd/tools/fairsim` are upstream Temporal code and stay. `tools/planindex` was retained by fn-66
and stays; its index gets relabeled. And the earlier generations' white-box seam is not confined
to `tools`: the history workflow cache calls an OpenTelemetry fact adapter under
`common/testing/umpire` with entity tags from `tools/umpire1`, six other history-service files
carry comments that point the reader at that observer, and every functional test cluster installs
the umpire2 monitor and its gRPC fault-injector interceptor. That seam has no consumer that
produces a Verdict, so this spec removes it and restores the cache file to upstream shape.

The Lean previous generations (Nexus v1, Nexus2, Umpire Artifact, Space, Exploration) are out of
scope. They are imported by live modules or reserved by open specs (fn-22, fn-33, fn-79, fn-80)
and need a roadmap decision, not a sweep.

## Architecture & Data Models
<!-- scope: technical -->

### Deletion set

- Go trees: `tools/gomad`, `tools/gomad1`, `tools/gomad2`, `tools/gomad3`, `tools/gomad3sim`,
  `tools/gomad3integration`, `tools/umpire1`, `tools/umpire2`, `tools/umpire3`,
  `tools/agentworkflow`, and `cmd/umpire-genmodels`, including their nested go.mod and go.sum
  files. The root go.mod `require` and `replace` for `github.com/temporalio/gomad` go with
  `tools/gomad2`.
- The legacy white-box seam: `common/testing/umpire` with every subpackage, the
  `tests/testcore/monitor` package, the monitor factory, purge, and interceptor wiring in the
  functional test base and test environment, the fact-adapter instrumentation in the history
  workflow cache (the only history-service importer), and the observer comments in six other
  history-service files. Those files keep their other branch changes; only the adapter imports,
  calls, and observer comments are removed.
- Tests: every file under `tests` whose name begins with `umpire2_` or `umpire3_`,
  `tests/lost_task_test.go` (a monitor-seam consumer written for this branch, not upstream), and
  the `tests/probe` and `tests/gomadfunctional` packages. The lost-task property it asserted is
  recorded in the ledger as a future Testpilot regression candidate.
- CI and tooling: the `umpire3`, `gomad3`, and `umpire-model-verification` workflows; the mise
  tasks and the `develop/umpire` install script that only serve `cmd/umpire-genmodels`; CODEOWNERS
  rows for deleted paths; `.gitignore`, `.gitattributes`, and yamlfmt entries for deleted paths.
- Makefile: every variable, target, and `.PHONY` entry whose only purpose is a deleted tree,
  including the prune clauses for a gomad3 toolchain directory that no longer exists on disk.
- The `umpire-check-live-tests` expected-failure list.

### Retention set

- `tools/fairsim` and `cmd/tools/fairsim` (upstream), `tools/planindex` and `.plans/index.json`
  (fn-66 decision, revalidated here), `.plans` history documents, `docs` research and design
  records, `.turbo` plan notes.
- `tools/umpire`, `tools/common`, `common/testing/testpilot`, `tests/testcore/testpilot`, and every
  other directory not named above stay untouched apart from reference edits.
- The `tools/umpire/regression` CI workflow test keeps asserting the retained commands and is
  updated to the new gate.

### Evidence rule

A removal decision needs repository-wide consumer evidence, following the ledger format fn-66
established in `tools/umpire/CLEANUP_INVENTORY.md`. Absence of a Go import alone is not
evidence. Each deleted root records its consumers found by searching Go imports (root module and
each nested module separately), Makefile, workflows, shell scripts, mise and lake configuration,
proto files, Lean sources, generated manifests, CODEOWNERS, ignore files, and documentation, and
the disposition of each consumer. The ledger records the fn-66 planindex carve-out as revalidated
and the fairsim exclusion as upstream ownership.

### Live-test gate

The gate keeps the whole-failure-set comparison the recorded pitfall requires and drops the
pinned list. The selector becomes the `^TestTestpilot` prefix so fn-77's new live tests join
without enumeration. The expected failure set is empty. Because an empty selector match would
also yield an empty failure set, the gate additionally requires at least one passing test
identity in the output. Passing identities are only printed under `go test -v`, so the gate
command becomes verbose and the pinned command in the CI workflow test changes with it. The CI
workflow test pins the selector, the verbose flag, the empty baseline, and the floor.

### Commit order

Each commit compiles under `go build -tags 'test_dep integration' ./...`:

The seam and the umpire trees form one dependency cycle: the history service imports
`tools/umpire1`, the harness imports `tools/umpire2`, and both trees import
`common/testing/umpire`. They are deleted together after every other importer is gone.

1. Legacy tests, `tests/lost_task_test.go`, `tests/probe`, `tests/gomadfunctional`.
2. `cmd/umpire-genmodels`, mise tasks, install script.
3. Seam removal and the umpire1, umpire2, umpire3 trees in one commit: history-service
   instrumentation, every monitor API in the functional test base and test environment, the
   testcore monitor package, `common/testing/umpire`, the three trees, and the retired-vocabulary
   test fixture path repoint.
4. gomad family trees plus the root go.mod require and replace, in one commit, followed by the
   tidy verification.
5. agentworkflow.
6. Makefile, workflows, live-test gate, and the CI workflow test, in one commit.
7. CODEOWNERS, ignore files, attributes, yamlfmt.
8. Documentation notes, `.plans/index.json` relabel, roadmap entry, flow record sync.

## Edge Cases & Constraints
<!-- scope: technical -->

- A retained file that references a deleted path only in prose (README, design record, doc
  comment, Lean comment) is edited to say the path was removed, never left dangling.
- The tidy gate is `go build -tags 'test_dep integration' ./...` and `go vet -tags test_dep
  ./...`, not a bare build, because retained importers live behind tags. Before tidy, the
  dependency closure of the retained tree is captured with `go list -deps`; every module tidy
  drops must have zero hits in that capture, and the table lands in the ledger.
- The default package set is captured with `go list ./...` before and after; the after set must
  be a subset of the before set.
- Removing the functional-harness interceptor changes the gRPC chain for every functional test.
  It lands in the seam commit and is verified by running the retained live gate plus one
  functional package that builds a cluster. The monitor API surface removed with it is the
  accessor, the rule-passed assertion, the passed-keys query, the violation-allowance toggle, the
  purge helpers, the factory option, and the default factory; the only retained reader,
  `tests/lost_task_test.go`, is deleted in commit 1.
- The live-test gate must not pass because the selector matched nothing. The floor assertion
  covers that.
- `.flow/tmp` may hold a duplicate tree with its own nested modules; the ledger records whether
  it is tracked and, if so, classifies it.

## Quick commands

```bash
go build -tags 'test_dep integration' ./... && go vet -tags test_dep ./...
CGO_ENABLED=0 go test -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/...
make lint-code
make umpire-check-regression
go run ./tools/planindex
git grep -n -E 'umpire[123]|gomad|agentworkflow|umpire-genmodels|common/testing/umpire' -- . ':!.plans' ':!.flow' ':!.turbo' ':!docs'
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A consumer-evidence ledger under `tools/umpire` classifies every deletion-set root and
  every retained neighbor named above with repository-wide consumers and dispositions, records
  the fairsim and planindex retentions with their reasons, and captures the dependency closure
  and package-set baselines, before any deletion lands. Errors: a root with an unclassified live
  consumer blocks deletion of that root.
- **R2:** The Go trees, nested modules, `cmd/umpire-genmodels`, the root go.mod require and
  replace for the gomad module, the mise tasks, and the install script are deleted; the tagged
  build and vet pass; tidy removes only modules absent from the retained dependency closure; the
  package set after is a subset of the package set before. Errors: a build, vet, or subset check
  failure blocks; a tidy diff that drops a module present in the closure blocks.
- **R3:** The legacy tests, the three workflows, every Makefile variable, target, prune clause,
  and `.PHONY` entry serving a deleted tree, the CODEOWNERS rows, and the ignore, attributes, and
  yamlfmt entries are removed; the retired-vocabulary test fixture no longer names a deleted
  path; no retained source references a deleted path except as an annotated historical note.
  Errors: the Quick commands grep returns only annotated historical notes.
- **R4:** `umpire-check-live-tests` runs `go test -v` with the `^TestTestpilot` prefix, compares
  the failure identity set against an empty baseline, requires at least one `--- PASS` identity,
  and the `tools/umpire/regression` workflow test pins the verbose command, the selector, the
  empty baseline, and the floor. Errors: any failing retained live test blocks; a run with no
  passing identity blocks.
- **R5:** `make lint-code`, `make umpire-check-regression`, `go run ./tools/planindex`, and the
  Go test set in Quick commands pass; the task receipt records `git diff --stat` line totals and
  `du -sh` before and after for `tools/umpire3` and `model/.lake`. Errors: a failed gate blocks
  completion.
- **R6:** No file under `service` imports `common/testing/umpire` or `tools/umpire1` or comments
  on the umpire observer, `tests/testcore` exposes no monitor API (accessor, rule-passed assertion,
  passed-keys query, violation allowance, purge helpers, factory option, default factory, or
  interceptor), `tests/lost_task_test.go` is deleted, and `common/testing/umpire` is gone in the
  same commit as the umpire trees; the retained live gate and one cluster-building functional
  package pass afterwards. Errors: a retained test that reads monitor facts blocks and is listed
  in the ledger; a functional package failure blocks.
- **R7:** Documentation and records are reconciled: historical banners on the research and design
  records that link into deleted trees, the Lean comment and ledger sentence rewritten,
  `.plans/index.json` relabels deleted-tree documents as historical and drops its stale entry,
  UMPIRE4_ORDER gains an fn-81 entry, fn-80 tasks .4 and .8 and fn-30 task .6 lose their
  references to the pinned list and the deleted workflow, and fn-14 gets a closure note. Errors:
  `go run ./tools/planindex` rejects a mislabeled or missing entry.

## Boundaries
<!-- scope: business -->

- No Lean deletions. Nexus v1, Nexus2, Umpire Artifact, Space, and Exploration stay.
- No `.plans`, `docs`, or `.turbo` deletions; those files gain notes only.
- No new CI workflow or dead-code detection gate.
- No refactoring of retained code beyond removing the seam and editing references.
- No deletion of `tools/fairsim`, `cmd/tools/fairsim`, or `tools/planindex`.
- No replacement monitor. The functional harness loses the seam rather than gaining a no-op.

## Decision Context
<!-- scope: both -->

Deleting rather than archiving to a branch was chosen because git history already preserves every
tree, and an archive branch invites the same CI wiring to return. Retaining the `.plans` history
documents keeps the reasoning that led to Umpire 4 readable without keeping code that competes
with it.

Removing the white-box seam rather than stubbing it was chosen because its only consumers are the
generations being deleted, every functional test currently pays for a monitor and interceptor
that produce no Verdict, and the Umpire 4 rules route evidence through declared Testpilot
Observations. A future white-box mode is a Testpilot design, not a revival of this adapter.
Rejected as overkill: a no-op monitor that keeps 26,000 lines alive for two call sites.

Retaining planindex reverses the first draft because fn-66 retained it deliberately and its
validator is the only thing that keeps `.plans/index.json` honest once the deleted-tree documents
are relabeled historical. Excluding fairsim is not a choice; it is upstream Temporal code.

The gate design keeps the recorded pitfall's whole-set comparison, and adds a floor because an
empty baseline alone cannot distinguish "all passed" from "nothing ran". Excluding the Lean
generations keeps this spec mechanical; their removal depends on fn-79 and fn-80 outcomes and
belongs in a later consolidation spec.

The declined ledger entry on generated API drift verification is unaffected: this spec adds no
drift gate and no CI workflow.

## Early proof point

Task fn-81-delete-the-pre-testpilot-go-generations.2 validates the core approach: the seam comes
out of the history service and the functional harness and the retained live gate still passes.
If it fails because a retained test depends on monitor facts, re-evaluate the seam disposition
before deleting any tree.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Consumer-evidence ledger and baselines | .1 | — |
| R2 | Go trees, modules, genmodels, tidy | .2, .3 | — |
| R3 | Tests, workflows, Makefile, dotfiles, references | .2, .3, .4 | — |
| R4 | Live-test gate redesign | .4 | — |
| R5 | Gates and measurements | .5 | — |
| R6 | White-box seam removal | .2 | — |
| R7 | Docs, index relabel, roadmap, flow record sync | .5 | — |

