# fn254.2 retained validation evidence

Frozen implementation checkpoint: be919e0999; pre-edit base: 3d54e19b1bfbce799ada92e4d5ffc7e8db636797.

`source-provenance.json` records the owner-waived fresh timing baseline, isolated cache/profile/concurrency settings and copied descriptor provenance. Historical timing reference: `model/design/gate-speed.md`; comparisons across historical modes, profiles and workloads do not establish an attributable speedup.

- `launch-red.log`: intended red control, healthy three-property check launched four model JVMs instead of two.
- `protocol-red.log`: intended red admission control, unfinished batch evaluation returned Evaluation rather than Refusal; also records an initial test-harness schema error corrected before proof.
- `tlc-equivalence.log` and `inherited-singleton-nondeterminism.txt`: inherited random-fingerprint singleton liveness trace variation, including all private observers and exact lasso differences. No trace fields were normalized.
- `tlc-bytecode.txt`, `liveworker-bytecode.txt`, `tlc-help.txt`: pinned TLC jar primary bytecode/help evidence. The constructor randomly selects `fpIndex`; process passes it to `FP64.Init`; documented `-fp` selects a fixed polynomial. Only `-fp 0` changed; heap, GC, disk fingerprints and one TLC worker remain.
- `tlc-equivalence-3.log`: completed TLC crate suite with automatic strategy comparison enabled: 13 unit tests, seven real controls, one scaling control; exit0. Zero doctests is expected (no Rust code examples).
- `clippy-third.log` and `fmt-final.log`: strict TLC all-target/all-feature clippy and workspace formatting, exit0.
- `tlc-consumer-inventory.json`: hashed source callsites beneath the automatic comparison and shared TLC golden fixture inventory.
- `workspace-required-equivalence.log` / `.status`: required workspace command, retain actual terminal status; never infer success from individual green binaries.
- `workspace-jvm-observation.json`: sampled owned Java process peak, including separate admission/model categories; sampling is not an exact launch inventory. Unit event accounting proves the configured recovery peak, ordered results, unique scratches and cleanup exactly.

Intermediate failed attempts (`controls-first.log`, `tlc-equivalence-2.log`, initial clippy/fmt logs) are retained and are not passes. No production baseline campaign was run. No review verdict or task completion is issued by the parallel worker.

## Final affected-scope completion and collector fix

Owner cadence amendment: conductor commit81f509b088226986e5ea2cbd8ee10a2942d430e4; exact policy copy/hash in `validation-cadence-policy.json` and `cadence-amendment-reference.json`. It retains all changed TLC controls and every existing TLC fixture/mutant comparison now, and defers unrelated workspace execution to fn254.8.

`frozen-corpus-completion.json` records the be919 original workspace-union corpus:171 selected cases across15 targets,168 passed and3 existing ignores. Shared fixture coverage uses46 completed case statuses before the original workspace timeout plus the sole remaining aggregate case with actualrc0; the enclosing workspace command remains124. `helper-a/021.json` records Temporal39passed/2ignored with actualrc0 (2307.66s); `helper-b/retained/results.json` records12 targets/69 cases allrc0; `frozen-000.json` records interpreted13passed/1ignored.

`validation-scope-ledger.json` carries every one of the923 original package/target/test identities and ignored status:171 retained and752 explicitly deferred. `tlc-consumer-proof-closure.json` maps all23 TLC caller source files to retained proof; `required-runtime-comparison-guards.json` verifies the comparison assertion is linked into all15 retained binaries. Raw original artifacts/features, selected identities, runtime bindings, binary/source SHA and copied descriptor provenance are retained separately.

The supplemental per-package Cargo runs are not original workspace-union proof. An unrelated Kettle UI attempt selected default rustc1.98 because the reconstruction omitted RUSTUP_TOOLCHAIN. The original mise1.99.1 bindings were captured and restored for subsequent retained commands; the unrelated UI failure remains deferred, not green. Inventory-helper attempts also remain retained; none executed test cases before their inventory failures.

Two orchestration drivers were stopped at owned child boundaries after actual childexit0 to change proof scheduling. Their143/130 statuses are not test passes. Frozen per-child receipts and the explicit completed-case prefix are authoritative. No active test was killed to split the scope.

The be919 collector precollected outputs before applying aggregate limits. `recovery-bounds-red.log` proves both regression symptoms. Current source2d596d0f5c fixes only check_strategy/recover_properties: bounded original-order waves, result/error and aggregate-cap admission before the next wave, reference width1. `collector-delta-reuse.json` verifies all other production runner bytes identical and explains qualified corpus reuse; old binaries are never labelled current source. `recovery-bounds-green.log` is the targeted green; `tlc-bounded-equivalence.log` is current-source23-case core exact comparison (15unit/7real/1scaling, allrc0), including a4-property mixed case crossing multiple waves. Current fmt/strictclippy are green in bounded-style.status. `current-core-validation.json` records exact command/environment/binary SHA.

Primary activity outputs show35 ordinary World blocks with14 model launches,2 added-negative World blocks with16, and8 reduced-module blocks with2. These counts exclude comparison launches/admission probes. Historical24/26 references are structural estimates, not newly measured baselines. Comparison-mode timings and changed threading are not an attributed routine-gate speedup. Peak events prove2 recovery workers; the initial live observer sampled peak4 model JVMs including a separately inventoried admission category.

Frozen1GiB binaries remain ignored at this worktree's `frozen-workspace-binaries/` for conductor-owned verified retirement/archive; the durable handover copies small manifests/logs/ledgers, not these binaries. No review verdict, flowctl done, task lifecycle mutation, push or integration was performed by this parallel worker.
