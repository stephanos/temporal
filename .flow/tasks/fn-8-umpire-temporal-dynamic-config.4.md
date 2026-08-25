---
satisfies: [R3, R9]
---
# fn-8-umpire-temporal-dynamic-config.4 Retain and integrate DynamicConfig generation

## Description
Materialize the generator-owned catalog and add only the generation/build/documentation integration required by R3 and R9. Keep generated files mechanical and the public model import explicit.

**Size:** M
**Files:** `model/Temporal/DynamicConfig.lean`, `model/Temporal/DynamicConfig/Types.lean`, `model/Temporal/DynamicConfig/Settings.lean`, `model/Temporal.lean`, `model/README.md`, `Makefile`
**Touches:** [model/Temporal/DynamicConfig.lean, model/Temporal/DynamicConfig/**, model/Temporal.lean, model/README.md, Makefile]

### Approach
- Add a generation-only Make command using the repository's `mise exec -- go run -tags test_dep` convention and the model output root. Do not add a check target, retained diff command, or CI wiring.
- Run the generator to create the complete retained facade and directory; never hand-edit or preserve authored content inside the owned output.
- Import the structural facade from the root Temporal model only after generation and candidate Lean verification succeed. Sequence this shared-root edit after the parent spec dependency completes.
- Extend model documentation with the generated structural ownership boundary, handwritten interpretation boundary, generation command, and focused Go/Lean verification. Avoid claims about YAML parsing, live runtime state, converter execution, or complete semantic classification.
- Verify a second generation is byte-identical without creating a repository drift-check interface.

### Investigation targets
**Required** (read before coding):
- `Makefile:79-121` — Lean and existing generator command variables
- `Makefile:984-999` — existing Umpire generation/check target layout
- `model/Temporal.lean:1-7` — public model import root
- `model/README.md:3-34` — current generated-versus-authored ownership documentation
- `model/Temporal/API.lean:1-7` — sibling generated facade convention

**Optional** (reference as needed):
- `.flow/memory/declined/generated-api-drift-verification.md` — binding exclusion of retained drift/CI verification

### Key context
The public operational dynamic-config README is not the authority for the current eight-policy registry and must remain untouched. Document source/runtime projection boundaries only in the model developer documentation.

### Quick commands
```bash
make umpire-gen-dynamic-config
go test -count=1 -tags test_dep ./cmd/tools/genleandynamicconfig
cd model && mise exec -- lake build
```

## Acceptance
- [ ] `make umpire-gen-dynamic-config` produces the retained three-module catalog through the new command.
- [ ] The root Temporal model imports the generated facade and the complete model build elaborates it.
- [ ] Model documentation states generated ownership, handwritten semantic ownership, generation, and focused verification without overclaiming runtime/config-converter behavior.
- [ ] Repeated manual generation is byte-identical.
- [ ] No `umpire-check-dynamic-config`, generated-drift command/prose, CI workflow, or unrelated operational-document change is introduced.

## Done summary
Retained the complete generated DynamicConfig facade, types, and 685-setting catalog; added generation-only Make integration, the public Temporal import, and model ownership/verification documentation. Task Quick commands passed, the parent Go suite passed, and repeated generation was byte-identical.

baseline: red (`make umpire-gen-dynamic-config` failed pre-edit because this task adds the target; `make lint-code` failed pre-edit when shared `/tmp` was full)
GATE_SKIPPED:unittest:green-receipt 754664ee - baseline reused from prior post-gate pass
GATE_SKIPPED:smoke:green-receipt 754664ee - baseline reused from prior post-gate pass

Canonical `make lint-code` remains inherited-red with 1828 branch-wide findings in pre-existing Go files; its auto-fix rewrites were reverted, and this task changes no Go source.

stage: impl-review - ran [2026-08-25T17:01:59Z..2026-08-25T17:04:17Z]
## Evidence
- Commits: 98ab54e9ab526012d550a1f0166aa01bc18985ea
- Tests: baseline: red (make umpire-gen-dynamic-config failed pre-edit: target absent), baseline: red (make lint-code failed pre-edit: shared /tmp exhausted), GATE_SKIPPED:unittest:green-receipt 754664ee - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt 754664ee - baseline reused from prior post-gate pass, make umpire-gen-dynamic-config, make umpire-gen-dynamic-config (second run byte-identical), go test -count=1 -tags test_dep ./cmd/tools/genleandynamicconfig, go test -count=1 -tags test_dep ./common/dynamicconfig ./cmd/tools/genleandynamicconfig, cd model && mise exec -- lake build, git diff --check c8a2b643cce1f575fad748f8cbea93eaa616c168..HEAD, make lint-code (inherited-red: 1828 branch-wide findings in pre-existing Go files)
- PRs: