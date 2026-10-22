# fn-232.10 qualification closure evidence

Candidate: `49ea645940c9df09ad2c84ce20bc18889f75f40f` plus the task-owned source patch retained at `/home/agent/wt/evidence/fn232-qualify/closure/candidate.patch` (SHA-256 `4ce94721949f46545ce53e5750f81f62cfc803f90ede816703df5e53f26d5fe7`). The integrated .12 generated/model source was present before building. The later .12 completion commit changed Flow/milestone records, not the compiled implementation inputs. Root froze source and top-level documents before these final lanes. This behavioral/golden evidence is scoped to that candidate and those cell/profile/limit selections; fn-232.13 owns ordinary-release whole-candidate qualification.

Host: Linux/aarch64. Rust `1.99.1 (58cf8f9c3 2026-10-13)`; Go `go1.27.1 linux/arm64`. Immutable binaries and their build logs are under `/home/agent/wt/evidence/fn232-qualify/closure/`:

| Binary | SHA-256 |
| --- | --- |
| `gomad-fixture` | `c82a1f44a49464c7933fe4aff1a0614f4e403c7660ce7d4cf50299a601eb1842` |
| `gomad-release` | `3ee1e77bc8b9ac67c3abd72f1b8f9265e7528a12952ba50194aad7dff49f861f` |

## Scope and requirement mapping

R9 preserves the existing WASM refusal of `qualify-set`, `merge-set` and `compare-support`. The live Go WASM dispatcher never implements these verbs, and its capability dispatcher refuses them before installation resolution. The generic/native help fixtures do not establish their WASM availability. No native port was added. Each command is exercised bare, with help and with its former generic arguments, using default and missing installations: 18 cases, each twice on both binaries. The reviewed projection retains status 2, unsupported-command classification and empty stdout; raw stderr/stdout hashes remain in command receipts.

Review found a genuine grammar difference: Go accepts final `--` after `--spec FILE` or `--result FILE`; Rust rejected it. The Rust parser now accepts only a final terminator, retaining positional refusal after it. Both lineages' grammar probes cover this. Successful fixture/canary qualification invocations use the terminator; other cells retain the original spelling. Report checks compare both spellings byte-for-byte and reject `-- extra`.

R13 uses actual fresh Go/Rust public qualification/report executions. Small cells use the frozen Go 294 reference; onebox uses the separately declared d414f0 Go producer and current guest source, as in the accepted v17 lane. Cluster retains Go's actual public qualification refusal and derives its verdict from repeated public scenario exploration and replay. No positive Go cluster-qualify availability is claimed.

R14 runs two independent fresh provisional Results and two independent fresh canonical Results for each cell. Each Result executes the unchanged repeat-2 specification and replays every retained execution. The first golden comes from attempt 0; the second is independently projected from attempt 1 and must be byte-identical. Original records, hashes and stream bytes remain exact; only the existing declared qualification projection excludes variable measurements and tokenizes declared operator paths. Regeneration is forwarded only to the child Python harness, then stripped from guest/operator command environments.

## Fresh commands and observed exits

Commands below run from the Temporal checkout; `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-qualify-closure` is isolated from other assigned targets.

