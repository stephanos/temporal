# fn233 interception candidate, 2026-10-23

The existing `fn233-interception` branch contains the implementation; no replacement worktree was created. Candidate `3e78dfa6ce749618a6322bf5affb93f66b5bc5bf` merges reviewed root `81a66b54b3` through `9fd36aedd8`. The runtime evidence below binds immutable ordinary/fixture executables built from **9fd36aedd8**, before the oracle-only follow-up. No final Rust qualification or Flow lifecycle completion is claimed.

## Source and review

New receiving fixes after initial `29f6a6f39f`: `97b2016708` preserves host routing for versioned `go run` operands, `0f457d19b0` preserves fn233 across helper/layout integration, `ca998081e9` fixes only four effective coverage diagnostics, `b26e49f484` admits actual registered cached `go run` executables, and `3e78dfa6ce` retains the real launcher golden and correct partial-artifact oracle expectations. Root integration merge commits are `8eb570d727`, `e25044f746`, and `9fd36aedd8`.

Cached admission remains restricted to the original registered Go process/invocation/compiler/argv and `go run`. The private 0700 GOTMPDIR remains pinned. Its optional pinned-compiler `go env -json GOCACHE` query uses the original deadline and a 16384-byte output bound. Cached modules must match the captured WASM bytes' SHA address, exact shard/address/basename layout, ownership, safe modes, no-follow directory/file handles, canonical containment and singleton executable directory. No production artifact permission rule was relaxed. New adversarial/unit tests include legacy private registration encoding and identity/deadline/environment negatives.

Independent same-family reviewer `/root/interception_finish/review_current` returned SHIP for the implementation, receiving merge, exact oracle corrections and input declarations. Its runtime authentication reports are retained below. It did no actual runtime/build/source edits during those evidence reviews.

## Verification

All Rust commands use `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn233-interception-new`, `CARGO_BUILD_JOBS=2`, `cargo +1.99.1`, `--release --offline --locked`, manifest `tools/gomad/rust/Cargo.toml`, package `gomad`.

| Gate | Exit / observed result |
|---|---|
| Cached-admission causal unit before fix | 101, expected original module-association refusal |
| Focused cache/query/boundaries | 0, 25 passed |
| Repair before root merge default / determinism-fixture lib | 0 / 0, 350 / 352 passed, one ignored each |
| Repair strict clippy all-targets default / determinism-fixture | 0 / 0 after local import-order correction |
| Merged 9fd default / determinism-fixture lib | 0 / 0, 403 / 405 passed, one ignored each |
| Merged ordinary / determinism-fixture binary build | 0 / 0 |
| Rust fmt / git diff check | 0 / 0 |
| Merged strict clippy default | 101: nine root stress findings, already fixed upstream by root e325783f6c; no clean combined-clippy claim for frozen 9fd |
| Pure launcher corpus tests | 0, 14 passed |
| Real public six-case corpus | 0 regeneration and 0 golden check, each internally twice, all six cases byte-identical |
| Closed output consumer exact fixture | 0 final reviewed harness: partial bytes authenticate, status3, inspect0, choices and verify-only refusals2, exact no-replay line |
| Synthetic shim cases | 0 for all 60 cases in each of two generations; 120 invocations, 60 byte-identical semantic pairs |

Logs before the current evidence root are retained at `/home/agent/evidence/fn233/{cached-run-*,cache-integration-*}`. Setup mistakes (wrong feature name, missing test import, missing merged fixture field and expected-status typos) remain in their original logs; final logs carry explicit v2/final suffixes.

## Frozen inputs and actual task1 proofs

Evidence root: `/home/agent/evidence/fn233/acceptance-cached-run` (0700). Ordinary `gomad-release` SHA `ffb73504deaecfdff7a89bee84f51890f46de474c4268ab8e85d84a58d8b215b`; fixture `gomad-fixture` SHA `0705862dada8fabbcf3303aa43a4a563b5baf08ade332faf6eee6f4abe0b51a0`. Both immutable private executable copies, independently installed through the genuine public install path. Full exact freeze is `binary-freeze.json`.

