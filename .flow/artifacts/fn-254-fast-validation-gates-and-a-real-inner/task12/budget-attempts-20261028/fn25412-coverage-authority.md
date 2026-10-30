# fn254.12 minimum TLC coverage authority

Use a compiled observer capability declaration and bind each run to its computed source/tool context. Do not require a checked-in allowlist of reviewed whole-tree source hashes. That earlier proposal would reject ordinary Model/test edits and defeat the edited-source inner loop; this recommendation supersedes it.

## Trust and specification

The current contract trusts checked-out development source and admitted toolchains to implement the producers correctly. Normal code review maintains the invariant that real TLC calls use the observed backend and that checker-free test adapters do not acquire an unobserved checker-launch path. Runtime recording establishes what that trusted source executed. A source digest identifies the actual input and detects changes during an attempt or retry; it is not a claim that every byte was separately reviewed in advance.

fn254 R8 requires actual work counts, hard limits and explicit reviewed budget changes. It does not require a source-review whitelist. The six-field receipt contract likewise requires complete producer evidence, not a preapproved SHA for every Model edit. A reviewed producer protocol/capability implementation is sufficient within the existing source trust boundary. This is not protection against malicious source deliberately bypassing its observer, just as current test counts do not defend against malicious count producers.

## Exact API

Compile the declaration into the existing observer implementation, with closed variants:

```rust
enum CoverageCapability {
    RustTlcObservedV1,  // nextest and doctest; all real TLC through observed backend
    GoCheckerFreeV1,   // actual admitted Go host test adapter/module closure
}
struct CoverageAdmission {
    capability: CoverageCapability,
    producer_protocol: u8,
    controller_image_sha256: String,
    inputs_sha256: String,          // current computed AdmittedInputs closure
    command_sha256: String,
}
fn admit_coverage(
    command: &Command,
    capability: CoverageCapability,
    inputs: &AdmittedInputs,
) -> io::Result<CoverageAdmission>;
fn verify_coverage_unchanged(
    admitted: &CoverageAdmission,
    inputs: &AdmittedInputs,
) -> io::Result<()>;
```

These can be private helpers in `work_counts`, or one narrow sibling module called by it. `AdmittedInputs` remains the existing frozen current source/tool/mode/selection context. Compute its digest from the actual checkout at admission. A normal edit before admission produces a different valid digest; changing inputs during the run or between comparable retries refuses. Bind the real early-pinned controller image and compiled protocol version separately from current checkout bytes; never rehash a mutable Cargo alias as the running image.

The adapter chooses the capability in code from the actual supported command/module/runner path. Neither environment, stage name nor budget table grants it. `admit_coverage` checks the command belongs to that supported path and its execution-changing settings fit the existing admitted tool policy. Uncovered wrappers, alternate executables and overlays stay unknown until the adapter explicitly covers them. This check protects the declared capability boundary without pinning every source revision.

## Concrete hooks and evidence

In `observe_streamed`, select `RustTlcObservedV1` only for actual nextest/doctest observation. In `observe_go`, resolve effective GOFLAGS/module/tool context first, then select `GoCheckerFreeV1` for the actual admitted host adapter. Open a coverage scope linked to the existing `tests-begin` ID before dispatch. Close only after owned completion, complete runner parsing, `tests-finish` publication and `verify_coverage_unchanged`.

The scope record contains `version, scope_id, test_producer_id, capability, producer_protocol, controller_image_sha256, inputs_sha256, command_sha256` and the completed producer reference. It duplicates no numeric counts. Every actual test producer in a completed dispatch needs a matched open/close before the collector can report observed TLC zero with no attempts. Missing/partial/crossed scopes stay unknown. Existing real/fixture/unclassified events remain authoritative; a coverage close cannot overwrite finish0 or an unmatched/corrupt attempt.

The current source basis is concrete. Real TLC probe/model sites use `run_process_observed` through typed `TlcInvocation`. Go native parity tests read `.pb`/`.methods` files, and Wasm calls execute in-process through `host.go`'s six-import allowlist with no process-spawn capability. Earlier guest/native build subprocesses belong to their Rust stages. Normal review must preserve these routing/capability invariants when source adds a new subprocess or checker path; a new producer implementation changes its capability/protocol version when the contract changes.

Current Go parsing accepts `-toolexec`, `-overlay`, `-modfile`, `-pkgdir` and compiler/linker flags. Their syntactic validity does not imply coverage. Preserve caller execution contracts, but require a supported admitted context before closing a checker-free scope. No toolchain or dependency-cache adversarial proof beyond the project's existing trusted tool/module policy is introduced.

Root needs to choose the two capability variants and command-policy boundary, then implement and control these hooks. No separate source certificate, per-edit certificate refresh or source SHA review gate is required. No code, runtime, build or Flow operation was performed for this revision.
