//! # What the two Models say
//!
//! `tests/pins.rs` of crate `temporal-model`. The claims below are about the tables, because the
//! tables are what Search, the Behavior Fingerprint and Contract lowering read. A `match` arm that
//! stopped saying what it says would fail here.
//!
//! Two tiers, like the Lean `#guard` pins but split by where Rust can run them:
//!
//! - `const _: () = assert!(..)` is decided by the compiler, no crate needed: this is a test file,
//!   so `cargo check --tests` or the build step of `cargo test`, before any test runs. Anything
//!   that is an associated `const` qualifies: state-space sizes via `Finite::CARDINALITY`, the
//!   attempt bound, a `const` state literal's fields.
//! - `#[test]` functions run in `cargo test`. Tables, reachability, refinement rows and Query
//!   outcomes need heap and closures, which `const fn` does not have, so they live here.
//!
//! A third tier is a `trybuild` compile-fail test (`tests/ui/*.rs` with a `.stderr` beside each),
//! which pins the text of a macro error the way Lean's `#guard_msgs` does; see the bottom.

use umpire::{Admission, AnyQuery, Finite, Step};

use temporal_model::nexus_caller as nexus;
use temporal_model::standalone_activity as activity;

/// The Lean `checked.run.result.outcome.name`, or the admission error: what every Query pin reads.
fn outcome(query: &dyn AnyQuery) -> Result<&'static str, Admission> {
    query.run_erased().map(|checked| checked.outcome.name())
}

// ---------------------------------------------------------------------------------------------
// Compile time
// ---------------------------------------------------------------------------------------------

// Six product phases; eight protocol phases, three attempt counts and three deadlines. These are
// the same numbers the table pins below check, decided when this file compiles, before any test runs.
const _: () = assert!(nexus::ProductState::CARDINALITY == 6);
const _: () = assert!(nexus::ProtocolState::CARDINALITY == 8 * (nexus::ATTEMPT_BOUND as usize + 1) * 2 * 2 * 2);

const _: () = assert!(activity::ProductState::CARDINALITY == 9);
const _: () = assert!(activity::ProtocolState::CARDINALITY == 12 * (activity::ATTEMPT_BOUND as usize + 1) * 2 * 2 * 2);

// A payload variant contributes one class per assignment: five constructors, six members.
const _: () = assert!(nexus::Reply::CARDINALITY == 6);
const _: () = assert!(activity::AttemptResult::CARDINALITY == 4);

// The count saturates at the bound, and the retry claim's state sits one below it.
const _: () = assert!(nexus::Attempts::LAST.saturating_succ().get() == nexus::ATTEMPT_BOUND);
const _: () = assert!(nexus::SUCCEEDED_ON_RETRY.attempts.get() == 1);
const _: () = assert!(activity::COMPLETED_ON_RETRY.attempts.get() == 2);

// ---------------------------------------------------------------------------------------------
// Nexus
// ---------------------------------------------------------------------------------------------

mod nexus_pins {
    use super::*;
    use nexus::*;

    /// A state written the way a reader names one: the phase, and whichever fields are not at the
    /// value the operation begins with.
    fn at(phase: Phase) -> ProtocolState {
        ProtocolState { phase, ..ProtocolState::first() }
    }

    // Six phases, and the four the design ends on.
    #[test]
    fn product_states_and_ends() {
        assert_eq!(nexusProduct.table().states.len(), 6);
        assert_eq!(nexusProduct.ends().len(), 4);
    }

    // Every action class the machine steps on: six replies, three resolutions, the two faults it
    // cannot see, and the one timer.
    #[test]
    fn product_action_classes() {
        assert_eq!(nexusProduct.action_keys().len(), 12);
    }

    // A retryable handler error is invisible here: it is the protocol machine that backs off.
    #[test]
    fn retryable_error_is_invisible_to_product() {
        let scheduled = ProductState { phase: ProductPhase::Scheduled };
        assert_eq!(handler_reply_step(&scheduled, Reply::HandlerError { retryable: true }), vec![]);
    }

    // What the Model actually reaches: every phase.
    #[test]
    fn product_reaches_every_phase() {
        let mut phases: Vec<_> = nexusProduct.reachable().into_iter().map(|s| s.phase).collect();
        phases.sort_by_key(|p| p.key());
        let mut all = ProductPhase::all();
        all.sort_by_key(|p| p.key());
        assert_eq!(phases, all);
        assert_eq!(nexusProduct.stuck(), None);
    }

    // Eight phases, three attempt counts and three deadlines, and the four phases the design ends on.
    #[test]
    fn protocol_states_and_ends() {
        assert_eq!(nexusProtocol.table().states.len(), 8 * 3 * 2 * 2 * 2);
        assert_eq!(nexusProtocol.ends().len(), 4 * 3 * 8);
        assert_eq!(nexusProtocol.starts, vec![at(Phase::Unscheduled)]);
    }

