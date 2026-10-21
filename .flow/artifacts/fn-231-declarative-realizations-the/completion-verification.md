# Conductor verification and closure

All three full conductor commands terminated with exit 0 after the parent SHIP,
against commit 55574918ac304158bd5ec0f3c8f2f4275a31d013. Adjacent conductor
receipts preserve their actual commit, command hashes and timestamps; they do
not certify later source edits. No source changed during these runs or closure.

| Command | Terminal session | Retained log under .flow/tmp |
| --- | --- | --- |
| `timeout 600 make model-check` | 44884 | fn231-conductor-model-check.log |
| `UMPIRE_BACKENDS=require timeout 600 make model-check` | 98373 | fn231-conductor-model-check-required.log |
| `timeout 600 make model-time` | 48155 | fn231-conductor-model-time.log |

Each command used CARGO_BUILD_JOBS=4,
CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo and
GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go. The cache remains active
and protected. Required-backend verification ran the admitted TLC controls;
unsupported activity source is not claimed as TLC agreement.

Timing was 87.140s against the 30s soft target, with an inner edit/test loop of
1.672s against 10s. Freshness took 33.840s, tests 24.825s, guest build 14.772s
and Temporal Go host 10.057s. The soft overrun remains visible, not waived.

The final guest SHA256 is
10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06;
the activity gap report remains
a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0.
Task verification retains the full 2,336-case corpus and privacy proof. These
conductor gates supplement rather than replace the task live runs and controls.

Normal `flowctl spec close fn-231` succeeded at 2026-10-21T13:15:37.306743Z.
A subsequent authoritative show reports the parent and both tasks done, parent
completion review ship; validation reports no errors. Closure is local only.
No push, PR or remote merge was performed. Foreign fn233 plans are untouched.
Scratch retirement has not yet occurred; raw logs remain at the paths above.
