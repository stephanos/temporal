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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
