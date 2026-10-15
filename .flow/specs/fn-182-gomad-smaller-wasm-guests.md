# Gomad: Smaller WASM guests

## Conversation Evidence

> user: "also file 7. Smaller WASM guests"

## Goal & Context

Reduce unnecessary dependencies in Temporal WASM guests and measure the resulting module size, startup cost and memory use. [paraphrase]

The existing optional-provider trimming idea names Elasticsearch as a candidate; actual dependency and registration analysis determines what can be excluded from the supported guest workload.

## Architecture & Data Models

Narrow the guest build closure at optional provider or registration boundaries. Keep required Temporal services, storage/network models and qualified workload behavior. The normal server and stock/native builds preserve their existing dependency selection and capabilities.

The qualified fn-178 candidate supplies the integrated baseline after fn-176 storage migration. Build-profile and dependency changes participate in preparation, provenance and cache identity; retained replay uses the recorded module rather than rebuilding it with a new dependency selection.

## API Contracts

Any guest-specific dependency selection must be explicit in the supported build profile and discoverable in retained provenance. An excluded provider selected by configuration produces a clear unsupported-configuration refusal, rather than silently using another provider. No new provider API or runtime scheduling behavior is introduced.

## Edge Cases & Constraints

An import that appears optional can register behavior through initialization or affect the test build closure. Confirm transitive dependencies and registration before removing it. Tests continue to exercise the same workload and failure witnesses; disabling a workload to shrink its binary is not a successful optimization.

Guest changes can rotate module, target-site or model identities. Measure repeatability and replay within each candidate's own identities, and compare functional outcomes across the baseline and candidate without assuming cross-build choice tapes are interchangeable.

## Implementation Design

### Baseline and exclusion decision (.1)

Freeze the qualified post-fn-178 integrated source/helper/runtime/model candidate before editing any provider selection. Inventory representative existing single-instance and cluster Temporal workloads with the provider's actual pinned `go list -deps -json`/test selection and build environment (`wasip1/wasm`, pure Go, existing adapters, tags and source inventory). Record complete command/tag/test/argument/configuration inputs, package/module edges and init registration dependencies; distinguish imported packages from reachable linked bytes. Reuse the final fn-178 workload/seed/repetition and resource selections rather than a smaller test package invented for this optimization.

On the same real Linux/arm64 host retain the full untrimmed R4 measurement baseline before making exclusions; missing phase data cannot be reconstructed from historical reports or a differently instrumented helper. Reuse `wasi` and `qualification` measurement collectors, but use the production fuel/GC/runtime profile. Existing diagnostic/fuel-off variants are useful attribution tools, not substitutes for production baseline/candidate acceptance. Match helper, runtime, engine, clocks, captured inputs, memory/fuel, resident nodes and replay selections across the pair; record the deliberately different dependency-selection tag/source closure as the treatment. Run heavy compilation/performance measurements alone with isolated cache roots, explicit cold/warm state and matched repetition order so parallel work or another candidate's caches cannot create the apparent gain.

Elasticsearch is a candidate, not a predetermined deletion. Trace `common/persistence/visibility/factory.go`'s constructors and its public `*elasticsearch.ProcessorConfig` signatures, test imports and init registrations; deleting a constructor import alone cannot sever all those edges. Before .2 record the smallest safe optional-provider seam and a disposable same-workload build demonstrating an actual module-byte reduction. Consider other optional providers if this seam requires broad API changes. Preserve required config types and default public APIs; if no safe exclusion exists, report that result and leave R2 unfulfilled rather than weaken workloads or declare completion.

### Explicit guest source selection and identity (.2)

