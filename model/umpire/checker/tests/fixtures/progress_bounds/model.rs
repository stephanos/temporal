use std::cell::Cell;
use std::fmt::Write;

use umpire_checker::Checker;
use umpire_dsl::{Assumption, Claim, Hole, Machine, Progress, ProgressEvaluation, Steps, Verdict, machine};

#[machine]
mod generated {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub struct Counter {
        #[starts(0..=0)]
        value: u8,
    }

    #[derive(Action, Copy)]
    pub enum Clock { Tick }

    impl Counter {
        #[on(Clock)]
        fn tick(&mut self, action: Clock) {
            match action {
                Clock::Tick if self.value < 2 => { self.value += 1; }
                Clock::Tick => {}
            }
        }

        #[progress]
        fn reaches_two(&self) -> LeadsTo {
            whenever(self.value == 0).eventually(self.value == 2).within(2)
        }
    }
}

#[derive(Clone, Copy)]
enum Metadata {
    Fixed,
    ChangingBound,
    ChangingAssumption,
    InvalidAssumption,
    PredicatePanic,
    EffectPanic,
    SafetyFailure,
}

struct World {
    starts: Vec<u8>,
    edges: Vec<(u8, u8, u8)>,
    trigger: u8,
    target: u8,
    hole: Option<u8>,
    bound: Option<u32>,
    fair: bool,
    metadata: Metadata,
    bound_override: Option<u32>,
    assumption_override: Option<Assumption<u8>>,
    calls: Cell<usize>,
}

impl World {
    fn new(edges: &[(u8, u8, u8)], bound: Option<u32>) -> Self {
        Self {
            starts: vec![0],
            edges: edges.to_vec(),
            trigger: 0,
            target: 3,
            hole: None,
            bound,
            fair: false,
            metadata: Metadata::Fixed,
            bound_override: None,
            assumption_override: None,
            calls: Cell::new(0),
        }
    }
}

impl Machine for World {
    type State = u8;
    type Action = u8;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<u8> {
        self.starts.clone()
    }

    fn actions(&self) -> Vec<u8> {
        vec![0, 1]
    }

    fn internal(&self, _: &u8) -> bool {
        false
    }

    fn step(&self, state: &u8, action: u8) -> Steps<u8, (), ()> {
        assert!(!matches!(self.metadata, Metadata::EffectPanic), "effect defect");
        let mut steps = Steps::new();
        for &(before, candidate, after) in &self.edges {
            if *state == before && action == candidate {
                steps.push(((), vec![], after));
            }
        }
        steps
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        let mut claim = Progress::evaluated("reaches_target", |world: &Self, state| {
            world.calls.set(world.calls.get() + 1);
            assert!(!matches!(world.metadata, Metadata::PredicatePanic), "predicate defect");
            ProgressEvaluation {
                trigger: (*state == world.trigger).into(),
                target: if world.hole == Some(*state) {
                    Verdict::Unknown(Hole { reason: "separate target hole" })
                } else {
                    (*state == world.target).into()
                },
                assumption: match world.metadata {
                    Metadata::ChangingAssumption => Assumption::without(&[*state % 2]),
                    Metadata::InvalidAssumption => Assumption::only(&[9]),
                    _ if world.fair => Assumption::fair(&[1]),
                    _ => Assumption::default(),
                },
                bound: if matches!(world.metadata, Metadata::ChangingBound) {
                    Some(u32::from(*state) + 1)
                } else {
                    world.bound
                },
            }
        });
        if let Some(bound) = self.bound_override {
            claim = claim.within_steps(bound);
        }
        if let Some(assumption) = &self.assumption_override {
            claim = claim.assuming(assumption.clone());
        }
        vec![claim]
    }

    fn invariants(&self) -> Vec<Claim<Self>> {
        if matches!(self.metadata, Metadata::SafetyFailure) {
            vec![Claim::new("safety", |_, _| Verdict::Fails)]
        } else {
            vec![]
        }
    }
}

