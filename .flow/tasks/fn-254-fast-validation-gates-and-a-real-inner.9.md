---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.9 Framework/Temporal split: inventory gate steps and couplings, decide the framework entry point

Touches: [model/design/gate-speed.md, model/design/framework-gate.md]

## Description
Decide where the framework's gate lives and how it stands alone (spec R7), before .3, .5 and .6 build on the Makefile. A short design note; no code moves here.

**Size:** S
**Files:** `model/design/framework-gate.md` (new).
**Touches:** [model/design/gate-speed.md, model/design/framework-gate.md]

### Approach
- Classify every model step in `Makefile:1270-1315`, every script in `model/scripts/`, every `model/dev/xtask` module and the `model/build/` caches as framework or Temporal. Examples: `temporal-guest-freshness`, `source-boundary`, `temporal-protobuf-freshness` and `temporal-go-host` are Temporal; `retired-names` is mixed, since it excludes a Temporal path.
- List the couplings with file:line: the 45 `"../../../.."` repo-root paths in `model/umpire`, `Scratch::new_in(root, "model/build/tlc")` (`backends/process/src/support.rs:52`) and the Apalache equivalents, the framework include of `dev/xtask/src/apalache_pins.rs`, protoc generators from the root `.bin/` (`scripts/proto-go.sh:31,41`), and `model-check`'s `$(API_BINPB)` prerequisite.
- **Owner decisions (ask, record in the spec's Decision Context):**
  1. Entry point: an `umpire` xtask subcommand (`cargo xtask gate`, one-language rule from fn-247) or a `model/umpire/Makefile`. Recommend the xtask.
  2. Workspace: `model/umpire` as its own Cargo workspace with its own lockfile, with the Temporal workspace depending on it by path (strongest standalone proof), or one shared workspace with the framework gate selecting `umpire` packages. Recommend its own workspace.
- Name where shared repo files go: `rust-toolchain.toml`, `rustfmt.toml`, `clippy.toml`, `deny.toml`, `mise` pins, `.config/nextest.toml`.

### Investigation targets
**Required:**
- `Makefile:1270-1315`, `model/scripts/`, `model/dev/xtask/src/`
- `model/Cargo.toml:1-60` — members and workspace dependencies
- `model/umpire/backends/process/src/support.rs:40-60` — cache root
- `.flow/specs/fn-247-one-language-for-the-framework-a-rust.md` — xtask conventions

### Acceptance
- [ ] Every gate step, script, xtask module and cache is classified framework or Temporal, with each coupling listed by file:line.
- [ ] Both owner decisions recorded in the fn-254 Decision Context.
- [ ] The note names the target layout that .10 implements, and what .3, .5 and .6 must place in it.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