| Command | Exit / evidence |
| --- | --- |
| `cargo +1.99.1 build --release --locked --manifest-path tools/gomad/rust/Cargo.toml --features determinism-fixture` | 0; `closure/build-final-fixture.log` |
| `cargo +1.99.1 build --release --locked --manifest-path tools/gomad/rust/Cargo.toml` | 0; `closure/build-final-release.log` |
| `cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml --features determinism-fixture qualify` | 0; one platform accounting invariant; `closure/test-final-qualify.log` |
| `cargo +1.99.1 clippy --release --locked --manifest-path tools/gomad/rust/Cargo.toml --features determinism-fixture --all-targets -- -D warnings` | 0; `closure/clippy-final.log` |
| `cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --check` | 0 |
| Python `compile()` on the two changed qualification scripts; `git diff --check` | 0 |
| `qualify-check.py --grammar-only` with current fixture and ordinary release binaries against the frozen Go binary | 0; `closure-grammar-final-strict`, `closure-release-grammar-strict` |
| Ordinary release fixture verb and `GOMAD_FIXTURE_TEST=1` probes | Each refused with 2 and empty stdout |
| `GOMAD_GOLDEN_REGENERATE=1 bash /home/agent/wt/evidence/fn232-qualify/closure/lane.sh /home/agent/wt/evidence/fn232-qualify/closure-final-small-regen fixture,fixture-canary,sqlite,cluster` | 0; `closure/final-small-regen.log` |
| `python3 /home/agent/wt/evidence/fn232-qualify/closure/strict-retained.py --binary /home/agent/wt/evidence/fn232-qualify/closure/gomad-release --work /home/agent/wt/evidence/fn232-qualify/closure-final-small-regen --cells fixture,fixture-canary,sqlite,cluster --output /home/agent/wt/evidence/fn232-qualify/closure-final-small-strict` | 0; 102 original command receipts, 16 Results and 48 fresh report calls (32 tamper refusals); `closure/final-small-strict.log` |
| `GOMAD_GOLDEN_REGENERATE=1 bash /home/agent/wt/evidence/fn232-qualify/closure/lane.sh /home/agent/wt/evidence/fn232-qualify/closure-final-onebox-regen onebox` | 0; authoritative session `92604`; `closure/final-onebox-regen.log` |
| `python3 /home/agent/wt/evidence/fn232-qualify/closure/strict-retained.py --binary /home/agent/wt/evidence/fn232-qualify/closure/gomad-release --work /home/agent/wt/evidence/fn232-qualify/closure-final-onebox-regen --cells onebox --output /home/agent/wt/evidence/fn232-qualify/closure-final-onebox-strict` | 0; four Results, 53 original receipts and 30 fresh reports (26 tamper refusals); `closure/final-onebox-strict.log` |
| `python3 tools/gomad/testdata/parity/qualify-accounting-check.py --binary /home/agent/wt/evidence/fn232-qualify/closure/gomad-fixture --work /home/agent/wt/evidence/fn232-qualify/closure-final-accounting-strict` | 0; 51 controlled cases, repeated real Linux freed/live heap phase checks; `closure/final-accounting-strict.log` |
| Final fixture `qualify-check.py --grammar-only` against frozen Go, work `closure-final-grammar-strict`; final scoped `git diff --check` | 0; `closure/final-grammar-strict.log` |

The retained `closure/lane.sh` holds every exact binary/helper/source/compiler/input/preparation argument. The retained integrated-source manifest used by the four small canonical cells has SHA-256 `4a795923eb56d26a557ed3900b41f02fc68c0f18e22a9084e99252309f8b552a`. Both attempts pass original budgets without pending attribution, cost failure or budget failure. Raw memory checks are in the lane's `memory.txt`.

Removing the refusal golden before the final small regeneration proved child flag propagation: the child regenerated it, both independent views agreed, and its bytes equal the preceding separately regenerated/strict-checked refusal golden. `closure/golden-diff-small.json` lists every old/new leaf difference: fixture 54, canary 54, SQLite 52, cluster 46. Go semantic verdicts and both Rust qualification outcome booleans are unchanged. Identity, retained metadata stream hashes/sizes and fixture command terminators remain explicit for review; no identity field was normalized.

## Prior evidence and diagnostic runs

Retained v17 evidence remains under `/home/agent/wt/evidence/fn232-qualify/d/v17`. Its five successful stage receipts (`lanes.txt`) retain the original candidate scope. A fresh provenance inspection authenticated all 369 stdout/stderr command hashes, 56 Rust Result/receipt/artifact sets and the byte-identical regen-a/regen-b golden snapshots. This does not substitute for the current candidate lanes.

Expected RED evidence: the strict refusal check failed with exit 1 before its new golden existed; `closure/flag-red.log` records the parent regeneration flag failing to reach its Python child; `closure/terminator-red-2.log` records Rust's final-terminator grammar failure (exit 1). Actual old Rust report returned 2 for a v17 Result with final `--`, while Go returned 0. After the fix, current Rust normal/final-terminator reports both returned 0 with identical 165758-byte output, and positional-after-terminator returned 2. Current ordinary release also reopens a fresh current fixture Result with a missing installation and final terminator (0, 166133 bytes).

The preliminary old-binary strict small run was intentionally stopped after the parser source fix and before obsolete canonical work; authoritative exit 143, raw outputs under `closure-lane-strict-small`. It supplies no acceptance pass. The final runs above were not restarted on observation timeouts.

## Completed mapping and handover

