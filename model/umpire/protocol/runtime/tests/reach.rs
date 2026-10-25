//! Bounded Reach preserves canonical typed evidence, including equal-state histories.
#![forbid(unsafe_code)]

use prost::Message;
use umpire_checker::search::Traversal;
use umpire_dsl::{Machine, Steps};
use umpire_runtime::{
    Evidence, EvidenceMatch, Exact, Projection, ReachTarget, Realize, Runtime, Scope,
    SingletonTarget, prost_types::Any, protocol,
};

#[derive(Clone, Debug, PartialEq, Eq, Hash)]
struct State(u8, Option<u8>);

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum Action {
    A,
    B,
    C,
}

#[derive(Debug, PartialEq)]
struct Outcome(u8);
#[derive(Debug)]
struct Fact(u8);

impl std::fmt::Display for State {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}
impl std::fmt::Display for Action {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(out, "{self:?}")
    }
}

struct Edge(u8, Action, u8, u8);
struct Graph(&'static [Edge]);

impl Machine for Graph {
    type State = State;
    type Action = Action;
    type Outcome = Outcome;
    type Fact = Fact;
    fn starts(&self) -> Vec<State> {
        vec![State(0, Some(3))]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::A, Action::B, Action::C]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &State, action: Action) -> Steps<State, Outcome, Fact> {
        self.0
            .iter()
            .filter(|edge| edge.0 == state.0 && edge.1 == action)
            .map(|edge| {
                (Outcome(edge.3), vec![Fact(edge.3), Fact(edge.3 + 10)], State(edge.2, state.1))
            })
            .collect()
    }
}

#[derive(Clone, Copy)]
enum Predicate {
    Any,
    Choice,
    History,
    Zero,
    Stutter,
    DeclaredStutter,
}
struct Binding {
    target: State,
    scope: Scope,
    predicate: Predicate,
}

impl Realize<Graph> for Binding {
    type Target = Exact<State>;
    fn scope(&self) -> Scope {
        self.scope
    }
    fn observe(
        &self,
        _: &Any,
        _: &State,
    ) -> Result<Projection<Action, Outcome, Self::Target>, protocol::Refusal> {
        let target = Exact(self.target.clone());
        Ok(if matches!(self.predicate, Predicate::DeclaredStutter) {
            Projection::Stutter(target)
        } else {
            Projection::Reach(target)
        })
    }
    fn reach_evidence(&self, _: &Any, path: &[&Evidence<Graph>]) -> EvidenceMatch {
        if path.is_empty() {
            return if matches!(self.predicate, Predicate::Zero) {
                EvidenceMatch::Match { cursor: 0 }
            } else {
                EvidenceMatch::Continue { cursor: 0 }
            };
        }
        assert_eq!(path[0].source, State(0, Some(3)));
        for adjacent in path.windows(2) {
            assert_eq!(adjacent[0].target, adjacent[1].source);
        }
        for edge in path {
            assert_eq!(
                edge.facts.iter().map(|fact| fact.0).collect::<Vec<_>>(),
                [edge.outcome.0, edge.outcome.0 + 10]
            );
        }
        match self.predicate {
            Predicate::Any | Predicate::Zero => EvidenceMatch::Match { cursor: path.len() },
            Predicate::Choice => {
                if path.len() == 1 && path[0].choice == 1 && path[0].outcome == Outcome(2) {
                    EvidenceMatch::Match { cursor: 1 }
                } else {
                    EvidenceMatch::Reject
                }
            }
            Predicate::History => match path {
                [first] if first.action == Action::A => {
                    EvidenceMatch::Continue { cursor: usize::from(first.outcome.0) }
                }
                [first, second]
                    if first.outcome == Outcome(2)
                        && first.choice == 1
                        && second.action == Action::C =>
                {
                    EvidenceMatch::Match { cursor: 3 }
                }
                _ => EvidenceMatch::Reject,
            },
            Predicate::Stutter => EvidenceMatch::Stutter { cursor: path.len() },
            Predicate::DeclaredStutter => EvidenceMatch::Reject,
        }
    }
}

