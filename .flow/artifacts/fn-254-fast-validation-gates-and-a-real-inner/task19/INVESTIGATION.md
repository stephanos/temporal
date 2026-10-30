# fn-254.19: released Rust cache normalization is insufficient

Task .19 remains open. No production sccache pin, wrapper, incremental policy or guest
compiler behavior was changed. The evaluated installed release is sccache 0.19.0.

The required cross-checkout Rust workspace artifact reuse is unavailable through the
release's supported configuration. Its [Rust hasher](https://github.com/mozilla/sccache/blob/v0.19.0/src/compiler/rust.rs#L1789-L1880)
retains physical arguments, Cargo environment values and working directory.
`SCCACHE_BASEDIRS` is advertised in general configuration but this release's Rust
backend does not consume it. [Upstream Rust normalization](https://github.com/mozilla/sccache/pull/2813)
and [request-scoped basedirs](https://github.com/mozilla/sccache/pull/2814) remain open
as reviewed on 2026-10-29. See [research.json](research.json) for the primary-source
review and local ownership constraints.

The project-pinned direct-rustc A/B/A control used identical library source and compiler
arguments, apart from initially empty output paths. Both checkout roots were supplied
in `SCCACHE_BASEDIRS` before the private daemon started. The actual Rust 1.99.1
binary was resolved through the framework mise environment.
The pinned control reports:

| Compile | Rust hit delta | Rust miss delta | Non-cacheable delta | Wall seconds |
| --- | ---: | ---: | ---: | ---: |
| Checkout A, empty output | 0 | 1 | 0 | 0.1182 |
| Equivalent checkout B, empty output | 0 | 1 | 0 | 0.0190 |
| Checkout A, another empty output | 1 | 0 | 0 | 0.0075 |

Raw statistics, argv, source and tool hashes, compiler version, environment and all
runs' timings are in [measurements.json](measurements.json). All compiler commands
succeeded; there were no cache errors, timeouts or unsupported-compiler events.
The A repeat establishes a functioning Rust cache without Cargo Fresh or a final
executable link. The B miss establishes the prerequisite failure. These are capability
controls, not Cargo/bootstrap/quick timings or speedup evidence. The two earlier
exploratory runs used a default Rust 1.98 rustup shim and are
retained separately; they do not establish the project-pinned result. The first
exploratory timing includes cold daemon/compiler preparation and a statistics query.
The pinned run above times compilation only. Shared-host timing is not a regression
result.

Reproduce from the repository root after explicitly installing sccache 0.19.0 through
mise; the script never installs or updates tools:

```sh
mise install sccache@0.19.0
python3 .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task19/reproduce.py .flow/tmp/fn-254.19/new-control
```

The output directory must be new. The script owns a local disk cache under that
directory and a unique temporary Unix socket; it stops only its own daemon in a
`finally` block. No run reset counters or contacted the default daemon. No
shared Cargo target, cache lease or existing cache was changed. Scratch caches and
sources remain under `.flow/tmp/fn-254.19`; durable receipts are copied here.

Suppressing checkout identity by hand is unsafe for this repository. In
`model/umpire/dev/guest-driver/src/context.rs`, `ToolRoot::local` embeds
`env!("CARGO_MANIFEST_DIR")` to select its default framework tool/pin root. A cache
entry compiled in A can retain A's authority when used in B. The proposed upstream
key normalization explicitly acknowledges that embedded-path issue. Rewriting keys,
forcing a fake cwd or replacing manifest values does not establish equivalent Rust
semantics. Registry dependency reuse alone would not complete this task.

Resume only with supported, semantics-preserving checkout normalization and a new
positive capability control. Production command admission, bootstrap isolation,
compiler/profile/target and changed-source negatives, cache failure controls,
workspace Cargo measurements, warm quick regression evidence and guest/native parity
remain unimplemented acceptance obligations. The final whole-gate proof remains .8.
