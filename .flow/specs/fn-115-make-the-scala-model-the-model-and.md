# Make the Scala model the model and archive the Lean-era work

> Local HTML render lens: `.flow/artifacts/fn-115-make-the-scala-model-the-model-and/spec.html` — open locally; ignored by git, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

`model/scalav2` is where Temporal features are authored. The repository layout still says otherwise. `model/` holds six sibling trees (`lean` with 431 files, `leanv2`, `go`, `quint`, `scala`, `scalav2`), and the name `scalav2` marks the live one as an experiment. `tools/umpire` holds 32,506 lines of Go from the Lean era, including 8,153 lines of generators that write Lean source, while the Go code that reads the Scala model's IR lives inside `model/scalav2`. The Lean toolchain is already removed from this checkout.

This spec makes the layout say what is true.

- `model/` is the Scala model and nothing else. It holds no Go code.
- `tools/umpire/` is the Go code that works with `model/`: it loads the IR, checks it, lowers it to Testpilot Cases and exports it to other backends.
- Everything else moves aside, to `model0/` and `tools/umpire0/`, as an archive nobody builds on.

A developer who opens `model/` finds one model. A developer who opens `tools/umpire/` finds only code the Scala pipeline runs.

The move is also the moment to get the module boundaries right. The model, the Umpire tooling and Testpilot each grew by accretion: one 2,736-line loader file, one 1,994-line lifter file, a Case runtime whose recorded-Run and deployment-binding helpers live in the Lean-era tool tree, and three shell scripts that orchestrate a Scala build. This spec revisits the packages and folders of all three, in Go and in Scala, so that each module has one job, a name that says it, and a small interface.

## Architecture & Data Models
<!-- scope: technical -->

**What is archived.** The live trees are placed by the module map below. Everything else moves aside.

| Today | After |
| --- | --- |
| `model/lean`, `model/leanv2`, `model/quint`, `model/scala` | `model0/` |
| `model/go`, except the parts of `umpire` and `caseproducer` that live code imports | `model0/` |
| `tools/umpire`, all of it as it is today | `tools/umpire0/` |
| `.plans/lean` | `.plans/archive/lean` |

`specimens`, `specs`, `README.md` and `SEMANTICS.md` of `model/scalav2` move to `model/` with the model.

**Module map.** The table is the proposal. The first task records each module's one-sentence job, public interface and permitted imports. The map receives independent review before any package is split, merged or renamed. The owner has delegated these decisions to the conductor's recommendations for this milestone.

| Module | Path | Job | Built from |
| --- | --- | --- | --- |
| Umpire IR | `proto/.../api/umpire/v1`, `api/umpire/v1` | The schema Scala lifts to and Go reads | `modelir/v1`, renamed to what everyone calls it |
| Testpilot IR | `api/testpilot/v1` | The Case, Run and Verdict schema | unchanged |
| DSL | `model/umpire` | What an author writes a Model with | `model/scalav2/scala/umpire` |
| Models | `model/temporal` | Temporal's Models, one folder per feature, and the kit they share | `model/scalav2/scala/temporal` |
| Lifter | `model/lifter` | Compiles typed Scala trees to the Umpire IR | `lifter/Lift.scala`, split by concern |
| Lifted IR | `model/ir` | The checked-in IR files | `model/scalav2/ir` |
| Gate | `model/gate` | Builds, lifts, compares the lifted IR with the checked-in files and runs the checks | `run.sh`, `gen.sh`, `scala.sh` |
| Model reader | `tools/umpire/model` | Loads, validates, interprets and checks an IR Model | `goir` and the checker from `model/go/umpire`, the checker behind `internal/` |
| Lowering | `tools/umpire/lower` | Turns a Query's witness into a Testpilot Case | `goir/testpilot` and what it needs of `caseproducer` |
| Conformance | `tools/umpire/conformance` | Says whether a Run's evidence is explained by the Model | `goir/conformance` |
| Export | `tools/umpire/export` | Writes the IR for Quint and P and holds them to Go's reading | `backends` |
| Exploration | `tools/umpire/explore` | Selects and reduces model-declared executable candidates | the exploration adapter added by fn-107 |
| Case runtime | `common/testing/testpilot` | Prepares, runs and evaluates a Case | unchanged location; recorded-Run and publishing helpers join it if the audit finds them live |
| Temporal Driver | `common/testing/testpilot/temporal` | Runs a Case against Temporal | gains deployment binding if the audit finds it live |
| Functional fixtures | `tests/testcore/testpilot` | Cases and cluster wiring for the functional tests | fixtures regenerated from the Scala model |
| Canary | `tools/canary` | Runs Cases against a deployment on a schedule | unchanged location |

