# Final worker timing evidence

Code checkpoint: `8f59fcf4e85cb7fc7cf6f9f519338f5bee4c1885`.
Command: `CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2177-rename GOCACHE=/home/agent/.cache/codex-build/fn2177-go make model-time`.
Worker session10347 terminated with exit0 on2026-10-21 at05:56UTC.
Log: `.flow/tmp/fn2176-final-model-time.log`.

Gate total170.532seconds exceeds the soft30second target; inner edit/test loop1.983seconds meets the soft10second target. Rust tests102.673seconds and Temporal Go host tests65.186seconds dominate. The exhaustive mutation/context corpus and assertions were not reduced to meet the soft budget. The timing command passes, but the full-gate soft performance goal is not achieved.

Before this run, default and required-backend full gates and the uncompromised live gap/privacy gate had terminated successfully. The root verification artifact records those exact results and production equivalence. Activity TLC remains source-refused; required-backend success does not establish activity TLC agreement.

The timing receipt check was not honored because preserved foreign MILESTONES.md changes make the shared checkout dirty. A receipt-writer refusal is recorded honestly as NO_RECEIPT, not a clean receipt. Subsequent pre-review checkpoint changes are Flow evidence only, with no executable or golden change.
