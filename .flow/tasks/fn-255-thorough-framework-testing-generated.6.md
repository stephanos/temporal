---
satisfies: [R5, R7]
---
# fn-255-thorough-framework-testing-generated.6 Coverage report per framework crate; aim generators and corpora

Touches: [model/umpire/mise.toml, model/design/coverage.md, model/umpire/dev/**, model/umpire/dev/decoder-testing/**, model/umpire/fuzz/corpus/**, model/umpire/fuzz/fuzz_targets/**]

## Description
Show which refusal and lowering branches no test reaches, then aim .2's generator and .3's decoder corpora at those branches (R5), with bounded smoke evidence and scheduled broader reports.

**Size:** S
**Files:** cargo-llvm-cov pin, native coverage command/parser and controls, `model/design/coverage.md`, generator weights, shared stable decoder harness and corpus additions.

### Prerequisites and ownership
- Consume committed fn-254.12 before execution or task closure, alongside the generator/corpus integration prerequisites. Admit coverage test/case/crate counts, smoke and scheduled timeouts, output limits and related workload in its single committed authoritative framework budget table. Do not retain a duplicate provisional coverage budget table or start evidence runs before admission.
- The coordinator serializes the shared mise pin, authoritative budget table, native command registration, generator weights and decoder harness/corpus edits with .2/.3/.4/.5/.7. Acquire exact exclusive ownership for overlapping files and preserve current native DAG, cancellation, resource admission and count authorities.

### Approach
- Pin cargo-llvm-cov through the framework mise file. Collect line coverage and explicitly enabled `--branch` coverage on .3's dated nightly with its matching llvm-tools component in an isolated target directory. Keep the stable production toolchain unchanged and record exact toolchain/components/flags. The cargo-llvm-cov usage documentation requires nightly for branch coverage.
- Exercise the stable production decoder harness at `model/umpire/dev/decoder-testing`, a member of the framework root workspace. Stable deterministic replay shares these same decoder/oracle bodies; thin nightly libFuzzer wrappers live in nested `model/umpire/fuzz` with its own lockfile. Do not introduce libFuzzer into stable root-workspace dependencies. Corpus improvements must reach the shared bodies used by both replay and fuzzing.
- Provide scheduled per-framework-crate coverage commands with explicit test/case manifests and authoritative total work/time caps. Reports preserve completed/remaining crates/tests when capped, exact manifest, counters, source/tool identity and resume inputs. Full campaigns remain scheduled-only and are not closure prerequisites.
- For closure run only the separately bounded representative test/case manifest. List its unreached refusal and lowering branches in `umpire-front`, `umpire-ir`, `umpire-tla` and the interpreter with file:line and recorded branch evidence.
- Adjust generator weights and add corpus seeds aimed at representative uncovered branches. Repeat only the bounded comparable manifest and record before/after branch evidence, keeping toolchain, flags, source/manifest changes and workload differences explicit. Label smoke coverage separately from broader scheduled reports.
- Enforce and report admitted work/time bounds with named over-cap controls; a cap or tool refusal cannot be reported as completed coverage. .7 owns final gate/tier wiring; coverage campaigns never enter quick/check/validate.

### Investigation targets
- `model/design/coverage.md`, .2's maintained generator and .3's shared stable decoder harness/corpora.
- fn-254.12 authoritative budgets and current native lifecycle/count receipts.
- Pinned cargo-llvm-cov usage and nightly branch-coverage requirements.

## Acceptance
- [ ] Per-crate scheduled commands and capped partial-report controls work; bounded smoke coverage records toolchain/components, flags, exact manifest, actual work counts and time in `model/design/coverage.md`.
- [ ] Representative unreached refusal/lowering branches drive a generator/corpus adjustment with comparable bounded before/after branch evidence. Both stable replay and nightly wrappers exercise the shared stable decoder bodies; libFuzzer stays outside the stable workspace.
- [ ] Coverage execution waits for committed fn-254.12 and admitted entries in its single authoritative budget table; over-cap controls fail by stage name and capped reports retain unfinished work. Full campaigns remain scheduled-only.

## Done summary
Blocked:
Execution and task closure wait for the committed fn-254-fast-validation-gates-and-a-real-inner.12 closure in umpire and admission of this task's work/time limits to its single authoritative budget table. Disjoint source preparation does not satisfy this gate. Flow task dependencies only accept same-spec IDs; after verifying that closure and admitted entries, the conductor may reset this block and admit the task normally.
## Evidence
- Commits:
- Tests:
- PRs:
