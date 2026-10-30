//! The differential check: a Model explored compiled and interpreted from its embedded tree
//! must produce the same states, edges, outcomes, claim verdicts and counterexamples.

use std::time::{Duration, Instant};

use umpire_checker::{Checker, Interpreted};
use umpire_dsl::{__lifted::Interprets, ActionValues, Claim, Machine, Values};

/// What one agreeing differential run explored, and how long each exploration took.
#[derive(Clone, Copy, Debug)]
pub struct Agreement {
    /// Admitted states.
    pub states: usize,
    /// Emitted edges.
    pub edges: usize,
    /// The compiled exploration.
    pub compiled: Duration,
    /// The interpreted exploration.
    pub interpreted: Duration,
}

/// Explores the World `world` builds compiled and interpreted from the same starts, or each
/// from its own starts when `starts` is `None`, under one state budget, and checks a fixture's
/// negative claim (built once for each side) the same way. Every rendered checker file must
/// be identical, and so must any refusal.
///
/// # Errors
/// Returns a World the interpreter refuses, or the first disagreement.
pub fn interpreted<M>(
    world: impl Fn() -> M,
    starts: Option<Vec<M::State>>,
    budget: usize,
    negative: Option<(Claim<M>, Claim<M>)>,
) -> Result<Agreement, Box<dyn std::error::Error>>
where
    M: Interprets + Values,
    M::Action: ActionValues<M>,
{
    let compiled = world();
    let (negative, interpreted_negative) = negative.unzip();
    let mut interpreted = Interpreted::new(world())?;
    if let Some(claim) = interpreted_negative {
        interpreted = interpreted.with_negative(claim);
    }
    compare(&compiled, &interpreted, starts, budget, negative)
}

/// [`interpreted`] for an already interpreted World, such as one of a seeded mutant tree.
///
/// # Errors
/// Returns the first disagreement.
pub fn compare<M>(
    compiled: &M,
    interpreted: &Interpreted<M>,
    starts: Option<Vec<M::State>>,
    budget: usize,
    negative: Option<Claim<M>>,
) -> Result<Agreement, Box<dyn std::error::Error>>
where
    M: Interprets + Values,
    M::Action: ActionValues<M>,
{
    let (expected, compiled_time) = explored(compiled, starts.clone(), budget, negative);
    let (actual, interpreted_time) = explored(interpreted, starts, budget, interpreted.negative());
    let (states, edges) = match (expected, actual) {
        (Ok((expected, counts)), Ok((actual, _))) => {
            if let Some(difference) = difference(&expected, &actual) {
                return Err(format!(
                    "interpreted exploration disagrees with compiled: {difference}"
                )
                .into());
            }
            counts
        }
        (Err(expected), Err(actual)) if expected == actual => (0, 0),
        (expected, actual) => {
            let outcome =
                |result: Explored| result.map_or_else(|refusal| refusal, |_| "explored".into());
            return Err(format!(
                "interpreted exploration disagrees with compiled:\n  compiled:    {}\n  interpreted: {}",
                outcome(expected),
                outcome(actual)
            )
            .into());
        }
    };
    if std::env::var("UMPIRE_QUICK_CACHE").as_deref() == Ok("1") {
        quick_agreement::<M>(states, edges);
    }
    Ok(Agreement { states, edges, compiled: compiled_time, interpreted: interpreted_time })
}

#[allow(clippy::print_stderr)]
fn quick_agreement<M>(states: usize, edges: usize) {
    eprintln!(
        "quick native/interpreter agreement {}: {states} states, {edges} edges",
        std::any::type_name::<M>()
    );
}

type Explored = Result<(crate::Files, (usize, usize)), String>;

