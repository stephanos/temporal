# Native World checkpoint verification

Task: fn-227-parameterized-models-declared-worlds.1. Implementation base: a76c3581620c39d8ccb760bc4af34d4c24b1d618. First frozen executable source: 9147fbdf6b81869bc8141ac120458509ac5f30b3.

The fresh pre-edit baseline was green for the literal Quick command and all three full gates; no handoff or green receipt substituted for it. Baseline logs are `.flow/tmp/fn2271-baseline-{quick,model-check,backends,time,artifacts}.log`. The baseline timing gate was 86.046s (soft 30s missed), with inner loop 1.596s (soft 10s met).

## Frozen observations before formal review

This first frozen gate record is historical after the formal NEEDS_WORK fixes. It is not substituted for post-review-fix verification.

Every command below completed with exit 0 against the frozen source. Build commands were serialized with `CARGO_BUILD_JOBS=4`, `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo`, and `GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go`; suites used foreground `timeout 600` calls and captured output. Green means the observed process exit, not a matched output line.

| Working directory | Command | Raw log under `.flow/tmp/` |
| --- | --- | --- |
| model | `mise exec -- cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-suite` | fn2271-verified-quick.log |
| model | `cargo clippy -p umpire-dsl-macros -p umpire-dsl -p umpire-suite --all-targets --all-features -- -D warnings` | fn2271-final-clippy.log |
| model | `cargo fmt --all --check` | terminal observation before final clippy |
| repository | `make model-check` | fn2271-verified-model-check.log |
| repository | `UMPIRE_BACKENDS=require make model-check` | fn2271-verified-backends.log |
| repository | `make model-time` | fn2271-verified-model-time.log |

Final timing: full gate **97.466s**, exceeding its **30s soft target**; inner edit/test loop **3.009s**, meeting its **10s soft target**. Timing exited 0, but the full-loop soft target remains missed. Its default-backend skip is not TLC execution evidence; the separate required-backend gate above is that proof.

All 571 pinned original artifacts and all 101 original UI stderr files pass fresh SHA-256 checks. Git comparisons against every original manifest path also pass, including metadata; no original golden, expansion, or stderr was changed. New expansion/UI artifacts are additive. `git diff --check` passes.

The actual guest freshness builder ran in the full gates. The embedded guest is unchanged: 3,329,990 bytes, SHA-256 `10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06`. The existing gaps golden is unchanged: 879,197 bytes, SHA-256 `a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0`. No regenerated guest or new live slice/gaps claim is made; the conditional changed-guest live gate did not activate.

## Controls and scope

`world`, `declared_world`, `world_dispatch`, and `generic_world` exercise conditional value derives, the selected typed World through each adapter/dispatch branch, computed starts, actual generic Machine well-formedness, renamed binders, impl-only bounds and type/const defaults. New trybuild pairs pin duplicate/mutable/misplaced World and conflicting starts, including empty static ranges. Existing diagnostics and snapshots are unchanged.

Suite `declared_world`, `explicit_world`, and `retained` exercise correlated ordered/deduplicated and empty starts, selected-world query/Run ownership after the source variable leaves scope, mismatched-world typing, non-Clone/non-Debug World storage, non-Clone outcomes, old factory laziness, unbounded legacy configuration/From/Clone/Send/Sync, and existing query generic argument counts. Algorithms are shared with the existing checker/suite paths.

RED/GREEN logs and the two audit axes are indexed in `task-1-audits.md`. Failed green-named intermediate attempts remain failures; only their specifically identified succeeding logs count. Earlier c3/b5 full gates predate the final fixes and are not substituted for final evidence. The first attempted post-check using unavailable `/usr/bin/time` was inconclusive (rc 127), followed by a proper foreground run.

This delivers native R1 and the native typed handoff portion of R3 only. Source lifting, compiler-checked private-field projection, the shared native/TLC fixture and parent completion remain assigned downstream. Authored field privacy is preserved; no reflection or duplicate field catalog was introduced.

## Raw log SHA-256

