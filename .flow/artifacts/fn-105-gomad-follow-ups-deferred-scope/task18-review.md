# fn-105.18 (D18) review rounds

Raw codex bridge on the working-tree deliverables (commits forbidden). Reviewer: gpt-5.6-sol at high reasoning effort, read-only sandbox. Command per round: `codex exec -s read-only --skip-git-repo-check -m gpt-5.6-sol -c model_reasoning_effort=high -o <scratch>/fn-105.18-review.md "<prompt>" </dev/null`. The prompt carried the task description and acceptance, spec R18, the documentation hunks, and the verification checklist; the reviewer read the report, evidence, and source from the repository.

## Round 1

1. **blocker** — [fn105-d18-evidence.json](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d18-evidence.json:56): the strict and traced manifest SHA-256 values are swapped. Strict is `574b1235…`, while traced is `fc105084…`. Swap the recorded hashes so each qualification outcome binds to the correct manifest.

2. **should-fix** — [fn105-d18-evidence.json](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d18-evidence.json:97): the index omits the cited `fn105-d18-gomad-suite-strict-seed11-sibling-no-poller.txt` and the third suite run that produced it. Add the file and that run’s command/outcome to the index.

VERDICT: NEEDS_WORK

## Round 2

Round-1 findings are resolved. Recomputed file/internal/report digests match the index for strict (`574b… / f1c8… / 3d5d…`), forward (`f7a3… / c4cb… / 1812…`), and traced (`fc10… / 20bf… / 650d…`). The third seed-11 run and sibling no-poller evidence are indexed.

1. **should-fix** — [GOMAD_D18_WORKER_CANCEL_DELIVERY.md:228](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md:228): Option A includes removing the skip, and line 240 says to apply it before qualification, while line 250 says the skip must only be removed after qualification, including linux/amd64. Make the sequence explicit: apply `clock_tick: forward` while retaining the skip, qualify an unskipped scratch candidate, then remove the tracked skip and regenerate only after all criteria pass.

2. **should-fix** — [fn105-d18-run.sh:4](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d18-run.sh:4): The retained regression script hardcodes an ephemeral `/private/tmp/.../d18` directory and repository path, then invokes `$S/timeline.py` rather than the retained `fn105-d18-timeline.py`. It cannot produce the promised evidence on a clean checkout. Resolve paths relative to the script/repository and use a caller-provided or temporary output directory.

VERDICT: NEEDS_WORK

## Round 3

Round-2 findings are resolved.

1. **nit** — `docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md:109`: the cited range `worker_commands_task.go:32-52` omits `GetKey`, which establishes the immediate key at lines 29–31. Change the citation to `:29-51`.

VERDICT: SHIP

## After the verdict

- Round 1 findings fixed before round 2: digests recomputed by file name, the three qualify-set reports retained, the third seed-11 run and the sibling no-poller file indexed.
- Round 2 findings fixed before round 3: skip-removal sequence made explicit in the report; the retained run script made path-independent and re-run once.
- Round 3 nit applied after the SHIP verdict and not re-reviewed: citation changed to `worker_commands_task.go:29-51`.
