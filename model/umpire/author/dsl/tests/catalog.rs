//! Complete declared metadata is independent of reached rows and path selection.

use umpire_dsl::{ActorProjection, Machine, Outcome, OutcomeClass, Rejection};

#[umpire_dsl::machine]
mod catalog {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(crate) struct Model {
        changed: bool,
    }

    #[derive(Action)]
    pub(crate) enum User {
        Apply,
    }

    #[derive(Action)]
    #[internal]
    pub(crate) enum Clock {
        Tick,
    }

    #[derive(Action)]
    #[fault]
    pub(crate) enum Network {
        Lose,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            self.changed = true;
        }

        #[on(Clock)]
        fn clock(&mut self, _: Clock) {
            self.changed = true;
        }

        #[on(Network)]
        fn network(&mut self, _: Network) {
            self.changed = false;
        }
    }
}

#[test]
fn fault_catalog_and_actor_projection_are_typed_and_independent_of_internal() {
    let world = catalog::world();
    let faults = world.fault_catalog().unwrap_or_else(|| panic!("undeclared fault catalog"));
    assert_eq!(faults.iter().map(ToString::to_string).collect::<Vec<_>>(), ["Network::Lose"]);
    assert!(!world.internal(&faults[0]));
    assert!(ActorProjection::<catalog::Network>::project(&faults[0]).is_some());
    assert!(ActorProjection::<catalog::User>::project(&faults[0]).is_none());
    assert_eq!(world.actions().len(), 3);
    let outcomes = world.outcome_catalog().unwrap_or_else(|| panic!("undeclared unit outcomes"));
    assert_eq!(outcomes.len(), 1);
    assert_eq!(outcomes[0].name, "()");
    assert!(outcomes[0].matches(&()));
}

#[derive(Debug)]
struct BorrowedOutcome(String);

struct Manual;

impl Machine for Manual {
    type State = u8;
    type Action = u8;
    type Outcome = BorrowedOutcome;
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        vec![0]
    }
    fn actions(&self) -> Vec<u8> {
        vec![0]
    }
    fn internal(&self, _: &u8) -> bool {
        false
    }
    fn step(&self, _: &u8, _: u8) -> umpire_dsl::Steps<u8, BorrowedOutcome, ()> {
        umpire_dsl::Steps::new()
    }
}

struct DeclaredEmpty;

impl Machine for DeclaredEmpty {
    type State = u8;
    type Action = u8;
    type Outcome = BorrowedOutcome;
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        Manual.starts()
    }
    fn actions(&self) -> Vec<u8> {
        Manual.actions()
    }
    fn internal(&self, action: &u8) -> bool {
        Manual.internal(action)
    }
    fn step(&self, state: &u8, action: u8) -> umpire_dsl::Steps<u8, BorrowedOutcome, ()> {
        Manual.step(state, action)
    }
    fn outcome_catalog(&self) -> Option<Vec<OutcomeClass<BorrowedOutcome>>> {
        Some(Vec::new())
    }
    fn fault_catalog(&self) -> Option<Vec<u8>> {
        Some(Vec::new())
    }
}

#[test]
fn borrowed_outcome_classification_needs_no_value_traits_and_missing_is_unknown() {
    assert!(Manual.outcome_catalog().is_none());
    assert!(Manual.fault_catalog().is_none());
    assert!(DeclaredEmpty.outcome_catalog().is_some_and(|classes| classes.is_empty()));
    assert!(DeclaredEmpty.fault_catalog().is_some_and(|faults| faults.is_empty()));
    let class = OutcomeClass::new("accepted", |outcome: &BorrowedOutcome| outcome.0 == "yes");
    assert!(class.matches(&BorrowedOutcome("yes".into())));
    assert!(!class.matches(&BorrowedOutcome("no".into())));
}

#[umpire_dsl::machine]
mod rejecting {
    use umpire_dsl::{Action, Rejection, State, reject};

    #[derive(State)]
    pub(crate) struct Model {
        changed: bool,
    }

    #[derive(Action)]
    pub(crate) enum User {
        Request,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {
            if !self.changed {
                reject!(Rejection::NotFound);
            }
        }
    }
}

#[test]
fn stock_catalog_declares_every_outcome_even_when_never_emitted() {
    assert!(rejecting::world().fault_catalog().is_some_and(|faults| faults.is_empty()));
    let classes = rejecting::world().outcome_catalog().unwrap_or_else(|| panic!("missing catalog"));
    let values = [
        Outcome::Accepted,
        Outcome::Rejected(Rejection::NotFound),
        Outcome::Rejected(Rejection::AlreadyExists),
        Outcome::Rejected(Rejection::FailedPrecondition),
        Outcome::Rejected(Rejection::InvalidArgument),
        Outcome::Rejected(Rejection::PermissionDenied),
        Outcome::Rejected(Rejection::Unavailable),
    ];
    assert_eq!(classes.len(), 7);
    for (class, value) in classes.iter().zip(values) {
        assert!(class.matches(&value));
        assert_eq!(classes.iter().filter(|candidate| candidate.matches(&value)).count(), 1);
    }
}

#[test]
fn one_typed_consumer_accepts_legacy_and_world_paths_for_each_outcome_type() {
    fn count<S: umpire_dsl::Registered, A, O, Kind>(_: &S, path: umpire_dsl::Path<S, A, O>) -> usize
    where
        umpire_dsl::Path<S, A, O>: umpire_dsl::IntoWorldPath<S, Kind>,
        <S::World as Machine>::Action: 'static,
    {
        umpire_dsl::IntoWorldPath::into_world_path(path)
            .alternative_count()
            .unwrap_or_else(|error| panic!("{error}"))
    }
    let state = catalog::world().starts().remove(0);
    assert_eq!(count(&state, umpire_dsl::path![state.clone(), .., true]), 1);
    assert_eq!(count(&state, umpire_dsl::path![catalog::User::Apply, true]), 1);
    let state = rejecting::world().starts().remove(0);
    assert_eq!(count(&state, umpire_dsl::path![state.clone(), .., true]), 1);
    assert_eq!(
        count(&state, umpire_dsl::path![rejecting::User::Request => Outcome::Accepted, true]),
        1
    );
}