fn explored<M: Machine>(
    world: &M,
    starts: Option<Vec<M::State>>,
    budget: usize,
    negative: Option<Claim<M>>,
) -> (Explored, Duration) {
    let starts = match starts.map_or_else(|| enumerated(world), Ok) {
        Ok(starts) => starts,
        Err(refusal) => return (Err(refusal), Duration::ZERO),
    };
    let started = Instant::now();
    let graph = Checker::explore_within(world, starts.clone(), budget, Checker::CATALOG_LIMIT);
    let elapsed = started.elapsed();
    let files = graph.map_err(|refusal| format!("refused: {refusal}")).and_then(|graph| {
        let mut files = crate::render(&graph).map_err(|refusal| refusal.to_string())?;
        files.insert(
            "rows.txt".into(),
            crate::render_rows(&graph).map_err(|refusal| refusal.to_string())?,
        );
        if let Some(claim) = negative {
            let report = Checker::refuted_from(world, starts, claim)
                .and_then(|negative| negative.verify_all())
                .map_err(|refusal| format!("negative claim refused: {refusal}"))?;
            files.insert("negative_report.txt".into(), report.to_string());
            crate::add_paths(&mut files, report);
        }
        Ok((files, (graph.counts.reached, graph.counts.emitted)))
    });
    (files, elapsed)
}

/// A World's starts, or the panic that enumerating them raised, so two sides can be compared
/// even where a start domain fails.
fn enumerated<M: Machine>(world: &M) -> Result<Vec<M::State>, String> {
    std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| world.starts())).map_err(|panic| {
        let message = umpire_checker::panic_message(panic.as_ref());
        format!("starts panicked: {message}")
    })
}

/// Every file that differs, with its first differing line on each side.
fn difference(expected: &crate::Files, actual: &crate::Files) -> Option<String> {
    let mut differences = Vec::new();
    let names: std::collections::BTreeSet<_> = expected.keys().chain(actual.keys()).collect();
    for name in names {
        let (Some(expected), Some(actual)) = (expected.get(name), actual.get(name)) else {
            let side = if expected.contains_key(name) { "compiled" } else { "interpreted" };
            differences.push(format!("only the {side} exploration writes {name}"));
            continue;
        };
        if expected == actual {
            continue;
        }
        let mut lines = expected.lines().zip(actual.lines()).enumerate();
        differences.push(match lines.find(|(_, (expected, actual))| expected != actual) {
            Some((line, (expected, actual))) => format!(
                "{name} line {}:\n  compiled:    {expected}\n  interpreted: {actual}",
                line + 1
            ),
            None => format!(
                "{name} has {} compiled and {} interpreted lines",
                expected.lines().count(),
                actual.lines().count()
            ),
        });
    }
    (!differences.is_empty()).then(|| differences.join("\n"))
}

/// Runs the differential check when a concrete World's `#[machine]` module was admitted.
#[doc(hidden)]
pub mod probe {
    use super::{Agreement, Claim, Interprets, Machine, Values};
    use crate::__probe::Probe;
    use umpire_dsl::ActionValues;

    type Checked = Result<Option<Agreement>, Box<dyn std::error::Error>>;

    pub trait Admitted<M: Machine> {
        fn interpreted(
            self,
            world: impl Fn() -> M,
            budget: usize,
            negative: Option<(Claim<M>, Claim<M>)>,
        ) -> Checked;
    }
    impl<M> Admitted<M> for &Probe<M>
    where
        M: Interprets + Values,
        M::Action: ActionValues<M>,
    {
        fn interpreted(
            self,
            world: impl Fn() -> M,
            budget: usize,
            negative: Option<(Claim<M>, Claim<M>)>,
        ) -> Checked {
            super::interpreted(world, None, budget, negative).map(Some)
        }
    }
    pub trait Unadmitted<M: Machine> {
        fn interpreted(
            self,
            world: impl Fn() -> M,
            budget: usize,
            negative: Option<(Claim<M>, Claim<M>)>,
        ) -> Checked;
    }
    impl<M: Machine> Unadmitted<M> for &&Probe<M> {
        fn interpreted(
            self,
            _: impl Fn() -> M,
            _: usize,
            _: Option<(Claim<M>, Claim<M>)>,
        ) -> Checked {
            Ok(None)
        }
    }
}
