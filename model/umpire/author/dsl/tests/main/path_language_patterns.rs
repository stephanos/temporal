//! Finite typed path grammar and pre-allocation admission controls.
#![forbid(unsafe_code)]

use umpire_dsl::{
    ActionDomain, Gap, InputDomain, Path, PathNode, Pattern, PatternError, Waypoint, path,
};

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
        Cancel,
        Send {
            #[inputs(0..=2)]
            amount: u8,
            #[inputs([false, true])]
            urgent: bool,
        },
    }
    impl Model {
        #[on(User)]
        fn user(&mut self, _action: User) {
            self.count = self.count.wrapping_add(1);
        }
    }
}
use world::{Model, User};

#[test]
fn enumerates_fresh_choices_in_repetitions_and_positional_permutations() {
    let pattern = path![Model { count: 0 }, [(User::Poll | User::Cancel); 0..=2]];
    assert_eq!(pattern.alternative_count(), Ok(7));
    let alternatives = pattern.alternatives(7).unwrap_or_else(|error| panic!("{error}"));
    let actions = alternatives
        .iter()
        .map(|alternative| {
            alternative
                .nodes
                .iter()
                .filter_map(|node| match node {
                    PathNode::Action { action, .. } => Some(action.to_string()),
                    _ => None,
                })
                .collect::<Vec<_>>()
                .join(",")
        })
        .collect::<Vec<_>>();
    assert_eq!(
        actions,
        [
            "",
            "User::Poll",
            "User::Cancel",
            "User::Poll,User::Poll",
            "User::Poll,User::Cancel",
            "User::Cancel,User::Poll",
            "User::Cancel,User::Cancel"
        ]
    );
    assert_eq!(alternatives[0].choices, ["[User::Poll | User::Cancel; 0]"]);
    let permutations = path![Model { count: 0 }, unordered![User::Poll, User::Poll, User::Cancel]];
    assert_eq!(permutations.alternatives(6).unwrap_or_else(|error| panic!("{error}")).len(), 6);
}

#[test]
fn empty_products_and_empty_repetitions_do_not_expand_unreachable_prefixes() {
    let enormous = Pattern::Action(
        ActionDomain::new(Ok(usize::MAX / 2), |_| -> () {
            panic!("empty product enumerated a prefix")
        }),
        None,
        None,
    );
    let empty = Pattern::Action(
        ActionDomain::new(Ok(0), |_| -> () { panic!("empty action domain indexed") }),
        None,
        None,
    );
    let product = Path::<u8>::pattern(Pattern::Sequence(vec![enormous, empty]));
    assert_eq!(product.alternative_count(), Ok(0));
    assert!(product.alternatives(0).unwrap_or_else(|error| panic!("{error}")).is_empty());
    let repeated = Path::<u8>::pattern(Pattern::Repeat(
        Box::new(Pattern::Sequence(vec![])),
        usize::MAX,
        usize::MAX,
    ));
    assert_eq!(repeated.alternative_count(), Ok(1));
    assert!(repeated.alternatives(1).unwrap_or_else(|error| panic!("{error}"))[0].nodes.is_empty());
}

#[test]
fn zero_products_and_zero_repetitions_annihilate_overflowing_subpatterns() {
    let overflowing = || {
        Pattern::Action(
            ActionDomain::new(Err(PatternError::ExpansionOverflow), |_| -> () {
                panic!("overflow domain indexed")
            }),
            None,
            None,
        )
    };
    let empty = Pattern::Either(vec![]);
    let product = Path::<u8>::pattern(Pattern::Sequence(vec![overflowing(), empty]));
    assert_eq!(product.alternative_count(), Ok(0));
    assert!(product.alternatives(0).unwrap_or_else(|error| panic!("{error}")).is_empty());
    let zero = Path::<u8>::pattern(Pattern::Repeat(Box::new(overflowing()), 0, 0));
    assert_eq!(zero.alternative_count(), Ok(1));
    assert!(zero.alternatives(1).unwrap_or_else(|error| panic!("{error}"))[0].nodes.is_empty());
}

