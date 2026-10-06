---
satisfies: [R7]
---
# fn-134-capabilities-own-their-properties.6 Docs, MILESTONES and close

## Description
Rewrite the docs for capabilities and capability Properties, run R7's vocabulary check, update MILESTONES.md, and close the spec.

**Size:** M
**Files:** `model/README.md`, `.plans/UMPIRE_MODULES.md`, `tools/umpire/README.md`, `MILESTONES.md`, source header comments
**Touches:** [model/README.md, .plans/UMPIRE_MODULES.md, tools/umpire/README.md, MILESTONES.md, model/umpire/**, model/temporal/capabilities/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `model/README.md`: rewrite "Capabilities and their laws" (:1013-1185) as "Capabilities and their Properties": the capability table, a worked `capabilities` section, a shared set, how a Property is brought (including from two capabilities), `except`/`overriding`, bounds in `queries`, and `origin`. Drop the law sidecar and "laws on the table" parts, and the lint law kinds (:181-193). Update the section lists (:419, :675-687, :1008) and the stray mentions (:376, :439-440, :534, :624, :752, :805-880, :888).
- `.plans/UMPIRE_MODULES.md:36, 43, 97-114` (the live module map), `tools/umpire/README.md:41`.
- Source header comments that still say law, catalog or `implements` (`model/umpire/Machine.scala:57, 303`; `model/temporal/capabilities/*.scala` headers; `check/checking.go:3`). Preserve the rest of each comment.
- R7 vocabulary check: grep `model/`, `tools/` and the READMEs (excluding `.flow/`, `.plans/archive`, `model/build/history`) for law, convention and catalog in this sense.
- MILESTONES.md: mark fn-134's rows done. Reword the intro line about `model/temporal/` ("capability vocabulary, laws, …") and the deferred item fn-122.7 (Pausable on fn-119's example).

## Acceptance
- [ ] The README describes capabilities, capability Properties, the `capabilities` section, bounds in `queries` and `origin`, with no law sidecar or law table.
- [ ] The vocabulary grep finds no law, convention or catalog in this sense outside `.flow/` and `.plans/archive`.
- [ ] MILESTONES.md rows for fn-134 are done; the intro and deferred items are reworded.
- [ ] Full gate: `make umpire-check-model`, `make umpire-check-cases`, `make umpire-check-lint`, `make lint-model` and the Go suite pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
