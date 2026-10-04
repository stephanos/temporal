# World terminal purity: admitted corrective design

Read-only world_terminal_purity_scout grounded this decision in recording.go,
errors.go, World error constructors, process/session.go, docs and all callers.
Requested gpt-6.1-sol / high; actual metadata unobservable; judge once unavailable
(no_key). No edits/tests/builds/package loading/Flow/Git/bridges by the scout.
The conductor also inspected the original FinishError body, concrete error
types, process forwarding, recording test and purity guidance.

## Actual contract conflict

Recorder.FinishError calls arbitrary Error before errors.Is, which may dispatch
custom Is/Unwrap. Unknown rejected errors also execute callbacks through %w.
Eight exported error-typed sentinel variables can be rebound, and model formatting
of those globals supplies a second callback path. Arbitrary callback semantics
inside this operation are inherently incompatible with a callback-free pure
World; they cannot both be retained through an exemption.

README promises typed World error reporting. The only direct recorder consumer
passes Register's CapacityError. The other consumer is Runner's child replay
fixture through Session.FinishError, passing Register/Quiesce errors or external
fmt.Errorf wrapping ErrReplayDivergence. Process Session is the sole production
forwarding site; Open is the sole production StartRecording caller. No application
production caller submits arbitrary custom errors to Recorder in this checkout.

## Selected repair and deliberate migration

Keep the direct FinishError signature and convenience for original immutable
World sentinels, concrete CapacityError/ReplayDivergenceError and private
model-generated classified wrappers. Project owned kind/detail without arbitrary
interface dispatch. Add FinishTerminal(Terminal) for detached validated terminal
data using the existing finishing transition. Use immutable private sentinel
identities/fixed messages internally; public default sentinel values and ordinary
typed Error/Unwrap/Is/As contracts remain usable.

FinishTerminal is explicit error-terminal input: capacity, replay-divergence or
invalid-input with nonempty detail. Reject an empty/inferred kind; the existing
Finish operation remains the unchanged quiescence-inference seam. No new detail
size bound is admitted. Reject typed-nil concrete errors safely without callbacks
or state change, rather than dereferencing them in the closed convenience path.

World-generated classified/context wrappers carry private detail, kind and
cause data; keep ordinary messages, typed cause relationships and recorded
bytes. General callback-based normalization belongs to the already effectful
world/process reporting boundary. Preserve its Error-before-Is category order
(capacity, replay, six invalid-input sentinels), unknown %w wrapping, nil input,
session validation and descriptor cleanup precedence, then pass detached Terminal
to Recorder. Do not perform classification earlier merely to simplify code.

Explicitly reconcile direct custom/external-wrapper recorder behavior and public
sentinel rebinding: arbitrary callbacks must run at the reporting/caller boundary,
not the pure model. Direct Recorder.FinishError cannot preserve the former custom
callback-derived message or acceptance, and must reject without callbacks or state
mutation. Existing code can classify outside World and call FinishTerminal; the
process seam continues general error reporting. This is an inventoried intentional
behavior/API migration, not a claim of unchanged arbitrary-input behavior.

Replacing FinishError's parameter outright would break direct source callers;
moving the whole recorder to an effectful package would widen ownership. The
selected closed convenience plus detached boundary preserves documented typed
capabilities and all identified callers without either alternative.

## TDD and evidence bounds

Before production repair, retain real callback/effect RED: custom Error/Is/single
and multi-Unwrap callbacks, rejected unknown input, and sentinel rebinding in a
serial restoring test. Corrected core must invoke no such callbacks, mutate no
recording state on rejection, and remain usable afterward. No blanket callback,
error, fmt, errors.Is or World purity exemption.

Preserve all eight original classifications/details, capacity/replay fields,
model-generated invalid/context messages and Is/As cause relationships. Reporting
tests cover multi-category joined/wrapped/custom-Is inputs and an Is method that
changes later text to detect reordered detail capture. Preserve nil/invalid-session/
unknown-error/cleanup ordering and wrapped error identity.

Compare complete recording bytes, snapshots/digests and composed manifest hashes,
including escaped/non-ASCII/long details without truncation or new bounds. Keep
matching/divergent child replay assertions and the complete terminal comparison;
incomplete replay remains accepted only for TerminalReplayDivergence. Supported
native process qualification is still required; compile/static/developmental
evidence cannot claim it.

Scope is limited to World core error/recording constructors, process terminal
reporting and their tests, existing replay test assertions, the relevant README/
ARCHITECTURE boundary text and interface inventory. Codec changes are only for
the same model-owned error conversion when required. No new model operation,
policy grant, replay format or Runner production change. The sole task-19 writer
owns implementation; the conductor owns Flow/docs-of-decision. User autonomy
chooses this grounded recommendation and leaves commits with the user.