#[test]
fn exact_singleton_repetition_expands_linearly_and_storage_has_its_own_limit() {
    let path = path![Model { count: 0 }, [User::Poll; 10_000]];
    assert_eq!(
        path.alternatives(1).unwrap_or_else(|error| panic!("{error}"))[0].nodes.len(),
        10_002
    );
    let too_large = path![Model { count: 0 }, [User::Poll; 1_000_001]];
    assert_eq!(too_large.alternative_count(), Ok(1));
    assert!(matches!(too_large.alternatives(1), Err(PatternError::ExpansionTooLarge { .. })));
}

#[test]
fn empty_and_reversed_domains_keep_zero_cases_and_zero_repetition_keeps_one() {
    assert_eq!(InputDomain::<u8>::values(vec![]).count(), Ok(0));
    assert_eq!(InputDomain::range(std::ops::RangeInclusive::new(3u8, 1)).count(), Ok(0));
    let repeated = path![Model { count: 0 }, [User::Poll; 2..=1]];
    assert_eq!(repeated.alternative_count(), Ok(0));
    assert!(repeated.alternatives(0).unwrap_or_else(|error| panic!("{error}")).is_empty());
    let zero = path![Model { count: 0 }, [User::Poll; 0]];
    assert_eq!(zero.alternative_count(), Ok(1));
}

#[test]
fn state_budget_and_structured_append_preserve_constraints() {
    let states: Path<u8> = path![true, .., false];
    assert!(matches!(states.alternatives(0), Err(PatternError::Budget { count: 1, budget: 0 })));
    let structured = path![Model { count: 0 }, [User::Poll; 1]]
        .then(Gap::AtMost(0), Waypoint::judged(|state: &Model| state.count == 1));
    let alternatives = structured.alternatives(1).unwrap_or_else(|error| panic!("{error}"));
    assert!(
        matches!(alternatives[0].nodes.last(), Some(PathNode::State(state)) if state.judge(&Model { count: 1 }))
    );
}

#[test]
fn refuses_counts_before_constructing_alternatives() {
    let pattern = path![Model { count: 0 }, [User::Poll; 0..=usize::MAX]];
    assert!(pattern.alternative_count().is_err());
    let pattern = path![
        Model { count: 0 },
        unordered![
            User::Poll,
            User::Poll,
            User::Poll,
            User::Poll,
            User::Poll,
            User::Poll,
            User::Poll
        ]
    ];
    assert!(pattern.alternatives(6000).is_err());
    assert_eq!(
        pattern
            .permutation_limit(7)
            .alternatives(5040)
            .unwrap_or_else(|error| panic!("{error}"))
            .len(),
        5040
    );
}

#[test]
#[allow(unused_qualifications, reason = "pins qualified state constructor grammar")]
fn state_paths_infer_closures_and_take_their_types_from_use() {
    fn accept(_path: Path<Model, world::Action>) {}
    /// The number of state waypoints and whether the first one holds at `first`.
    fn shape<S: PartialEq, A: 'static, O>(path: &Path<S, A, O>, first: &S) -> (usize, bool) {
        let nodes = path.alternatives(1).unwrap_or_else(|error| panic!("{error}")).remove(0).nodes;
        let states: Vec<_> = nodes
            .iter()
            .filter_map(|node| match node {
                PathNode::State(state) => Some(state),
                _ => None,
            })
            .collect();
        (states.len(), states[0].judge(first))
    }
    #[derive(Clone, PartialEq)]
    enum Coin {
        Unseen,
        Tails,
    }
    let closure_first: Path<Model> =
        path![|state: &Model| state.count == 0, .., |state| state.count == 1];
    assert_eq!(shape(&closure_first, &Model { count: 0 }), (2, true));
    let typed: Path<Model, world::Action> = path![Model { count: 0 }, .., |state| state.count == 1];
    assert_eq!(shape(&typed, &Model { count: 0 }), (2, true));
    accept(path![Model { count: 0 }, .., |state| state.count == 1]);
    accept(path![User::Poll, |state| state.count == 1]);
    let qualified: Path<Model> = path![world::Model { count: 0 }, .., |state| state.count == 1];
    assert_eq!(shape(&qualified, &Model { count: 0 }).0, 2);
    let enum_states: Path<Coin> = path![Coin::Unseen, .., Coin::Tails];
    assert!(shape(&enum_states, &Coin::Unseen).1);
}