    // Eight schedule commands, six replies, three resolutions, the two faults and the four timers.
    // The catalog is in canonical order, so it opens on the backoff timer rather than on `schedule`.
    #[test]
    fn protocol_action_classes() {
        let keys = nexusProtocol.action_keys();
        assert_eq!(keys.len(), 8 + 6 + 3 + 1 + 1 + 4);
        assert_eq!(keys.iter().take(2).collect::<Vec<_>>(), ["backoff", "complete-canceled"]);
    }

    // A retryable handler error backs the operation off and raises the attempt count; the count
    // saturates rather than wrapping.
    #[test]
    fn retryable_error_backs_off() {
        let backed_off = ProtocolState { phase: Phase::BackingOff, attempts: Attempts::new(1).unwrap(), ..at(Phase::Scheduled) };
        assert_eq!(
            protocol_handler_reply_step(&at(Phase::Scheduled), Reply::HandlerError { retryable: true }),
            vec![Step { outcome: ProtocolOutcome::Accepted, state: backed_off, facts: vec![ProtocolFact::PendingAttempts] }]
        );
        let last = ProtocolState { attempts: Attempts::LAST, ..at(Phase::Scheduled) };
        let steps = protocol_handler_reply_step(&last, Reply::HandlerError { retryable: true });
        assert_eq!(steps.iter().map(|s| s.state.attempts).collect::<Vec<_>>(), vec![Attempts::LAST]);
    }

    // The refinement: every protocol row is a product step or a stutter.
    #[test]
    fn refinement_passes() {
        let refinement = nexusProtocol.refinement().expect("nexusProtocol declares refines:");
        assert_eq!(refinement.rejected, None);
        assert_eq!(refinement.rows.len(), nexusProtocol.table().transitions.len());
        // A reply the product sees is that reply's step; a retry it cannot see is a stutter.
        assert_eq!(refinement.lookup("scheduled-0-unset-unset-unset-handlerReply-async"), Some(Some("handlerReply-async")));
        assert_eq!(refinement.lookup("scheduled-0-unset-unset-unset-handlerReply-handlerError-true"), Some(None));
        // A deadline firing is the product's one timer, whichever deadline it was.
        assert_eq!(refinement.lookup("started-0-unset-unset-expires-startToClose"), Some(Some("timeout")));
    }

    // Each functional Query finds its claim on its path.
    #[test]
    fn find_queries_find() {
        let finds: [&dyn AnyQuery; 7] = [
            &*syncCompletion, &*asyncCompletion, &*asyncFailure, &*handlerError, &*retry,
            &*scheduleToStartTimeout, &*startToCloseTimeout,
        ];
        for query in finds {
            assert_eq!(outcome(query), Ok("found"), "{}", query.name());
        }
    }

    // The product claim is verified over every trace of the asynchronous path.
    #[test]
    fn terminal_holds() {
        assert_eq!(outcome(&*terminalHolds), Ok("verified-within-limits"));
    }

    // A timer is named under `when:` like any action, and a Scenario lists it where it fires.
    #[test]
    fn scenarios_name_their_classes() {
        assert_eq!(
            retriedThenSucceeded.action_keys(),
            ["schedule-unset-unset-unset", "handlerReply-handlerError-true", "backoff", "handlerReply-syncSuccess"]
        );
    }

    // Verified over the composition: the one reply comes before the stop.
    #[test]
    fn stopped_worker_replies_nothing() {
        assert_eq!(outcome(&*stoppedWorkerRepliesNothing), Ok("verified-within-limits"));
        assert_eq!(nexusCaller.table().states.len(), 2 * nexusProtocol.table().states.len());
    }
}

// ---------------------------------------------------------------------------------------------
// Standalone activity
// ---------------------------------------------------------------------------------------------

mod activity_pins {
    use super::*;
    use activity::*;

    fn at(phase: Phase) -> ProtocolState {
        ProtocolState { phase, ..ProtocolState::first() }
    }

    // Nine phases the caller can read, and the five the design ends on.
    #[test]
    fn product_states_and_ends() {
        assert_eq!(activityProduct.table().states.len(), 9);
        assert_eq!(activityProduct.ends().len(), 5);
    }

    // Twelve phases, three attempt counts and three deadlines.
    #[test]
    fn protocol_states_and_ends() {
        assert_eq!(activityProtocol.table().states.len(), 12 * 3 * 8);
        assert_eq!(activityProtocol.ends().len(), 5 * 3 * 8);
    }

