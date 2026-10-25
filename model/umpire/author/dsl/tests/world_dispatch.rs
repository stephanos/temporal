//! World-aware adapters retain ordinary, staying and rejecting enabledness.

#![forbid(unsafe_code)]

use umpire_dsl::{Machine, Outcome, Rejection, StepError, Verdict};

#[umpire_dsl::machine]
mod staying {
    use umpire_dsl::{Action, State, World, stay};
    #[derive(World)]
    pub(super) struct World {
        enabled: bool,
    }
    #[derive(State)]
    pub(super) struct Model {
        count: u8,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Pulse,
    }
    impl Model {
        #[on(User)]
        fn pulse(&mut self, _: User, w: &World) {
            if w.enabled && self.count == 0 {
                stay!();
            }
        }
    }
}

#[umpire_dsl::machine]
mod rejecting {
    use umpire_dsl::{Action, Outcome, Rejection, State, World, check, reject};
    #[derive(World)]
    pub(super) struct World {
        reject: bool,
        mutate: bool,
    }
    #[derive(State)]
    pub(super) struct Model {
        count: u8,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Probe,
    }
    impl Model {
        #[on(User)]
        fn probe(&mut self, _: User, w: &World) {
            if w.mutate {
                self.count += 1;
            }
            if w.reject {
                reject!(Rejection::NotFound);
            }
        }
        #[postcondition]
        fn old_before_outcome(&self, before: &Self, outcome: &Outcome) {
            check!(matches!(outcome, Outcome::Accepted) || self == before);
        }
        #[postcondition]
        fn selected_outcome(&self, before: &Self, outcome: &Outcome, w: &World) {
            check!(matches!(outcome, Outcome::Rejected(_)) == w.reject);
            check!(self.count == before.count + u8::from(w.mutate));
        }
    }
}

#[umpire_dsl::machine]
mod combined {
    use umpire_dsl::{Action, Rejection, State, World, reject, stay};
    #[derive(World)]
    pub(super) struct World {
        reject: bool,
        mutate: bool,
    }
    #[derive(State)]
    pub(super) struct Model {
        count: u8,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Probe,
    }
    impl Model {
        #[on(User)]
        fn probe(&mut self, _: User, w: &World) {
            stay!();
            if w.mutate {
                self.count += 1;
            }
            if w.reject {
                reject!(Rejection::NotFound);
            }
        }
    }
}

#[test]
fn world_access_keeps_all_step_and_rejection_branches() {
    for enabled in [false, true] {
        let world = staying::World::new(enabled);
        let state = world.starts().remove(0);
        assert_eq!(world.step(&state, staying::User::Pulse.into()).len(), usize::from(enabled));
    }
    for (reject, mutate, expected) in [
        (false, false, None),
        (false, true, Some(Outcome::Accepted)),
        (true, false, Some(Outcome::Rejected(Rejection::NotFound))),
    ] {
        let world = rejecting::World::new(reject, mutate);
        let state = world.starts().remove(0);
        let action = rejecting::User::Probe.into();
        let edges = world.checked_step(&state, action).unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(edges.first().map(|edge| edge.0), expected);
        if let Some(edge) = edges.first() {
            for post in world.postconditions() {
                assert_eq!(
                    post.holds(&world, &state, &rejecting::User::Probe.into(), edge),
                    Verdict::Holds
                );
            }
        }
        let both = combined::World::new(reject, mutate);
        let state = both.starts().remove(0);
        let edges = both
            .checked_step(&state, combined::User::Probe.into())
            .unwrap_or_else(|error| panic!("{error}"));
        assert_eq!(edges.len(), 1);
        assert_eq!(edges[0].0, expected.unwrap_or(Outcome::Accepted));
    }
    let world = rejecting::World::new(true, true);
    assert!(matches!(
        world.checked_step(&world.starts()[0], rejecting::User::Probe.into()),
        Err(StepError::RejectedAfterMutation)
    ));
    let world = combined::World::new(true, true);
    assert!(matches!(
        world.checked_step(&world.starts()[0], combined::User::Probe.into()),
        Err(StepError::RejectedAfterMutation)
    ));
}