All five fresh public-command cell lanes passed: fixture, intentional-failure canary, SQLite, cluster and onebox. There are 20 independent current Rust Results (two provisional and two canonical per cell); each executes the unchanged repeat-2 spec and replays every retained execution. The two independently derived golden views are byte-identical. Strict retained checks authenticate 155 original lane command receipts and all 20 raw Result/receipt/record/campaign/artifact chains, reproject them against current goldens, and make 78 fresh ordinary-release report calls including 58 expected tamper refusals. The separate grammar children retain 87 receipts per lane, including all 18 refusal cases twice per lineage.

The 51-case accounting oracle and golden cover process CPU counted once, concurrent/overlapping store windows, cold/warm/resident and operator ceilings, exact threshold crossings, missing measurements, inconsistent intervals and terminal receipts. Fresh real Linux phase processes prove freed store heap leaves the operator phase and live heap remains charged. No missing measurement or pending attribution was accepted. Source/helper/module/model/compiler/host/profile/limit identities remain in the records and retained provenance; report independently recomputes verdict, costs, budgets and receipt consistency. Its read path opens retained Result/receipt data rather than resolving installation, source or compiler. Public tests use non-existent installations, and current report also reopens v17 Results whose original source worktree is gone. Ordinary release refuses fixture commands/control environment; it successfully recomputes the fixture-produced retained Results.

The onebox integrated-source manifest differs from the small-cell union manifest because the selected input-file sets differ. For small cells, the raw manifest-file SHA-256, embedded manifest-byte SHA-256 and declared `integrated_source_sha256` are all `4a795923eb56d26a557ed3900b41f02fc68c0f18e22a9084e99252309f8b552a`. For both canonical onebox attempts, those same three quantities are all `e7540b3b4eba618cadab094b10acbb3f7ae3e3dc1d724907dd3cd05693d268aa`. There is no raw/declared digest mismatch.

Both onebox canonical runs pass every original bound. Operator peaks are 795000832 and 793214976 bytes (2 GiB ceiling); store peaks 3792535552 and 3791847424 bytes (4 GiB warm / 6 GiB cold ceilings); compiler peaks 2243706880 and 2383675392 bytes (6 GiB ceiling). Preparation walls are 83.04 and 80.22 seconds (600-second ceiling); largest store walls are 121.50 and 257.63 seconds (900-second ceiling). These are the named fixture candidate's measured checks, not a whole-candidate R12 or one-hour-ladder pass.

`closure/golden-diff-all.json` retains every explicit old/new leaf value: fixture 54, canary 54, SQLite 52, onebox 50, cluster 46. Go verdicts and Rust qualification outcome booleans are unchanged. Root's independent `qualify_review` returned SHIP for source and, in a fresh follow-up, SHIP with no findings for onebox goldens/evidence: it independently authenticated the four raw chains, all 53 original / 30 ordinary-release report receipts, both independent views and actual source-manifest bytes. It attributed all 50 onebox leaf changes to identity/source-root metadata and consequent hashes/addresses; source/compiler inventory file hashes and retained inspection/replay JSON otherwise agree. No record or identity normalization was applied to the generated goldens.

**Release:** all worker qualification, strict-report, accounting and grammar commands are terminal with the numeric exits above. The measured host window and main source freeze are explicitly released to root. No task-10 implementation/verification blocker remains; root alone owns final acceptance, Flow task/spec updates and the local commit. Ordinary-release whole-candidate qualification remains with fn-232.13. No push, history rewrite or Flow completion was performed by this worker.

Exact task-owned paths for the root commit:

- `.flow/artifacts/fn232-10-qualification-closure-2026-10-22.md`
- `tools/gomad/rust/gomad/src/cli/qualify.rs`
- `tools/gomad/testdata/parity/qualify-check.py`
- `tools/gomad/testdata/parity/qualify-corpus.json`
- `tools/gomad/testdata/parity/qualify-lane.py`
- `tools/gomad/testdata/parity/qualify/README.md`
- `tools/gomad/testdata/parity/goldens-qualify/refusals.json`
- `tools/gomad/testdata/parity/goldens-qualify/cells/{fixture,fixture-canary,sqlite,onebox,cluster}.json`

No worktree was created by this worker. Kept roots are the assigned build target (regenerable, isolated), the open fn-232 qualification evidence/preparation roots (required current/prior results, expected RED/interruption diagnostics and corpus provenance), and shared frozen reference sources/binaries (required R13 inputs). No unrelated worktree/cache was removed; untracked `tools/gomad/.turbo` and other workers' Flow artifacts were untouched. The temporary README scratch file was removed. Root owns the repository-wide worktree inventory and cleanup. Final disk: 158 GiB available on the checkout filesystem, 57 GiB on `/home/agent`.