    // A cancel response with no cancel requested is not enabled.
    #[test]
    fn cancel_without_request_is_not_enabled() {
        assert_eq!(protocol_attempt_result_step(&at(Phase::Started), AttemptResult::Canceled), vec![]);
    }

    // The refinement: every protocol row is a product row between its mapped states, or a stutter.
    // A retryable failure is visible here (started -> backingOff reads as started -> scheduled), a
    // pause request is a stutter (pauseRequested reads as started), and a retryable failure under a
    // requested pause reads as the product's pause.
    #[test]
    fn refinement_passes() {
        let refinement = activityProtocol.refinement().expect("activityProtocol declares refines:");
        assert_eq!(refinement.rejected, None);
        assert_eq!(refinement.rows.len(), activityProtocol.table().transitions.len());
        assert_eq!(
            refinement.lookup("started-1-unset-unset-unset-attemptResult-failed-true"),
            Some(Some("attemptResult-failed-true"))
        );
        assert_eq!(refinement.lookup("started-1-unset-unset-unset-control-pause"), Some(None));
        assert_eq!(refinement.lookup("pauseRequested-1-unset-unset-unset-control-unpause"), Some(None));
        assert_eq!(
            refinement.lookup("pauseRequested-1-unset-unset-unset-attemptResult-failed-true"),
            Some(Some("control-pause"))
        );
        assert_eq!(refinement.lookup("backingOff-1-unset-unset-unset-control-pause"), Some(Some("control-pause")));
    }

    // Each functional Query finds its claim on its path.
    #[test]
    fn find_queries_find() {
        let finds: [&dyn AnyQuery; 8] = [
            &*completion, &*nonRetryableFailure, &*retry, &*cancel, &*terminate, &*pauseResume,
            &*scheduleToStartTimeout, &*startToCloseTimeout,
        ];
        for query in finds {
            assert_eq!(outcome(query), Ok("found"), "{}", query.name());
        }
    }

    // The two product claims are verified over every trace of their paths.
    #[test]
    fn product_claims_verify() {
        assert_eq!(outcome(&*terminalHolds), Ok("verified-within-limits"));
        assert_eq!(outcome(&*pauseHolds), Ok("verified-within-limits"));
    }

    // Verified over the composition path: the one attempt starts while the worker polls, and the
    // claim fires on it, so this is not a vacuous verify.
    #[test]
    fn stopped_worker_starts_nothing() {
        assert_eq!(outcome(&*stoppedWorkerStartsNothing), Ok("verified-within-limits"));
    }
}

// ---------------------------------------------------------------------------------------------
// The errors, pinned
// ---------------------------------------------------------------------------------------------

/// The Lean `#guard_msgs` equivalent: each `tests/ui/*.rs` is a file that must fail to compile,
/// and the `.stderr` beside it is the exact diagnostic, macro errors and rustc errors alike.
///
/// - `ui/unobservable_not_a_timer.rs`: `unobservable: [backof]` -> our `syn::Error` text.
/// - `ui/steps_undeclared_action.rs`: `steps: { handlerRepli: .. }` -> rustc E0433 at the token.
/// - `ui/step_signature.rs`: `handlerReply: transport_fault_step` -> rustc E0593 at the fn name.
/// - `ui/non_exhaustive_reply.rs`: a `match reply` missing `HandlerError { retryable: false }`
///   -> rustc E0004 with the missing pattern spelled out.
/// - `ui/transition_claim_arity.rs`: `terminalIsFinal` with a one-argument `holds:` -> our text.
#[test]
fn authoring_errors_are_pinned() {
    let t = trybuild::TestCases::new();
    t.compile_fail("tests/ui/*.rs");
}

/// A product Property about an action the protocol machine does not have cannot be read there.
/// Lean rejects the `query` block at elaboration; here the Query is admitted when it runs, so the
/// pin is on the `Err`.
#[test]
fn product_property_on_missing_action_is_not_admitted() {
    use umpire::{Claim, Property, Query, QueryKind};
    static TIMES_OUT: std::sync::LazyLock<Property<nexus::ProductState, nexus::ProductOutcome, nexus::ProductFact>> =
        std::sync::LazyLock::new(|| Property {
            name: "timesOut",
            machine: &nexus::nexusProduct,
            claim: Claim::SameStep {
                when: nexus::timeout::any(),
                holds: |step| step.state.phase == nexus::ProductPhase::TimedOut,
            },
        });
    let query = Query {
        name: "timesOutOnProtocol",
        kind: QueryKind::Find,
        property: &*TIMES_OUT,
        scenario: &nexus::asyncThenSucceeded,
        limits: &nexus::three,
    };
    assert_eq!(
        query.run().map(|c| c.outcome.name()),
        Err(Admission::UnknownAction { property: "timesOut", action: "timeout", machine: "nexusProtocol" })
    );
}
