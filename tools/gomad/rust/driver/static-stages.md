`static_stages.py` assembles concrete reviewed commands and a separate raw-output
reader for the first 24 original driver labels. It does not run them while
assembling, claim qualification, or supply runtime results. Its output resets the
other 72 labels to explicit pending/UNEXECUTED entries; compose reviewed runtime
owner bindings before freezing the final configuration.

Supply a JSON `gomad.driver-static-inputs/v1` object with exactly:

- `repo`: this consolidated checkout's absolute path.
- `driver_config`: absolute future frozen configuration path.
- `lint_base`: exact 40-character base commit, preserving original lint scope.
- `tools`: exact `path`/`sha256` bindings for `python`, `go`, `cargo`, `rustc`,
  `rustdoc`, `rustfmt`, `cargo_clippy`, `clippy_driver`, `make`, `golangci`,
  `errortype`, and the original compile-only `true` executable. Rust tools share
  their pinned tool directory; Clippy keeps its original sibling executable names.
- `caches`: existing canonical external directories named `cargo_target`,
  `cargo_home`, `go_cache`, `go_mod_cache`. No cache is deleted by these stages.

Invoke the caller's pinned Python:

```
python static_stages.py assemble --inputs INPUTS.json --base BASE-BINDINGS.json \
  --plan /absolute/static-plan.json --output /absolute/static-bindings.json
```

Then compose runtime endpoints and run the ordinary `configure.py` freeze. The
base must already include explicit scratch/temporary bindings and the original
candidate/installations/identity/deadline fields. The frozen source includes this
module and `static-ignored.json`; its plan and tool hashes are additional frozen
inputs. Run/read recompute the fixed recipe, check all frozen bytes, and refuse
changed recipes. The assembler accepts tool/cache paths, not arbitrary commands.

The stages retain protocol checking, two independent Go replay regenerations,
retained Go model/conformance suites with original repeat counts, ordinary host
and conformance/reference helper builds, dynamic Go inventory/run checks,
compile-only admission, original actionlint 1.7.7 workflow selection, scoped
strict lint, all-platform architecture, harness and world race checks. Rust
lint covers the consolidated host's six and helper's seven feature combinations;
tests cover the six original helper modes and ordinary host. Conformance tests
run the actual library containing the transferred tests, rather than relying on
the now-thin helper binary's zero-test target. The helper corpus is independently
regenerated twice and must retain exact checked-in bytes. No obsolete Go host or
Go qualification producer is built. Both helper build roles are immutable copies
with actual feature/compiler/source receipts; the ordinary release rebuild must
match the already frozen candidate hash. These static roles do not substitute
for later public/helper installation and admission obligations.

Each stage retains `stages/LABEL/static/result.json` with schema
`gomad.driver-static-execution/v1`, bound `stage`, `source_sha256`, `identity`,
`config_sha256`, `plan_sha256`. Its `actions` contain each exact `recipe`, its
`results`, and the ordered `commands` it owns. Global `commands` contain unique
`command-NNNN` names, actual argv/cwd, and hashes for adjacent `.stdout`,
`.command.json`, `.process.json`. Those original command/process receipts retain
actual PID, interval, numeric exit and completion. Go results identify their raw
`listing` and `run`, exact package/root inventory and repetitions. Rust results
identify actual built test executables, binary hashes/target/cwd, raw `listing`,
`ignored_listing`, `run`, passed/ignored native names and repeat index; doctests
have a separate actual raw command. The independent reader reparses these logs
and checks exact dynamic inventory, repetitions, zero failures/skips, and all
known ignored dispositions. Its standard driver check adds `testcases`,
`raw_reports` (absolute paths/hashes), `static_execution` (path/hash), and
`deferred_ignored` for the final receiving reader. A raw log mutation, invented
root, missing/duplicated command, changed binary or unknown ignore refuses.

`static-ignored.json` inventories 14 exact original source/name/reason rows. The
seven gzip oracle tests and retained actual-WASM metadata test require actual
separate receiving evidence under `test-wasm/complete-behavioral-reach`.
The two original root34 aggregate broker tests map to
`transferred-live/TestClusterBrokerDurableRestartParity`; private35 maps to
`transferred-live/TestClusterShutdownLiveNodesEvidenceOrder`. The final live
reader must require their exact actual subvariants and raw evidence; a broad
root label alone cannot discharge them. Three original manual throughput,
calibration and diagnostic profiling exclusions remain unexecuted exclusions.
Every deferred entry retains native test ID, source/hash, original reason,
disposition, `obligation`, binary hash and `status: UNEXECUTED`. They never earn a
PASS testcase. This source inventory does not manufacture a complete original
Go callable/subtest reach map; that separately reviewed map remains required by
the final reader, along with actual later executions. No required-final ID or
semantic obligation is removed or weakened.

Portable tests cover recipe/matrix completeness, frozen tool mutations, all 96
labels with exactly 24 bound, exact ignores, dynamic inventories and refusals,
and independent raw re-reading. No static recipe has yet been executed against
this prospective source. Final runtime endpoints, full test reach map, role
admission, strict integrated lint/runtime tests, final source freeze and
qualification remain pending with their respective owners.