fn record(text: &mut String, name: &str, world: &World, budget: usize) -> umpire_testing::TestResult {
    world.calls.set(0);
    writeln!(text, "{name}:")?;
    match Checker::explore(world, budget) {
        Err(error) => writeln!(text, "refused: {error}")?,
        Ok(result) => {
            let report = result.verify_all()?;
            let before_query = world.calls.get();
            let queried = result.verify_progress(world, &world.progress().remove(0))?;
            let original = report.claims.iter().find(|claim| claim.name == "reaches_target").ok_or("missing original progress claim")?;
            let expected = original.path.as_ref().map(ToString::to_string);
            let actual = queried.path.as_ref().map(ToString::to_string);
            assert_eq!(actual, expected, "query preserves {name}");
            assert_eq!((queried.outcome, &queried.verdict, &queried.hint, &queried.holes), (original.outcome, &original.verdict, &original.hint, &original.holes));
            assert_eq!(world.calls.get() - before_query, if matches!(world.metadata, Metadata::PredicatePanic) { 1 } else { result.counts.reached });
            if matches!(world.metadata, Metadata::ChangingBound) {
                let overridden = result.verify_progress(world, &Progress::new("reaches_target").within_steps(3))?;
                assert_eq!(overridden.verdict, Some(umpire_checker::ProgressVerdict::Verified { longest: Some(3) }));
            }
            if matches!(world.metadata, Metadata::ChangingAssumption) {
                let overridden = result.verify_progress(world, &Progress::new("reaches_target").assuming(Assumption::default()))?;
                assert_eq!(overridden.verdict, Some(umpire_checker::ProgressVerdict::Verified { longest: Some(3) }));
            }
            world.calls.set(before_query);
            let claim = &report.claims[0];
            text.push_str("outcome: ");
            write!(text, "{:?}", claim.outcome)?;
            text.push_str("\nverdict: ");
            write!(text, "{:?}", claim.verdict)?;
            text.push_str("\nhint: ");
            write!(text, "{:?}", claim.hint)?;
            text.push('\n');
            for hole in &claim.holes {
                writeln!(text, "hole: {hole}")?;
            }
            if let Some(path) = &claim.path {
                write!(text, "{path}")?;
            }
            let before = world.calls.get();
            for claim in &report.claims {
                if claim.name != "reaches_target" || claim.path.as_ref().is_some_and(|path| path.failure.is_some()) {
                    assert!(claim.verdict.is_none() && claim.hint.is_none());
                }
            }
            result.verify_all()?;
            writeln!(text, "reached: {}; cached: {}", result.counts.reached, before == world.calls.get())?;
        }
    }
    writeln!(text, "calls: {}\n", world.calls.get())?;
    Ok(())
}

