# fn232 minimizer custody after fn231 closure

fn231 closed at `4e1a90755e`. Open fn232 owns the following original source
history and external evidence at their existing paths. No evidence was copied,
moved or deleted; actual artifact_path bindings remain valid. This branch uses
an `ours` merge for archival reachability only, preserving the closure source
tree; it must not be mistaken for replaying the obsolete delegate source.

## Source reachability

Branch `fn232-minimizer-source-custody` holds archival merge
`aa322850d09969110c358a4f6cb20dce3f9c7237` and all these original commits,
verified with `git merge-base --is-ancestor` and branch containment before
worktree removal:

- `6506164bdaf5348a004404cf4aad9ff5c3e11ee2` (type-sensitive display check)
- `6e299e88ba9b0b434444e776d2158b0b19796b47` (display coverage)
- `7d9a371d492c0bb06c915dc6931a147c08b5df6d` (optional provenance fix)
- `a1e8891f4c290b77d76bf6009add452f07096e68` (helper cleanup)
- `54747bf0eab38f05b27352bfe24c322e30ff8f04` (sparse uint fix)
- `4c9bdad25bf4d64eb6a80e80c6898649f6a5108f` (original feature/cache history)
- `87f2da8315215b41e4df0a9c44cefb2d51856711` (dirty proposals snapshot committed)
- `670b533ecbbe2bf61b179a5a6dd053dde85fdc02` (dirty provenance snapshot committed)
- `e5ea438c90e381cf1d49b126ce18b0bdc32d2ce3` (both snapshot parents)

Cleanup covers only `/home/agent/wt/fn231-proposals`, `fn231-provenance`,
`fn231-rust-minimizer`, `fn231-decoder-diagnostic` and
`fn231-semantic-diagnostic` under that same parent. All are clean and held by
this open branch before removal. No worker retains a target lease. A /proc
scan checks cwd, exe, maps and fd references before removal. The closed branch
`fn231-rust-minimizer` is removed only after containment verification; the two
snapshot branches may be removed after their commits are held here. Root main
and unrelated .turbo/user files are unchanged by this cleanup.

## Evidence retained at exact existing paths

Under `/home/agent/wt/evidence/`, fn232 takes custody of:

- `fn231-public-provenancefix-01/` and `fn231-public-provenancefix-02/` in full:
  accepted scope55 + semantic119 each, raw retained artifacts, terminal/source
  receipts and `-02/two-run-comparison.json`. Display offline authentication
  still follows the actual artifact_path values in these roots.
- `fn232-integrated-provenancefix-01/` in full: immutable a07dd6e2 host and
  build/source receipts (compiled source8536a84a85).
- `fn231-public-modulefix/` in full: authenticated actual module for uint
  causal diagnosis, including the scope-01 campaign execution6334a9aa/context
  00389332 target/module.wasm referenced by the external diagnostic source.
- `fn231-provenance-host-red-02/`, `fn231-provenance-host-green-01/` and
  `fn231-public-storagewait-01/` in full: genuine-parent causal records and
  input chain, including `minimize-01/rust-input.json`.

Retain `/home/agent/evidence/fn231-uint-retained-diagnostic-test.rs`,
`fn231-uint-retained-red-2.log`, `fn231-uint-retained-green.log`,
`fn231-uint-symbolize-suite.log`, `fn231-uint-metadata-suite.log`;
`fn231-provenance-genuine-green-test.rs`, `fn231-provenance-host-red-2.log`,
`fn231-provenance-host-green.log`, `fn231-provenance-minimize-tests-final.log`,
`fn231-provenance-replay-tests.log`, `fn231-provenance-clippy-final.log`.
The existing fn231 handover records exact commands and causal counters. Keep
its other raw review/RED/GREEN logs pending root's separate evidence disposition.

Shared fn232 inputs remain unchanged:
`/home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a`,
`/home/agent/wt/fn232-go-source/temporal-1d7b37f00a`,
`/home/agent/wt/evidence/fn232-replay/stable-reference2/replay-input.json`,
`/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go`,
that `.build-cache/fn232-refresh-minimize-go-cache` and
`.build-cache/fn232-refresh-minimize-preparation`, and
`/home/agent/.cache/go-build-local`. Preparation caches are regenerable; no
cache cleanup is authorized by this transfer.

## Separate evidence deletion proposal; nothing deleted

Candidate obsolete roots under `/home/agent/wt/evidence/`:
`fn231-integrated-fixture-00a97c42`, `fn231-minimize-focused-01`,
`fn231-provenance-controlled-01`, `fn231-provenance-host-red-01`,
`fn231-public-00a97c42`, `fn231-public-poolfix-01`.
A read-only scan of retained proof trees' text receipts/source/logs (recognized
extensions, files at most10MB) and symlinks found one cross-reference:
`fn231-public-modulefix/preparation.json` names
`fn231-public-00a97c42` as `previous_attempt`. It records historical preparation
rather than an artifact input, but root must decide whether to retain that
history before deleting it. This bounded scan is not a proof that arbitrary
binary or larger files contain no path strings. Preserve all candidates until
root's final dependency check and deletion authorization. No committed portable
test requires these external roots; actual forensic reproductions do.

## Obsolete evidence cleanup after complete dependency scan

Root scanned every byte of7833 files (2545170984B, including binary and >10MB payloads) and every symlink in the seven retained accepted/causal/build roots against the five obsolete candidate absolute paths; no references found. A fresh /proc cwd/exe/maps/environment/fd scan found no live references. Deleted exactly /home/agent/wt/evidence/{fn231-integrated-fixture-00a97c42,fn231-minimize-focused-01,fn231-provenance-controlled-01,fn231-provenance-host-red-01,fn231-public-poolfix-01},324828124B. These are superseded closed-spec outputs, not accepted or receiving inputs; required raw proofs, shared Go inputs and original source history remain. Detailed scan/removal receipt: /home/agent/evidence/fn232-obsolete-minimizer-evidence-cleanup.json. Kept fn231-public-00a97c42 explicitly in open fn232 causal custody because retained public-modulefix/preparation.json cites it as previous_attempt; no immutable receipt rewritten. Other earlier-listed cleanup candidates are dispositions, not runtime passes.
