use std::{
    collections::BTreeMap,
    error::Error,
    fmt::{self, Write},
    panic,
};

use framework_checker::{Checker, Result as Exploration};
use framework_dsl::{Claim, Hole, Machine, StepClaim, Steps, Verdict};

#[allow(unreachable_pub)]
mod kettle {
    include!("../fixtures/kettle_trait.rs");

    impl std::fmt::Display for Kettle {
        fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            write!(
                f,
                "temperature={}, heating_on={}, beeping={}",
                self.temperature, self.heating_on, self.beeping
            )
        }
    }

    pub(super) fn negative() -> Claim<KettleWorld> {
        Claim::<KettleWorld>::new("never_beeps", |_, k| (!k.beeping).into())
    }
}

type Files = BTreeMap<String, String>;
type Case = (&'static str, fn() -> Result<Files, Box<dyn Error>>);

pub(super) fn registered() -> Vec<Case> {
    vec![
        ("kettle", || {
            let mut files = render(&Checker::explore(&kettle::KettleWorld, 1 << 20));
            let negative = Checker::refuted(&kettle::KettleWorld, kettle::negative());
            files.insert("negative_report.txt".into(), negative.verify_all().to_string());
            add_paths(&mut files, negative.verify_all());
            Ok(files)
        }),
        ("kettle_cut", || Ok(render(&Checker::explore(&kettle::KettleWorld, 82)))),
        ("starts_cut", || Ok(run(Mode::SharedStarts, 1))),
        ("shared_starts", || Ok(run(Mode::SharedStarts, 3))),
        ("edge_checks", || Ok(run(Mode::Edges, 3))),
        ("refuted_cut", || Ok(run(Mode::Fails, 1))),
        ("post_refuted_cut", || Ok(run(Mode::FailsOnEdge, 1))),
        ("vacuity", || Ok(run(Mode::Vacuity, 3))),
        ("choices", || Ok(run(Mode::Choices, 3))),
        ("effect_panic", || Ok(run(Mode::EffectPanic, 3))),
        ("effect_overflow", || Ok(run(Mode::Overflow, 3))),
        ("invariant_panic", || Ok(run(Mode::InvariantPanic, 3))),
        ("postcondition_panic", || Ok(run(Mode::PostconditionPanic, 3))),
        ("holes", || Ok(run(Mode::Holes, 3))),
        ("hole_then_failure", || Ok(run(Mode::HoleThenFailure, 3))),
        ("zero_budget", || {
            let error = panic::catch_unwind(|| Checker::explore(&Tiny(Mode::Edges), 0))
                .err()
                .ok_or("zero budget was accepted")?;
            let message = error
                .downcast_ref::<&str>()
                .copied()
                .or_else(|| error.downcast_ref::<String>().map(String::as_str))
                .ok_or("budget refusal has no message")?;
            Ok(BTreeMap::from([("report.txt".into(), format!("refused: {message}\n"))]))
        }),
    ]
}

fn run(mode: Mode, budget: usize) -> Files {
    render(&Checker::explore(&Tiny(mode), budget))
}

fn render<M>(result: &Exploration<M>) -> Files
where
    M: Machine,
    M::State: fmt::Display,
    M::Outcome: fmt::Debug,
    M::Fact: fmt::Debug,
{
    let mut files = BTreeMap::from([("report.txt".into(), result.verify_all().to_string())]);
    let mut states = String::new();
    for (id, state) in result.states() {
        writeln!(states, "{}: {state}", id.0).ok();
    }
    files.insert("states.txt".into(), states);
    if result.counts.reached <= 3 {
        let mut rows = String::new();
        for (id, _) in result.states() {
            for edge in result.rows(id) {
                writeln!(
                    rows,
                    "{}: {:?} choice={} outcome={:?} facts={:?} -> {}",
                    id.0, edge.action, edge.choice, edge.emitted.0, edge.emitted.1, edge.emitted.2
                )
                .ok();
            }
        }
        files.insert("rows.txt".into(), rows);
    }
    add_paths(&mut files, result.verify_all());
    files
}

fn add_paths<M: Machine>(files: &mut Files, report: framework_checker::Report<M>)
where
    M::State: fmt::Display,
{
    for claim in report.claims {
        if let Some(path) = claim.path {
            files.insert(format!("paths/{}.txt", claim.name), path.to_string());
        }
    }
}

#[derive(Clone, PartialEq, Eq, Hash)]
struct Value(u8);

impl fmt::Display for Value {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "state={}", self.0)
    }
}