fn binding(target: u8, depth: usize, work: usize, predicate: Predicate) -> Binding {
    Binding { target: State(target, Some(3)), scope: Scope { depth, work, prefix: 128 }, predicate }
}
fn judge<R: Realize<Graph>>(runtime: &mut Runtime<Graph, R>) -> protocol::Response
where
    R::Target: SingletonTarget<State>,
{
    runtime.dispatch(
        protocol::Method::Judge as u32,
        &protocol::JudgeRequest {
            observation: Some(Any { type_url: "test/Reach".into(), value: vec![1] }),
        }
        .encode_to_vec(),
    )
}
fn explanation(response: protocol::Response) -> protocol::Explanation {
    let Some(protocol::response::Outcome::Judge(response)) = response.outcome else {
        panic!("expected judgment: {response:?}")
    };
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = response.judgment else {
        panic!("expected explanation: {response:?}")
    };
    explanation
}
fn unexplained(response: &protocol::Response) {
    assert!(matches!(
        response.outcome,
        Some(protocol::response::Outcome::Judge(protocol::JudgeResponse {
            judgment: Some(protocol::judge_response::Judgment::Unexplained(_)),
            ..
        }))
    ));
}
fn bounded(response: protocol::Response, cause: &str) {
    let Some(protocol::response::Outcome::Refusal(refusal)) = response.outcome else {
        panic!("expected bound refusal: {response:?}")
    };
    assert_eq!(refusal.code(), protocol::RefusalCode::Budget);
    assert!(refusal.message.contains(cause), "{}", refusal.message);
}

#[test]
fn canonical_actions_choices_outcomes_and_ordered_facts_are_retained() {
    static EDGES: &[Edge] =
        &[Edge(0, Action::A, 1, 1), Edge(0, Action::A, 1, 2), Edge(0, Action::B, 1, 3)];
    for (predicate, expected) in [(Predicate::Any, "Outcome(1)"), (Predicate::Choice, "Outcome(2)")]
    {
        let mut runtime = Runtime::new(Graph(EDGES), binding(1, 1, 32, predicate));
        let report = explanation(judge(&mut runtime));
        assert_eq!(report.steps, ["A"]);
        assert_eq!(report.outcomes, [expected]);
    }
}

#[test]
fn equal_states_with_different_choices_outcomes_and_consumption_keep_their_continuations() {
    static EDGES: &[Edge] =
        &[Edge(0, Action::A, 1, 1), Edge(0, Action::A, 1, 2), Edge(1, Action::C, 2, 3)];
    let mut runtime = Runtime::new(Graph(EDGES), binding(2, 2, 64, Predicate::History));
    let report = explanation(judge(&mut runtime));
    assert_eq!(report.steps, ["A", "C"]);
    assert_eq!(report.outcomes, ["Outcome(2)", "Outcome(3)"]);
}

#[test]
fn exact_depth_complete_absence_and_live_depth_or_work_frontiers_differ() {
    static CHAIN: &[Edge] = &[Edge(0, Action::A, 1, 1), Edge(1, Action::B, 2, 2)];
    let mut exact = Runtime::new(Graph(CHAIN), binding(2, 2, 64, Predicate::Any));
    assert_eq!(explanation(judge(&mut exact)).steps, ["A", "B"]);
    let mut beyond = Runtime::new(Graph(CHAIN), binding(2, 1, 64, Predicate::Any));
    bounded(judge(&mut beyond), "depth");
    let mut absent = Runtime::new(Graph(CHAIN), binding(3, 2, 64, Predicate::Any));
    unexplained(&judge(&mut absent));
    let mut work = Runtime::new(Graph(CHAIN), binding(2, 2, 2, Predicate::Any));
    bounded(judge(&mut work), "work");
}

#[test]
fn a_short_canonical_match_survives_a_longer_cycle_but_no_match_refuses_at_cut() {
    static CYCLE: &[Edge] =
        &[Edge(0, Action::A, 1, 1), Edge(1, Action::A, 1, 2), Edge(0, Action::B, 2, 3)];
    let mut short = Runtime::new(Graph(CYCLE), binding(2, 1, 16, Predicate::Any));
    assert_eq!(explanation(judge(&mut short)).steps, ["B"]);
    let mut cut = Runtime::new(Graph(CYCLE), binding(3, 2, 64, Predicate::Any));
    bounded(judge(&mut cut), "depth");
}