`goir`, `scalav2` and `modelir` do not survive as names, because "Go", "v2" and "model" distinguish nothing once there is one model, one reader and one name for its IR.

**Rules the map must satisfy.**

- **Deep modules.** A module hides a lot behind a small interface. The model reader is one package a caller uses to load and check a Model, with its table checker behind `internal/`. A package that only forwards to another is merged into it.
- **One direction.** Testpilot imports nothing of Umpire: a Case runtime knows no Model. The model reader imports nothing of Testpilot. Lowering, conformance and model-owned exploration may combine the reader and Testpilot. Exploration uses lowering rather than defining a second producer. Export's only model-domain dependency is the reader; standard libraries and the generated IR schema remain permitted. The canary and the tests import these modules and nothing imports them. The map also assigns the live producer-neutral campaign/replay protocols and the Case-file helpers they use.
- **Scala mirrors it.** The DSL imports nothing of the Models. A Model imports the DSL, the shared kit and the generated Temporal API types, and by default nothing else. The lifter depends on the IR classes and reads Models as compiled trees, never as source imports.
- **Files follow concerns.** No source file mixes two concerns because of history. The loader and the validator are separate files with separate tests. The lifter is one file per kind of thing it lifts (types, expressions, declarations, realizations) plus its entry point.
- **Names say the job.** A package is named for what it does for its caller. Names that mark history (`v2`, `0`, `new`, `scala` as a qualifier on a Go test) appear only on the archives.

**What "archive" means.** Code under `model0/` and `tools/umpire0/` is kept for reference. No live code imports it, no Makefile target or CI job reads it, and the main Go module does not build it. It leaves the build through a nested `go.mod` in each archive root, which makes `./...` skip the tree.

**Live code that uses the old `tools/umpire` today.** `tools/canary` (19 files), nine `tests/testpilot_*_test.go` files and two test files under `common/testing/testpilot` import its `evaluation`, `recordedrun`, `replay`, `publish` and `binding` packages. Those five packages are 6,553 lines. Each is either needed by the Scala pipeline and stays live, or its importer is itself Lean-era. An audit decides, package by package, before anything is detached.

**Parity oracles become goldens.** Several Go tests compare what the IR yields against a second implementation: the hand-written Go models in `model/go/standaloneactivity`, `model/go/nexuscaller` and `model/go/worker`, and the Lean dumps. With those archived, each comparison is replaced by a checked-in golden of what the interpreter derives (tables, Definition IDs, refinement rows, fingerprints, Query answers, lowered Case bytes). This golden set is the baseline that fn-113 R13, fn-117 R9, fn-112 R1, fn-114 R1 and fn-116 R2 name. It is written here, once, because this spec runs first.

## API Contracts
<!-- scope: technical -->

**Import paths.** `go.temporal.io/server/model/...` ceases to exist as a Go import path. The live packages are under `go.temporal.io/server/tools/umpire/...`.

**Source positions and artifact identity.** The IR records each declaration's Scala file. The first task fixes a closed old-to-new path map, including source labels, and identifies every artifact checksum or identity derived from those fields. Golden comparison applies exactly that deterministic transformation to the captured baseline and independently re-derives the affected hashes. It never ignores whole provenance records, paths, identities or hashes. Declaration line and column positions are preserved, including when historical comment references are removed. Existing source-confinement checks move with the model; this refactor adds no new loader admission rule.

Semantic values, Definition IDs, Behavior Fingerprints, ordering and Query outcomes remain unchanged. Stable persisted identity domains, including Case namespaces `scala.` and `scala.explore.` and the conformance model-identity hash domain `goir.model/v1`, remain compatibility data rather than package names to rename. Removing historical names from package/module names does not rename those protocol values. R21 separately permits the generated IR namespace/type-name change while freezing field numbers, types, enum values and wire meaning. Any effect beyond these closed transformations stops the responsible task.

