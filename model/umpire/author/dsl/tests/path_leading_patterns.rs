//! Leading structural nodes retain the owning state's closure inference.
use umpire_dsl::{Machine, Path, Registered, path};

#[umpire_dsl::machine]
#[doc(hidden)]
pub mod world {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model {
        pub count: u8,
    }
    #[derive(Action)]
    pub enum User {
        Advance,
        Disabled,
        Send {
            #[inputs(0..=1)]
            amount: u8,
        },
    }
    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            self.count += 1;
        }
    }
}
use world::{Model, User};

#[allow(clippy::needless_pass_by_value, reason = "takes a path by value as find does")]
fn consume<S: Registered>(
    _: &S,
    pattern: Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>,
) -> usize
where
    <S::World as Machine>::Action: 'static,
{
    pattern.alternative_count().unwrap_or_else(|error| panic!("{error}"))
}

#[test]
fn leading_structures_infer_nested_and_trailing_state_predicates() {
    assert_eq!(
        consume(
            &Model { count: 0 },
            path![(User::Advance | User::Disabled), |_| true, |s| s.count == 1]
        ),
        2
    );
    let state = Model { count: 0 };
    assert_eq!(consume(&state, path![(User::Disabled | User::Advance), |s| s.count == 1]), 2);
    assert_eq!(
        consume(&state, path![(User::Advance | User::Disabled), User::Advance, |s| s.count == 2]),
        2
    );
    assert_eq!(
        consume(&state, path![(User::Advance | [|s| s.count == 0; 1]), |s| s.count == 1]),
        2
    );
    assert_eq!(consume(&state, path![[User::Advance; 2], |s| s.count == 2]), 1);
    assert_eq!(
        consume(&state, path![unordered![User::Advance, |s| s.count == 1], |s| s.count == 2]),
        2
    );
    assert_eq!(consume(&state, path![User::Send { amount: _, .. }, |s| s.count == 1]), 2);
    assert_eq!(consume(&state, path![User::*, |s| s.count == 1]), 4);
    assert_eq!(consume(&state, path![User::Advance => (), |s| s.count == 1]), 1);
}

#[test]
fn structural_seeding_preserves_source_order_and_evaluates_values_once() {
    let mut events = Vec::new();
    let pattern = path![
        unordered![
            {
                events.push("state");
                Model { count: 0 }
            },
            {
                events.push("action");
                User::Advance
            },
            [{
                events.push("child");
                User::Disabled
            }; {
                events.push("count");
                0
            }]
        ],
        |s| s.count == 1
    ];
    assert_eq!(events, ["state", "action", "child", "count"]);
    assert_eq!(pattern.alternative_count(), Ok(6));

    let mut calls = 0;
    let mut next = || {
        calls += 1;
        User::Advance
    };
    let pattern = path![unordered![next(), next()], |s| s.count == 2];
    assert_eq!(calls, 2);
    assert_eq!(pattern.alternative_count(), Ok(2));
}

#[test]
fn outer_state_members_in_shadowing_initializers_still_infer_the_owner() {
    let guarded = path![(User::Advance | User::Disabled), |state| (match 0 {
        _ if state.count == 0 => 1,
        _ => 0,
    }) == 1];
    assert_eq!(guarded.alternative_count(), Ok(2));
    let pattern = path![(User::Advance | User::Disabled), |state| {
        let state = Model { count: state.count };
        state.count == 0
    }];
    assert_eq!(pattern.alternative_count(), Ok(2));
    let nested = path![(User::Advance | User::Disabled), |state| {
        let predicate = |state: &Model| state.count == 0;
        predicate(&Model { count: state.count })
    }];
    assert_eq!(nested.alternative_count(), Ok(2));
    let block = path![(User::Advance | User::Disabled), |state| {
        {
            let state = Model { count: 0 };
            let _ = state.count;
        }
        state.count == 0
    }];
    assert_eq!(block.alternative_count(), Ok(2));
}
