# fn-254.12 count design

Recommendation for the root-owned API decision. This source-grounded design closes the known synthetic attribution gap with typed invocation provenance and explicit fixture authority, and represents actual inapplicability separately from missing measurements. It needs no change to the audited child exec path, Model semantics, test selection or stage DAG. No runtime, build or acceptance claim accompanies this report.

## Decision

1. Keep the canonical `WorkCounts` in `gate/work_counts.rs`. Adapt budget admission to consume its observation evidence. Do not import the duplicate six-field donor type as another source of truth.
2. Retain all TLC-looking attempts. Give new attempts an explicit origin, and preserve the legacy unclassified path. A private `Tlc` origin distinguishes admitted checker work from closed protocol fixtures. A synthetic origin needs a linked, integrity-checked fixture authority; a bool, environment variable, script path or missing banner does not qualify.
3. Interpret checker startup from recognized TLC startup output associated with an admitted invocation. Preserve three actual-checker outcomes, `Started`, `ProvenNotStarted` and `Unknown`. Synthetic protocol applicability is a separate positively evidenced classification. Legacy `tlc-finish ... 0` remains `Unknown`.
4. Add explicit observation status for test counts, TLC launches and TLC configurations. Structural N/A comes from the actual completed native dispatch contract. A missing observer stays unknown. Budget tables declare each metric as bounded or inapplicable and must match producer evidence.
5. Implement a trusted pre-exec error channel only if fresh evidence finds a required real invocation that must be proven not started after the supervisor fork. The known 84 unconfirmed historical events are source-attributed synthetic families, so such a channel does not address their cause.

## Why these seams fit current code

`process/work.rs:81` labels any command with an argument equal to `tlc2.TLC`. `support.rs:469-472` calls it after spawning the owned guardian. `work.rs:103` checks captured startup text and writes a boolean finish. The collector at `work_counts.rs:324` makes the entire aggregate unknown when any finish is false. It correctly refuses unmatched attempts and preserves the distinction between an attempted fork and a confirmed start.

`runner.rs:139-169` admits Java/JAR and immediately flattens `PinnedJar` into path fields. Its two executable sites are the admission probe at line 146 and the real/synthetic common execution at line 561. `batch_tests.rs:17` and `cache_tests.rs:121` directly build `Tlc` with scripts. Their commands therefore look identical to real Java at the current observation boundary. `cache_tests.rs:389-424` selects a real-Kettle branch using `Tlc::discover`; synthetic authority must be created only after that branch returns.

`PinnedJar` is useful provenance, not an existing capability. Its fields are public (`jar.rs:11-18`), and its documentation explicitly says admission does not recheck later replacement (`jar.rs:44-49`). Simply storing `PinnedJar` or accepting a caller-supplied enum does not prove executable identity or synthetic nonexecution.

The same owned process runner already supports a clean target environment (`support.rs:193-217`). Ordinary `spawn` ignores `Command::env_clear` and inherits the environment (`support.rs:170-186`). `java_command` removes only JVM variables (`support.rs:109-116`). Current fake Bash scripts can consequently run inherited `BASH_ENV` or exported functions. Their script bytes alone cannot support a closed synthetic authority. Select `spawn_clean` in the parent runner for admitted fixture commands and provide only required fixed values. This reuses existing lease/lifecycle custody.

## Concrete invocation design

Use a private backend-owned origin on `Tlc`, conceptually:

```rust
enum InvocationOrigin {
    AdmittedTlc(CheckerIdentity),
    ProtocolFixture(FixtureAuthority),
    Unclassified,
}
```

`discover` constructs the admitted origin inside the successful `PinnedJar` admission closure, before running the `-h` probe. Both the probe and `execute` pass typed observation context through one new parent-side runner entry point. Existing `run_process` callers keep their behavior. An unclassified command containing `tlc2.TLC` retains the current conservative observation path. Neither script basename nor environment chooses the origin.

Keep process support generic. A process-owned observation context can contain checker name, admitted tool identity, configuration/probe identity and an optional fixture-authority reference. The TLC adapter owns which context it supplies. `process` must not import TLC or any system Model; module-map ownership already puts bounded execution and event publication there.

The producer keeps one unique attempt ID across origin, terminal outcome and authority reference. Version new event records rather than changing the meaning of v1. The collector rejects unknown versions, mixed versions for one attempt, orphan or duplicate terminal events, malformed hashes and crossed scopes. The retained v1 collector controls remain unchanged.

An admitted-origin attempt plus recognized startup output yields `Started` even if later evaluation, resource refusal, cancellation or cleanup fails. A started probe increments launches and contributes no model configuration. Started configuration `-` leaves configurations unknown. A real admitted attempt without startup evidence stays `Unknown`, including successful Java exec, exit 1/127, empty output, timeout, output truncation and nonUTF8 output. Terminal publication should occur through the parent finalization path for all captured results; observer failure remains a count error and cannot be swallowed by an expected backend refusal. Where safe UTF8 decoding fails, record an unknown terminal cause rather than silently inventing zero.