Pinned compiler: `/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go`, SHA `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. Every actual command retains original 4GiB / 2^50 fuel / 300s guest / 600s overall / 5s cleanup ceilings. Small qualifier reader baseline preserves its original stricter 60s/180s time bounds.

- **Cached Go run regression:** genuine retained previously-failing WASM SHA `3ee590f1436d512e57b3fac85f8cee53027b68d633599c0be249fddcafee5b41`, 2469178 bytes. Four original real driver modes now return 0,0,1,0 as expected. Their first two runs were both already cached; `cached-run-authentication.json` says so. A separate fresh XDG pair proves actual private GOTMPDIR cold launch then cached launch with different `seed-failure` guest args, both exit0. Before-cache executable list empty; publication occurs during the first run; second uses unchanged cache inode/mtime and the same module bytes. `cold-cached-run-authentication-final.json` SHA `16046d2ea38a723e1ce74d5df22127b5985a021760098dbb8f84fb901be0e435`, accepted/gaps[].
- **Bare manual fixture-role Go proof:** exact shared exporter recipe, full authenticated overlay/tags and hand-written Go `-exec '<fixture> run-guest'`; actual Go386682 → launcher387019; `TestFrontendSystemInfo` PASS and package ok, driver0 in138.734s. Module205879495 bytes SHA `2464f00de3dee433a3c82a84b8a5c8057f006287321feb38808425c76aaa5777`. `manual-authentication-current.json` SHA `0de037cd07c5649c1ebe2d49e42837d97e112edba3e5f33a294cba3f3b657790`, accepted/gaps[]. Exact explicit matching-installation replay0/reproduced=true in4.927s. Inspect0; successful artifact correctly refuses minimize3 because minimization requires a replayable target failure. Separate genuine intercepted failure corpus minimizes0 with sealed origin preserved in all four rounds.
- **Actual concurrent package launchers:** one `go test -p=2`, distinct Frontend/tasktoken package operands and union selector, actual Go427557 → distinct launchers427814/427815, shared driver/wrapper/artifact root and distinct launch identities. Both selected tests PASS, driver0 in14.853s. Process-wide observer unions children of every Go OS thread; 169 samples contain both sealed launcher argv prefixes (peak2). `parallel-observed-authentication.json` SHA `c52095a95b9dd526c59d0b5d0d2912c70d8819238bf6ab494da03470787e283f`, accepted/gaps[]. Two earlier passing p2 generations used an incomplete main-thread-only observer and carry no overlap claim.
- **Ordinary complementary wrapper proof:** genuine ordinary binary + its matching ordinary installation, caller supplies readonly/PGO/trimpath/buildvcs/fulltags/count1/verbose/exact Frontend selector. Caller supplies neither `-exec` nor overlay; the legitimate shim inserts ordinary `run-guest` and authenticated temporal overlay, bound in the retained original and rewritten argv. Go434560 → launcher434721, same captured Frontend module, named PASS/package ok, driver0 in124.032s. `ordinary-manual-authentication.json` SHA `fb92740bdcae3ce7a40caa6a714abc8e6347c0e7bd9bc283e0b39416dc41a871`, accepted/gaps[]. This is wrapper orchestration, not a claim of bare ordinary manual execution. Prior wrong-original-PATH invocation correctly refused2 before guest and remains retained.

## Corpus and conditional replay semantics

`public-corpus-corrected-regenerate` and `public-corpus-golden-check` each run six cases twice. They authenticate every actual module/preparation receipt, complete campaign, sealed origin, record/manifest/journal, exact output stream, seed-prefix and logical byte accounting before semantic projection. First failure skips a later seed separately proven successful; target failure has exact replay argv, replay/inspection and minimization. Qualification probes first accept the unmodified actual Result before transplanting intercepted provenance and requiring refusal.

`launcher-golden-authentication.json` SHA `bb938a0b56b714944a554eef8f9bde2c6b539061d9139467061fa88f699b50c7` independently authenticates 96 command receipts, 24 complete campaigns and both byte-identical golden observations. The committed launcher golden was generated only from these real runs. The original failed `public-corpus-regenerate` is retained: only the exact output-limit1 capacity fixture changes expected replayability to false. It still requires status3/capacity/one attempt/one unexecuted seed/authenticated artifact and exact no-replay diagnostic. Replayable target-failure expectations are unchanged.

`callback-refusal-final` authenticates actual compiled-Go marker delivery before external stdout closure, plus retained failed-delivery bytes, actual infrastructure3, one unexecuted seed, inspect0, no-choice inspection2 and verify-only2 with exact missing-choice incompatibility. It claims no exact reproduction of the external consumer failure. Earlier oracle-failed callback roots remain untouched. No production trace or replay availability was fabricated.

Both supplied historical qualification Results were rejected by the current reader before tampering; those baseline failures are preserved. Root authorized a fresh **provisional**, actual original small fixture recipe, two repeats and matching ordinary installation solely as an authentic reader boundary baseline. `qualify-reader-baseline`0 and `report-baseline-current`0 produce/result-read `report-baseline-fixture/rust-qualifications/v1/result-7b03ae71229292007ae0d0d3c72e8d86dd47f910c69728eda7d6490740aa7c8e.json`. This supplies no final support/qualification claim. `manual-reader-verbs` also proves inspect0, appropriate success-minimize3, report baseline0 then intercepted provenance3, and source-only qualify3.

## Remaining ownership and preservation

All scoped task1 actual commands are terminal. `manual-hidden-replay` exits0 in110.352s with `reproduced=true`, `choice-replay=exact`. `parallel-hidden-replays` exits0 in123.678s: the exact observed parallel Frontend artifact replays0 in111.073s and Worker artifact replays0 in12.552s, both reproduced=true/exact. The private user/PID/network namespace proves all16 build-input locations unavailable: source/compiler roots, Go caches, both preparation roots, selected compile cache, all invocation roots and both installation files. The retained binary and authenticated artifact/target-pool bytes remain accessible. Full mount/command argv and the checks are retained in `hidden-replay-argv.json`, `parallel-hidden-replay-argv.json`, `hidden-inputs-proof/` and `parallel-hidden-inputs-proof/`.

Independent `hidden-replay-authentication.json` SHA `2e220fac198ecd75725395f27eff2943c2a95f5a4b7094c2fcfe6d7fc8934544` accepts all three artifact bindings and namespace proofs, gaps[]. The input-absence checks run before replay and the harness restores no inputs; no post-replay filesystem write audit is claimed.

These retained-only explicit replay commands omit the optional `--installation` selector deliberately. Current replay validates an explicitly selected installation and uses its operator compile cache; hiding that selected file while retaining the flag would be a valid missing-input refusal. The separate installed manual replay above proves the matching explicit installation path; these three namespace runs prove retained-only replay without installation/source/compiler/cache inputs. No claim is made that the installed replay argv worked with its selected file absent.

Root/conductor retains final source freeze, all-feature Rust gates and final Rust qualification. Tasks2/3 still need their 21 real shim cases/two regeneration/full81-case golden, genuine no-explicit-install Frontend success and final coherent-candidate Make wrapper gates. Ten reviewed real input declarations are prepared under `real-inputs/` but were not executed as the 21-case shim corpus. Current task1 ordinary complement is additional real wrapper evidence, not a blanket task2/3 close.

Earlier unchanged-Make six stages/two regenerations (12 accepted receipts, six actual Frontend campaigns) and the original manual proof remain at `/home/agent/evidence/fn233/acceptance-current`, bound to ordinary b12dfeac… / fixture ee1d6203… from0f457d19b0. They are historical scoped acceptance, not acceptance of9fd or a future final source. Root explicitly defers another full Make batch until the coherent global freeze. Root Makefile remains unchanged by fn233.

Keep worktree `/home/agent/wt/fn233-interception` on its branch because fn233 is open and this implementation is not yet integrated; preserve assigned target `/home/agent/.cache/codex-build/fn233-interception-new` until conductor releases it. Keep current and prior fn233 evidence roots, including failed attempts, for open acceptance and causal evidence. `worktrees-at-checkpoint.txt` records the shared registry; other worktrees/targets remain under their respective owners and were not deleted. No pushes, history rewrite or Flow lifecycle transitions were performed.