Prefer the existing `--build-tag`/`target.Spec.BuildTags` path for an opt-in dependency selection such as `gomad_wasm_trim`, constrained to cooperative `gomad && wasip1 && wasm` guest builds. Name the actual tag(s) in .1's decision and use complementary source constraints so default production, native Gomad and stock WASI builds keep their previous provider implementation. Reject an incompatible stock-profile trim request explicitly; never enable trimming implicitly from the broad `gomad` tag or make a provider exclusion apply to native callers. Keep ordinary test cases and type/API contracts; avoid a new runtime profile registry or dependency upgrades.

At the measured provider boundary isolate only heavyweight constructor/registration/type references that keep the optional closure reachable. An excluded provider selected as primary or secondary configuration must take an explicit deterministic unsupported-configuration path before serving requests, with the provider named; never silently substitute SQLite, disable visibility, change precedence or skip a test. Package files that preserve light public types must not import the removed heavy implementation transitively. .1's proof selects the exact files; do not preapprove a large factory redesign.

`backend/provider.go` already binds BuildTags, build environment, compiler and source inventory into provenance/build keys; `sourceInventory` and prepared validation must bind all actual selected files, embeds, module language versions and generated/adapter inputs. Extend only a demonstrated missing input. The compiled cache already binds actual module/helper/engine compatibility, so a different build profile with identical module bytes may safely share that compatible entry; R3 forbids incompatible reuse, not content-identical reuse. Prove changed tags and changed provider source with unchanged go.mod invalidate preparation when needed, and that recorded module bytes, not today's tags or source tree, drive source-less retained replay/resume. Keep unsupported historical identity handling and the default nil/empty provenance conventions intact.

### Matched measurements and behavioral acceptance (.3)

Report module bytes and verified compiled-cache artifact bytes separately, cold compilation, warm preparation versus compiled load, fresh instantiation, common authorization-to-terminal execution and full-run wall costs, and actual helper peak RSS. For clusters use fn-178's one-process accounting and final resident budget; do not duplicate one helper peak per incarnation or compare it with obsolete per-helper cluster limits. Record CPU/architecture evidence, source/tool/module/helper/engine/runtime identities, raw per-seed/repetition rows, median/range, exact cache state and rerun commands. Compare the matched distributions and report unavailable phases, regressions and differences within spread plainly. Smaller bytes establish R2, not an assumed startup/execution/RSS improvement or a new arbitrary speedup threshold.

Compare success/oracle/intentional-failure witnesses across untrimmed and trimmed builds and prove deterministic repetition plus fresh exact retained replay within each candidate's own identities, including storage-crash/reopen and network/cluster effects used by the existing selections. Carry fn-176's final full crash-selection operations/digest context wherever a selected workload uses explicit crash choices; preserve context admission, default nil behavior and real pending mismatch classification. Do not compare cross-build tape/site identity as if it were a replay contract. Qualification must retain every existing resource bound and original workload/canary; no missing case is excused by the reduced closure.

Tasks .1–.3 form one measurement-led optimization batch with an actual pre-exclusion baseline as a gate to source changes. Run focused provider/source/provenance and default/selected guest configuration checks during implementation; run affected lint/architecture/identity validation, full `test-wasm` and selected production guest/cluster qualification on the joined candidate. Native execution remains deferred under fn-128/fn-149; portable native/default source controls remain required. Retain one R1–R5 handover with the exclusion decision, before/after rows, commands/exits, current identities, independent review and truthful documentation. This planning pass performs no source modification, build, measurement or runtime test and makes no size/performance claim.

## Acceptance Criteria

