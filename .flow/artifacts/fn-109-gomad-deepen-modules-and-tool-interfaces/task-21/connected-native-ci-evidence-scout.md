# Connected native CI evidence scout

Read-only GitHub inspection on 2026-10-04. No external or Flow state was changed.

## Branch and exact-source relationship

- Local candidate/source checkpoint: `4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7` (`gomad: checkpoint task 19 architecture and purity boundaries`).
- Connected public repo: [`stephanos/temporal`](https://github.com/stephanos/temporal), repository id `731354515`, default branch `main`.
- Remote `gomad` branch currently points to `0dd05b313acd0986312da7fd3159520e6a21f1bf`, the local baseline checkpoint, not the task-19 source checkpoint. Run metadata and Actions logs identify subsequent runs at commits `8789deab055d1b72ac6bc86711d74f3fd7313fa2`, `67dbe02666afd68c064c6c3cb9197b03d4664687` and others; none establishes execution of `4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7`.
- I attempted the exact-source commit endpoint and GitHub returned 422 `No commit found for SHA`; compare against that SHA returned 404. Thus the exact task-19 checkpoint is not available in this connected repo for native qualification. Historical pass is not transferable across this source delta.

## Retrieved run evidence

- Latest recent successful full manual dispatch inspected: [run 36968858553](https://github.com/stephanos/temporal/actions/runs/36968858553), `workflow_dispatch`, SHA `8789deab055d1b72ac6bc86711d74f3fd7313fa2`, completed 2026-10-02. Jobs `core`, `core-linux`, `host-tools-linux`, both functional smoke jobs, and `temporal-integration` all report success. It includes native Darwin/arm64 and Linux/amd64 execution, with successful qualification artifact metadata. Linux temporal artifact id `11211377694` (`gomad3-linux-temporal-qualification-36968858553`, SHA256 `41fb0570...`), Darwin temporal artifact id `11213436175` (`gomad3-temporal-qualification-36968858553`, SHA256 `c30b6084...`); artifacts remain listed unexpired. The Linux core qualification artifact is id `11211123021`.
- This run is not exact-source evidence: run SHA `8789de...` is remote and the local candidate `4a6467...` is neither in the fetched history nor found by the commit API. Workflow job summaries/logs prove named CI steps ran; the artifact listing by itself does not prove embedded toolchain/source hashes. Full qualification artifact (4.7 GB) was not downloaded.
- More recent dispatch [run 37027023410](https://github.com/stephanos/temporal/actions/runs/37027023410), SHA `67dbe02666afd68c064c6c3cb9197b03d4664687`, failed Linux host tier: `TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/cancellation/simulation-exploration` observed `Reason:"target_supervision"`, expected `"cancelled"`. Linux core qualification upload had no files, so it cannot support a Linux qualification claim. Darwin core, both smoke jobs, and temporal integration succeeded on that different commit. Run’s 28 representative-workload artifact listing exists but its quality/expected counts were not inspected.
- Runs at `cdc86ed...` (run 36748617624), `adb0b1...` (36497204705) and `cfba287...` (36493869196) are also successful dispatches in the queried page, but older than 36968858553; not inspected further because they cannot solve the exact-source gap.

## Coverage and limits

- Read-only Actions runs query was `GET /repos/stephanos/temporal/actions/runs?branch=gomad&per_page=100` (two pages). It exposed 22 Gomad workflow manual dispatches in page 1 and no runs page 2; no schedule run appeared. This is a bounded page query, not proof that no historical scheduled run exists. The `fetch_commit_workflow_runs` wrapper filters only PR-triggered runs and first page only; its empty result for candidate SHA is not absence evidence for scheduled/manual runs.
- The workflow at the successful run SHA includes `workflow_dispatch` and schedule triggers, native core jobs, and Linux/Temporal qualification on schedule/dispatch. A follow-up inspection of the exact run workflow and its complete six-job response corrected the initial assumption about omitted soak jobs: this workflow predates the soak jobs and ran none. The soak-enabled workflow is present at the later remote branch tip. See `../../fn-112-gomad-determinism-assurance-and-test/task-10/connected-soak-evidence.md`; no soak report was opened or native bound established.
- Native toolchain-build success, named qualification steps, and retained artifact metadata are evidence for the corresponding remote source SHA only. They do not qualify the local source candidate and do not establish the current source/toolchain hashes.

## Safe next step / blocker

Have the source candidate commit become available on remote `gomad` (or an appropriately scoped remote branch), then run the currently configured `workflow_dispatch` native qualification and collect the per-platform artifacts. Inspect embedded source/toolchain hashes and (for task 10) retain soak report contents before accepting. Until then, native qualification of task-19 source and task-10 soak evidence for this source are unestablished. No active run was found or left to poll.
