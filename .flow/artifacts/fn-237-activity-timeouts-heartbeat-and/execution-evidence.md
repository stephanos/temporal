# Optional handler facts checkpoint F

Task: `fn-237-activity-timeouts-heartbeat-and.1`. This checkpoint supplies the bounded fact-result interface, independently of stored optional values and Activity policy.

Tier: session jev-unavailable(no_key), explicit AGENTS implementer gpt-6.1-sol at medium / reviewer gpt-6.1-sol at high. The explicit Codex review uses the same model family as implementation.

The pre-edit base is `8ca8756c653c40ad8aa945ee480bd58262bc2132`. The focused baseline from `model/` was `mise exec -- cargo test -p umpire-dsl-macros -p umpire-front -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures`: exit 0, 157 passing tests across reported suites; raw log `.flow/tmp/fn2371-baseline.log`. Doc-test groups collecting zero tests are separate from those executed suites.

The shared interface checkpoint `a7e9bfcad8` passed `mise exec -- cargo check -p umpire-fixtures --tests` (exit 0). Declaration metadata is `Method.fact` / `Machine.fact`; the admitted tree carries `Lifted.fact_type` and typed sequence `Branch.facts`; the decode-only hidden bridge defaults to a fail-closed panic. Legacy branches initially use Unit and empty sequences.

`preexisting-artifact-manifest.json` preserves all 799 original artifact hashes and complete byte sizes, including snapshot metadata. The supplemental frozen-original audit covers all 15 retained Activity artifacts, including `gaps.txt` outside the 799-file scope. Initial comparisons match the manifest base, original historical authority, committed HEAD and working bytes.

Red controls before production fixes:

- Lane A: optional macro handler and independent source both refused with exit 101 at the unsupported return. Raw logs `.flow/tmp/fn2371-a-macro-red.log` and `.flow/tmp/fn2371-a-source-red.log`.
- Lane B: two intended failures (forged facts discarded, disabled-handler fact-tail arithmetic skipped), exit 101. Raw log `.flow/tmp/fn2371-b-red.log`.
- Lane C: source fact positives refused at handler return, exit 101. Raw log `.flow/tmp/fn2371-c-red.log`.
- Lane D: new generic fixture and replay source refused at the optional handler result, exit 101. Raw logs `.flow/tmp/fn2371-d-initial-red.log` and `.flow/tmp/fn2371-d-replay-red.log`.

The integrated focused six-crate command passed (`fn2371-focused-green.log`). New source controls also explicitly preserve legacy `Seq<Unit>` metadata in both emitter paths. Isolated generation rendered only `optional_facts` into a temporary expected tree, then copied its new snapshots; the ordinary fixture runner validates them. The Model has 7 projected states and 31 edges, and required TLC has 57 distinct observer states (248 generated) in its complete exploration. Five separately refuted properties force A, B, C, D and clearing witnesses; raw TLC traces are retained in `tlc-witnesses.txt`.

The fresh correctness audit identified acceptance of noncanonical absolute short `::Option` constructors. The added source refusal table failed on `::Option::Some(Fact::A)` before the fix (`fn2371-absolute-option-red.log`), then passed after restricting absolute paths to core/std identities. Canonical constructor paths, prefix SSA, both emitters, ordinary clearing and exact legacy metadata now pass 9 source tests (`fn2371-source-facts-green.log`). The independent standards audit reported no findings; both raw reports are retained in `quality-audits.md`.

Required-mode fixture exploration and its selected-World/source-mutant controls passed in the first integrated required command, whose overall exit was 101 because the new private replay control needed a declared World and native action Display identities. Those test setup errors were corrected. `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-testing --lib` then passed all 8 tests, including actual four-value TLC replay, omitted/duplicate/other-valid/ordinary-extra vector mutations, missing observers, source-only fact substitution, unknown/wrong-kind decoding and nonempty observer preservation (`fn2371-replay-green.log`).
