# fn254.3 preparation, 2026-10-25

Pin `github:nextest-rs/nextest` to `0.9.149` with `version_prefix = "cargo-nextest-"`. The installed mise 2026.10.11 downloaded and admitted the official Linux ARM64 precompiled binary in this scratch tree. `--version`, run/list help, and schema help passed. No project build, test execution, fresh baseline, source edit, task start, or Flow mutation occurred.

The official changelog lists 0.9.149, released 2026-10-20, as current stable. The release also fixes the self-updater's compromised certificate trust. Use mise admission rather than invoking an older nextest self-updater. [Nextest changelog](https://www.nexte.st/changelog/)

## Mise resolver proof

Both local registry queries, `mise registry cargo-nextest` and `mise registry nextest`, returned tool-not-found. The installed builtin `github` backend succeeded; no plugin or Cargo compilation was needed. Its version prefix option matches the release tags. [Mise GitHub backend](https://mise.jdx.dev/dev-tools/backends/github.html)

```toml
[tools]
"github:nextest-rs/nextest" = { version = "0.9.149", version_prefix = "cargo-nextest-" }
```

This candidate belongs in each task-owned mise file. Scratch `configcwd/mise.lock` resolves seven default platforms, including the universal macOS archive for both macOS architectures. Only Linux ARM64 was executed. The archive SHA-256 is `6f82e375187cf3a1f3e336879a51940c315a5d0ce7253f7e517ad28295b74189`; installed binary SHA-256 is `5b5d894f7770fa61e1061dc6c8338fe9c98ed80de3614035f70fe5bd0e133df7`. URLs/checksums and provenance entries remain in that scratch lock. Official nextest release URLs support exact versions and list Linux ARM64 and universal macOS binaries. [Release URLs](https://nexte.st/docs/installation/release-urls/)

`probe-env.json` retains isolated data/config/cache/state/global/system/trust/ceiling paths. HOME stayed unchanged. The config cwd is `configcwd/`; its ceiling is this probe directory, a parent. `config-list.log` lists only the scratch config. Ceiling directories themselves are excluded from discovery. [Mise configuration](https://mise.jdx.dev/configuration)

Initial backend queries preceded full isolation. Initial installation used the isolated data/config/cache/state, global/system config-file overrides and ceiling; the later help/lock probe additionally set explicit system-config-dir, shims and tmp directories. `MISE_SYSTEM_DIR` in the initial environment was replaced with documented `MISE_SYSTEM_CONFIG_DIR`. No global tool installation or mise upgrade occurred.

## Actual runner seam

At task10 source `3858df60b843619c35ca06e51aee01017dc4bbb7`, both leaves select `cargo test --workspace --locked --quiet`. No feature, target, release, package exclusion or ignored-test flag is present. Framework `Gate::command` supplies cwd, tool root, Cargo home/target, cache-disable, no-auto-install, and required-backend mode; Temporal `gate::stage` supplies cwd/home/cache-disable and inherits the remaining environment. Preserve these contexts on both replacement commands.

Temporal `test_run::run(root)` can execute nextest and one Cargo doctest invocation sequentially, returning the first failing status. Framework `test_run::configure(&mut Command)` only appends argv to one command. `gate.rs` calls it for both `tests` and `update`; update adds INSTA_UPDATE/TRYBUILD. That signature cannot directly sequence two processes. Before .3 implementation, narrowly amend ownership/seam or agree how the leaf owns orchestration. Preserve gate timing and update environment for both commands. This is a preparation finding, with no proposed dispatcher redesign.

Nextest explicitly excludes doctests. Each successful test stage must then run exactly one `cargo test --doc --workspace --locked --quiet`; retain Cargo for doctests and stop before that command when nextest fails. [Nextest doctest limitation](https://nexte.st/)

## Selection and parity controls

Candidate run argv is `cargo nextest run --workspace --locked --fail-fast --run-ignored default --status-level slow --final-status-level slow --success-output never --failure-output immediate-final`. These reporter settings retain failure details and slow-test status without pass-per-test noise. The exact slow threshold must be chosen and recorded during implementation. [Running tests](https://nexte.st/docs/running/), [Reporting](https://nexte.st/docs/reporting/)

For inventory, use `cargo nextest list --workspace --locked --message-format json --run-ignored all --ignore-default-filter`. This lists both ignored states. Execution must keep `--run-ignored default`. Do not copy clippy's `--all-features` into tests. Match any explicitly admitted feature flags (`--features`, `--all-features`, `--no-default-features`), target selectors, target triple, Cargo profile, environment/backend mode and Cargo configuration on both inventories. Nextest `--profile` selects a nextest profile; `--cargo-profile` selects the Cargo build profile. List/run help retained here confirms these controls. [Listing tests](https://nexte.st/docs/listing/)

Normalize each `rust-suites` entry using `package-id` joined to frozen Cargo metadata, `binary-name` plus `kind` for target identity, each `testcases` key for the full test name, and its `ignored` boolean. Retain suite ID/build platform and filter-match diagnostics so filtered suites cannot disappear silently. Canonicalize workspace-relative package identity because frozen metadata contains old worktree roots. Compare identities as a multiset and reject missing, extra, duplicated or unmatched entries. Doctests require separate Cargo list/execution accounting. [Machine-readable list fields](https://nexte.st/docs/machine-readable/list/)

Task10 `index.json` proves a frozen 930-identity baseline, four ignored and 19 new controls, so 949 is the expected joined logical count before any later producer additions. It is not a freshly observed nextest or joined Cargo list count. Keep .4's checked mapping and task10's declared delta. Reference fallback is the original Cargo command; when a serial diagnostic is necessary use the same selection with `cargo test --workspace --locked --quiet -- --test-threads=1`. No fresh serial timing baseline is required. A nextest diagnostic can use `--test-threads 1`; that still uses nextest's per-test process model. [Cargo test semantics](https://doc.rust-lang.org/cargo/commands/cargo-test.html)

## Source-derived JVM candidates

Package ownership below comes from task13 revision1 frozen framework/outer metadata, normalized from its former worktree root, and current Rust source. These are candidate filter scopes for .3 to verify against its actual list.

| Workspace | Candidate package/target filter | Source route |
| --- | --- | --- |
| Framework | `package(=umpire-tlc) & binary(=main)` | controls and scaling call `Tlc::discover`; runner unit batch controls mostly use fake Java scripts |
| Framework | `package(=umpire-apalache) & binary(=main)` | controls call `Apalache::discover`, selected by UMPIRE_APALACHE |
| Framework | `package(=umpire-tla) & binary(=main)` | emission, world, vector, optional_ticks, phased, shadowing, narrow_remainder, sequential_checks call TLC |
| Framework | `package(=umpire-testing) & kind(lib) & test(actual_tlc_fact_traces_replay_exact_vectors)` | `src/backend_tests.rs:428` calls real TLC when required |
| Framework | `package(=umpire-fixtures) & binary(=fixtures)` | generated registry -> TLC consumer -> generated check_backend -> check/bound -> backend::check_exploration -> Tlc; explicit interpreted/bound/fixture test modules also call TLC |
| Temporal | `package(=umpire-temporal-tests) & binary(=fixtures)` | same generated TLC consumer plus activity tests, policy_checks and proof_checks; names alone omit conditional JVM paths |

Conservative grouping of the three mixed integration binaries and fixture binaries is justified until exact-name refinement proves every transitive route. Keep all other packages parallel. Catalog admission/macros/build tests and front world_fairness were searched; their inspected paths admit/emit syntax or use native consumers and did not expose Java launch routes. Merely depending on umpire-testing does not imply JVM execution. Its default CHECKER/SUITE consumers are native; the explicit TLC consumer and check/bound APIs introduce Java. Author DSL/suite and native checker fixture harnesses use those native consumers in inspected sources. No complete semantic callgraph is claimed. Generated dispatch and arbitrary future consumer closures remain implementation inventory checks. Search evidence is in `jvm-source-search.txt` and `jvm-callers.txt`.

Illustrative framework grouping, seven lines, pending actual list validation. Exact matching and kind predicates are documented in the [filterset DSL](https://nexte.st/docs/filtersets/reference/).

```toml
[test-groups]
jvm = { max-threads = 1 }
[profile.default]
test-threads = "num-cpus"
[[profile.default.overrides]]
filter = '(package(=umpire-tlc) | package(=umpire-apalache) | package(=umpire-tla)) & binary(=main) | package(=umpire-fixtures) & binary(=fixtures) | package(=umpire-testing) & kind(lib) & test(actual_tlc_fact_traces_replay_exact_vectors)'
test-group = 'jvm'
```

A separate Temporal config needs its fixtures predicate. This candidate was not parsed against workspace metadata or exercised. Nextest groups bound concurrently running tests within one invocation, and leave other tests unaffected. Separate nextest invocations do not share a host semaphore. [Test group semantics](https://nexte.st/docs/configuration/test-groups/)

Current TLC defaults to at most two singleton-recovery property workers per test; each execution has `-Xmx512m` by default. Apalache defaults to the same 512 MiB limit. Thus a group size of one can still mean two simultaneous JVM heaps, plus nonheap/process memory. Stage5 overlap must budget the sum across framework and Temporal invocations and other JVM consumers, including per-test worker overrides. .3 representative execution must observe the combined peak. This research establishes no host-wide bound, scheduler change, memory measurement or completed parity proof. Unrelated full workspace/combined execution remains final .8 work under owner cadence.
