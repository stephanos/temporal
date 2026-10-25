---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.11 Standalone control: framework gate passes on model/umpire alone


## Description
Prove the framework stands alone and stays Temporal-agnostic (spec R7 errors), as a check the framework gate runs itself.

**Size:** S
**Files:** standalone control under .10's real framework gate module, gate-local stage/child-mode registration, exact declarative boundary policy, framework xtask integration controls. Task .8 owns consolidated README/tier documentation.
**Touches:** [model/umpire/dev/xtask/src/gate.rs, model/umpire/dev/xtask/src/gate/**, model/umpire/dev/xtask/tests/main.rs, model/umpire/dev/xtask/tests/standalone/**, model/umpire/dev/gate/policy/framework-boundary.txt]

### Honest source and proof boundary
- Add the real standalone stage/control inside .10's existing gate module, with gate-local module registration; keep the general CLI, Makefile, test-runner policy and cache source unchanged. Register standalone integration controls in the owned existing aggregate test root. No new task or dormant placeholder is needed. If .10 lands different paths, correct Touches before admitting this task.
- The complete real isolated-copy framework entry-point run below remains mandatory on this task's exact candidate. It proves that recorded current input, not future .3/.5/.6/.14/.15/.16 changes. Preserve source/tool/stage manifests and execution scope. Task .8 already consumes all final producers and renews standalone proof whenever relevant inputs changed; reuse this proof only for equivalent unchanged inputs. Never substitute synthetic controls or describe an earlier candidate as the final tree.
- Source ownership may permit .3/.6/.11 together, but actual complete standalone execution still needs its own compilation target and admitted resources/JVM budget. Source parallelism is not authorization for concurrent unbounded required-backend gates. .5/.15 full Touches overlap this source, so their integration remains serial.

### Approach
- Copy the exact current candidate bytes of `model/umpire` into scratch outside the repository, retaining a source manifest; `git archive HEAD` alone omits candidate edits. Exclude generated caches and reject source symlink escapes. No Temporal tree, root Makefile/.bin, Git metadata or repository-specific environment may supply an undeclared source/tool dependency. Use a separately owned compilation target, not the main checkout's target.
- The isolated copy builds and invokes both framework-local xtask and its dedicated guest driver through their normal entry points. Clear repository-specific prebuilt-driver/root overrides, require the copied local pin/configuration inputs, and verify the shared context resolves only the copied framework and explicitly owned outputs. Missing copied local configuration refuses instead of finding the parent checkout. Do not use a parent prebuilt driver to conceal a source dependency.
- The outer control runs the framework entry point once in a named child mode that skips only this copy-control stage. All other framework stages run exactly once; recursion and a silently skipped stage fail. Record cost and tier, then register the stage for .12.
- Agnosticism checks follow .9's policy ownership: filesystem-based scanning works without Git; generic dependency/path checking has no Temporal knowledge; Temporal deny-vocabulary and any narrowly justified policy-data exceptions are explicit. Preserve the existing Temporal boundary test. Check both declared and resolved path dependencies; do not let copied caches supply absent inputs.
- Negative controls: a planted path dependency on a Temporal crate fails, a planted `temporal` identifier fails, and a planted `../../../..` repo-root path fails.

### Investigation targets
**Required:**
- `model/design/framework-gate.md`
- `model/dev/xtask/src/retired_names.rs` (or its moved successor)
- `model/scripts/test-capabilities.sh` — existing control pattern

## Owner validation cadence (2026-10-25)

The real directory-only framework entry-point run is the isolation gate being changed and remains required, alongside planted violation controls. Reuse equivalent evidence at .8 only if all relevant inputs remain unchanged. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The framework gate passes on an isolated copy of `model/umpire`, building its own local xtask/guest driver and consuming local context/pins without parent-source/configuration fallback.
- [ ] All three planted violations fail with a named reason.
- [ ] Cost and control tier (quick, check or validate) recorded with exact source/tool/stage inputs; task .8 states the tier in the consolidated README and renews changed-input standalone proof.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
