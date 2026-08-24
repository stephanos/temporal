---
satisfies: [R6, R7]
---
# fn-2-agentworkflow-configuration-and-cli.5 Update Agentworkflow documentation and run final verification

## Description
Update current product documentation and perform the final cleanup/verification pass for R6 and R7. Keep historical design records historical and link the approved replacement design where useful.

**Size:** M
**Files:** `tools/agentworkflow/README.md`, `docs/research/agentworkflow-oss-landscape.md`, `docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md`, any current references found by the final scan
**Touches:** [tools/agentworkflow/README.md, docs/research/agentworkflow-oss-landscape.md, docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md]

### Approach
- Rewrite the tutorial, configuration reference, provider selection, protection/snapshot, resume, and extensibility sections around the new path, model mapping, Cobra CLI, and internal-only implementation.
- Remove current migration guidance and former project-config detection/flag/path prose; retain runtime JSON/JSONL documentation.
- Update current implementation links in the OSS landscape document.
- Scan the active tool and current docs for obsolete `.spec`, former config, public root API, and backendtest references.
- Run focused tests first, then full module tests/build, import formatting, diff checks, and repository lint.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/README.md:52-179` — tutorial, tasks, and resume
- `tools/agentworkflow/README.md:179-389` — YAML contract, protection, providers, and migration
- `tools/agentworkflow/README.md:442-463` — snapshot safety and public extension prose
- `docs/research/agentworkflow-oss-landscape.md:19-71` — current implementation references
- `docs/superpowers/specs/2026-08-24-agentworkflow-configuration-cli-design.md` — approved contract

**Optional** (reference as needed):
- `docs/superpowers/specs/2026-08-23-agentworkflow-yaml-workflows-design.md` — historical superseded design

### Acceptance
- [ ] Current user documentation consistently uses `.agentworkflow/config.yml` and explains stage models plus `--model` precedence.
- [ ] Current docs no longer promise a public Go API or backend test helper.
- [ ] No current tool code/test/doc contains former project-config migration behavior or obsolete `.spec` examples.
- [ ] Runtime JSON/JSONL and `--json` documentation remains accurate.
- [ ] Full tests, build, formatting/import checks, diff checks, and repository lint have fresh recorded results.

## Acceptance
- [ ] R6 current documentation and reference cleanup is complete.
- [ ] R7 verification evidence is recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