- **R1:** Record the dependency closure for representative qualified Temporal guests, identify removable optional providers and demonstrate which dependency edges the selected exclusions eliminate. Errors: any candidate needed by a supported workload remains included or is rejected as an optimization candidate.
- **R2:** Implement the justified guest-only exclusions and demonstrate a smaller module for the same supported workload selection; default production server and stock/native build behavior remain unchanged. Errors: explicitly requesting an excluded provider fails clearly without fallback; unsupported exclusions do not ship.
- **R3:** Preparation keys, provenance and compiled-cache admission bind the selected dependency profile and resulting module. Errors: different profiles cannot share incompatible cache entries, and retained replay with incompatible execution identities remains a classified refusal.
- **R4:** Retain matched baseline/candidate measurements on the same real Linux/arm64 host for module bytes, compiled-cache bytes, cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS; report repetitions, median/range, host/build identities and rerun commands. Errors: unavailable measurements or differences within measurement spread are labeled as such, with no assumed startup or memory gain.
- **R5:** The affected qualified workloads retain success and intentional-failure witnesses, deterministic repetition and fresh retained replay on the trimmed candidate; existing capacity and resource budgets still pass. Errors: behavior changes, missing canary failures or failed replay prevent adoption of the exclusion.

## Boundaries

- No GC, fuel, scheduler, engine or runtime-policy changes; no dependency upgrades or broad provider redesign.
- No removal of production-server features, stock/native support, differential models or test cases needed to retain workload coverage.
- No promise that smaller modules improve every phase; performance conclusions follow matched measurements.

## Decision Context

Optional dependency trimming targets compilation, cache and memory costs without changing simulated execution semantics. Elasticsearch is a concrete starting point, not a required removal if closure analysis shows it is needed. The implementation follows fn-178 so measurements include the settled cluster execution architecture.

Planning confirmed optional Elasticsearch references in the visibility factory and existing build-tag/source-closure provenance. Reuse those mechanisms; an explicit dependency selection does not require a new runtime profile framework. The accepted measurement contract follows the existing Linux/arm64 performance host policy. Tasks .1 through .3 are sequential.

Planning refinement (2026-10-15): capture the unchanged-workload production baseline after fn-178 and before exclusion, choose a measured optional-provider seam without making Elasticsearch removal mandatory, preserve native/stock selection through explicit guest-only tags, and retain compatible content-addressed cache reuse. No exclusion or performance conclusion is approved without the R1/R2/R4/R5 evidence.

## Early proof point

fn-182-gomad-smaller-wasm-guests.1 proves that a safe optional-provider exclusion shrinks a same-selection guest build and captures the baseline. If it fails, reconsider the exclusion rather than weakening workload coverage or proceeding to task .2.

## Quick commands

Use pinned tool versions and the documented helper inputs for actual guest gates.

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/backend
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Record the dependency closure for representative qualified Temporal guests, identify removable optional providers and demonstrate which dependency edges the selected exclusions eliminate. Errors: any candidate needed by a supported workload remains included or is rejected as an optimization candidate. | fn-182-gomad-smaller-wasm-guests.1 | — |
| R2 | Implement the justified guest-only exclusions and demonstrate a smaller module for the same supported workload selection; default production server and stock/native build behavior remain unchanged. Errors: explicitly requesting an excluded provider fails clearly without fallback; unsupported exclusions do not ship. | fn-182-gomad-smaller-wasm-guests.2 | — |
| R3 | Preparation keys, provenance and compiled-cache admission bind the selected dependency profile and resulting module. Errors: different profiles cannot share incompatible cache entries, and retained replay with incompatible execution identities remains a classified refusal. | fn-182-gomad-smaller-wasm-guests.2 | — |
| R4 | Retain matched baseline/candidate measurements on the same real Linux/arm64 host for module bytes, compiled-cache bytes, cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS; report repetitions, median/range, host/build identities and rerun commands. Errors: unavailable measurements or differences within measurement spread are labeled as such, with no assumed startup or memory gain. | fn-182-gomad-smaller-wasm-guests.1, fn-182-gomad-smaller-wasm-guests.3 | — |
| R5 | The affected qualified workloads retain success and intentional-failure witnesses, deterministic repetition and fresh retained replay on the trimmed candidate; existing capacity and resource budgets still pass. Errors: behavior changes, missing canary failures or failed replay prevent adoption of the exclusion. | fn-182-gomad-smaller-wasm-guests.3 | — |
