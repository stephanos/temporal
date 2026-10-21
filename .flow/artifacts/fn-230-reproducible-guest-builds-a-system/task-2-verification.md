# Task 2 verification ledger

All required pre-review checks and post-SHIP final worker verification passed.
This ledger supersedes the pending gate statements in the earlier source-freeze
artifact. Formal verdict provenance is in `task-2-formal-review.md`.

- `make umpire-check-model-slice umpire-check-model-gaps`: exit 0. Slice package
  1.111 s; complete gaps package 355.473 s. All 2336 unique gaps subtests passed
  once: 2238 generic cases, 79 strict cases and 19 scope-boundary controls.
  Existing full observation, 1 MiB frame, privacy and timeout assertions remain.
  The actual report matches the unchanged golden byte-for-byte, SHA-256
  `a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0`.
  Regular log/status: `.flow/tmp/fn2302/live-final.log` and `live-final.rc`.
  Report snapshot: `.flow/tmp/fn2302/root-verified-live-gap-report.txt`.
- `make model-check`: exit 0. Generic and Temporal Go hosts passed, including
  unchanged caught-panic/overflow, Reset and trap/import-denial controls.
  `.flow/tmp/fn2302/model-check-final.log` and `model-check-final.rc`.
- `UMPIRE_BACKENDS=require make model-check`: exit 0, including actual required
  TLC execution/agreement and negative controls. Temporal host 9.889 s.
  `.flow/tmp/fn2302/model-check-required-final.log` and its `.rc`.
- Initial `make model-time`: exit 2, not a pass. It reached freshness in 33.431 s
  then failed the new nested-Make test's stale-summary assertion. The timing
  wrapper exports `UMPIRE_TIME_STEPS=1` and intentionally suppresses step output.
  A focused run with that environment reproduced exactly one failing test out of
  six, exit 1. The fixture now removes only that timing flag from its nested Make
  environment, retaining every existing stale/status/offset assertion. The same
  six-test timing-environment command then passed, exit 0. No production wrapper,
  builder, budget or assertion changed. Full retry passed, exit 0: gate 51.936 s
  versus the unchanged 30 s soft target; inner loop 1.607 s versus 10 s. Native
  tests took 24.041 s, guest builds 14.353 s and Temporal host 9.862 s.
  Logs/statuses: `model-time-final.*`, `timing-fixture-red.*` and
  `timing-fixture-green.*` and successful `model-time-retry.*` under `.flow/tmp/fn2302/`.
- Post-fixture full default and required-backend `make model-check` reruns both
  passed, exit 0, without receipt reuse. Temporal hosts took 9.856 s and 9.803 s.
  Logs/statuses: `model-check-postfix.*` and `model-check-required-postfix.*`.
- The current 3326027-byte committed asset failed real `make model-check` after
  one byte changed at offset 8, Make exit 2 with existing/built sizes and offset.
  `make umpire-guest` returned 0 and restored exactly the original bytes.
  `current-stale-control.*`, `current-stale-model-check.log` and
  `current-refresh-after-stale.log` retain this separate post-remediation proof.
- After all full gates, both built stock and Temporal guests still compare equal
  to the renewed independent proof, and Temporal equals the committed asset.

The two original static quality audits and two timing-fixture delta audits found
zero findings. Their complete reports are preserved in `task-2-quality-audits.md`.
They are same-family static assessments, not substitutes for formal review.

## Final worker verification after SHIP

Classification required FULL (`.gitattributes` is unmatched). Actual default and
required-backend gates both passed again, exit 0, with Temporal host durations
9.795 s and 9.822 s. Actual final `make model-time` passed, exit 0: full gate
85.295 s, above the unchanged 30 s soft target; inner loop 1.648 s, below 10 s.
Freshness took 33.026 s, native tests 24.218 s, guest builds 14.538 s and Temporal
Go tests 9.941 s. Logs/statuses are `verify-model-check.*`,
`verify-model-check-required.*` and `verify-model-time.*` under `.flow/tmp/fn2302/`.

Changed-Go formatting, integration-tag root lint, whole-changed-file host lint,
host/integration vet, script syntax and diff checks passed again. After all gates,
the built stock/Temporal outputs still compare byte-identically with the renewed
proof, and Temporal equals the committed embedded asset. There are no surviving
task-owned commands. The isolated metadata worktree and branch were normally
retired after archival; proof clones/targets and actual report snapshot remain
retained for parent completion review. Latest disk observation: 61 GiB free.

All three receipt-writing attempts correctly refused foreign milestone dirt
(`NO_RECEIPT`); none was represented as a green reusable receipt. CLI configuration
confirmed plan sync false, tracker disabled with no linked tracker, and QA mode
unset. No external lifecycle action or parent completion is performed by this task.

Changed-Go formatting, integration-tag root lint, scoped whole-changed-file host
lint and both host/integration vet passed. The earlier unfiltered host lint found
inherited issues in untouched annotations/probes production files; it is not
represented as green. The newly touched Go surfaces have no lint waiver.

Gate receipt checks decline reuse because foreign `MILESTONES.md` changes remain
present. Actual complete commands run; no dirt is hidden and no clean receipt is
manufactured. Foreign milestone/fn-233 files remain excluded from every commit.
The broad legacy Testpilot umbrella's seven previously documented failures are
not a passing full-suite claim; its failure accounting and unrelated IR bridge
behavior remain unchanged.
