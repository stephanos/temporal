# Task9 source checkpoint review

Checkpoint-safe as a source-progress commit: **Yes, with the final affected-default RED retained explicitly.** The inspected candidate has no identified new source correctness, design, security or scope defect. Verified progress covers the reviewed source corrections and inspected focused checks; the final default gate is not GREEN. This is an independent progress review, not task acceptance, formal SHIP, full R18 preservation or native qualification.

The requested reviewer is `gpt-6.1-sol/high`, from the same GPT family as the writer. This fresh reviewer read AGENTS.md, the complete Gomad README, milestone ordering, task9's retained requirements and the incremental conductor admission. No Go, build, test, lint, generation or bridged command ran in this review.

## Binding and scope

HEAD and diff base are `29c80199cdf1a7444f3a3aa99388e2f408e3cf73`. The selected manifest SHA-256 is `0cc145589ba5b275ce6b66312e8d301ba02539353cd4d3d29312cdbef7b6f6b9`. Its bytes hash to that identity, and all 1,042 declared inputs match the checkout. The inspected admission hashes to `df6d86a0f86a3db821ad18b32a54df1130690213ea703f929b47a2bd51741e57`.

The review covers the seven tracked Gomad diff files and all four new focused test files. It compares original query, download, build, standard-inventory and capability-list boundaries with coherent Git `d635e23f00d926a43b942f25a9d05bd0ccb72025`. The separately introduced adapter helper has the explicitly corrected whole-graph baseline `0c2e091b73325368b22ec142e76da5ed5ab4cad2`; no d635 adapter proof is inferred from its failed compilation.

## Finding

- **P2, final-gate verification issue.** `tools/gomad3/target/internal/gocommand/command_test.go:410` and `:430`. The frozen final default-command receipt exits 1 with 90 named passes and one named failure, `TestStructuredCancellationRemovesDescendant`, reporting `descendant  survived cancellation`. Its receipt SHA-256 is `2e6a6c5cb7b19a24e5b1c87343e6c74308fbc500ca5bf5e64a1b189fd8518085`; both recorded stream hashes match. File-existence acknowledgement can cancel after shell redirection creates the PID file but before printf fills it. `processAlive` at line 463 treats an empty, unparsable PID as alive. This supports a fixture-race explanation and does not prove a surviving descendant, but that explanation alone does not turn the final RED green. The same frozen candidate's focused receipt passes this test. The sole execution owner must retain explicit reconciliation before claiming default-gate acceptance. Do not alter the original fixture, waive cleanup proof, or count an earlier differently bound run as the final default gate. The conductor reports one exact matched HEAD baseline control pending after the final batch; this reviewer has not inspected its outcome.

## Source assessment

Removing exactly the added Request field and three-line opt-in stderr assignment restores both hostexec files byte-for-byte to the review base. All four pipe defers, explicit closes, capture collection and default behavior remain unchanged. Both child descriptors share the existing stdout pipe only when requested. The actual combined capture retains ordering, byte counts and hashes.

Existing Structured, Diagnostic, Compatibility and execute bodies are byte-identical to base. StructuredCommand refuses operation errors, stdout-first overflow, stderr overflow and watchdog before exposing stdout; complete bounded raw outcomes reach each owning original decoder. OutputError changes only the same direct actual exec.ExitError's detached Stderr, preserving its ProcessState and exact 32 KiB prefix, decimal omission marker and 32 KiB suffix above 64 KiB. Download and List keep their original explicit-stderr treatment. List keeps ctx.Err override on any failure. Standard inventory restores literal argv and inherited cwd. Compiler handling releases the real cache lock before projecting either failure; its legacy fallback applies only when raw CommandError is absent. Preparation restores the original Background identity-query timing. No public API, dependency, pin, native input or existing test/comment changes appear in this diff.

The new tests cover real ordered bounded capture, actual raw process errors, error-object identity and detached stderr, boundary sizes, decoder/context precedence and real cache-lock reacquisition. The corrected helper receipt's raw events independently confirm 55 named passes, zero failures and zero skips. Frozen listing and standard-stderr receipts have 23 and five named passes respectively with matching streams. Those characterization passes record observations; they do not themselves assert original/current equivalence.

## Acceptance remains open

Public review's two oversized-stream outcomes still change the original generic overflow text/type to typed per-stream OverflowError (`target/internal/capabilityreview/list.go:110`, `target/internal/gocommand/command.go:77`). The admission explicitly leaves that R18 question unresolved. Standard inventory's new complete-data bound is required by R10 and does not grant a public-review preservation exception.

Task40's genuine close-fault/dependency acceptance, the unchanged ST1005 sites at `target/internal/build/context.go:40` and `:59`, and retained upgrade-publication RED attribution remain open. Native execution stays deferred and unverified with fn128/fn149. This report grants no waiver, completion, push, PR or CI authority.

All reviewer-owned command handles terminated with returned exit status. The reviewer launched no persistent process or child execution gate and changed only this new report. The worker's shared execution lane and final handover remain the conductor's responsibility. Both user-owned .turbo files remain untouched by this reviewer. Stop after the task9 checkpoint/handover; no following task is admitted.
