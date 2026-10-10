# Memory findings for scripted preparation

| Track | Category | Entry | Why relevant |
|---|---|---|---|
| bug | integration | `engine-summary-fields-need-the-runner-2026-10-03` | A prior exploration counter was present in the campaign journal but absent from the public runner summary and both `gomad explore` and `gomad inspect` lines. For the fresh exploration/inspection calls, assert the public `CampaignResult.ChoiceExploration` projection and cover both CLI output paths; journal-only assertions missed the drift. |
| bug | integration | `go-mod-download-inside-a-target-module-2026-09-28` | Preparation that fetches modules can mutate a target's `go.sum` when run from its module directory. Preserve existing sum validation before downloads, run download commands outside the target module, and retain a fixture with `go.mod` to catch writes. This is relevant to scripted preparation calls that might fetch dependencies. |
| bug | integration | `recompute-mapping-summaries-after-final-2026-10-08` | When preparation replaces detailed rows, recompute every derived count from the final rows and assert the row count alongside the aggregate. Retaining an earlier summary after changing inputs previously produced inconsistent preserved evidence. |

Search boundary: `flowctl memory search` ran through `/home/agent/.codex/scripts/flowctl`, after one initial `flowctl usage` attempt failed because the command was not on PATH. Plain targeted queries for scripted preparation/retained sidecar/diagnostics, capacity/bounds/calibration/policy/exploration/inspection, and retention characterization/fresh calls returned the runner projection, module-download, and recomputed-summary entries listed above. No returned entry specifically covers capacity-two calls, one-call bounds, retained sidecars, or calibration/policy tables. The memory tree contains six bug entries and no knowledge entries in this checkout; this no-hit is limited to active memory search results, not specs or source.

Consumed memory input SHA-256 before and after writing this artifact:

| File | Before | After |
|---|---|---|
| `.flow/memory/bug/integration/engine-summary-fields-need-the-runner-2026-10-03.md` | `3dc71f5098257ced1d6f0534cb044b6735738162f5e859a5ab83c46ebfff611e` | `3dc71f5098257ced1d6f0534cb044b6735738162f5e859a5ab83c46ebfff611e` |
| `.flow/memory/bug/integration/go-mod-download-inside-a-target-module-2026-09-28.md` | `74434ee0a6e989ddf785de4c692514a76fbc2dbbaa30fd72fd7fab6da3375f25` | `74434ee0a6e989ddf785de4c692514a76fbc2dbbaa30fd72fd7fab6da3375f25` |
| `.flow/memory/bug/integration/recompute-mapping-summaries-after-final-2026-10-08.md` | `2ab164c0f4f1480225a7725d747fa9da864ee987fccdc1477b1d5bee231bd501` | `2ab164c0f4f1480225a7725d747fa9da864ee987fccdc1477b1d5bee231bd501` |
| `.flow/memory/README.md` | `a05ef78e1bca711c3dbce3e3e64eecd544a2742548206a2cf6909f694188acba` | `a05ef78e1bca711c3dbce3e3e64eecd544a2742548206a2cf6909f694188acba` |

Tooling note: the initial PATH-only `flowctl usage` failed. The documented explicit executable path then successfully returned usage and ran all three targeted plain memory searches. No rerank was requested.
