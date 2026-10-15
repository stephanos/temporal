# fn-182.3 trimmed-guest qualification (provisional) and R4 rerun procedure

Status: these are provisional functional/replay results on pre-fn-178 source. Matched timed R4
measurements have **not** been run because they need a quiet host. Nothing here is acceptance on
the joined fn-178 candidate.

## Functional/replay cells (R5, partial)

- Candidate: branch `fn182` at `52a9413803` plus uncommitted docs that do not affect execution
  (`candidate-1-binding.txt` records HEAD, index digest, helper and compiler). Helper SHA256
  `a238ec8c…` is byte-identical to the helper built in the baseline worktree at `01c419c40a`.
- Driver: `run-cells.sh <worktree> <evidence-root>`. It runs one fresh `go test` process per cell
  with `GOMAD_WASM_QUALIFICATION_PROFILE=cooperative`, a provisional manifest and the integrated
  digest = sha256 of `git ls-files -s`. Multi-instance inputs are the retained fn-176 producer
  configs `gomad-fn176-qualification-live-20261015.1hg1RG/clusters`, and the staged fixture is
  byte-copied from `testdata`.
- Evidence: `/home/agent/wt/fn182-tmp/qual/candidate-1` (local). Stage log: `candidate-1-stages.log`.

| Cell | Exit | Qualified | Deterministic | Target success | Execs/replays | Replayable bugs | Budget violations | Module |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| onebox success | 0 | true | true | true | 2/2 | 0 | 0 | 306,370,445 B `53e844ff…` |
| onebox canary | 0 | false (expected) | true | false (expected) | 2/2 | 1 | 0 | same |
| multi-instance success | 0 | true | true | true | 8/8 | 0 | 0 | 209,079,548 B `b8f1e71b…` |
| multi-instance canary | 0 | false (expected) | true | false (expected) | 8/8 | 1 | 0 | same |

The success witnesses pass and the intentional-failure canaries stay observable as one replayable
bug each. Repeated execution is deterministic, and fresh exact replays ran within each candidate's
own module identity. Existing budgets passed. The onebox cold helper peak was 5,044,191,232 B
against a 6 GiB limit, and the warm peak was 1,276,428,288 B against 4 GiB.

The cooperative trimmed one-box module is 306,370,445 B. The last qualified full-closure module
(`5835781d…`, checkpoint 2026-10-14) was 337,961,968 B, a difference of −31,591,523 B. That
comparison is historical, not a matched measurement. The fn-176 retained multi-instance module is
the full-closure baseline for the cluster cell.

Wall times are from a loaded shared host and are **non-acceptance**.

Not yet run: the baseline (default-tag) cells from the same parent for a same-source functional
A/B. Prior fn-176 frozen results supply only historical full-closure witnesses. Also not run:
`make -C tools/gomad test-wasm` and lint on this branch.

## R4 matched measurement (prepared, NOT run)

The driver runs on a quiet Linux/arm64 host, after the post-fn-178 baseline worktree and the
rebased candidate exist and each has its helper built (`CARGO_BUILD_JOBS=4 make -C tools/gomad wasm-helper`):

```sh
.flow/artifacts/fn182-guest-qualification-2026-10-15/measure-r4.sh \
  /home/agent/wt/fn182-base /home/agent/wt/fn182 /home/agent/wt/fn182-tmp/measure/r4-<date> 5 "onebox multi-instance"
```

- **Rotation:** it alternates ABBA order per rep and waits for load < 1.5 and MemAvailable ≥ 8 GiB
  before each cell.
- **Fresh roots:** every rep uses a fresh root, so the first execution is a cold engine compile and
  later executions and replays are warm cache hits.
- **Host record:** it writes `host.txt` (boot id, CPU, revisions) and `stages.log`.
- **Output:** `summarize.py` writes `summary.md` with median [min, max] (n) per side. It labels
  each delta as beyond or within spread.
- **Metrics:** module bytes, compiled-cache bytes, Go preparation wall, cold compile, warm cache
  load and engine prepare, cold/warm fresh instantiation, cold/warm execution wall, cold/warm
  helper peak RSS, qualification and budget.

Expected duration is about 2.5 h for 5 reps (one-box ≈ 4 min and multi-instance ≈ 9–20 min per
cell). The baseline worktree must be at the joined fn-178 commit, and the candidate rebased onto
it.
