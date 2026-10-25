//! World arguments are typed before interpreting the existing positional callback signatures.

#![forbid(unsafe_code)]

use umpire_dsl::{Machine, Verdict};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct Owned(String);

#[umpire_dsl::machine]
mod counter {
    use super::Owned;
    use umpire_dsl::{Action, State, World, check, path};

    #[derive(World)]
    pub(super) struct World {
        pub(super) initial: Owned,
        pub(super) limit: u8,
    }

    #[derive(State)]
    pub(super) struct Model {
        pub(super) count: u8,
        pub(super) token: Owned,
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Advance,
    }

    impl Model {
        #[starts]
        fn starts(selected: &World) -> impl IntoIterator<Item = Self> {
            (selected.limit > 0).then(|| Self { count: 0, token: selected.initial.clone() })
        }

        #[on(User)]
        fn user(&mut self, _: User, selected: &World) {
            if self.count < selected.limit {
                self.count += 1;
            }
        }

        #[invariant]
        fn bounded(&self, selected: &World) {
            check!(self.count <= selected.limit && self.token == selected.initial);
        }

        #[postcondition]
        fn before_only(&self, world: &Self) {
            check!(self.count == world.count + 1);
        }

        #[postcondition]
        fn world_only(&self, before: &World) {
            check!(self.count <= before.limit);
        }

        #[postcondition]
        fn before_and_world(&self, world: &Self, before: &World) {
            check!(self.count > world.count && self.count <= before.limit);
        }

        #[postcondition]
        fn outcome_and_world(&self, old: &Self, outcome: &(), selected: &World) {
            let () = *outcome;
            check!(self.count > old.count && self.count <= selected.limit);
        }

        #[scope]
        fn admitted(selected: &World) -> Assumption {
            if selected.limit == 0 { only([] as [User; 0]) } else { only([User::Advance]) }
        }

        #[assumption]
        fn fair_user(selected: &World) -> Assumption {
            if selected.limit == 0 { fair([] as [User; 0]) } else { fair([User::Advance]) }
        }

        #[progress]
        fn finishes(&self, selected: &World) -> LeadsTo {
            path![self.count == 0, ..=3, self.count == selected.limit]
                .assuming(Self::fair_user(selected))
        }
    }
}

#[test]
fn selected_world_reaches_handlers_claims_scopes_and_computed_starts() {
    for limit in [1, 2] {
        let world = counter::World::new(Owned(format!("world-{limit}")), limit);
        let initial = world.starts();
        assert_eq!(initial.len(), 1);
        assert_eq!(initial[0].token.0, format!("world-{limit}"));
        let mut state = initial[0].clone();
        for _ in 0..limit {
            let action = counter::User::Advance.into();
            let edge = world.step(&state, action).pop().unwrap_or_else(|| panic!("missing step"));
            for post in world.postconditions() {
                assert_eq!(
                    post.holds(&world, &state, &counter::User::Advance.into(), &edge),
                    Verdict::Holds
                );
            }
            state = edge.2;
        }
        assert!(world.step(&state, counter::User::Advance.into()).is_empty());
        assert_eq!((world.invariants()[0].predicate)(&world, &state), Verdict::Holds);
        assert_eq!(
            world.assumptions().iter().map(|entry| entry.0).collect::<Vec<_>>(),
            ["admitted", "fair_user"]
        );
        let progress = world.progress();
        let evaluate = progress[0].evaluator.unwrap_or_else(|| panic!("missing evaluator"));
        assert_eq!(evaluate(&world, &initial[0]).trigger, Verdict::Holds);
        assert_eq!(evaluate(&world, &state).target, Verdict::Holds);
    }
    assert_eq!(counter::World::new(Owned("empty".into()), 0).starts(), []);
}
