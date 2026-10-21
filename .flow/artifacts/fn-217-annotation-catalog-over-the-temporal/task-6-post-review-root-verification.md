# Post-review verification at 06caf99b

Source checkpoint: `06caf99bcc0614463c4316db7414eb0f184ae71d`.
The real formal-review Number finding is fixed, with an actual seed-3 RED/GREEN
regression, fractional-interval controls and impossible zero-only refusal.
Ordinary Number generation keeps its previous candidates, order and RNG behavior.
The two guard findings were refuted against the actual locked syn 3.0.9 API
and a freshly compiled exact negative control (one test passed).
The delivered first-round NEEDS_WORK remains recorded; no review reset occurred.

Root ran both complete gates with `CARGO_BUILD_JOBS=4`, Rust target
`/home/agent/.cache/codex-build/fn2177-rename` and Go cache
`/home/agent/.cache/codex-build/fn2177-go`:

- `make model-check`: original session 82482, terminal exit 0;
  generic Go host 1.081s, Temporal Go host 79.163s. Default TLC skip is explicit.
- `UMPIRE_BACKENDS=require make model-check`: original session 62470,
  terminal exit 0; generic Go host 1.011s, Temporal Go host 63.375s.
  Real-TLC-required banner present.

Logs: `.flow/tmp/fn2176-post-review-root-default.log` and
`.flow/tmp/fn2176-post-review-root-required.log`.
These are actual complete runs, not receipt-reused checks. Dirty MILESTONES.md
prevents clean-worktree receipts; foreign plans and milestone edits were preserved.

Worker full live gap session 52071 exited 0 in 563.085s. Root inspected the
complete regular log `.flow/tmp/fn2176-post-review-final-live.log`, both passing
top-level tests including privacy, and the immutable actual report
`.flow/tmp/fn2176-post-review-final-report.txt`.
Root matched all 2,317 strict/generic report identities to final PASS subtests:
zero missing, 2,336 PASS subtests including the boundary controls.
There are 79 strict confirmations, 754 tested generic observations, 1,484 local
untested outcomes, and 217 explicitly retained dated Unannotated findings.
Worker-reported inventory/request/response sizes are 692785/503633/879201,
below the unchanged 1 MiB bound.

Root independently hashed the actual rebuilt Temporal Wasm:
`c2d2eb6af8bdabacdc13ae31e6237e39d74d3b3bd753a145ae540c73c78be46c`.
Actual report and committed activity gap golden both hash to
`a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0`.
The changed guest was tested afresh; pre-review passes were not substituted.

All root build commands are terminal. Renewed timing and the single resumed
formal implementation review remain separate gates; this artifact claims neither
SHIP nor task/spec completion.
