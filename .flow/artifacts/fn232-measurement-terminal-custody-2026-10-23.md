# Native terminal admission, publication timing and preparation custody

This additive source batch follows `9c1b44db3b`. It does not certify complete
119-root preservation, full Temporal execution or fn-185 qualification.

## Source behavior

Terminal catalogue reports now use `gomad.native-terminal-exit/v2`. Six ordinary
Release successes retain their original helper admission. Two controls execute
against a separately admitted diagnostic-profile helper, supplied explicitly by
`--terminal-exit-diagnostic-helper` and
`--terminal-exit-diagnostic-helper-receipt`. Their admission is retained as
`diagnostic_control_admission`; distinct role implementation identities are
preserved. These controls do not assert an unobserved Release-only predicate.

Both controls use the current actual module. The digest control changes only its
declared SHA and receives the actual `module SHA-256 mismatch` before engine
creation. The loaded descriptor control uses a 64KiB request bound against the
same module's larger declared initial memory, receiving actual `capacity` with
`initial guest memory exceeds limit` before Store/instantiation. A module with at
most one initial page cannot supply this control and is refused explicitly.
Raw request, module catalogue, report and actual diagnostic profile are retained.
The descriptor refusal also retains its actual Outcome and measured compilation
metrics. Digest refusal has no fabricated Outcome or phase metrics.

The special admission-refusal profile validator retains the complete existing
schema, bounded distribution and reconciliation checks, and additionally requires
zero instantiation, execution and import observations. Ordinary execution profile
validation still requires an execute interval. Independent profile parsing is
available to an ordinary controller; execution collection remains compiled only
for diagnostic features. Attribution names share a single feature-independent
inventory, rather than copying the collector or relabeling its role.

`linked::Execution` and the raw report retain the actual terminal kernel timestamp
as `terminal_monotonic_nanos`, with `terminal_clock: CLOCK_MONOTONIC`. The generic
process observer records `reaped_monotonic_nanos` immediately after actual child
wait/reap. The production terminal reader checks the matching child PID, the
shared clock, clean process cleanup and exact positive subtraction bounded by
1,000,000,000ns. This interval includes raw stream/profile/report publication and
child exit; it does not substitute linked-return cleanup for process exit.
The existing linked cleanup interval remains checked separately. Terminal v2
retains `process_exit_observations` for ordinary successes and `process_exit` in
each admission-refusal observation, including the raw driver observation path.

Measurement preparation publishes actual `guest.auxiliary` through the existing
private, bounded preparation observation sidecar and reopens it before returning.
`preparation-observations.json` binds its artifact root, the actual provenance
record digest and receipt path. Module roots are private and descriptor-pinned.
Reconstructed namespace samples require nonempty observations and authenticate
an exact complete receipt digest anchored in their common receipt. This permits
a distinct immutable baseline executable to verify the originating producer's
host/helper identities without claiming to be that producer. The default
preparation opener remains strict about the current host/helper origin. The
explicit cross-origin opener verifies original receipt bytes first, then retains
all module, provenance, compiler/source inventory, record and closed payload
commitments.

## Executed checks

All Cargo commands used pinned Rust 1.99.1 and only the assigned target
`/home/agent/.cache/codex-build/fn232-measurement-new`.

- Full native library, original bounds, serial execution: **462 passed, 3 ignored,
  exit 0**, `/home/agent/measurement-terminal-followup-full-serial.log`.
- Strict all-target Clippy under ordinary, diagnostic-profile, guest-profile,
  diagnostic-fuel-off and determinism-fixture: **all 5 commands exit 0**,
  `/home/agent/measurement-terminal-followup-clippy.log`.
- Actual public native/profile corpus: diagnostic **6+1 passed**, guest
  **7+1 passed**, fuel-off **7+1 passed**, all exit 0,
  `/home/agent/measurement-terminal-followup-public-matrix.log`.
- Actual terminal production observation/supervisor/publication controls:
  **15 passed, 1 child-only helper ignored, exit 0**,
  `/home/agent/measurement-terminal-followup-fixture.log`. The new real WASM child
  test accepts prompt publication and actual 1.1s preparation, but rejects actual
  1.1s post-terminal publication through the production terminal-to-reap consumer.
