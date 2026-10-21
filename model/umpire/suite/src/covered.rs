use umpire_dsl::{Alternative, Associated, IntoWorldPath, Machine, Path};

use crate::{
    Coverage, Refusal, Run, SpaceBinding,
    coverage::{ObservedStep, WitnessObservation, coverage},
    pattern_search::{Graph, Rng, Witness},
    search::{self, Rooted, Trace},
};

type WorldPath<S> = Path<
    S,
    <<S as Associated>::World as Machine>::Action,
    <<S as Associated>::World as Machine>::Outcome,
>;
type Choice<S> = Alternative<
    S,
    <<S as Associated>::World as Machine>::Action,
    <<S as Associated>::World as Machine>::Outcome,
>;
const MAX_RECORDS: usize = 1_000_000;

/// The number of authored alternatives a test expects before any Model search.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Cases(pub usize);

/// Selected ordinary runs, their concrete expansion and typed witness coverage.
pub struct Covered<S: Associated> {
    runs: Vec<Run<S>>,
    coverage: Coverage,
    expansion: String,
}

impl<S: Associated> Covered<S> {
    /// The runs in authored alternative order, or sampled experiment order.
    pub fn runs(&self) -> &[Run<S>] {
        &self.runs
    }

    /// The number of selected runs.
    pub fn len(&self) -> usize {
        self.runs.len()
    }

    /// Whether the finite experiment selected no runs.
    pub fn is_empty(&self) -> bool {
        self.runs.is_empty()
    }

    /// Checks one assertion against every selected ordinary run.
    pub fn all(&self, predicate: impl FnMut(&Run<S>) -> bool) -> bool {
        self.runs.iter().all(predicate)
    }

    /// Actual witness observations against the world's declared catalogs and registrations.
    pub const fn coverage(&self) -> &Coverage {
        &self.coverage
    }

    /// Deterministic choices, exact retained outcomes, actions and final states for goldens.
    pub fn expansion(&self) -> &str {
        &self.expansion
    }
}

fn label<S: Associated>(alternative: &Choice<S>) -> String {
    if alternative.choices.is_empty() {
        "state pattern".into()
    } else {
        alternative.choices.join(", ")
    }
}

fn admit<S: Associated>(
    path: &WorldPath<S>,
    budget: usize,
    expected: Option<Cases>,
) -> Result<Vec<Choice<S>>, Refusal>
where
    <S::World as Machine>::Action: 'static,
{
    let count = path.alternative_count()?;
    if let Some(Cases(expected)) = expected
        && count != expected
    {
        let alternatives = path.alternatives(budget).map_or_else(
            |error| format!("alternatives not expanded: {error}"),
            |alternatives| {
                alternatives
                    .iter()
                    .enumerate()
                    .map(|(index, alternative)| {
                        format!("alternative {index}: {}", label::<S>(alternative))
                    })
                    .collect::<Vec<_>>()
                    .join("\n")
            },
        );
        return Err(Refusal::Cases { actual: count, expected, alternatives });
    }
    if count == 0 {
        return Err(Refusal::Empty);
    }
    Ok(path.alternatives(budget)?)
}

fn alternative<S: Associated>(index: usize, choice: &Choice<S>, failure: Refusal) -> Refusal {
    Refusal::Alternative { index, choices: label::<S>(choice), failure: Box::new(failure) }
}

/// Finds one shortest witness per authored alternative over one retained Model graph.
/// A case-count mismatch precedes Model stepping; a cut graph always refuses coverage.
pub fn cover<S: Associated, A, O, K>(
    space: &impl SpaceBinding<S>,
    cases: Cases,
    path: Path<S, A, O>,
) -> Result<Covered<S>, Refusal>
where
    Path<S, A, O>: IntoWorldPath<S, K>,
    <S::World as Machine>::Action: 'static,
{
    let path = path.into_world_path();
    let alternatives = admit::<S>(&path, space.alternative_budget(), Some(cases))?;
    let world = space.world();
    let rooted = Rooted { world: world.as_ref(), starts: space.starts().to_vec(), query: None };
    let explored = rooted.explore(space.state_budget(), space.catalog_limit())?;
    if explored.counts.cut {
        return Err(Refusal::Cut { budget: space.state_budget() });
    }
    search::failed(&explored.verify_all()?, 0, None)?;
    let graph = Graph::new(&explored)?;
    let mut witnesses = Vec::with_capacity(alternatives.len());
    let mut records = 0usize;
    for (index, choice) in alternatives.iter().enumerate() {
        let found =
            graph.shortest(choice, true).map_err(|error| alternative::<S>(index, choice, error))?;
        charge(&mut records, found.rows.len() + 1)?;
        witnesses.push((index, found));
    }
    let header = format!("cover: {} alternatives, {} cases\n", alternatives.len(), witnesses.len());
    assemble::<S>(&world, &explored, &alternatives, witnesses, header, space.catalog_limit())
}

