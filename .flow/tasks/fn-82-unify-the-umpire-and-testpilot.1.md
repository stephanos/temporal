---
satisfies: [R7]
---
# fn-82-unify-the-umpire-and-testpilot.1 Harden the retired-vocabulary gate and buf breaking config

## Description
Harden the retired-vocabulary gate before any rename lands (R7, spec §R7). Every later task
adds its retired names here, so the gate must fail closed and scan the right specs first.

**Size:** S
**Files:** `tools/umpire/internal/retiredvocabulary/check.go`, `tools/umpire/vocabulary/retired_vocabulary_test.go`, `proto/internal/buf.yaml`
**Touches:** [tools/umpire/internal/retiredvocabulary/check.go, tools/umpire/vocabulary/retired_vocabulary_test.go, proto/internal/buf.yaml]

### Approach
- `addFile` at `check.go:155-165` returns nil on `os.IsNotExist`; make it return an error naming the path. `addOpenFlowRecord` may keep its silent skip for closed specs, but a listed id whose `.json` is absent must error too.
- Fix `downstreamSpecs` at `check.go:26-42`: replace the deleted `fn-28-authorized-remote-staging-black-box` and the misspelled `fn-33-run-resumable-semantic-exploration` with the real ids (`flowctl specs --json` is the source of truth), and add `fn-46-export-lean-model-module-impact-index`, `fn-70-scheduled-canary-proof-of-concept-as-a`, `fn-74-deepen-testpilot-worker-activation`, `fn-78-typed-temporal-authoring-and-checked`, `fn-79-deferred-nexus-operation-cancellation`.
- Add a guard in `buildRetiredRules` (`check.go:264`) that rejects a rule whose token is a single bare word (no uppercase after the first character, no `.`, `/`, `_`, or `%`), because the lowerCamel variant would ban ordinary English; test it.
- Add `model/Testpilot` and `model/Shared` (`.lean`, `.md`) to the tree roots in `scopedPaths` (`check.go:93-107`); today `Testpilot/Scoped.lean` and `Shared/Scoped*.lean` are unscanned, so later retirements would not be enforced there.
- Add a `breaking.ignore` entry for `temporal/server/api/testpilot/v1` in `proto/internal/buf.yaml` so `develop/buf-breaking.sh:63` stays green through the R4 rename; confirm with `make buf-breaking` (`lint-protos` runs `buf lint` only and proves nothing about breaking changes).
- Extend `retired_vocabulary_test.go` (three tests at `:13`, `:86`, `:102`) with: missing listed path fails with the path in the message; bare-word rule rejected; stale spec id fails; a retired token placed under `model/Testpilot` is reported.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/retiredvocabulary/check.go:26-42,93-131,155-165,264-338` — scan list, scoped paths, silent skip, rule builder
- `tools/umpire/vocabulary/retired_vocabulary_test.go` — existing command-level tests to extend
- `develop/buf-breaking.sh:55-70` — the internal-proto invocation and its config path

**Optional** (reference as needed):
- `Makefile:1135-1137` — the make target that runs the command
- `tools/umpire/regression/ci_workflow_test.go:28,295-307` — CI test that pins the target name and retired legacy paths

### Key context
- The gate is the enforcement point for the whole spec; a bug here silently weakens every later task.
- fn-81 .3 to .5 also edit `check.go` and its test; this spec depends on fn-81, so start only after it closes.
## Acceptance
- [ ] `addFile` and a listed open-spec id with no `.json` fail `umpire-check-retired-vocabulary` with the missing path named
- [ ] `downstreamSpecs` contains only ids that resolve under `.flow/specs`, including the five added specs
- [ ] A retired rule for a bare English word is rejected and covered by a test
- [ ] The gate scans `model/Testpilot` and `model/Shared`, proven by a test that plants a retired token there
- [ ] `proto/internal/buf.yaml` ignores the testpilot package for breaking checks and `make buf-breaking` passes
- [ ] `go test -tags test_dep ./tools/umpire/...` and `make umpire-check-retired-vocabulary` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