- Explicit pinned Go compiler/cache/custody probe: **1 passed, exit 0, 43.73s**,
  `/home/agent/measurement-actual-preparation-custody.log`. Actual first/second
  preparation observations survive measurement publication and exact readback.
  Retained root:
  `/home/agent/wt/evidence/fn232-measurement-preparation-custody-20261023-01`.
- Actual production Runtime preparation, full namespace reconstruction and
  sidecar mutation/removal refusals: **1 passed, exit 0, 36.94s**,
  `/home/agent/measurement-actual-runtime-custody.log`. The effective prepared
  choice limit is asserted as the original 64MiB. This uses the real
  `module::prepare` and `namespace_driver::load_prepared` consumer, not a publisher
  fixture alone. Current cooperative runtime module/provenance/installation/raw
  observations are retained under
  `/home/agent/wt/evidence/fn232-measurement-runtime-custody-20261023-01/module`.
- Pinned foreign-origin receipt acceptance plus eight refusal mutations:
  **1 test passed, exit 0**, `/home/agent/measurement-origin-custody-tests.log`.
  The normal opener rejects a foreign origin. Exact pinned bytes permit it;
  changed receipt/module/provenance/payload, missing directory, extra file,
  wrong record and invalid digest are refused.

An earlier parallel full suite retained **461 passed, 1 failed, 3 ignored** in
`/home/agent/measurement-terminal-followup-full.log`: an existing cold-cache trap
fixture hit its unchanged 5s watchdog under concurrent build/test load. The
serial rerun passed with that same budget and unchanged expected terminal.
No timeout was enlarged and no canonical result was normalized to hide it.
Formatting and diff whitespace checks also passed.

Fresh independent review in the existing separate gpt-6.1-sol/high context read
the source and completed green outputs and returned **READY**. It found the
reconstructed namespace custody bypass during development; the actual consumer
mutation/removal probe above verifies its repair. This is an independent context,
not a different model family. Source file hashes are retained separately at
`/home/agent/measurement-terminal-followup-source-sha256.txt`.

## Integration and remaining acceptance

Root helper-role fixes after `9c1b44db3b` must remain intact when receiving this
batch, including conformance/socket guards, retained-only World assertions and
fixture cleanup. Root moved the old diagnostic name functions above its tests;
this batch instead extracts those unchanged functions into `profile_inventory`.
Resolve that ordering overlap by retaining the common inventory and diagnostic
re-export, not duplicate function bodies. New Execution fields need `None` in
any additional root-only mechanical initializers.

Actual full Temporal catalogue terminal execution with sealed ordinary and
separate diagnostic admissions, complete frozen publication qualification and
final profile-reader execution remain pending. The actual helper API/control
fixtures above do not replace those runs. Runtime's original Go-owner/helper-owner
comparison belongs to the independently integrated immutable reference wrapper.

The five original `^TestTerminalExit` tests in frozen
`tools/gomad/internal/backends/wasm/wasi/process_test.go` remain separately
accounted for: MeasurementControls, ObserverReceivesCompleteAcceptedFrameBeforeValidation,
AuthorizationMismatchAfterStarted, MeasurementSendsCapturedInputsBeforeStarted,
and ExpiredContextDoesNotLaunchHelper. Native controls cover actual timing,
missing/invalid typed timing, death/watchdog/reaping and publication. Original
partial/duplicate frame, authorize-frame, captured-input handshake and expired
transport launch predicates retain their explicit immutable Go reference proof,
owned by the final-profile worker. No native Started frame is invented. Wire-only
private revision strings have no mutable linked counterpart and are not silently
covered by role admission tests. Catalogue stress adaptation, remaining original
producer matrix, fn-232.13 frozen qualification and fn-185.2/.3 remain open.

The actual Go compiler lease was released after both probes. Only this worker's
inactive incremental build cache was removed to reclaim 16GiB; binaries, logs,
retained source worktrees and evidence roots remain available. No other worker's
target was used or removed, and no Flow lifecycle or push operation was performed.