/// Performs a finite seeded experiment of actual pattern-admitted random walks.
/// Each selected alternative must finish within `Space::walk_bound`; cuts refuse the experiment.
/// Sampling establishes only this experiment, never exhaustive alternative coverage.
pub fn sample<S: Associated, A, O, K>(
    space: &impl SpaceBinding<S>,
    path: Path<S, A, O>,
    n: usize,
    seed: u64,
) -> Result<Covered<S>, Refusal>
where
    Path<S, A, O>: IntoWorldPath<S, K>,
    <S::World as Machine>::Action: 'static,
{
    let path = path.into_world_path();
    let alternatives = admit::<S>(&path, space.alternative_budget(), None)?;
    if n > MAX_RECORDS {
        return Err(Refusal::SearchLimit { limit: MAX_RECORDS });
    }
    let world = space.world();
    let rooted = Rooted { world: world.as_ref(), starts: space.starts().to_vec(), query: None };
    let explored = rooted.explore(space.state_budget(), space.catalog_limit())?;
    if explored.counts.cut {
        return Err(Refusal::Cut { budget: space.state_budget() });
    }
    search::failed(&explored.verify_all()?, 0, None)?;
    let graph = Graph::new(&explored)?;
    let mut rng = Rng::new(seed);
    let mut witnesses = Vec::with_capacity(n);
    let mut records = 0usize;
    for _ in 0..n {
        let index = rng.index(alternatives.len());
        let choice = &alternatives[index];
        let found = graph
            .walk(choice, true, space.walk_bound(), &mut rng)
            .map_err(|error| alternative::<S>(index, choice, error))?;
        charge(&mut records, found.rows.len() + 1)?;
        witnesses.push((index, found));
    }
    let header = format!(
        "sample: seed {seed}, {n} requested walks, walk bound {} steps, {} alternatives; finite experiment only\n",
        space.walk_bound(),
        alternatives.len(),
    );
    assemble::<S>(&world, &explored, &alternatives, witnesses, header, space.catalog_limit())
}

fn charge(records: &mut usize, count: usize) -> Result<(), Refusal> {
    *records = records.checked_add(count).ok_or(Refusal::SearchLimit { limit: MAX_RECORDS })?;
    if *records > MAX_RECORDS {
        return Err(Refusal::SearchLimit { limit: MAX_RECORDS });
    }
    Ok(())
}

fn assemble<S: Associated>(
    world: &Rc<S::World>,
    explored: &umpire_checker::Result<Rooted<'_, S::World>>,
    alternatives: &[Choice<S>],
    witnesses: Vec<(usize, Witness<Rooted<'_, S::World>>)>,
    mut expansion: String,
    catalog_limit: usize,
) -> Result<Covered<S>, Refusal> {
    let mut observations = Vec::with_capacity(witnesses.len());
    for (case, (index, witness)) in witnesses.iter().enumerate() {
        append(
            &mut expansion,
            format_args!(
                "case {case}, alternative {index}: [{}]; start {}",
                label::<S>(&alternatives[*index]),
                witness.trace.start
            ),
        );
        let mut observation = WitnessObservation::<S::World> {
            states: vec![&witness.trace.start],
            rows: Vec::with_capacity(witness.rows.len()),
        };
        for (source, ordinal) in &witness.rows {
            let row = explored
                .rows(*source)?
                .nth(*ordinal)
                .unwrap_or_else(|| unreachable!("witness retains an existing row"));
            let before = explored
                .states()
                .nth(source.0)
                .unwrap_or_else(|| unreachable!("row source is admitted"))
                .1;
            observation.states.push(&row.emitted.2);
            observation.rows.push(ObservedStep {
                before,
                action: &row.action,
                emitted: &row.emitted,
            });
            append(
                &mut expansion,
                format_args!(
                    "; {} (choice {}) => {:?}, facts {:?} -> {}",
                    row.action, row.choice, row.emitted.0, row.emitted.1, row.emitted.2
                ),
            );
        }
        let final_state =
            witness.trace.steps.last().map_or(&witness.trace.start, |step| &step.after);
        append(&mut expansion, format_args!("; final {final_state}\n"));
        observations.push(observation);
    }
    let coverage = coverage(world.as_ref(), catalog_limit, &observations)?;
    let runs = witnesses
        .into_iter()
        .map(|(_, witness)| Run::retained(world.clone(), witness.trace))
        .collect();
    Ok(Covered { runs, coverage, expansion })
}

fn append(output: &mut String, arguments: fmt::Arguments<'_>) {
    output.write_fmt(arguments).unwrap_or_else(|_| unreachable!("String formatting is infallible"));
}

pub(crate) fn find_pattern<S: Associated>(
    space: &impl SpaceBinding<S>,
    path: &WorldPath<S>,
) -> Result<Trace<S::World>, Refusal>
where
    <S::World as Machine>::Action: 'static,
{
    let alternatives = admit::<S>(path, space.alternative_budget(), None)?;
    let world = space.world();
    let rooted = Rooted { world: world.as_ref(), starts: space.starts().to_vec(), query: None };
    let explored = rooted.explore(space.state_budget(), space.catalog_limit())?;
    let graph = Graph::new(&explored)?;
    let mut best = None;
    let mut refusal = None;
    for choice in &alternatives {
        match graph.shortest(choice, true) {
            Ok(found)
                if best.as_ref().is_none_or(|old: &Trace<S::World>| {
                    found.trace.steps.len() < old.steps.len()
                }) =>
            {
                best = Some(found.trace);
            }
            Ok(_) => {}
            Err(error @ Refusal::Unbridged { .. }) => {
                if refusal.is_none() || matches!(error, Refusal::Unbridged { cut: true, .. }) {
                    refusal = Some(error);
                }
            }
            Err(error) => return Err(error),
        }
    }
    search::failed(&explored.verify_all()?, 0, best.as_ref().map(|trace| trace.steps.len()))?;
    best.ok_or_else(|| {
        refusal.unwrap_or(Refusal::Unbridged {
            waypoint: 0,
            state: "no start state".into(),
            cut: explored.counts.cut,
        })
    })
}
use std::{
    fmt::{self, Write},
    rc::Rc,
};
