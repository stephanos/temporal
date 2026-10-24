use std::cell::Cell;
use std::fmt::Write;

use umpire_checker::{Checker, Refusal};
use umpire_dsl::{Assumption, AssumptionExt, Machine, Progress, ProgressEvaluation, Steps, Verdict};

#[derive(Clone, Copy)]
enum Source {
    Registered,
    RegisteredWithProgress,
    Manual,
    Generated,
}

struct World {
    source: Source,
    assumption: Assumption<u8>,
    starts: bool,
    actions: Vec<u8>,
    steps: Cell<usize>,
    calls: Cell<usize>,
}

impl World {
    fn new(source: Source, assumption: Assumption<u8>, starts: bool) -> Self {
        Self {
            source,
            assumption,
            starts,
            actions: vec![0, 1],
            steps: Cell::new(0),
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
        if self.starts { vec![0] } else { vec![] }
    }

    fn actions(&self) -> Vec<u8> {
        self.actions.clone()
    }

    fn internal(&self, _: &u8) -> bool {
        false
    }

    fn step(&self, _: &u8, _: u8) -> Steps<u8, (), ()> {
        self.steps.set(self.steps.get() + 1);
        Steps::new()
    }

    fn assumptions(&self) -> Vec<(&'static str, Assumption<u8>)> {
        if matches!(self.source, Source::Registered | Source::RegisteredWithProgress) {
            vec![("registered", self.assumption.clone())]
        } else {
            vec![]
        }
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        match self.source {
            Source::Registered => vec![],
            Source::Manual => vec![Progress::new("pending")
                .whenever(|world: &Self, _| {
                    world.calls.set(world.calls.get() + 1);
                    Verdict::Holds
                })
                .eventually(|_, _| Verdict::Fails)
                .assuming(self.assumption.clone())],
            Source::Generated | Source::RegisteredWithProgress => {
                vec![Progress::evaluated("pending", |world: &Self, _| {
                    world.calls.set(world.calls.get() + 1);
                    ProgressEvaluation {
                        bound: None,
                        trigger: Verdict::Holds,
                        target: Verdict::Fails,
                        assumption: if matches!(world.source, Source::Generated) {
                            world.assumption.clone()
                        } else {
                            Assumption::default()
                        },
                    }
                })]
            }
        }
    }
}

fn record(text: &mut String, name: &str, world: &World) -> umpire_testing::TestResult {
    write!(text, "{name}: ")?;
    match Checker::explore(world, 8) {
        Err(error @ (Refusal::ScopeAction { .. } | Refusal::EmptyAssumption)) => {
            write!(text, "refused: {error}")?;
        }
        Err(error) => return Err(error.into()),
        Ok(result) => write!(text, "{:?}", result.verify_all()?.outcome())?,
    }
    writeln!(text, "; steps={} calls={}", world.steps.get(), world.calls.get())?;
    Ok(())
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut text = String::new();
    for (name, assumption) in [
        ("only_unknown", Assumption::<u8>::only(&[3_u8])),
        ("excluded_unknown", Assumption::<u8>::without(&[3_u8])),
        ("fair_unknown", Assumption::<u8>::fair(&[3_u8])),
        ("fair_outside_only", Assumption::<u8>::only(&[0_u8]).fair([3_u8])),
        ("fair_excluded_unknown", Assumption::<u8>::fair(&[3_u8]).without([3_u8])),
    ] {
        for (source_name, source) in [
            ("registered", Source::Registered),
            ("manual", Source::Manual),
            ("generated", Source::Generated),
        ] {
            for starts in [true, false] {
                record(
                    &mut text,
                    &format!("{name}/{source_name}/starts={starts}"),
                    &World::new(source, assumption.clone(), starts),
                )?;
            }
        }
    }
    record(
        &mut text,
        "registered_before_evaluation",
        &World::new(Source::RegisteredWithProgress, Assumption::only(&[3_u8]), true),
    )?;
    for (name, assumption) in [
        ("only_empty", Assumption::<u8>::only(&[] as &[u8])),
        ("exclude_all", Assumption::<u8>::without(&[0_u8, 1])),
        ("only_then_without", Assumption::<u8>::only(&[0_u8]).without([0_u8])),
    ] {
        for (source_name, source) in [
            ("registered", Source::Registered),
            ("manual", Source::Manual),
            ("generated", Source::Generated),
        ] {
            for starts in [true, false] {
                record(
                    &mut text,
                    &format!("{name}/{source_name}/starts={starts}"),
                    &World::new(source, assumption.clone(), starts),
                )?;
            }
        }
    }
    record(
        &mut text,
        "disabled_press_only",
        &World::new(Source::Manual, Assumption::only(&[0_u8]), true),
    )?;
    let mut no_actions = World::new(Source::Manual, Assumption::default(), true);
    no_actions.actions.clear();
    record(&mut text, "unrestricted_zero_actions", &no_actions)?;
    record(
        &mut text,
        "valid_zero_starts",
        &World::new(Source::Manual, Assumption::only(&[0_u8]).fair([0_u8]), false),
    )?;
    Ok(umpire_testing::Files::from([("report.txt".into(), text)]))
}
