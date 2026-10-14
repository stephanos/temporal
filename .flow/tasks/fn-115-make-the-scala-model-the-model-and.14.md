---
satisfies: [R2, R11, R20]
---
# fn-115-make-the-scala-model-the-model-and.14 Store lifter fixtures as plain Scala in an excluded testdata directory

## Description
Store the lifter's fixtures as plain `.scala` files and keep `model/lifter/testdata` out of compilation by excluding the directory, instead of hiding the sources behind a `.scala.fixture` suffix. Owner request of 2026-10-02.

**Size:** S
**Files:** model/lifter/project.scala, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala, tools/umpire/internal/golden path substitutions, Makefile lint roots if needed
**Touches:** [model/lifter/**, model/.scalafmt.conf, model/gate/**, tools/umpire/internal/golden/**, tools/umpire/model/**, Makefile]

### Approach
- Add `//> using exclude` for `testdata` to the lifter's `project.scala`; confirm compile and test of `model/lifter` skip the directory, including the two fixtures that must not compile.
- Scalafix applies to `testdata/lifts` as its own project with all rules (it compiles and has no forbidden construct); `lint-model` and `fix-model` gain that root and the packaged Model jar it builds against. `unsupported` (its `var` and `while` are what the lifter must refuse), `werror` and `crossed` (must not compile, so `RemoveUnused` cannot run) stay out of scalafix. A scalafix rewrite that would shift a recorded position stops the task.
- `model/.scalafmt.conf` still applies to `testdata` (owner decision): `fmt-model` and `lint-model` format the fixtures. All fixtures but `unsupported/Unsupported.scala` already conform; formatting that one rewraps lines 22-25, below its asserted refusal at line 18, and it has no expected IR. Any format change that would shift a recorded position stops the task.
- Rename the eleven `*.scala.fixture` files to `*.scala`. The tests still copy a fixture into scratch before building it, without stripping a suffix.
- The six expected IR files record source paths ending in `.scala.fixture`. Regenerate them through the gate's update path and prove the diff is that path suffix only. Extend the golden helper's closed path substitutions and the affected goldens' handling by exactly these paths; no golden content is regenerated.
- Check every other reader of the suffix (gate, vocabulary and isolation tests, docs).
- Line 1 of `lifts/Admission`, `lifts/CloseReset` and `lifts/Realizations` names a retired path; fix it in the same line (no line added or removed, so recorded positions stay).

## Acceptance
- [ ] No `*.scala.fixture` file remains; `model/lifter/testdata` is excluded from the lifter's compile and test, and a fixture that must not compile does not break them.
- [ ] Scalafix checks `testdata/lifts`; the three refusal fixtures are outside it with the reason recorded.
- [ ] The fixtures are formatted by `fmt-model` and checked by `lint-model`.
- [ ] Expected IR differs from before only in the recorded source path suffix; line and column positions and all other bytes are unchanged.
- [ ] The migration goldens pass with the path mapping extended by exactly the renamed files; model gate, lifter suite, `lint-model` and the `tools/umpire` Go tests pass.


## Done summary
The lifter's eleven fixtures are plain `.scala` files; `//> using exclude testdata` in `model/lifter/project.scala` keeps them out of the lifter's compile and test (negative proof: removing the directive fails on `Crossed.scala`). The tests copy fixtures to scratch without suffix handling, and the lifter's suffix-only `%s` path substitution is gone. Line 1 of three fixtures now names current paths without moving a line.

The six expected fixture IR files and `rejects.txt` differ from before only by the `.scala.fixture` to `.scala` suffix; `model/ir`, `model/cases` and the 1,411 goldens are unchanged. The golden helper applies the six renames as an exact `source_path_renames` list to the current IR (unit-tested), so the frozen goldens keep their captured spelling. Scalafix runs on `testdata/lifts` as its own root with all rules; the three refusal fixtures stay outside it. `lint-model` and `fix-model` depend on the packaged Model jar through a new Make rule.

Deviation from the owner's request: scalafmt checks nine of the fourteen testdata files. Formatting Admission, Channels, CloseReset, Realizations and Rejects would move recorded positions, which the spec forbids, so they are excluded with the reason in `model/.scalafmt.conf` and fn-113 task 5 formats them once its golden projection compares positions by file. Full model gate, `lint-model`, a no-op `fix-model` and `lint-code-fast` pass. Independent review (Claude Fable, fresh context) returned SHIP in round 1. Handover: .flow/tmp/fn115-14-summary.md; evidence: .flow/tmp/fn115-14-evidence.json; review: .flow/tmp/fn115-14-review/round1-review.md. No agent commits.
## Evidence
- Commits:
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks, CC=/usr/bin/clang make umpire-check-model, mise exec -- make lint-model, make fix-model (hash-identical before/after), GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast, CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/internal/golden/ ./tools/umpire/model/ ./tools/umpire/lower/ ./tools/umpire/conformance/, scala-cli compile model/lifter --test (with and without //> using exclude testdata), Independent review round 1 SHIP (claude-fable-5-1); .flow/tmp/fn115-14-review/round1-review.md
- PRs: