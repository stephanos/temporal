# System workspace tool bridge

From `model/`, use `mise exec -- cargo xtask <command>`. The bridge delegates reusable
commands to the independent framework CLI at `umpire/dev/xtask`. Guest commands route
the five framework guest packages to `umpire/`; `umpire-temporal-guest` and custom
packages use the system workspace. Output, Cargo home and target paths resolve
against the caller before changing directories. Compiler wrapping and cache admission
stay in the dedicated framework guest driver.

`gate` runs the combined framework and system graph. `proto-go temporal [--check]`
owns the system's protobuf selector; run `core` or `kettle` from `model/umpire` instead.
`build-guests` builds the Temporal guest. `time` measures the combined gate and the
edited-package benchmark; `make model-validate` selects the TLC-required combined
run. The bridge's forwarded `validate` command retains framework-only scope.

`timing-controls` and `proto-controls` run selected Rust controls through the shared
test runner. Repository Make freshness, guest routing, embedded-guest refusal and
retired-name controls remain system-owned. The two minimal `scripts/xtask.sh` files
bootstrap current native tools; the former orchestration and control scripts are
native commands and Rust tests.