#[derive(Clone, PartialEq, Eq, Hash, Debug)]
enum Action {
    Advance,
    Reset,
    Idle,
    Unused,
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Mode {
    SharedStarts,
    Edges,
    Fails,
    FailsOnEdge,
    Vacuity,
    Choices,
    EffectPanic,
    Overflow,
    InvariantPanic,
    PostconditionPanic,
    Holes,
    HoleThenFailure,
}

struct Tiny(Mode);

impl Machine for Tiny {
    type State = Value;
    type Action = Action;
    type Outcome = bool;
    type Fact = &'static str;

    fn starts(&self) -> Vec<Value> {
        if self.0 == Mode::SharedStarts {
            vec![Value(0), Value(1), Value(0)]
        } else {
            vec![Value(0)]
        }
    }
    fn actions(&self) -> Vec<Action> {
        vec![Action::Advance, Action::Reset, Action::Idle, Action::Unused]
    }
    fn internal(&self, _: &Action) -> bool {
        false
    }
    fn step(&self, state: &Value, action: Action) -> Steps<Value, bool, &'static str> {
        let mut steps = Steps::new();
        if state.0 == 0 && action == Action::Advance {
            match self.0 {
                Mode::EffectPanic => panic!("broken heater"),
                Mode::Overflow => {
                    let limit = std::hint::black_box(u8::MAX);
                    steps.push((true, vec![], Value(limit + 1)));
                    return steps;
                }
                Mode::Choices => {
                    steps.push((false, vec!["rejected"], Value(2)));
                    steps.push((true, vec!["accepted"], Value(1)));
                    steps.push((false, vec!["same_target"], Value(1)));
                    return steps;
                }
                _ => {}
            }
        }
        if self.0 == Mode::Choices {
            return steps;
        }
        let next = match (state.0, action) {
            (0 | 1, Action::Advance) => Some(Value(state.0 + 1)),
            (1, Action::Reset) => Some(Value(0)),
            (2, Action::Idle) => Some(state.clone()),
            _ => None,
        };
        if let Some(next) = next {
            steps.push((true, vec![], next));
        }
        steps
    }
    fn invariants(&self) -> Vec<Claim<Self>> {
        vec![Claim::<Self>::new("safe", |world, state| match world.0 {
            Mode::InvariantPanic if state.0 == 1 => panic!("broken invariant"),
            Mode::Holes => Verdict::Unknown(Hole { reason: "temperature unavailable" }),
            Mode::HoleThenFailure if state.0 == 0 => {
                Verdict::Unknown(Hole { reason: "initial gap" })
            }
            Mode::Fails | Mode::HoleThenFailure => Verdict::Fails,
            _ => Verdict::Holds,
        })]
    }
    fn postconditions(&self) -> Vec<StepClaim<Self>> {
        match self.0 {
            Mode::Edges => vec![
                StepClaim::<Self>::new("never_decreases", |_, before, after, _, _, _| {
                    (after.0 >= before.0).into()
                }),
                StepClaim::<Self>::after(
                    "idle_changes",
                    Action::Idle,
                    |_, before, after, _, _, _| (before != after).into(),
                ),
            ],
            Mode::FailsOnEdge => {
                vec![StepClaim::<Self>::new("unchanged", |_, before, after, _, _, _| {
                    (before == after).into()
                })]
            }
            Mode::Vacuity => vec![StepClaim::<Self>::after(
                "unused_claim",
                Action::Unused,
                |_, _, _, _, _, _| Verdict::Fails,
            )],
            Mode::PostconditionPanic => {
                vec![StepClaim::<Self>::new("panicking_edge", |_, _, _, _, _, _| {
                    panic!("broken postcondition")
                })]
            }
            Mode::Holes => vec![StepClaim::<Self>::new("edge_hole", |_, _, _, _, _, _| {
                Verdict::Unknown(Hole { reason: "edge unavailable" })
            })],
            Mode::Choices => {
                vec![StepClaim::<Self>::new("choice_facts", |_, _, after, _, outcome, facts| {
                    ((*outcome && after.0 == 1 && facts == ["accepted"])
                        || (!outcome
                            && ((after.0 == 2 && facts == ["rejected"])
                                || (after.0 == 1 && facts == ["same_target"]))))
                    .into()
                })]
            }
            _ => vec![],
        }
    }
}