struct NoPredicate(u8);
impl Realize<Graph> for NoPredicate {
    type Target = Exact<State>;
    fn observe(
        &self,
        _: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome, Self::Target>, protocol::Refusal> {
        Ok(Projection::Reach(Exact(State(self.0, state.1))))
    }
}

#[test]
fn positive_zero_step_model_self_loop_and_declared_stutter_require_separate_evidence() {
    static LOOP: &[Edge] = &[Edge(0, Action::A, 0, 1)];
    static CHANGE: &[Edge] = &[Edge(0, Action::A, 1, 1)];
    let mut zero = Runtime::new(Graph(LOOP), binding(0, 0, 16, Predicate::Zero));
    assert_eq!(explanation(judge(&mut zero)).steps, [] as [String; 0]);
    for predicate in [Predicate::Any, Predicate::Stutter] {
        let mut edge = Runtime::new(Graph(LOOP), binding(0, 1, 16, predicate));
        assert_eq!(explanation(judge(&mut edge)).steps, ["A"]);
    }
    for (graph, target) in [(LOOP, 0), (CHANGE, 1)] {
        let mut absent = Runtime::new(Graph(graph), NoPredicate(target));
        unexplained(&judge(&mut absent));
    }
    let mut changed = Runtime::new(Graph(CHANGE), binding(1, 1, 16, Predicate::Stutter));
    unexplained(&judge(&mut changed));
    let mut declared = Runtime::new(Graph(CHANGE), binding(0, 0, 16, Predicate::DeclaredStutter));
    assert_eq!(explanation(judge(&mut declared)).steps, [] as [String; 0]);
    for target in [State(1, Some(3)), State(0, Some(2))] {
        let mut changed = binding(0, 0, 16, Predicate::DeclaredStutter);
        changed.target = target;
        let mut changed_target = Runtime::new(Graph(CHANGE), changed);
        unexplained(&judge(&mut changed_target));
    }
}

#[derive(Clone, Copy)]
enum ObservedBudget {
    Unknown,
    Absent,
    Present(u8),
}
struct PartialTarget {
    position: u8,
    budget: ObservedBudget,
}
impl ReachTarget<State> for PartialTarget {
    fn matches(&self, candidate: &State) -> bool {
        candidate.0 == self.position
            && match self.budget {
                ObservedBudget::Unknown => true,
                ObservedBudget::Absent => candidate.1.is_none(),
                ObservedBudget::Present(value) => candidate.1 == Some(value),
            }
    }
}

struct Budgets;
impl Machine for Budgets {
    type State = State;
    type Action = Action;
    type Outcome = Outcome;
    type Fact = Fact;
    fn starts(&self) -> Vec<State> {
        vec![State(0, Some(2))]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::A]
    }
    fn internal(&self, _: &Action) -> bool {
        true
    }
    fn step(&self, state: &State, _: Action) -> Steps<State, Outcome, Fact> {
        match (state.0, state.1) {
            (0, Some(2)) => [
                (Outcome(1), vec![Fact(1)], State(1, Some(1))),
                (Outcome(2), vec![Fact(2)], State(1, Some(0))),
            ]
            .into_iter()
            .collect(),
            _ => Steps::new(),
        }
    }
}

#[test]
fn partial_filter_matches_distinct_checked_budgets_without_substituting_unknown_absent_or_zero() {
    let unknown = PartialTarget { position: 1, budget: ObservedBudget::Unknown };
    let mut search = Traversal::new(1, 32);
    search.root(Budgets.starts().remove(0)).unwrap_or_else(|error| panic!("{error}"));
    let mut reached = Vec::new();
    while let Some(id) = search.next() {
        let state = search.label(id).position.clone();
        if unknown.matches(&state) {
            reached.push(state.clone());
        }
        for action in Budgets.actions() {
            search.charge().unwrap_or_else(|error| panic!("{error}"));
            for (outcome, facts, candidate) in
                Budgets.checked_step(&state, action).unwrap_or_else(|error| panic!("{error}"))
            {
                search.charge().unwrap_or_else(|error| panic!("{error}"));
                search
                    .push(id, candidate, Some((outcome, facts)))
                    .unwrap_or_else(|error| panic!("{error}"));
            }
        }
    }
    assert_eq!(reached, [State(1, Some(1)), State(1, Some(0))]);
    let absent = PartialTarget { position: 1, budget: ObservedBudget::Absent };
    let zero = PartialTarget { position: 1, budget: ObservedBudget::Present(0) };
    let one = PartialTarget { position: 1, budget: ObservedBudget::Present(1) };
    for (index, candidate) in reached.iter().enumerate() {
        assert!(unknown.matches(candidate));
        assert!(!absent.matches(candidate));
        assert_eq!(zero.matches(candidate), index == 1);
        assert_eq!(one.matches(candidate), index == 0);
        assert!(!PartialTarget { position: 2, budget: ObservedBudget::Unknown }.matches(candidate));
        assert!(Exact(candidate.clone()).matches(candidate));
    }
}