A fixture completion does not become a real `Started` event. It supplies positive synthetic applicability evidence and remains recorded as executed protocol work. Aggregate actual TLC starts sum confirmed real starts only after every attempt is accounted for by real evidence or validated fixture authority. Any unclassified or real unknown still poisons certainty. Do not subtract a historical expected 84 or hardcode 752.

## Fixture authority requirements

The constructor must be test-private in the TLC crate, reachable only from the reviewed batch/cache fixture helpers. Passing an arbitrary script into a public `synthetic=true` factory is insufficient. Pin a closed fixture family/template and typed, safely quoted data. Retain the exact rendered bytes or their retained artifact reference plus hash. The trusted constructor identifies why the template can produce protocol bytes and filesystem/control events but cannot launch Java/TLC. Hashing an arbitrary script is identity evidence, not a proof of this property.

The linked authority binds at least version, source/template identity, test identity/case, current test executable identity, canonical owned script path, rendered script digest, interpreter identity, environment policy, attempt IDs, process terminal dispositions and the fixture assertion-completion outcome. Bind it to the invocation/scope and include its digest in the parent work evidence. Validate script identity before execution and after owned cleanup; reject replacements, symlinks, missing authority, duplicate links and crossed identities. The source/tool/budget pin used for retry also binds this authority.

Use the existing `OwnedProcess::spawn_clean` for synthetic target execution. Preserve parent observation scope and receipt publication; those are parent operations and need no fake-process exemption variable. The cache script needs a fixed trusted PATH for `dirname`/`cat`, and already uses absolute `/usr/bin/grep`. No `BASH_ENV`, exported Bash functions or caller-controlled command lookup enters this target. Keep scripts in uniquely owned scratch. This design assumes the same trusted source/system tools and private-scratch ownership as the current runner. Defending against an adversary rewriting executable bytes between checks would require stronger execution identity custody; this task must not claim that extra guarantee.

Batch helper bodies are currently open strings assembled at five reviewed test families. Prefer a small test-only closed family enum that owns rendering over a generic body-taking authority factory. Cache helpers share one static script; mutable `mode`, jar identity and Java-version text are typed fixture data. The identity-refresh control deliberately changes jar/version data, so authority must pin the script/template while recording the intended data changes. Treat those changes as fixture cases, not integrity violations of the immutable executable.

Publish a fixture authority that can describe an expected refusal/evaluation/output-limit outcome as a successfully completed test assertion. Existing gate `child_count_receipts` only admit successful/skipped stage receipts (`work_counts.rs:198-238`); do not pretend an expected failed process is a failed gate child. Add a narrow fixture-evidence shape under the work observation collector, or an explicitly typed linked authority variant. Reuse its path/hash integrity rules, without weakening ordinary child gate validation. Parent Rust discovered/executed/ignored/skipped counts remain the unchanged nextest/doctest totals. Protocol processes get no additional Rust test count, and no protocol event disappears by clearing receipt scope.

Keep the real cache-child branch outside the fixture constructor. Selecting `UMPIRE_CACHE_TEST_REAL_KETTLE` chooses the test's real behavior; it must always use admitted `discover` context and count real probes/configurations. The environment variable never grants a synthetic classification.

## Positive applicability and producer completeness

Preserve the six nullable numeric fields for compatibility. Add one receipt-level evidence object with three independently validated statuses, because configurations can remain unknown while launch totals are known:

```rust
struct CountEvidence {
    tests: MetricEvidence,       // covers all four test counters together
    tlc_launches: MetricEvidence,
    tlc_configurations: MetricEvidence,
}
enum MetricEvidence {
    Observed { producer_authority: AuthorityRef },
    NotApplicable { dispatch_authority: AuthorityRef, reason: ApplicabilityReason },
    Unknown { reason: String },
}
```

Names are illustrative; the root owns the final API. `Observed` requires matching numeric data and complete bounded producer evidence. `NotApplicable` requires null numeric data plus a closed typed reason and completed dispatch authority. `Unknown` requires null and fails applicable budget admission. `count_error` continues to describe corrupted/incomplete publication, rather than doubling as applicability. All-four-test totals remain atomic and consistent.

Supply dispatch applicability from the same actual framework/Temporal native match branches that execute each stage (`gate.rs:171-265`, system `gate.rs:240-330`). The branch returns or publishes a typed completed count contract. For example, `cargo fmt --all --check` establishes no test execution and no TLC work; compile-only wasm/clippy/docs dispatches establish their own audited contracts. Generic or unfamiliar commands default to unknown. Avoid an extra stage-name registry whose answer can drift from the executable branch. Keep `tasks.toml` and scheduler's mise inventory as the only graph; count contracts introduce no dependencies, scheduling or runner.

A test-bearing branch requires actual test producer receipts even when its name is `timing-controls`, `capability-controls`, `protobuf-controls` or Temporal `source-boundary`. Go stages require the Go adapter's complete actual inventory/terminal evidence. Do not mark them N/A because the current Rust observer does not recognize them. Likewise, a native boundary helper can claim N/A only if its actual completed operation contains no counted tests/checker dispatches.

