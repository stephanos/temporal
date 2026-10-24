---
satisfies: [R10]
---
# fn-166-mini-activity-a-small-complete-model-to.7 Rename the model-side Make targets out of the umpire- family

## Description
**Recut 2026-10-18 (one-language series):** Several targets disappear (`umpire-check-lint`, `umpire-check-equivalence`, `umpire-gen-cases`); coordinate the renames with fn-208.

Rename the model-side Make targets out of the `umpire-` family and into the repo's verb-noun convention, leaving the runtime family alone (R10). The prefix is the Go binary's name, not what the target does; the model workflow now has three naming styles (`umpire-<verb>-<noun>`, `<verb>-model`, `model`/`model-mini`), and the owner asked for `model` to be the clear one (2026-10-11).

**Size:** S
**Touches:** [Makefile, .github/workflows/**, AGENTS.md, model/README.md, model/framework/SEMANTICS.md, .plans/*.md, model/check/*.scala, tools/umpire/**/*.go, tools/umpire/export/README.md, tools/canary/**]

### Approach
- Mapping (R10): `umpire-check-model` -> `check-model`, `umpire-gen-model` -> `gen-model`, `umpire-check-lint` -> `lint-ir`, `umpire-gen-cases`/`umpire-check-cases` -> `gen-cases`/`check-cases`, `umpire-gen-fixtures`/`umpire-check-fixtures` -> `gen-fixtures`/`check-fixtures`, `umpire-check-equivalence`/`umpire-capture-baseline` -> `check-equivalence`/`capture-baseline`. `umpire-ir-bridge` builds a binary and may keep its name or become `build-ir-bridge`; decide on the task. The runtime targets keep `umpire-`: they run Cases against a cluster and know nothing of the model.
- No aliases: an old name is removed, not kept, as the owner rejected special casing in this spec. One `sed` over the live references, then a grep for the old model-side names outside `.flow/` (Makefile, `*.md`, `*.go`, `*.scala`, `*.yml`) must be empty. The gate's own messages (`rerun with --update (make umpire-gen-model)`) are references too.
- `.PHONY` and the Makefile comments follow. Flow's tasks and handoffs are history and are not rewritten.
- After task 1, since task 1 edits the same targets for `FEATURES`; a Makefile and tooling change, so it lands in the owner's sweep with the rest of the non-Scala work.
## Acceptance
- [ ] The model-side targets exist under their new names and `make` fails naming an old one.
- [ ] The runtime targets (`umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`, `umpire-assess`, backends, live tests, scratch) are unchanged.
- [ ] A grep over the live tree outside `.flow/` for the old model-side names is empty: Makefile, CI workflows, `AGENTS.md`, READMEs, `.plans/`, gate messages, Go and Scala comments.
- [ ] `model` and `model-mini` still run the loop; `make check-model` and `make gen-model` run what `umpire-check-model` and `umpire-gen-model` ran.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: