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

#[umpire_dsl::machine]
mod comparison {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(crate) struct Model {
        reject: u8,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Apply,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            if self.reject != 1 {
                self.reject = 1;
            }
        }
    }
}

#[test]
fn a_field_comparison_named_reject_keeps_unit_outcomes() {
    use umpire_dsl::Machine;

    fn unit_outcomes<M: Machine<Outcome = ()>>(_: &M) {}

    let world = comparison::world();
    unit_outcomes(&world);
    let state = &world.starts()[0];
    let steps = world
        .checked_step(state, world.actions()[0].clone())
        .unwrap_or_else(|error| panic!("unexpected Model error: {error}"));
    assert_eq!(steps.len(), 1);
    assert_ne!(steps[0].2, *state);
}

/// The lifted back ends encode the stock outcome by these names; replay compares them with
/// `Debug`, so they must be exactly the native catalog's spelling.
#[test]
fn the_lifted_stock_outcome_spells_every_native_outcome() {
    use umpire_dsl::__lifted::Type;
    let Type::Enum { variants, .. } = Type::outcome() else { panic!("stock outcome is an enum") };
    let catalog: Vec<_> = Outcome::catalog().into_iter().map(|class| class.name).collect();
    let lifted: Vec<_> = variants.iter().map(|name| name.trim_start_matches("Outcome::")).collect();
    assert_eq!(lifted, catalog);
    let rejections = [
        Rejection::NotFound,
        Rejection::AlreadyExists,
        Rejection::FailedPrecondition,
        Rejection::InvalidArgument,
        Rejection::PermissionDenied,
        Rejection::Unavailable,
    ];
    // A new Rejection variant fails to compile here until the lifted spelling lists it.
    let position = |rejection: Rejection| match rejection {
        Rejection::NotFound => 0,
        Rejection::AlreadyExists => 1,
        Rejection::FailedPrecondition => 2,
        Rejection::InvalidArgument => 3,
        Rejection::PermissionDenied => 4,
        Rejection::Unavailable => 5,
    };
    assert!(rejections.into_iter().map(position).eq(0..rejections.len()));
    for (rejection, outcome) in rejections.into_iter().zip(&lifted[1..]) {
        assert_eq!(format!("{:?}", Outcome::Rejected(rejection)), *outcome);
    }
    assert_eq!(format!("{:?}", Outcome::Accepted), lifted[0]);
}