```text
50cfc603ef2a4a212374bba590b847c0f675c027ec5c1c0bc6731cc3b3aee890  .flow/tmp/fn2271-verified-quick.log
80c88484f3ed055c08d88d28f59dfb7410f836dc84c40e256611632c6a434a3e  .flow/tmp/fn2271-final-clippy.log
0cad19a919a9c562b2d8c4cb8371a16da8fe51e742501ae636586ca31c7f15f3  .flow/tmp/fn2271-verified-model-check.log
e842aba75a8f56bc8e5d90847fe1787041e0e9a18cbd4cb69910439dc732d248  .flow/tmp/fn2271-verified-backends.log
699ccfa6967b4def2b8d7ebf32401c236dbb0562abc71b97a8e790dd8fb5d316  .flow/tmp/fn2271-verified-model-time.log
d49edfca665f5b5f0c2fb7a4583bc18317b01141f6455a6d382f5b01ca67084a  .flow/tmp/fn2271-verified-artifacts.log
be2e44ec799527d1354524030d60b8654e1539e896de8158673cf69c620a2c19  .flow/tmp/fn2271-verified-ui.log
b77af37d1190b1d75f81d501a1f3857858ec004af6754b0270590340c1e2b905  .flow/tmp/fn2271-default-generics-red.log
9f0311423412433b94427e09172049fd6412fd5708a54251ac4770d5fe786a92  .flow/tmp/fn2271-default-generics-green3.log
```

## Final post-SHIP verification

Executable source remains f0adb5afdb659614cf418a7f420106a85718ba06. The resumed reviewer returned SHIP at 2026-10-21T15:48:14.374964Z with all four findings fixed. The permanent receipt and verbatim report are `task-1-review-round2-receipt.json` and `task-1-review-round2.md`. The review used gpt-6.1-sol/high in read-only mode; the worker's commands below provide the fresh suite evidence.

The canonical Quick command passed after the fixes (`fn2271-round2-quick2.log`, rc 0). Final formatting and strict scoped clippy passed (`fn2271-finalize-clippy.log`, rc 0). All three full gates ran again after SHIP and exited 0. Their final logs are `fn2271-finalize-model-check.log`, `fn2271-finalize-backends.log` and `fn2271-finalize-model-time.log`. The required-backend run measured 182.436s wall time. Final timing is **93.288s gate total**, missing the **30s soft target**, and **5.035s inner loop**, meeting the **10s soft target**. These numbers supersede the historical timing above. Timing's default TLC skip remains distinct from the successful required-backend run.

Normal receipts for model-check, model-backends and model-time were written at 155a7b29, then all three `gate check` commands returned HONORED with exit 0. No full gate was skipped or waived; the checks validated the receipts after the observed runs. Earlier dirty foreign documents caused RUN, and no dirty-receipt bypass occurred.

Root-owned commits bbf0835fbf3f536a954adae8dca3ef19795627d4 and 155a7b295990f457cd85ec9e77ecfd18a1cc96eb are interleaved metadata/evidence work, not native implementation. The first records the user's three approved parent/later-task contracts. The second integrates separately verified fn232 evidence and documentation. Neither changes this task's executable source. Foreign fn233 files remain untouched and untracked.

Final SHA-256 and Git path/metadata comparisons pass for all original 571 artifacts and 101 UI diagnostics. The original manifest hash is `5799233527eff38f1a342d5189436b3f2d4346217244aa5afd5ff47052412d9d`; the UI manifest is `206d97b91916171401117faff6a9bc9565a76d24c536cd3348cc719164cf3d92`. Guest/gap hashes and sizes remain exactly as recorded above. No new live or cross-OS identity claim is made.

Both task-owned code lanes were integrated and removed normally with patch backups retained under `.flow/tmp/`. Root received an explicit cleanup sublease for only the regenerable Cargo `debug/incremental` directory after all builds stopped. Root preserved dependencies, build products, proof logs, guest/Go data and foreign proof. Its capacity record is `.flow/tmp/fn2271-incremental-cleanup.md`; final free space was 34GiB. All worker-started commands are terminal.

```text
72ff6baed76a0a251b384639ca89035609368c9a68eb9b8b11fcc266a14b60e9  .flow/tmp/fn2271-round2-quick2.log
f591e097f22f91dd5c31fbbf7e587d80c0a4e0ed594f9c9012a544f38c52d926  .flow/tmp/fn2271-finalize-clippy.log
43dceaefefd1ea00d7bcfb3531b4a240f549164650b4f754ca02931ab051c6b8  .flow/tmp/fn2271-finalize-model-check.log
3e383aefa72bdb66d7dbb198445d477c7f837e0655ed38b0b730981377a2a59b  .flow/tmp/fn2271-finalize-backends.log
4d247cf86eec1da8fcca3726eac35a48975dbfaa1047c7260f3e477de20b099f  .flow/tmp/fn2271-finalize-model-time.log
d49edfca665f5b5f0c2fb7a4583bc18317b01141f6455a6d382f5b01ca67084a  .flow/tmp/fn2271-finalize-artifacts.log
be2e44ec799527d1354524030d60b8654e1539e896de8158673cf69c620a2c19  .flow/tmp/fn2271-finalize-ui.log
ac685fedaa1857b30773d404e10ca6d33949d3005c21633d1dc4295aece8c4e0  task-1-review-round2-receipt.json
```