#[test]
fn structured_inputs_and_actor_wildcards_keep_full_declared_domains() {
    let exact_outside = path![Model { count: 0 }, User::Send { amount: 5, .. }];
    let alternatives = exact_outside.alternatives(2).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(alternatives.len(), 2);
    for alternative in alternatives {
        assert!(
            matches!(&alternative.nodes[2], PathNode::Action { action, .. } if matches!(action.as_ref(), world::Action::User(User::Send { amount: 5, .. })))
        );
    }
    let set = path![Model { count: 0 }, User::Send { amount: (0 | 2), urgent: _ }];
    assert_eq!(set.alternative_count(), Ok(4));
    let range = path![Model { count: 0 }, User::Send { amount: 0..=2, .. }];
    assert_eq!(range.alternative_count(), Ok(6));
    let wildcard = path![Model { count: 0 }, User::*];
    assert_eq!(wildcard.alternative_count(), Ok(8));
    assert_eq!(User::actions().len(), 8);
}

#[test]
fn exclusion_predicates_are_typed_and_do_not_change_explicit_endpoint() {
    let pattern = path![Model { count: 0 }, ..!(User::Poll), User::Poll];
    let alternatives = pattern.alternatives(1).unwrap_or_else(|error| panic!("{error}"));
    let exclusion = alternatives[0]
        .nodes
        .iter()
        .find_map(|node| match node {
            PathNode::Gap { exclude: Some(exclude), .. } => Some(exclude),
            _ => None,
        })
        .unwrap_or_else(|| panic!("missing exclusion predicate"));
    assert!(exclusion(&User::Poll.into()));
    assert!(!exclusion(&User::Cancel.into()));
    assert!(
        matches!(alternatives[0].nodes.last(), Some(PathNode::Action { action, .. }) if **action == User::Poll.into())
    );
}

#[derive(Clone, PartialEq, Eq, Hash)]
struct ManualState;
impl std::fmt::Display for ManualState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str("manual")
    }
}
struct ManualWorld;
#[derive(Debug)]
enum BorrowedOutcome {
    Accepted,
    Refused,
}
impl umpire_dsl::Registered for ManualState {
    type World = ManualWorld;
    fn world() -> Self::World {
        ManualWorld
    }
}
impl umpire_dsl::Machine for ManualWorld {
    type State = ManualState;
    type Action = User;
    type Outcome = BorrowedOutcome;
    type Fact = ();
    fn starts(&self) -> Vec<ManualState> {
        vec![ManualState]
    }
    fn actions(&self) -> Vec<User> {
        User::actions()
    }
    fn internal(&self, _: &User) -> bool {
        false
    }
    fn step(
        &self,
        _: &ManualState,
        _: User,
    ) -> umpire_dsl::Steps<ManualState, BorrowedOutcome, ()> {
        umpire_dsl::Steps::new()
    }
}

#[test]
fn borrowed_outcome_alternatives_need_neither_clone_nor_equality() {
    let actor_led: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |_: &ManualState| true];
    assert_eq!(actor_led.alternative_count(), Ok(2));
    let no_closure: Path<ManualState, User, BorrowedOutcome> = path![(User::Poll | User::Poll)];
    assert_eq!(no_closure.alternative_count(), Ok(2));
    let constant: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |_| true];
    assert_eq!(constant.alternative_count(), Ok(2));
    let named_constant: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |_state| true];
    assert_eq!(named_constant.alternative_count(), Ok(2));
    let nested: Path<ManualState, User, BorrowedOutcome> = path![
        (User::Poll | unordered![|_: &ManualState| true, |state| state.to_string() == "manual"]),
        |state| state.to_string() == "manual"
    ];
    assert_eq!(nested.alternative_count(), Ok(3));
    let state_led: Path<ManualState, User, BorrowedOutcome> =
        path![(ManualState | User::Poll), |_| true];
    assert_eq!(state_led.alternative_count(), Ok(2));
    let unit = path![Model { count: 0 }, User::Poll => ()];
    assert_eq!(unit.alternative_count(), Ok(1));
    let pattern =
        path![ManualState, User::Poll => (BorrowedOutcome::Accepted | BorrowedOutcome::Refused)];
    let alternatives = pattern.alternatives(2).unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(alternatives.len(), 2);
    for (index, alternative) in alternatives.iter().enumerate() {
        let predicate = alternative
            .nodes
            .iter()
            .find_map(|node| match node {
                PathNode::Action { outcome, .. } => outcome.as_ref(),
                _ => None,
            })
            .unwrap_or_else(|| panic!("missing outcome predicate"));
        assert_eq!(predicate(&BorrowedOutcome::Accepted), index == 0);
        assert_eq!(predicate(&BorrowedOutcome::Refused), index == 1);
    }
    assert_ne!(alternatives[0].choices, alternatives[1].choices);
    let adapted: Path<ManualState, User, BorrowedOutcome> =
        path![ManualState, .., |_: &ManualState| true];
    assert_eq!(adapted.alternatives(1).unwrap_or_else(|error| panic!("{error}")).len(), 1);
}