**Fixture replacement.** R22's replacement of a legacy pinned fixture by a Scala-produced Case is a separate, explicit artifact migration, not permissible interpreter drift. Its inventory names the replacement Query, Program, Contract, provenance/Case identity, consumer policy and binding consequences. Historical recorded Runs are never rewritten to impersonate executions of replacement Cases; existing recordings retain their established compatibility decision. A fixture that cannot yet lower uses R22's named exception until its missing primitive is supplied.

**Commands.** The model's gate is one command with the same `--update` behavior `run.sh` has today. It is a Scala program run through scala-cli (see Edge Cases, Scripts). Make targets lose the `scala` qualifier where it no longer distinguishes anything: the reviewed map selects `make umpire-check-model` and `make umpire-gen-model`, invoking `scala-cli run model/gate` without/with `-- --update`. The complete command, module and fixture contracts are recorded in `.plans/UMPIRE_MODULES.md` and its two machine-readable inventories. `make lint-scala`, `fmt-scala` and `fix-scala` keep their names, since they name the language they lint.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Scripts.** `run.sh` (236 lines), `gen.sh` and `scala.sh` are bash around a Scala build. `scala.sh` exists because scala-cli exits 0 when `-Werror` turns a warning into an error. `gen.sh` runs protoc and packages a jar. `run.sh` compiles, tests, packages, materializes lifter fixtures with `sed`, lifts, diffs against the checked-in IR and runs the Go tests. Nothing in that needs a shell. The fixture checks are tests of the lifter and belong in the lifter's test suite. The rest is one Scala program that spawns scala-cli, protoc and go. It may use a process and file library, since it is tooling and no Model can import it. The scala-cli exit-code defect is then handled in one function instead of one wrapper script. The same applies to the export module's `run.sh`, which becomes an opt-in of its Go tests.
- **Fixtures and the canary's pinned Case were rendered by Lean.** `tests/testcore/testpilot/testdata` and `tools/canary/casebinding` hold Case bytes the Lean renderer produced. They are regenerated by lowering from the Scala model, or the consumer that needs them is archived. fn-107.22 generates lowered Case files; this spec uses its output.
- **Moves, renames and edits are separate reviewable phases.** The user reserves commits and staging. Capture before/after file manifests and move mappings, use filesystem renames without staging, and distinguish path/import-only moves from cleanup. Do not manufacture commits or rewrite the user's index. Original source bytes retained in the archives preserve the material needed for later history-aware commits.
- **Authorized baseline.** fn-107 is closed with all 23 tasks and an independent completion review. Its uncommitted work is part of this task's starting state. Capture that current state, including untracked implementation files, rather than using HEAD as the source baseline. Unrelated concurrent edits remain protected.
- **The other open specs name old paths.** fn-112, fn-113, fn-114, fn-116, fn-117, fn-118, fn-119 and fn-120 run after this spec and name some of today's paths and packages (`model/scalav2/...`, `scala/umpire`, `goir`) to say what they mean. The closing task rewrites those references in the open specs and in `MILESTONES.md` to the names the approved module map gives. Closed specs and their tasks are history and are not touched.
- **What the checker loses.** `model/go/umpire` is 9,201 lines with tests. It mirrors Lean in places the interpreter never calls (canonical encodings, sets, coverage, replay). A file moves to `tools/umpire` only if a live package imports something from it, and an exported declaration with no live caller is deleted after the move.
- **Lean parity tests.** The Go tests that read Lean dumps skip today without the dumps. They are removed with their dump paths.
- **Makefile.** About forty `umpire-*` targets exist. Those that build, generate or check Lean, regression views, evaluation profiles or anything else under the archive are deleted, with their variables. Targets are never left pointing at `model0/` or `tools/umpire0/`.
- **CI.** `.github/workflows/umpire.yml` sets `working_directory: model/lean` and runs archive targets. It ends up running what is live: the Go tests of `tools/umpire`, `common/testing/testpilot`, `tools/canary` and the live tests, minus anything archived. Whether CI also runs the model gate, which needs a JVM, is a separate decision.
- **Ignore rules.** `.gitignore` has entries for `model/lean`, `tools/umpire/**/testdata`, generated `umpire-*` binaries and Lean generator directories that no longer exist. Each entry is kept, moved or removed to match the new layout, so that no tracked testdata under `tools/umpire` becomes ignored.
- **Lean is forgotten inside `model/`.** The archive is where the history lives. Inside `model/` the model is described on its own terms, with no reference to what it was ported from. This is a deliberate exception to the repository rule that refactors preserve comments. The Go code that moves to `tools/umpire` mentions Lean on 50 lines in 15 files; those go when R3 removes the parity tests and R12 removes unused declarations, and the closing task reports what is left.
- **Agent instructions.** `AGENTS.md` tells agents to read the Lean guidelines before Lean work and lists a project structure without `model/`. It is updated to name the Scala model, its gate and `tools/umpire`, and the Lean mandate is removed.
- **Disk.** Go's build cache grows during wide refactors. Workers check free space and clean the cache at task boundaries.
- **No recursive delete, no worktrees.** Workers use guarded filesystem renames and copies with captured source hashes, never `rm -rf` or a git worktree. Freeze the original old tooling and archived model bytes before extraction; promote live copies while keeping the originals for the eventual archive. Only required archive-root module/README metadata and archive indexes are added to frozen material.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A recorded audit classifies every package under today's `tools/umpire`, `model/go`, `model/quint`, `model/leanv2`, `model/scala` and `model/lean` as live or archive, with the evidence: its importers outside its own tree, the Makefile targets and CI steps that invoke it, and the checked-in files it generates. Errors: a package with a live importer is never classified archive without naming what happens to the importer; a row with no evidence blocks R4.
- **R2:** A golden set records, for every IR file, the tables, Definition IDs, refinement rows, fingerprints, Query answers and lowered Case bytes the interpreter derives, and a Go test fails on any difference. It is captured before any move and passes after every task. Errors: only the closed transformations in API Contracts are allowed; any other difference stops the task. The golden inventory includes every-row Property results and explicit unsupported standings, not only named Query answers.
- **R3:** Each test that compares the IR against `model/go/standaloneactivity`, `model/go/nexuscaller`, `model/go/worker` or a Lean dump is listed with the golden of R2 that now carries its claim, and is then removed. Errors: a comparison whose claim no golden carries gets a golden added before the test is removed.
- **R4:** `model/` contains the Scala model and no `.go` file, laid out as the approved module map says. `model/scalav2` no longer exists. Errors: a Go test that reads a file under `model/` reads it by a path resolved from the repository root or by embedding, and fails with the missing path named when the file is absent.
- **R5:** `tools/umpire/` contains the packages that load, check, lower and export the IR, as the approved module map says, and every package in it is imported by live code or is a command a Makefile target or CI step runs. Errors: an unused package found at the closing task moves to the archive or is deleted, and the done summary says which.
- **R6:** Everything R1 classifies as archive is under `model0/` or `tools/umpire0/`, moved with history-preserving renames. Errors: a move phase that also edits file content beyond paths and imports is split; the captured original bytes remain available in the archive.
- **R7:** No Go file outside `model0/` and `tools/umpire0/` imports a package inside them, and a test enforces it. Errors: a live importer that this spec cannot cut loose is listed with its imports, and the owner chooses between promoting the package into `tools/umpire`, archiving the importer, or a follow-up spec; R8 waits for that choice.
- **R8:** The archives are outside the main Go module's build. `go build ./...`, `go vet ./...` and `make lint-code` pass without compiling anything under `model0/` or `tools/umpire0/`. Errors: if R7 has an open exception, the affected archive root stays in the build, and the done summary states which and why.
- **R9:** The Makefile has no target, variable or prerequisite that names `model0/`, `tools/umpire0/`, Lean, or a file this spec removed, and every remaining `umpire-*` or model target runs to success. Errors: a target whose only purpose was archived work is deleted, never stubbed.
- **R10:** `.github/workflows/umpire.yml` names no archived path and runs only live targets. Errors: a job with no live step left is removed.
- **R11:** The model gate passes from the new layout, with the lifter's position prefix and the interpreter's position check updated together, and its `--update` rewrites only source-position paths in the checked-in IR. Errors: a position that does not resolve inside `model/` is rejected as it is today.
- **R12:** The moved checker and lowering code carry no exported declaration without a live caller. The done summary states the line counts of `tools/umpire` (live) before and after this cleanup. Errors: a declaration kept for a test only is unexported or the test is removed with it.
- **R13:** `AGENTS.md`, `model/README.md`, `tools/umpire`'s own README, `common/testing/testpilot/README.md`, `tests/testcore/testpilot/README.md`, `MILESTONES.md` and the open specs name only paths that exist. `.plans/lean` is under `.plans/archive`. Each archive root has a short README that says what it is, when it was archived and that nothing builds it. Errors: closed specs, `.turbo/plans` and `docs/superpowers` are history and stay as written.
- **R14:** `.gitignore` matches the new layout: no rule names a path that no longer exists, and a full run of the model gate and `go test` of the live packages adds no generated or untracked drift relative to the authorized uncommitted starting state (no error surface beyond that check).
- **R15:** At the closing task, the model gate, `make lint-scala`, `make lint-code`, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...` and the live tests that ran before this spec pass. Errors: a live test that cannot run without an archived artifact is reported with the artifact named, and the owner decides whether the test or the artifact stays.
- **R16:** A recorded and independently reviewed module map covers the model, the Umpire tooling and Testpilot, in Go and Scala. For each module it states the job in one sentence, the public interface and the modules it may import. The conductor exercises the owner’s delegated decision authority, and independent review precedes any package split, merge or rename. Errors: a module whose job needs "and" to state is split or the map says why not; a deviation from the proposal in this spec is recorded with its reason.
- **R17:** The import rules of the module map are enforced by a Go test over the import graph and by the Scala build: Testpilot imports nothing under `tools/umpire`, the model reader imports nothing of Testpilot, export follows the domain dependency rule in the map, and the DSL compiles without the Models. Model-owned exploration follows the explicit reader/lowering/runtime allowance. Errors: a violation fails the test with the importing file and the forbidden import named.
- **R18:** Each module's exported surface is what the map lists. Errors: an exported Go declaration with no caller outside its package is unexported or deleted; a helper package that only forwards is merged into its caller.
- **R19:** The lifter and the IR loader are split by concern as the module map says: the loader and the validator are separate files with separate tests, and the lifter is one file per kind of thing it lifts plus its entry point. No file-size limit applies. Errors: a file that still mixes two concerns after the split is listed with the reason.
- **R20:** `model/` contains no shell script. One Scala program is the gate and replaces `run.sh`, `gen.sh` and `scala.sh`, and the lifter's fixture and refusal checks are tests in the lifter's own suite. Errors: a compile that prints an error and exits 0 fails the gate; a missing tool (scala-cli, protoc, go) fails with the tool named.
- **R21:** The Umpire IR's proto package, Go package and generated Scala classes are named `umpire/v1`. Errors: if the rename changes any byte Go derives (R2) other than type names inside the IR files, it is reverted and the map records `modelir` as kept.
- **R22:** Every checked-in Case fixture and the canary's pinned Case are produced by lowering from the Scala model, or their consumer is archived under R1. Errors: a fixture that cannot be lowered yet is listed with the Query and the missing primitive, and its consumer keeps running on the old bytes until it can.
- **R23:** `model/README.md` describes the whole system and not only the Scala DSL: what a Model is, the layers it passes through (authoring in Scala, lifting, the Umpire IR, reading and checking in Go, lowering, the Testpilot IR, running and assessing a Case, export to other checkers), which module owns each layer, and which command runs each step. A Mermaid diagram shows the layers, the two IRs between them and the direction data flows. It links to `SEMANTICS.md`, the module map and each module's own README instead of repeating them. Errors: the diagram names only modules and artifacts that exist at the closing task, and a reader can follow one Query from its Scala declaration to a Verdict using the README alone.
- **R24:** `model/README.md` is written for a reader who has never seen the model. It opens with what the model is for and what a developer gets from it, in plain words, before any layer or module is named. Every project term (Model, machine, Property, Scenario, Query, realization, Case, Run, Verdict, IR, lifting, lowering) is explained in one sentence where it first appears. One small worked example runs through the page: a few lines of a real Model, what they lift to, and the Case and Verdict they end in. It assumes no knowledge of Lean, of this repository's history or of any flow spec, and cites none. Errors: a fresh reader with no context (a subagent given only the README) answers what the model is for, what the two IRs are and how to run the gate; each question it cannot answer from the page is fixed in the page, and the done summary records the questions and answers.
- **R25:** Nothing inside `model/` mentions Lean. No source file, comment, test name, document, script, fixture or file name under `model/` contains the word, a `.lean` path, or a reference to the Lean toolchain or its commands. Today 68 lines in 20 files under `model/scalav2` do (61 in the Scala sources, the rest in `README.md`, `SEMANTICS.md` and `run.sh`). A comment that explains a rule keeps the explanation and loses the reference; a comment or sentence that only points at Lean is deleted; a test named after Lean is renamed for what it checks. A check in the model gate fails on any new mention. Errors: a rule whose only stated justification was "as Lean does" gets its real reason written down (the Go consumer that depends on it), or the rule is listed for removal in fn-113.

## Boundaries
<!-- scope: business -->

- No change to what any Model says or to IR wire semantics; the namespace/type-name relocation in R21 is the sole schema naming change.
- No behavior change in the interpreter, the checker, the lowering or Testpilot. Packages move, split, merge, get renamed and lose unused parts; what they compute stays the same (R2).
- No rewrite of `tools/canary` or of Testpilot's execution and verification internals. They change import paths, take in helpers the audit finds live, and lose Lean-era dependencies.
- No cleanup inside the archives. They are frozen as moved.
- No change to what the Scala DSL offers or how a Model is written. Cleaning the Scala layer is fn-113, fn-112 and fn-114. This spec moves Scala folders, splits the lifter by concern and replaces the shell scripts.
- Closed flow specs, `.turbo/plans` and `docs/superpowers` keep their old paths.
- Deleting the archives is a later decision.
- Adding the JVM gate to CI is a later decision.

## Decision Context
<!-- scope: both — conditionally substructured -->

**Archive by moving aside.** The owner asked for the previous work to be kept and moved to `model0/` and `tools/umpire0/`. Deleting it would lose nothing git does not keep, but a visible archive answers "where did the Lean model go" without archaeology.

**Out of the build.** An archive that still compiles is still maintained: every change to `common/testing/testpilot` would have to keep 30,000 lines of frozen code building and linting. A nested `go.mod` removes that cost. The price is that the archives stop compiling as the live code moves on, which is what frozen means.

**This spec runs before the Scala cleanup.** The Scala cleanup specs (fn-113, fn-117, fn-112, fn-114) have no tasks yet, so moving first costs only path edits in those specs. Moving last would mean every task of those specs lands on paths that are about to change. fn-107 has now closed, satisfying this spec’s prerequisite.

**Goldens over a second implementation.** The hand-written Go models and the Lean dumps were the reference when Scala was the newcomer. Scala is now the reference, so a second implementation no longer says which side is right. A golden says the only thing still worth knowing: that a change did not alter what the model means.

**A module map before any move.** Moving tens of thousands of lines twice is the expensive mistake. The map is small, reviewable in one sitting, and records names and boundaries under the owner’s delegated decision authority.

**One deep model reader.** Today a caller loads with `goir` and gets results typed by `model/go/umpire`, two packages for one job. Putting the checker behind `internal/` gives callers one package and lets the checker change freely.

**Recorded Runs and binding belong to Testpilot.** They describe a Run and how a Case meets a deployment, and they know no Model. They sit in the old Umpire tree for historical reasons. Whether they are still used is for the audit.

**Scala for the gate.** A typed program can share the lifter's own knowledge of fixtures and roots, can be tested, and removes `sed` and exit-code scraping. Go was the other candidate; Scala wins because the gate's subject is a Scala build and fn-114 moves the root lists into Scala anyway.

**Rejected:** keeping `model/scalav2` as a name and only archiving its siblings. The suffix would outlive the thing it was version two of. **Rejected:** leaving the interpreter under `model/` in a `go` subdirectory. The owner's rule is that `model/` holds no Go.

## Scheduled decisions

The first task resolves live canary/helper ownership, final Scala project placement, public reader surfaces, command names, and the reserved locations for the later example and explorer. Testpilot stays under its existing root; a broader runtime relocation is outside this spec. These decisions are recorded before any restructuring.

The scope retains focused generation checks and repairs existing CI. Broad generated API drift verification and new CI coverage remain declined; see `.flow/memory/declined/generated-api-drift-verification.md`.

## Execution sequence

The audit and compatibility map precede immutable goldens and claim-by-claim parity transfer. Runtime helper promotion and reader encapsulation keep the current layout working before the one coherent namespace swap. The reader cleanup and lifter extraction can then proceed independently; schema renaming, gate replacement and consumer-fixture migration follow. Final import/build reconciliation and documentation close the work. Essential path, import and command repairs accompany the task that changes them, never a later cleanup task.

The thirteen tasks separate substantial implementation concerns rather than administrative steps. Mechanical moves may touch many files but add no new behavior. The first task must narrow any implementation task that its audited map shows cannot fit one worker iteration.

## Early proof point

Tasks fn-115.1 and fn-115.2 prove that the live dependency closure and every semantic/artifact claim can be captured before movement. If either inventory is incomplete or a source-only relocation cannot be distinguished from semantic drift, repair the map and baseline before extracting or archiving anything.

## Quick commands

Before relocation:

```bash
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/...
GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make umpire-check-scala
make lint-scala
```

The first task records the final command names; each move updates its affected commands immediately. Closing verification includes the model gate, deterministic generation comparison, both language linters, full Go build/vet, the live tooling/runtime/canary Go suites and the complete previously live Testpilot test selection. Gates run sequentially when they share build caches or local servers.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Package and consumer audit | fn-115.1 | — |
| R2 | Semantic and artifact goldens | fn-115.1, fn-115.2, fn-115.3, fn-115.5, fn-115.6, fn-115.7, fn-115.8, fn-115.9, fn-115.10, fn-115.11, fn-115.12, fn-115.13 | — |
| R3 | Parity claim transfer | fn-115.3 | — |
| R4 | Scala-only model layout | fn-115.6 | — |
| R5 | Live Go tooling ownership | fn-115.4, fn-115.5, fn-115.6, fn-115.12 | — |
| R6 | Frozen archives and reviewed moves | fn-115.1, fn-115.4, fn-115.6, fn-115.13 | — |
| R7 | No live archive imports | fn-115.4, fn-115.6, fn-115.11, fn-115.12 | — |
| R8 | Archives excluded from main Go module | fn-115.6, fn-115.12 | — |
| R9 | Live Make targets only | fn-115.6, fn-115.12 | — |
| R10 | Existing CI follows live targets | fn-115.6, fn-115.12 | — |
| R11 | New-layout gate and exact path migration | fn-115.2, fn-115.6, fn-115.10 | — |
| R12 | Unused live exports removed | fn-115.7 | — |
| R13 | Active docs and open-spec references | fn-115.13 | — |
| R14 | Ignore rules and no generated drift | fn-115.6, fn-115.12, fn-115.13 | — |
| R15 | Complete final gates | fn-115.11, fn-115.13 | — |
| R16 | Reviewed module and command map | fn-115.1, fn-115.4, fn-115.5 | — |
| R17 | Executable import rules | fn-115.4, fn-115.5, fn-115.10, fn-115.12 | — |
| R18 | Intentional public surfaces | fn-115.1, fn-115.4, fn-115.5, fn-115.7, fn-115.12 | — |
| R19 | Loader and lifter concern splits | fn-115.7, fn-115.8 | — |
| R20 | Scala gate and lifter tests | fn-115.10, fn-115.12 | — |
| R21 | Umpire IR namespace | fn-115.2, fn-115.9 | — |
| R22 | Scala-produced consumer fixtures | fn-115.1, fn-115.11 | — |
| R23 | End-to-end system README | fn-115.13 | — |
| R24 | Newcomer README reader audit | fn-115.13 | — |
| R25 | No historical-front-end references in model | fn-115.12, fn-115.13 | — |
