The fresh original ringpop reference uses recovered commit
`92c69594f21ec09d82b716b04a8eed8352a8fdd4`. It does not restore missing historical
fn235 evidence. The available `completion-fn235-reference{-v2}` artifacts are
2-node echo-loop failures, so they cannot supply the required successful 3-node
ringpop World, helper network wire or runtime choice traces.

Preparation is separate from qualification. Acquire the coordinator's heavy Go
build lease and an exclusive Cargo target before executing:

```
python3 tools/gomad/rust/driver/original_ringpop.py \
  --prepare-plan /absolute/original-build-plan.json --root /absolute/fresh-original-build
```

The plan contains `schema: gomad.original-ringpop-build-plan/v1`, the exact
`reference_commit` above, absolute `repo`, `git`, `compiler`, `cargo` and
`cargo_target` paths, plus `files` mapping each of the three executable paths to
its SHA256. The `compiler` is the pinned stock Go compiler. `cargo` must be a
regular, pinned rustup Cargo proxy accepting `+1.99.1`; a copied proxy must retain
the basename `cargo`. The matching Rust toolchain and locked offline dependency
cache must already exist. Optional `environment` and `temporary_paths` use the
normal driver's environment rules. No target or cache is silently selected.

Preparation archives the full original repository because its Go module replaces
the root server module. It changes exactly one Go host file: after the original
network decoder accepts the complete helper bytes, an explicit environment
variable permits a fresh exclusive file write of those bytes. The old host
otherwise discards the wire after decoding it into network records. No Rust
helper or model source changes. The receipt records before/after source file
inventories, symlink targets, the original archive, exact patch, all three build
commands, actual process receipts and executable hashes. This is explicitly an
observed original-source reference with an observation patch.

Before the global qualification freeze, seal the generated `inputs.json`, source
archive, every regular source file, all executables referenced by its build plan,
and every build command/process/stdout/stderr receipt. Set:

```
final_retained.ringpop_original.inputs = /absolute/fresh-original-build/inputs.json
```

The separately sealed current fn224 ringpop fixture and scenario remain explicit
workload inputs; that fixture did not exist in the recovered original archive.
The final section produces the original run once under
`GLOBAL_ROOT/original-ringpop`, then the candidate run and fresh replay, within
the existing global deadline. It uses seed7, exactly 3 nodes, the original scenario
with only the three duration arguments changed to120s, and the original explicit
40m/45m watchdog arguments. The driver coordinator's global deadline still bounds
the entire attempt. Future artifacts and run receipts stay dynamic in that same
cycle directory and never require prefreeze hashes.

The reference receipt schema is `gomad.ringpop-original-reference/v1`, with
`fresh: true` and the recovered commit. The reader verifies a genuine retained
success, original public inspect/result reports, module/helper identities, all
three phase witnesses per owner, complete raw World/network/96-byte choice trace
streams and command custody. Candidate module identity and raw streams must equal
the original bytes; there is no projection, first-replay comparison, echo-loop
substitution or historical-custody claim. `--read-only` launches no producer and
refuses incomplete reference evidence.
