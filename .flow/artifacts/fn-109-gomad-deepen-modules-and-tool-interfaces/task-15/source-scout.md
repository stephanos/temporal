# Simulation progress source reconnaissance

Fresh read-only thinking scout, requested Codex pin gpt-6.1-sol at high.
Inspected current execution sources after task 13 at 0dd05b313a plus its
uncommitted time-wire candidate. No tests, generation or edits were performed.
This is input to task 15, not its characterization/design acceptance evidence.

The existing arbiter uses aggregate delivered credits and counted external and
handling work. `settleLocked` prevents advancement globally while handling or
delivered work remains; externally blocked participants are excluded from timer
arbitration. Admission, forwarding and transfer normally validate acknowledgements
and act atomically under its mutex. Participant pointer equality protects restarted
incarnations. Transport owns bounded pending/abandoned correlation and cancellation.

Recommend comparing operation handles with a closed typed transition function,
then selecting complete semantic transitions on the existing private progress owner.
Aggregate acknowledgements cannot identify individual operations, so operation
handles would introduce extra correlation without stronger wire semantics. Move
response phases into the progress owner under the same mutex; preserve transport
cancellation and domain mutation with their current owners. Request IDs, host reply
order and map iteration must remain private correlation, never replay identity.

Current sources expose two concrete validation-order concerns for task 16:

- `handleWaitAcceptance` in simulation_unix.go calls runnable before validating
  Arrivals. An unknown acknowledgement can wake an installed waiter with Retry
  while the acceptance returns an error. Task 15 should characterize the current
  negative behavior explicitly; task 16 should add a strengthened regression
  requiring rejection before waking the waiter. Valid-path pins stay unchanged.
- Response-barrier duplication is detected after admitting work and consuming
  acknowledgement credits. Rollback removes new work but cannot restore already
  consumed credits. Validate the complete event before any accounting mutation.

Task 15's missing strongest behavioral cases are two blocking operations on the
same participant (both reply orders); partial acknowledgement of two deliveries;
arrival to an installed quiescent waiter; cancellation with an already committed
model operation and late discarded response connected to coordinator credits;
participant death with a waiter/in-flight work; stale callbacks after restart;
unknown acknowledgement across admission/transfer/wait paths; unknown or duplicate
response rejected before callbacks. Known legitimate abandonment cleanup must not
be confused with rejecting malformed/unknown responses.

The typed owner should collapse admission plus response reservation, forwarded
admission, response delivery, wait suspension/resumption, completion reservation,
model dispatch/arrival/discard and participant removal into complete validated
events. Merely renaming individual counter methods to events does not fulfill R11.
Keep simultaneous operations as counts plus scoped response phases, not one flag.

Existing state-machine, transport and process tests provide useful valid-path pins,
but task 15 still needs executable characterization and a two-design decision.
Task 16 still needs implementation, unchanged characterization tests, strengthened
validation regressions and real process conformance on the required platforms.
The root transport integration includes the existing model-delay watchdog finding;
preserve and attribute its disposition instead of weakening assertions.