For an applicable observer with genuinely zero work, numeric zero needs a positive complete observer-scope authority, with begin/end identity, admitted producer version/source and completed dispatch/runner coverage. It is not derived from an empty event directory. An all-synthetic scope can establish zero real TLC through its complete set of admitted fixture attempts and terminals. An arbitrary empty directory still yields the existing default unknown `WorkCounts`. Test the complete-scope zero case separately from missing-observer controls.

Delegated child work keeps its existing exclusive authority. A standalone parent may declare only its own structural applicability and link independently validated child stage evidence. Verify setup work before claiming parent TLC N/A; `standalone.rs:410-436` includes fetch/build preparation and then a complete child gate. Budget/count aggregation must not charge child elapsed time twice. Child counts never get flattened into parent own counts merely to populate six cells.

Budget declarations should have an explicit closed status for each modality. A bounded entry needs a numeric maximum and `Observed` actuals; an inapplicable entry needs the matching producer N/A reason/contract. Unknown, omitted or mismatched statuses fail immediately. The budget table cannot grant applicability authority, and no `null => 0`, `null => N/A` or observed-positive-with-N/A case passes. Keep hard count failures out of the wall-time retry path. The donor's all-six-numbers requirement is too strict for true build/format N/A and must change, while retaining its refusal of missing applicable counts.

## Why an exec channel is optional here

`syscall.rs:797-846` is the only general point that knows every candidate `execve` failed. `child_error` currently writes stderr and exits 1. A private bounded channel could establish a positive pre-exec failure, but status 1, stderr text and pipe EOF cannot do so. Successful exec does not establish TLC startup and would leave every currently successful fake script ambiguous without fixture provenance anyway.

If later needed, the channel must be prepared before fork, use allocation-free raw child operations, survive the guardian/target descriptor layout, identify the owned target attempt, report pre-exec failure with a complete record, and retain Linux/Darwin ownership/lease/cancellation/completion rules. A killed or partial writer stays unknown. Do not make EOF mean exec success, or success mean TLC startup. Adding this machinery now would expand the sensitive proof surface without resolving the known synthetic classification gap. Before-launch admission refusal can already be positive non-start evidence when published by the owning typed caller before any process launch; it does not require a child handshake.

## Required control matrix

| Control | Expected count evidence |
| --- | --- |
| Retained v1 attempt only | Missing completion refusal |
| Retained v1 finish0, or unclassified script success/exit1/127 | TLC unknown |
| Admitted real probe and model start | Launches include probe; configurations exclude probe and deduplicate source+cfg |
| Real checker starts then refutes/evaluates/refuses | Count actual start; preserve original result |
| Real command without captured startup, including cancel/limit/nonUTF8 | Unknown, never zero |
| Every existing batch/cache protocol success/refusal/evaluation/partial/output case | Full Rust execution count; linked completed fixture assertions; zero real starts only from closed authority |
| Real cache-child selection | Real probe/model attribution; no synthetic authority |
| Modified script/interpreter policy, BASH_ENV or exported-function injection, forged fixture kind/reference | Reject or cleanly exclude injected environment; never lower real counts |
| Missing/corrupt/cross-scope fixture authority or orphan synthetic terminal | Count refusal |
| Fixture identity-refresh mutations | Intended data changes retain original assertions and distinct recorded cases |
| Empty collector directory | Unknown, retained current control |
| Positive complete observer scope with no TLC dispatch | Observed zero with explicit producer authority |
| fmt/build completed contract | N/A with dispatch evidence; no numeric substitution |
| Applicable Rust/Go observer missing | Unknown and budget refusal |
| N/A declared with observed counted activity | Contract contradiction and refusal |
| Known real + synthetic + unknown mixture | Unknown aggregate; all evidence retained |
| Budget count overrun/unknown | Immediate failure; no timing retry |
| Time-only retry | Identical source/tool/fixture/budget/tier; both attempts retained and included in tier cost |

## Implementation boundary and remaining proof

This is a bounded parent-side producer/collector/API change. It needs process support wiring, TLC origin and fixture helpers, work receipt schema/validation, native dispatch contracts and budget adaptation. It does not justify a syscall rewrite, a second runner or a stage graph. Avoid broad `PinnedJar` capability refactoring; a TLC-private origin with invocation identity checks meets this task's attribution need.

Fresh evidence must still prove exact classification of the full selected optimized real test inventory, including all ten source-identified synthetic test families. Historical 836 attempts / 752 confirmed / 84 unknown is context, not a proposed expected-count assertion. The current source search found only two TLC command construction sites and the private direct constructors; real integration controls use `discover`, while the resource-limited scripted cases live in batch tests. No current source requirement demonstrates a necessary post-fork real non-start classification. If fresh execution produces any real unknown, keep the metric unknown and investigate its cause before budget admission.

The immutable fixture authority and completed observer-scope evidence are real implementation requirements, not facts already supplied by current `PinnedJar` or receipts. Root can choose simpler concrete serialization while preserving these distinctions. Full-gate/final timing remains owner-deferred to .8. Migration remains owner-deferred. No tests, builds, Git mutations, Flow mutations, bridges, reviews or remote writes were performed in this lane.
