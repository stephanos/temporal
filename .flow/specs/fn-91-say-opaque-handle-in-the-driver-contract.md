# fn-91-say-opaque-handle-in-the-driver-contract Say opaque handle in the Driver contract

> HTML render lens (local): open `.flow/artifacts/fn-91-say-opaque-handle-in-the-driver-contract/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

fn-87 renamed the protocol's `OpaqueCapabilityType` and `capability_slot_id` to `OpaqueHandleType`
and `handle_slot_id`. Its reason: Capability is `Umpire.Capability`, and the runtime concepts say
effect handle. It deliberately left out the hand-written Driver contract. The Go seam that mints,
publishes, consumes and invokes that handle still says capability. The fn-87 `.3` summary lists the
names it kept (`OpaqueCapability`, `CapabilityEffect`, `CapabilityBridge`, `CapabilityFactory`,
`NewCapability`, `InvokeCapability`, the server's `opaqueCapability`), and `UMPIRE4_ORDER.md` carries
the gap forward as "not yet a spec".

This breaks SEM-19 (one word per concept) at the one seam every Driver implements. A reader holds
`Slot_OpaqueHandle` from the wire and passes it through `OpaqueCapability`. The glossary
(`tools/umpire/CONTEXT.md`, **Slot**) already lists "opaque capability" under _Avoid_, and the
READMEs already say "opaque handle" and "handle bridge", so the code is the only layer still out of
step. The rename is mechanical and preserves behavior. It lands before fn-79's cancellation work
resumes against this seam.

## Architecture & Data Models
<!-- scope: technical -->

An **opaque handle** is an effect handle that has not started yet. A Driver mints it around a
`HandleEffect`, publishes it into a handle Slot, and the Executor consumes and invokes it, which
returns the `EffectHandle` of the started effect. The Go names use "handle", not "effect handle":
`EffectHandle` already names the started effect's `Wait`/`Cancel`/`Drain` handle, and the wire
(`OpaqueHandleType`), the IR (`Catalog.OpaqueHandleType`) and the READMEs already say "opaque
handle".

### Renames

| Today | After | Where |
| --- | --- | --- |
| `OpaqueCapability` | `OpaqueHandle` | contract leaf, facade alias |
| `CapabilityEffect` | `HandleEffect` | contract leaf, facade alias |
| `CapabilityBridge` | `HandleBridge` | contract leaf, facade alias |
| `Session.InvokeCapability` | `Session.InvokeHandle` | Driver `Session` interface and every implementation |
| `worker.CapabilityFactory` | `worker.HandleFactory` | SDK worker Driver |
| `worker.SessionOptions.NewCapability` | `NewHandle` | SDK worker Driver |
| server `(*Session).NewCapability` | `NewHandle` | server Driver |
| server `opaqueCapability`, `capabilitySlot`, `capabilityClaim` | `opaqueHandle`, `handleSlot`, `handleClaim` | server Driver |
| server `Session.capabilities`; the `capability` fields of the slot and claim | `handles`; `handle` | server Driver |
| locals, parameters and fake-bridge fields named `capability` that hold an opaque handle | `handle` | Executor scheduler, worker interpreter, composite Session, test fakes |
| server test helpers (`capabilitySession`, `capabilityEffectFunc`, `successfulCapabilityEffect`, `blockingCapabilityEffect`, `capabilityValue`) and the five `Test*Capability*` functions | the same names with `Handle` | server Driver tests |
| test Slot ID literals `"capability"` and `"private-capability"`, scheduler test mode `"nil-capability"` | `"handle"`, `"private-handle"`, `"nil-handle"` | Go-built test Programs only |
| server source and test files `capability.go`, `capability_test.go` | `handle.go`, `handle_test.go` | server Driver (`ownership_test.go` keeps its name) |

`Bridge`, `Publish`, `Await` and `Consume` keep their names.

**Local-name collisions.** A renamed local or parameter never takes a name another binding in the
same scope (parameters included) already has, and no name is reused for a different handle kind.
Some scopes already bind `handle` to a started `EffectHandle` or `handles` to `ReservationHandle`s;
those keep their names, and the opaque-handle binding there becomes `opaque`. The server's
`InvokeHandle` holds three kinds at once: its parameter (the consumed claim) becomes `claimed`, the
claim's minted handle local becomes `opaque`, and the started `EffectHandle` stays `handle`. A
scope with no competing binding uses `handle`, including the composite Session's `InvokeHandle`
parameter.

### Measured surface (2026-09-26)

| Class | Occurrences | Files |
| --- | --- | --- |
| Exported identifiers (six contract/worker/server names above) | 123 | 25 |
| Unexported server types and test helpers | 54 | 5 |
| `capability`/`capabilities` locals, fields, comments, diagnostics in the handle sense | ~115 | 14 |
| Test Slot ID literals | 22 | 6 |
| Lean (`model/`), conformance corpus JSON, proto, Umpire lowering | 0 | 0 |

Lean and the conformance corpus need no edits: fn-87 already renamed the wire, Authoring
(`handleSlot`, `Types.opaqueHandle`) and the fixtures. The only Umpire hit is the scripted replay
test Session.

## API Contracts
<!-- scope: technical -->

```go
// package contract (re-exported unchanged in shape by package testpilot)
type OpaqueHandle interface{}
type HandleEffect interface {
	Accepts(context.Context, *testpilotspb.Instruction, proto.Message) bool
	Invoke(context.Context, proto.Message, int64) EffectResult
}
type HandleBridge interface {
	Publish(context.Context, Coordinate, string, OpaqueHandle) error
	Await(context.Context, string) error
	Consume(context.Context, string) (OpaqueHandle, error)
}
type Session interface {
	// ...unchanged methods...
	InvokeHandle(context.Context, Coordinate, OpaqueHandle, proto.Message) (EffectHandle, error)
	Bridge(context.Context) (HandleBridge, error)
}

