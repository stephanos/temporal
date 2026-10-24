---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.2 xtask crate replaces model/scripts/*.py with identical arguments, exit codes and output; guest rebuilt byte-identical

## Description
After .1. Create `model/dev/xtask` (binary crate, `cargo xtask` alias in `.cargo/config.toml`) with subcommands `guest-rustc`, `guest-lock`, `guest-freshness`, `fetch-apalache` and `retired-names` porting `model/scripts/{guest-rustc,guest-lock,guest-freshness,fetch-apalache,retired-name-matches}.py` one for one: same arguments, exit codes and stdout bytes, proved by golden tests over recorded invocations. `guest.sh` writes a launcher that execs the xtask binary with the same arguments; `retired-names.sh` and the Makefile (`temporal-guest-freshness`, `guest-build-controls`) call xtask; `python3 -m unittest discover` becomes `cargo test -p xtask`. Port every `test_*.py` case (guest, guest-lock, guest-freshness, apalache, retired names). Fold `feature/activity/compatibility/verify.py` into `checkpoint.rs`. Delete all `.py` files under `model/`; `git ls-files 'model/**/*.py'` is empty. Prove `make umpire-guest` yields the committed guest bytes and the freshness check passes on Linux (macOS certification is renewed by the existing fn-232 recipe when the owner next runs it; record if not run).

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
