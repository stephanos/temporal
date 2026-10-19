//! Scoped outcomes preserve rejection, return boundaries and evaluation isolation.

use std::{cell::Cell, panic::catch_unwind};

use umpire_dsl::{Outcome, Rejection, claim, outcome, reject};

#[test]
fn rejecting_returns_only_from_the_helper_and_keeps_the_first_rejection() {
    fn helper() {
        reject!(Rejection::NotFound);
    }
    let continued = Cell::new(false);
    assert_eq!(
        outcome::evaluate(|| {
            helper();
            continued.set(true);
            outcome::reject(Rejection::Unavailable);
        }),
        Outcome::Rejected(Rejection::NotFound)
    );
    assert!(continued.get());
    assert_eq!(outcome::evaluate(|| {}), Outcome::Accepted);
}

#[test]
fn inactive_nested_and_unwinding_frames_restore_the_enclosing_outcome() {
    fn inactive() {
        reject!(Rejection::InvalidArgument);
    }
    inactive();
    outcome::reject(Rejection::Unavailable);
    claim::evaluate(inactive);
    assert_eq!(outcome::evaluate(|| {}), Outcome::Accepted);
    assert_eq!(
        outcome::evaluate(|| {
            outcome::reject(Rejection::NotFound);
            assert_eq!(
                outcome::evaluate(|| reject!(Rejection::AlreadyExists)),
                Outcome::Rejected(Rejection::AlreadyExists)
            );
            assert!(
                catch_unwind(|| outcome::evaluate(|| {
                    outcome::reject(Rejection::PermissionDenied);
                    panic!("handler panicked");
                }))
                .is_err()
            );
        }),
        Outcome::Rejected(Rejection::NotFound)
    );
    assert_eq!(outcome::evaluate(|| {}), Outcome::Accepted);
}

#[test]
#[allow(
    clippy::disallowed_methods,
    clippy::disallowed_types,
    reason = "native test of outcome isolation across threads"
)]
fn frames_do_not_leak_between_threads() {
    assert_eq!(
        outcome::evaluate(|| {
            let child =
                std::thread::spawn(|| outcome::evaluate(|| reject!(Rejection::FailedPrecondition)))
                    .join();
            assert_eq!(child.ok(), Some(Outcome::Rejected(Rejection::FailedPrecondition)));
        }),
        Outcome::Accepted
    );
}

#[umpire_dsl::machine]
mod wrapped {
    use umpire_dsl::{Action, Rejection, State};

    #[derive(State)]
    pub(crate) struct Model {
        changed: bool,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Request,
    }

    macro_rules! decline {
        () => {
            umpire_dsl::reject!(Rejection::PermissionDenied)
        };
    }

    fn helper() {
        decline!();
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            if !self.changed {
                helper();
            }
        }
    }
}

#[test]
fn a_literal_rejection_inside_a_helper_macro_selects_step_outcomes() {
    use umpire_dsl::Machine;

    let world = wrapped::world();
    let state = &world.starts()[0];
    let steps = world
        .checked_step(state, world.actions()[0].clone())
        .unwrap_or_else(|error| panic!("unexpected Model error: {error}"));
    assert_eq!(steps.len(), 1);
    assert_eq!(steps[0].0, Outcome::Rejected(Rejection::PermissionDenied));
    assert_eq!(steps[0].2, *state);
}