// package worker
type HandleFactory func(context.Context, testpilot.Coordinate, testpilot.HandleEffect) (testpilot.OpaqueHandle, error)
type SessionOptions struct { Bridge testpilot.HandleBridge; NewHandle HandleFactory /* ...unchanged... */ }

// package server
func (s *Session) NewHandle(context.Context, testpilot.Coordinate, testpilot.HandleEffect) (testpilot.OpaqueHandle, error)
```

Signatures, method sets and error values are otherwise identical. No alias under the old names
stays behind. Diagnostic text in the handle sense changes: `nil capability` becomes
`nil opaque handle`, `capabilities cannot be inspected` becomes `opaque handles cannot be inspected`
(expression and path binding), and the two test-Session error strings say handle. No checked-in
fixture or `expected.json` contains any of these strings.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Importers.** All importers are in this module. Implementations of `Session` are the server,
  worker and composite Sessions, plus the test Sessions in the facade conformance test, the
  composite Driver test, the scheduler test, two `tests/testcore/testpilot` artifact tests and the
  Umpire replay test. `tools/canary`'s `fencedSession` embeds `testpilot.Session` and overrides only
  `InvokeRPC`, so it compiles unchanged. There is no external consumer to keep aliases for.
- **Other senses stay.** These uses of "capability" name something else and are not edited:
  `Umpire.Capability` and every Lean use (`workerStopCapabilityId`, capability requirements),
  `KNOWN_GAP_KIND_CAPABILITY` and the known-gap kind `"capability"`,
  `CapabilityRequirementDefinitionIDs`, the "correlated capability" feature (comments,
  `TestCorrelatedCapability*`, `TestCorrelatedPrepareRejectsUnsupportedCapability`), the Temporal
  API's `WorkerVersionCapabilities`, and the Opcode-sense leftovers (`realizedNexusCapabilities`,
  `TestWorkerProfileAdmitsTheFaultCapability`, "unsupported instruction context or Driver
  capability", "controller capability required", "the capabilities its opcodes require").
- **Gate self-collisions.** A retired `OpaqueCapability` also matches the split spelling
  `"OpaqueCapability" + "Type"` in the protocol test's retired-descriptor list and in the gate's own
  test (the gate skips only its rule file, not its test), so both splits move to a boundary the rule
  does not match, and every new positive pin is itself written split. The gate scans task records
  whose committed status is not `done`; every fn-78 task record is committed as `todo` and
  `fn-78 .1`'s Done summary names `InvokeCapability` and `CapabilityEffect`. fn-78 is closed, so it
  leaves the downstream list rather than having its history rewritten; the gate stops scanning those
  ten records for every retired token, which is accepted for a closed spec.
- **What the gate cannot see.** The gate holds compound identifiers only. The unexported test
  helpers, the `capabilities` field and `capability` locals are not gate tokens, so R1's "none
  remains" is checked by a case-sensitive search of the touched Go trees whose every remaining hit is
  one of the senses listed under **Other senses stay**.
- **Concurrent work.** fn-79 (deferred) plans cancellation "through task 1's generic server seam"
  in prose. It is not edited, and its re-plan reads the new names. fn-90's live loops run on the
  same packages; the rename commit rebases over whatever fn-90 has landed and does not sweep in
  another session's uncommitted `.plans` or `.flow` edits.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The contract leaf, the facade aliases, every `Session` implementation, the worker factory
  and options, and the server Driver expose exactly the API Contracts shapes; no exported or
  unexported Go identifier in the Renames table's "Today" column remains, and no scope binds two
  different handle kinds to one local name (Local-name collisions). Errors: an implementation left
  on the old method name fails to compile as a `Session`, which is the check; an alias kept under an
  old name fails R3; an unexported helper, field or local the gate cannot see is caught by the
  case-sensitive search in Edge Cases, whose remaining hits must all be other-sense uses.
- **R2:** Behavior is unchanged. `make umpire-check-case-runtime-conformance`,
  `make umpire-check-testpilot-protocol` and `make umpire-check-testpilot-authoring` pass with no
  fixture, `expected.json` or generated file changed, and every Driver ownership, closure,
  quarantine and race test passes under its new name. Errors: any diff under the conformance corpus
  or generated protocol code is a failure, not a regeneration.
- **R3:** The retired-vocabulary gate holds `OpaqueCapability`, `CapabilityEffect`,
  `CapabilityBridge`, `CapabilityFactory`, `NewCapability`, `InvokeCapability`, `CapabilitySlot` and
  `CapabilityClaim` (their lowerCamel rule covers the server's unexported types), and its test pins
  them with live negatives `OpaqueHandleType`, `Umpire.Capability` and `KNOWN_GAP_KIND_CAPABILITY`.
  The comment calling `CapabilityBridge` and `CapabilityEffect` the live seam is replaced. Errors: a
  reintroduced old name in any scanned tree fails `make umpire-check-retired-vocabulary` with its
  path and line; a token rejected as a bare word is not added.
- **R4:** Doc comments and READMEs in the handle sense say opaque handle, handle effect, handle
  bridge or handle factory, including "generic capability factory" in the Temporal Driver README
  and "the capability bridge" in `UMPIRE4_COMPONENTS.md`'s Driver-vocabulary sentence. Diagnostic
  text follows API Contracts. Errors: none beyond R3; prose in the other senses listed in Edge Cases
  is left as is.
- **R5:** `UMPIRE4_ORDER.md` carries no "Driver contract still says capability" bullet (removed
  when fn-91 was queued, so this holds before the work starts), and fn-78 leaves the gate's
  downstream list. Errors: no `.plans` document other than R4's `UMPIRE4_COMPONENTS.md` phrase, and
  no spec or task record, is edited.
- **R6:** `make umpire-check-regression` exits 0 with the passing live-identity count it reports at
  the base commit (recorded in the evidence before the change), and `make lint-code-fast` reports
  no new issues. Errors: a live identity that also fails at the
  base commit (the intermittent failures fn-90 tracks) is re-run, not waived.

## Boundaries
<!-- scope: business -->

- Hand-written Go and prose only; no protocol, Lean, fixture or generated-code change.
- No rename of the Opcode-sense or correlated-capability-sense leftovers; they belong to a separate
  SEM-19 sweep if one is wanted.
- No change to `EffectHandle`, `ReservationHandle`, `Bridge`, `Publish`, `Await` or `Consume`.
- No compatibility aliases, no glossary amendment (CONTEXT.md already states the rule).

## Decision Context
<!-- scope: both — conditionally substructured -->

The contract uses "handle" rather than "effect handle" in identifiers because `EffectHandle` is taken
by the started effect. An `OpaqueEffectHandle` beside `EffectHandle` would put two nearly identical
names on one seam for things with different lifecycles. `OpaqueHandle` matches the wire's
`OpaqueHandleType` word for word. `HandleEffect` can read as a verb phrase, but it keeps the
`Handle*` family (`HandleBridge`, `HandleFactory`) that the READMEs already use. Rejected alternatives:
`DeferredEffect` (a new word, which SEM-19 forbids adding), keeping deprecated aliases (every
importer is in this module and the gate would have to allow them), and folding the Opcode-sense
leftovers in (a different concept, and the request limits scope to the effect-handle sense).


Ordering against fn-79 is carried by `UMPIRE4_ORDER.md` rather than a spec edge: fn-79 is deferred
and its re-plan reads whatever names are live. `.plans/index.json` is not edited for it here.

## Quick commands

```bash
go vet -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/replay/... ./tools/canary/...
go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/replay/... ./tools/umpire/internal/retiredvocabulary/...
make umpire-check-retired-vocabulary
```

## Early proof point

Task fn-91-say-opaque-handle-in-the-driver-contract.1 proves the rename is mechanical: every
`Session` implementation compiles on `InvokeHandle`, the Driver ownership tests pass under their new
names, and no fixture or generated file moves. If it forces a behavior or fixture change, stop and
re-check the Renames table against the seam before .2 adds gate tokens.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | API Contracts shapes; no Today identifier remains; no local-name collision | fn-91-say-opaque-handle-in-the-driver-contract.1 | — |
| R2 | Behavior unchanged; conformance, protocol and authoring checks with no fixture diff | fn-91-say-opaque-handle-in-the-driver-contract.1, fn-91-say-opaque-handle-in-the-driver-contract.2 | — |
| R3 | Gate holds the eight tokens with pins and live negatives; live-seam comment replaced | fn-91-say-opaque-handle-in-the-driver-contract.2 | — |
| R4 | Doc comments, READMEs and the COMPONENTS phrase say handle | fn-91-say-opaque-handle-in-the-driver-contract.1, fn-91-say-opaque-handle-in-the-driver-contract.2 | — |
| R5 | No ORDER bullet; fn-78 leaves the downstream list; no other planning edits | fn-91-say-opaque-handle-in-the-driver-contract.2 | ORDER half already holds |
| R6 | Regression exit 0; lint-code-fast no new issues | fn-91-say-opaque-handle-in-the-driver-contract.2 | — |


