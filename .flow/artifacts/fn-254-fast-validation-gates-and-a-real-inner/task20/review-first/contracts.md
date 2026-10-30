**Review scope:** fn-254.20 adopts a mise-pinned mold linker for the two Linux GNU host triples in both Cargo workspaces, through a framework-local bash adapter, with one new guest-driver integration control and retained evidence. I read the spec, the eleven production/doc paths, the proof scripts and measurement data, and traced the standalone gate and bootstrap paths the adapter now sits under.

**What checks out against the stated contracts:**
- Target scoping is real. Both `.cargo/config.toml` files add only `[target.<linux-gnu>] linker`, no `rustflags`. Cargo resolves the slash-containing path against the config's parent, so rustc receives an absolute adapter path (confirmed in `standalone.log`), which keeps `BASH_SOURCE`-based mise root resolution valid under Cargo's package-root cwd.
- Fail-hard and sanitation hold. The adapter forces both auto-install variables off, propagates `mise where` failure status (test asserts exit 37), and refuses a missing `bin/ld.mold` before any `cc` call. Argument bytes, including spaces and non-UTF-8, pass through unchanged.
- WASI isolation holds. `guest_flags` in `guest.rs:219` adds no linker flag; `proof.py` asserts every `wasm32-wasip1` rustc invocation lacks `mold-linker`/`fuse-ld` and every host-unit invocation carries the adapter. Baseline vs restored stock builds in the same driver/lock/compiler namespace produced identical WASM bytes and identical sidecars, matching the doc's claims.
- Standalone/framework-only operation survives. `gate/standalone.rs` copies via a full inventory walk preserving the executable bit, keeps out-of-repo PATH entries (so `mise` stays reachable) and trusts the copied root for `mise -C`.
- Measurement is valid as described. I recomputed both medians from `measurement.json` rows 2–13 (0.2302 s / 0.0814 s); the schedule is one warmup per arm then six balanced observations; both arms replay identical captured objects/rlibs and force a real link; `.comment` checks separate the arms.

## Findings

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/design/gate-speed.md:277`
- **R-IDs**: [R9]
- **Problem**: The decision record says raw logs, scripts, matched linker arguments and `measurement.json` "remain in `.flow/tmp/fn25420/`", but the durable copies are the committed `task20/` artifacts (`measurement.json`, `measure.py`, `capture-linker.py`, all logs). `.flow/tmp` is scratch; a reader following the doc after cleanup finds nothing, while the committed evidence goes unmentioned.
- **Suggestion**: Point the sentence at the committed `task20/` directory and note `.flow/tmp/fn25420/` only as the original working location.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/guest-driver/tests/linker.rs:14`
- **R-IDs**: [R9]
- **Problem**: The doc contract is "Both workspaces select the framework's adapter", but the permanent control reads only `model/umpire/.cargo/config.toml`. The Temporal workspace's `model/.cargo/config.toml` is verified once by the one-shot `proof.py` and has no standing guard, so a later edit there (for example adding `rustflags` to a target table) would silently break the "no global Rust flags / target-scoped" promise for the Temporal tool build.
- **Suggestion**: Add a matching assertion for `model/.cargo/config.toml` (path `umpire/scripts/mold-linker.sh`) in a Temporal xtask control, or have the existing test read both files.

**FYI (not findings):**
- The mold arm's per-link `mise where` process spawn is disclosed and included in the 0.08 s figure, but it was only measured against one large link. Small build-script/proc-macro links in guest builds pay the same fixed cost; the matched stock guest builds (12.81 s GNU ld host units vs 12.25 s mold, in the retained logs) are consistent with a net gain but were not reported. The doc correctly makes no full-build claim.
- `-fuse-ld=mold` requires GCC ≥ 12 or clang as `cc`. The doc says "existing cc driver" without that floor; on an older x86_64 toolchain the build fails hard and visibly, which the fail-hard policy accepts. x86_64 execution is not claimed.
- The test's `targets.len() == 2` / `settings.len() == 1` assertions are a deliberate scope guard but will need updating for any future legitimate `[target.*]` entry.
- Code-smell baseline: `linux_linker_is_scoped_and_requires_the_pinned_tool` is one long method with inline shell fakes; acceptable for an integration control.

## Requirements coverage
| R-ID | Status | Evidence |
| --- | --- | --- |
| R9 | met (mold portion of R9; debuginfo and cache-reuse portions belong to .18 and sibling tasks) | Matched relink medians in `measurement.json`; `proof.py` WASI/host invocation and namespace assertions; `linker.rs` fail-hard and argument controls; WASM hash unchanged across baseline/restored builds; no speedup claimed beyond the focused link. |
| R7 | met for this task's scope (full standalone gate renewal deferred to .8 per spec) | Adapter resolves `mise -C <framework root>`; `standalone.rs` copy preserves the executable bit and external PATH; `standalone.log` shows a copied framework-local config linking with mold outside the parent repository. |

Unaddressed R-IDs: []
Suppressed findings: 1
Classification counts: 2 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":1},"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>