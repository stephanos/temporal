//! Authored Rust expressions retain their values, ownership and inference in paths.
#![forbid(unsafe_code)]

use umpire_dsl::{Path, PathNode, Waypoint, path};

/// The state waypoints of a path's one alternative, in order.
fn waypoints<S, A: 'static, O>(path: &Path<S, A, O>) -> Vec<std::rc::Rc<Waypoint<S>>> {
    let mut alternatives = path.alternatives(1).unwrap_or_else(|error| panic!("{error}"));
    alternatives
        .remove(0)
        .nodes
        .into_iter()
        .filter_map(|node| match node {
            PathNode::State(state) => Some(state),
            _ => None,
        })
        .collect()
}

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
        Poll,
        Label {
            #[inputs([String::new()])]
            text: String,
        },
    }
    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            self.count += 1;
        }
    }
}

#[test]
fn waypoint_expressions_are_evaluated_in_authored_order() {
    let mut count = 0u8;
    let mut next = || {
        count += 1;
        count
    };
    let pattern: Path<u8> = path![0u8, ..=0, next(), ..=0, next()];
    let values = waypoints(&pattern)
        .iter()
        .map(|state| match state.as_ref() {
            Waypoint::Is(value) => *value,
            _ => panic!("expected exact state"),
        })
        .collect::<Vec<_>>();
    assert_eq!(values, [0, 1, 2]);
}

#[test]
fn exact_repetition_evaluates_its_count_once() {
    let mut calls = 0;
    let pattern = path![
        world::Model { count: 0 },
        [world::User::Poll; {
            calls += 1;
            calls
        }]
    ];
    assert_eq!(calls, 1);
    assert_eq!(pattern.alternative_count(), Ok(1));
}

#[test]
fn outcome_alternatives_share_one_owned_action_expression() {
    let text = String::from("authored");
    let pattern = path![world::Model { count: 0 }, world::User::Label { text } => (() | ())];
    let alternatives = pattern.alternatives(2).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(alternatives.len(), 2);
    for alternative in alternatives {
        assert!(alternative.nodes.iter().any(|node| matches!(node,
            PathNode::Action { action, .. } if action.to_string().contains("authored"))));
    }
}

#[test]
fn qualified_state_struct_expressions_and_nested_closures_keep_rust_inference() {
    let count = 0;
    let start = world::Model { count: 0 };
    let shorthand: Path<world::Model> =
        path![world::Model { count }, .., world::Model { count: 1 }];
    assert_eq!(waypoints(&shorthand).len(), 2);
    let update: Path<world::Model> =
        path![world::Model { ..start }, .., world::Model { count: (1) }];
    assert_eq!(waypoints(&update).len(), 2);
    let nested = path![world::Model { count: 0 }, [|state| state.count == 1; 1]];
    assert_eq!(nested.alternative_count(), Ok(1));
    let branches = path![
        world::Model { count: 0 },
        ([|state| state.count == 1; 1] | unordered![|state| state.count == 2, world::User::Poll])
    ];
    assert_eq!(branches.alternative_count(), Ok(3));
}
