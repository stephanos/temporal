use std::fmt;

use umpire_dsl::{Machine, Outcome};

use crate::Counts;

/// One emitted choice in a counterexample.
#[derive(Clone)]
pub struct PathStep<S, A> {
    /// The action taken from the preceding state.
    pub action: A,
    /// The choice's zero-based position within the action's ordered results.
    pub choice: usize,
    /// The state reached by the choice.
    pub after: S,
}

/// A discovered counterexample rooted in an initial state.
#[derive(Clone)]
pub struct Path<S, A> {
    /// The initial state from which the trace starts.
    pub start: S,
    /// Emitted choices in execution order.
    pub steps: Vec<PathStep<S, A>>,
    /// A caught effect or predicate panic following the trace, if any.
    pub failure: Option<String>,
}

impl<S: fmt::Display, A: fmt::Debug> fmt::Display for Path<S, A> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "start: {}", self.start)?;
        for (index, step) in self.steps.iter().enumerate() {
            writeln!(
                f,
                "{}: {:?} choice={} -> {}",
                index + 1,
                step.action,
                step.choice,
                step.after
            )?;
        }
        if let Some(failure) = &self.failure {
            writeln!(f, "failure: {failure}")?;
        }
        Ok(())
    }
}

/// A named claim's aggregate verdict and first counterexample.
#[derive(Clone)]
pub struct ClaimResult<S, A> {
    /// The model's declaration name, or an effect failure's discovery identity.
    pub name: String,
    /// Whether the claim holds, fails, or remains undecided.
    pub outcome: Outcome,
    /// A concrete path witnessing refutation.
    pub path: Option<Path<S, A>>,
    /// Reached hole reasons, in first-observed order.
    pub holes: Vec<&'static str>,
}

/// Safety/step results and vacuity diagnostics for an exploration.
pub struct Report<M: Machine> {
    /// Graph statistics shared with the exploration result.
    pub counts: Counts,
    /// Claims in declaration order, followed by caught effect failures in discovery order.
    pub claims: Vec<ClaimResult<M::State, M::Action>>,
    /// Action-catalog warnings followed by filtered-claim warnings in declaration order.
    pub warnings: Vec<String>,
}

pub(crate) fn aggregate<S, A>(cut: bool, claims: &[ClaimResult<S, A>]) -> Outcome {
    if claims.iter().any(|claim| claim.outcome == Outcome::Refuted) {
        Outcome::Refuted
    } else if cut || claims.iter().any(|claim| claim.outcome == Outcome::Unknown) {
        Outcome::Unknown
    } else {
        Outcome::Verified
    }
}

impl<M: Machine> Report<M> {
    /// The aggregate verdict, preserving refutations despite cuts or holes.
    pub fn outcome(&self) -> Outcome {
        aggregate(self.counts.cut, &self.claims)
    }
}

impl<M: Machine> Report<M>
where
    M::State: fmt::Display,
{
    /// Panics for refuted or unknown verification, printing the first refutation's path.
    pub fn assert_verified(&self) {
        if let Some(claim) = self.claims.iter().find(|claim| claim.outcome == Outcome::Refuted) {
            let path = claim.path.as_ref().map_or_else(String::new, ToString::to_string);
            panic!("claim {} refuted:\n{path}", claim.name);
        }
        assert!(self.outcome() == Outcome::Verified, "verification unknown:\n{self}");
    }
}

impl<M: Machine> fmt::Display for Report<M> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "starts: {}", self.counts.starts)?;
        writeln!(f, "reached: {}", self.counts.reached)?;
        writeln!(f, "emitted: {}", self.counts.emitted)?;
        writeln!(f, "evaluated: {}", self.counts.evaluated)?;
        writeln!(f, "refused: {}", self.counts.refused)?;
        writeln!(f, "depth: {}", self.counts.depth)?;
        writeln!(f, "cut: {}", self.counts.cut)?;
        writeln!(f, "outcome: {}", verdict(self.outcome()))?;
        for claim in &self.claims {
            writeln!(f, "claim {}: {}", claim.name, verdict(claim.outcome))?;
            for hole in &claim.holes {
                writeln!(f, "hole {}: {hole}", claim.name)?;
            }
            if let Some(failure) = claim.path.as_ref().and_then(|path| path.failure.as_ref()) {
                writeln!(f, "failure {}: {failure}", claim.name)?;
            }
        }
        if self.warnings.is_empty() {
            writeln!(f, "vacuity: none")?;
        }
        for warning in &self.warnings {
            writeln!(f, "vacuity: {warning}")?;
        }
        Ok(())
    }
}

fn verdict(outcome: Outcome) -> &'static str {
    match outcome {
        Outcome::Verified => "verified",
        Outcome::Refuted => "refuted",
        Outcome::Unknown => "unknown",
    }
}