fn query_contracts() -> umpire_testing::TestResult {
    let mut obligation_cycle = World::new(&[(0, 0, 1), (1, 1, 0)], None);
    obligation_cycle.trigger = 1;
    let cycle = Checker::explore(&obligation_cycle, 8)?;
    let query = cycle.verify_progress(&obligation_cycle, &Progress::new("reaches_target"))?;
    let path = query.path.as_ref().ok_or("cycle has no witness")?;
    assert_eq!(path.cycle_start, Some(1));
    assert_eq!(path.steps.iter().map(|step| step.action).collect::<Vec<_>>(), [0, 1, 0]);
    assert_eq!(path.steps.iter().map(|step| step.after).collect::<Vec<_>>(), [1, 0, 1]);
    let independent = World::new(&[(0, 0, 0), (0, 1, 3)], Some(1));
    let retained = Checker::explore(&independent, 8)?;
    let fair = Progress::new("reaches_target").assuming(Assumption::fair(&[1]));
    assert_eq!(retained.verify_progress(&independent, &fair)?.verdict, Some(umpire_checker::ProgressVerdict::MissedDeadline));
    let mut fair_world = World::new(&[(0, 0, 0), (0, 1, 3)], None);
    fair_world.fair = true;
    let retained = Checker::explore(&fair_world, 8)?;
    assert_eq!(retained.verify_progress(&fair_world, &Progress::new("reaches_target").within_steps(2))?.verdict, Some(umpire_checker::ProgressVerdict::MissedDeadline));
    Ok(())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    query_contracts()?;
    umpire_testing::interpreted(generated::world, None, 16, None)?;
    let mut text = String::new();
    let generated = Checker::explore(&generated::world(), 16)?.verify_all()?;
    text.push_str("generated_exact_bound: ");
    write!(text, "{:?}", generated.claims[0].verdict)?;
    text.push_str("\n\n");
    let chain = [(0, 0, 1), (1, 0, 2), (2, 0, 3)];
    for (name, bound) in [("finite_longest", None), ("exact_bound", Some(2)), ("missed_bound", Some(1))] {
        let mut world = World::new(&chain, bound);
        world.trigger = 1;
        record(&mut text, name, &world, 16)?;
    }
    let mut vacuous = World::new(&chain, Some(1));
    vacuous.trigger = 4;
    record(&mut text, "no_obligation", &vacuous, 16)?;
    vacuous.trigger = 3;
    record(&mut text, "target_already_holds", &vacuous, 16)?;

    let escaping_cycle = [(0, 0, 0), (0, 1, 3)];
    for (name, bound) in [("fair_unbounded", None), ("fair_bounded", Some(2))] {
        let mut world = World::new(&escaping_cycle, bound);
        world.fair = true;
        record(&mut text, name, &world, 16)?;
    }
    record(&mut text, "cycle_before_deadline", &World::new(&escaping_cycle, Some(1)), 16)?;
    let competing = [(0, 0, 0), (0, 1, 1), (1, 0, 2), (2, 0, 3), (0, 1, 4)];
    record(&mut text, "deadlock_before_cycle_and_deadline", &World::new(&competing, Some(1)), 16)?;
    let mut intermittent = World::new(&[(0, 0, 1), (1, 0, 0), (0, 1, 3)], Some(1));
    intermittent.fair = true;
    record(&mut text, "intermittently_disabled_fair_action", &intermittent, 16)?;

    let mut hole = World::new(&[(0, 0, 0)], Some(1));
    hole.starts.push(4);
    hole.hole = Some(4);
    record(&mut text, "separate_hole_preserves_cycle", &hole, 16)?;
    for (name, edges) in [
        ("separate_hole_preserves_deadlock", vec![]),
        ("separate_hole_preserves_deadline", chain.to_vec()),
    ] {
        let mut world = World::new(&edges, Some(1));
        world.starts.push(4);
        world.hole = Some(4);
        record(&mut text, name, &world, 16)?;
    }
    let mut unproved = World::new(&chain, Some(3));
    unproved.hole = Some(2);
    record(&mut text, "hole_prevents_success", &unproved, 16)?;
    record(&mut text, "cut_prevents_success", &World::new(&chain, Some(3)), 1)?;
    record(&mut text, "cut_preserves_cycle", &World::new(&escaping_cycle, Some(1)), 1)?;
    record(&mut text, "cut_preserves_deadline", &World::new(&chain, Some(1)), 2)?;

    for (name, metadata) in [
        ("predicate_defect", Metadata::PredicatePanic),
        ("effect_defect", Metadata::EffectPanic),
        ("safety_failure", Metadata::SafetyFailure),
    ] {
        let mut world = World::new(&chain, Some(3));
        world.metadata = metadata;
        record(&mut text, name, &world, 16)?;
    }

    for (name, metadata) in [
        ("inconsistent_bound", Metadata::ChangingBound),
        ("inconsistent_assumption", Metadata::ChangingAssumption),
    ] {
        let mut world = World::new(&chain, Some(3));
        world.metadata = metadata;
        record(&mut text, name, &world, 16)?;
        if matches!(metadata, Metadata::ChangingBound) {
            world.bound_override = Some(3);
        } else {
            world.assumption_override = Some(Assumption::default());
        }
        record(&mut text, &format!("{name}_overridden"), &world, 16)?;
    }
    let mut invalid_assumption = World::new(&chain, Some(2));
    invalid_assumption.metadata = Metadata::InvalidAssumption;
    invalid_assumption.assumption_override = Some(Assumption::default());
    record(&mut text, "assumption_override_keeps_bound", &invalid_assumption, 16)?;
    invalid_assumption.assumption_override = Some(Assumption::only(&[0]));
    invalid_assumption.bound_override = Some(3);
    record(&mut text, "both_overrides", &invalid_assumption, 16)?;
    let mut bound_only = World::new(&escaping_cycle, Some(0));
    bound_only.fair = true;
    bound_only.bound_override = Some(2);
    record(&mut text, "bound_override_keeps_fairness", &bound_only, 16)?;

    for starts in [true, false] {
        for numeric_override in [false, true] {
            let mut world = World::new(&chain, Some(0));
            if !starts {
                world.starts.clear();
            }
            if numeric_override {
                world.bound = Some(3);
                world.bound_override = Some(0);
            }
            record(&mut text, &format!("zero_bound/starts={starts}/override={numeric_override}"), &world, 16)?;
        }
    }
    text.pop();
    Ok(umpire_testing::Files::from([("report.txt".into(), text)]))
}