#[derive(Clone, Copy)]
enum Fault {
    Catalog,
    Step,
}
struct Faulty(Fault);
impl Machine for Faulty {
    type State = State;
    type Action = Action;
    type Outcome = Outcome;
    type Fact = Fact;
    fn starts(&self) -> Vec<State> {
        vec![State(0, Some(3))]
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::A]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn checked_catalog(
        &self,
        limit: usize,
    ) -> Result<umpire_dsl::ActionCatalog<Action>, umpire_dsl::CatalogError> {
        if matches!(self.0, Fault::Catalog) {
            Err(umpire_dsl::CatalogError::Overflow { limit })
        } else {
            umpire_dsl::ActionCatalog::checked(self.actions(), limit)
        }
    }
    fn step(&self, _: &State, _: Action) -> Steps<State, Outcome, Fact> {
        Steps::new()
    }
    fn checked_step(
        &self,
        _: &State,
        _: Action,
    ) -> Result<Steps<State, Outcome, Fact>, umpire_dsl::StepError> {
        Err(umpire_dsl::StepError::RejectedAfterMutation)
    }
}
struct FaultBinding;
impl Realize<Faulty> for FaultBinding {
    type Target = Exact<State>;
    fn observe(
        &self,
        _: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome, Self::Target>, protocol::Refusal> {
        Ok(Projection::Reach(Exact(state.clone())))
    }
    fn reach_evidence(&self, _: &Any, _: &[&Evidence<Faulty>]) -> EvidenceMatch {
        EvidenceMatch::Continue { cursor: 0 }
    }
}
#[test]
fn reach_catalog_and_checked_step_failures_refuse_without_advancing_or_freezing() {
    for (fault, code, message) in [
        (Fault::Catalog, protocol::RefusalCode::Budget, "catalog count overflows"),
        (Fault::Step, protocol::RefusalCode::MalformedRequest, "rejecting step changed state"),
    ] {
        let mut runtime = Runtime::new(Faulty(fault), FaultBinding);
        let request = protocol::JudgeRequest {
            observation: Some(Any { type_url: "test/Reach".into(), value: vec![1] }),
        }
        .encode_to_vec();
        for _ in 0..2 {
            let response = runtime.dispatch(protocol::Method::Judge as u32, &request);
            let Some(protocol::response::Outcome::Refusal(refusal)) = response.outcome else {
                panic!("expected checked refusal")
            };
            assert_eq!(refusal.code(), code);
            assert!(refusal.message.contains(message), "{}", refusal.message);
        }
        assert!(matches!(
            runtime.dispatch(protocol::Method::Reset as u32, &[]).outcome,
            Some(protocol::response::Outcome::Reset(_))
        ));
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum RetainedEvidence {
    Original,
    Outcome,
    Fact,
    Cursor,
}
struct PrefixBinding(std::rc::Rc<std::cell::Cell<RetainedEvidence>>);
impl Realize<Graph> for PrefixBinding {
    type Target = Exact<State>;
    fn observe(
        &self,
        actual: &Any,
        state: &State,
    ) -> Result<Projection<Action, Outcome, Self::Target>, protocol::Refusal> {
        match actual.value.as_slice() {
            [1] => Ok(Projection::Reach(Exact(State(1, Some(3))))),
            [2] if state.0 == 1 => Ok(Projection::Current),
            _ => Err(protocol::Refusal::malformed("unexpected observation")),
        }
    }
    fn reach_evidence(&self, _: &Any, path: &[&Evidence<Graph>]) -> EvidenceMatch {
        let [edge] = path else {
            return if path.is_empty() {
                EvidenceMatch::Continue { cursor: 0 }
            } else {
                EvidenceMatch::Reject
            };
        };
        let mode = self.0.get();
        let outcome = if mode == RetainedEvidence::Outcome { 2 } else { 1 };
        let facts = if mode == RetainedEvidence::Fact { [1, 12] } else { [1, 11] };
        if edge.outcome != Outcome(outcome) || edge.facts.iter().map(|fact| fact.0).ne(facts) {
            return EvidenceMatch::Reject;
        }
        EvidenceMatch::Match { cursor: if mode == RetainedEvidence::Cursor { 2 } else { 1 } }
    }
}

#[test]
fn later_current_revalidates_retained_typed_outcome_fact_and_consumption_cursor() {
    static EDGES: &[Edge] = &[Edge(0, Action::A, 1, 1)];
    for changed in [RetainedEvidence::Outcome, RetainedEvidence::Fact, RetainedEvidence::Cursor] {
        let mode = std::rc::Rc::new(std::cell::Cell::new(RetainedEvidence::Original));
        let mut runtime = Runtime::new(Graph(EDGES), PrefixBinding(std::rc::Rc::clone(&mode)));
        let call = |runtime: &mut Runtime<Graph, PrefixBinding>, value| {
            runtime.dispatch(
                protocol::Method::Judge as u32,
                &protocol::JudgeRequest {
                    observation: Some(Any {
                        type_url: "test/ReachPrefix".into(),
                        value: vec![value],
                    }),
                }
                .encode_to_vec(),
            )
        };
        let first = explanation(call(&mut runtime, 1));
        assert_eq!(first.steps, ["A"]);
        assert_eq!(first.outcomes, ["Outcome(1)"]);
        let retained_report = first.encode_to_vec();
        mode.set(changed);
        let refused = call(&mut runtime, 2);
        let Some(protocol::response::Outcome::Refusal(refusal)) = refused.outcome else {
            panic!("old typed evidence must still validate")
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::State);
        assert!(refusal.message.contains("retained prefix"));
        mode.set(RetainedEvidence::Original);
        assert_eq!(explanation(call(&mut runtime, 2)), first);
        assert_eq!(first.encode_to_vec(), retained_report);
    }
}