#[test]
fn manual_world_predicates_keep_lexically_shadowed_state_bindings() {
    let local: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            let state = &Model { count: 0 };
            state.count == 0
        }];
    let renamed: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |_outer| {
            let state = &Model { count: 0 };
            state.count == 0
        }];
    let closure: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            let predicate = |state: &Model| state.count == 0;
            predicate(&Model { count: 0 })
        }];
    assert_eq!(local.alternative_count(), Ok(2));
    assert_eq!(renamed.alternative_count(), Ok(2));
    assert_eq!(closure.alternative_count(), Ok(2));
    let binders: [Path<ManualState, User, BorrowedOutcome>; 4] = [
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            for state in [Model { count: 0 }, Model { count: 1 }] {
                if state.count > 1 {
                    return false;
                }
            }
            true
        }],
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            match Some(Model { count: 0 }) {
                Some(state) if state.count == 0 => state.count == 0,
                _ => true,
            }
        }],
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            if let Some(state) = Some(Model { count: 0 })
                && state.count == 0
            {
                state.count == 0
            } else {
                true
            }
        }],
        path![(User::Poll | User::Poll), |state| {
            let _ = state;
            let mut values = vec![Model { count: 0 }];
            while let Some(state) = values.pop() {
                if state.count != 0 {
                    return false;
                }
            }
            true
        }],
    ];
    assert!(binders.iter().all(|pattern| pattern.alternative_count() == Ok(2)));
}

#[test]
fn manual_world_selection_keeps_authored_predicate_and_waypoint_constraints() {
    fn valid(_: &ManualState) -> bool {
        true
    }
    let later: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), ManualState, |state| state.to_string() == "manual"];
    let helper: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |state| valid(state) && state.to_string() == "manual"];
    let matched: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |state| matches!(state, ManualState)
            && state.to_string() == "manual"];
    let nested: Path<ManualState, User, BorrowedOutcome> = path![
        (User::Poll | User::Poll),
        unordered![ManualState, |state| state.to_string() == "manual"]
    ];
    let bound: Path<ManualState, User, BorrowedOutcome> =
        path![(User::Poll | User::Poll), |state @ ManualState| state.to_string() == "manual"];
    let helper_then_bare: Path<ManualState, User, BorrowedOutcome> = path![
        (User::Poll | User::Poll),
        |state| valid(state) && state.to_string() == "manual",
        |state| state.to_string() == "manual"
    ];
    let nested_helper_then_bare: Path<ManualState, User, BorrowedOutcome> = path![
        (User::Poll | User::Poll),
        unordered![|state| valid(state) && state.to_string() == "manual", |state| state
            .to_string()
            == "manual"]
    ];
    assert_eq!(later.alternative_count(), Ok(2));
    assert_eq!(helper.alternative_count(), Ok(2));
    assert_eq!(matched.alternative_count(), Ok(2));
    assert_eq!(nested.alternative_count(), Ok(4));
    assert_eq!(bound.alternative_count(), Ok(2));
    assert_eq!(helper_then_bare.alternative_count(), Ok(2));
    assert_eq!(nested_helper_then_bare.alternative_count(), Ok(4));
}
