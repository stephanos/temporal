# Connected CI soak evidence scout

Read-only GitHub Actions inspection, 2026-10-04. No workflow was dispatched and no external state was changed.

## Result

No actual `gomadtool soak` report or soak ledger was retrievable for either native platform from the bounded run set inspected. The two successful/failed recent dispatches that were previously found predate addition of the soak jobs in their workflow source. They ran core, host-tool, functional-smoke and integration jobs, not soak jobs. Their retained artifacts are qualification and functional-smoke archives, not soak report/ledger artifacts.

## Runs and source

- Local task-10 gate source is present on remote branch `gomad` at `0dd05b313acd0986312da7fd3159520e6a21f1bf` (same as branch tip observed in the earlier native scout). The workflow at that ref contains `determinism-soak-darwin` and `determinism-soak-linux`, but the workflow commit itself has no corresponding run in the queried recent branch run set.
- Successful dispatch [36968858553](https://github.com/stephanos/temporal/actions/runs/36968858553), SHA `8789deab055d1b72ac6bc86711d74f3fd7313fa2`, completed 2026-10-02. Direct jobs endpoint listed only `core`, `host-tools-linux`, `core-linux`, two functional-smoke jobs and `temporal-integration`; no soak matrix jobs. Artifact endpoint lists six artifacts, all `gomad3-*qualification*` or `gomad3-functional-smoke*`; zero soak artifacts. Its workflow source at that SHA has no `determinism-soak` jobs.
- Failed dispatch [37027023410](https://github.com/stephanos/temporal/actions/runs/37027023410), SHA `67dbe02666afd68c064c6c3cb9197b03d4664687`, completed 2026-10-02. Direct jobs endpoint likewise lists only the six non-soak jobs; artifacts list contains qualification/smoke names only. Linux `core-linux` fails the host tier as recorded in the earlier native CI report, so it cannot serve as successful Linux qualification either.
- `/actions/runs?branch=gomad&per_page=100` pages 1–3 were inspected; page 1 exposed 22 workflow dispatches through 2026-09-28 plus one PR run, and pages 2–3 were empty. No schedule run or soak matrix job appeared in those pages. This bounded query is not a full historical audit, but the available runs page contains no task-10 evidence. `fetch_commit_workflow_runs` is PR-only/first-page and was not treated as a schedule/dispatch inventory.

## Qualification and bound

The conductor independently fetched run 36968858553's jobs with `per_page=100`:
the response reports `total_count: 6`, every returned job is completed/success,
and none is a soak job. The exact workflow at `8789deab...` contains dispatch
and schedule triggers but no `determinism-soak` job. This confirms the earlier
native scout's omitted-soak assumption was wrong; that report is corrected.
These read-only responses are not native qualification reports.

There is no report content from which to read platform, workload, seed, execution/toolchain identity, fresh or cumulative clean repetitions, diagnostic/load settings, `execution_wall_nanos`, divergence, overflow, target failure, or infrastructure failure. Therefore no native first-bound count, per-cohort bound, or task-10 completion claim can be made. The local handover's linux/arm64 stand-in run explicitly measured no Gomad bound. Earlier successful native qualification artifacts do not satisfy task 10: they are from earlier workflow/source revisions and are not soak artifacts.

## Retrieval handles and next action

Run IDs checked: `36968858553`, `37027023410`; artifact inventories: respectively six and four entries, with no `gomad3-soak-*` names. Latest remote soak workflow source ref: `0dd05b313acd0986312da7fd3159520e6a21f1bf`. No live run/job handle exists to poll. The needed action is a schedule/dispatch of this soak-enabled workflow followed by inspection of its Darwin and Linux matrix jobs and each uploaded per-workload/per-seed report and ledger. Keep qualification status separate: the current local source checkpoint `4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7` remains unqualified by the previously located older remote runs.
