use std::fmt::{self, Display};

use umpire_dsl::{CheckOutcome, Machine};

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
    /// Index of the first edge in a repeating closed walk, when this is a lasso.
    pub cycle_start: Option<usize>,
    /// A failed check or caught effect or predicate panic at the end of the trace, if any.
    pub failure: Option<String>,
}

impl<S: Display, A: Display> Display for Path<S, A> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.cycle_start.is_some() {
            out.write_str("prefix:\n")?;
        }
        writeln!(out, "start: {}", self.start)?;
        for (index, step) in self.steps.iter().enumerate() {
            if self.cycle_start == Some(index) {
                out.write_str("cycle:\n")?;
            }
            writeln!(
                out,
                "{}: {} choice={} -> {}",
                index + 1,
                step.action,
                step.choice,
                step.after
            )?;
        }
        if self.cycle_start.is_some() {
            out.write_str("repeat cycle\n")?;
        }
        if let Some(failure) = &self.failure {
            writeln!(out, "failure: {failure}")?;
        }
        Ok(())
    }
}

/// A logical progress claim's progress classification.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ProgressVerdict {
    /// An outstanding obligation reaches a state with no admitted successor.
    Deadlock,
    /// A target-avoiding closed walk satisfies the explicit weak fairness assumptions.
    Cycle,
    /// A bounded obligation can avoid its target for the declared number of transitions.
    MissedDeadline,
    /// Every outstanding obligation reaches its target.
    Verified {
        /// Longest first-target distance, absent when fairness allows arbitrary delay.
        longest: Option<u32>,
    },
    /// Missing behavior or inconsistent metadata prevents a conclusion.
    Unknown,
}

impl Display for ProgressVerdict {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        fmt::Debug::fmt(self, out)
    }
}

/// A named claim's aggregate verdict and first counterexample.
#[derive(Clone)]
pub struct ClaimResult<S, A> {
    /// The model's declaration name, or an effect failure's discovery identity.
    pub name: String,
    /// Whether the claim holds, fails, or remains undecided.
    pub outcome: CheckOutcome,
    /// Progress classification for a logical progress result; absent for safety or defects.
    pub verdict: Option<ProgressVerdict>,
    /// Advice attached to a logical progress refutation.
    pub hint: Option<String>,
    /// A concrete path witnessing refutation.
    pub path: Option<Path<S, A>>,
    /// Reached hole reasons, in first-observed order.
    pub holes: Vec<&'static str>,
}

/// Claim results and vacuity diagnostics for an exploration.
pub struct Report<M: Machine> {
    /// Graph statistics shared with the exploration result.
    pub counts: Counts,
    /// Claims in declaration order, with supplemental query claims in their registration slots,
    /// followed by caught effect failures in discovery order.
    pub claims: Vec<ClaimResult<M::State, M::Action>>,
    /// Action-catalog warnings followed by filtered-claim warnings in declaration order.
    pub warnings: Vec<String>,
}

pub(crate) fn aggregate<S, A>(cut: bool, claims: &[ClaimResult<S, A>]) -> CheckOutcome {
    if claims.iter().any(|claim| claim.outcome == CheckOutcome::Refuted) {
        CheckOutcome::Refuted
    } else if cut || claims.iter().any(|claim| claim.outcome == CheckOutcome::Unknown) {
        CheckOutcome::Unknown
    } else {
        CheckOutcome::Verified
    }
}

impl<M: Machine> Report<M> {
    /// The aggregate verdict, preserving refutations despite cuts or holes.
    pub fn outcome(&self) -> CheckOutcome {
        aggregate(self.counts.cut, &self.claims)
    }
}

impl<M: Machine> Report<M> {
    /// Panics for refuted or unknown verification, printing the first refutation's path.
    pub fn assert_verified(&self) {
        if let Some(claim) = self.claims.iter().find(|claim| claim.outcome == CheckOutcome::Refuted)
        {
            let path = claim.path.as_ref().map(ToString::to_string).unwrap_or_default();
            panic!("claim {} refuted:\n{path}", claim.name);
        }
        assert!(self.outcome() == CheckOutcome::Verified, "verification unknown:\n{self}");
    }
}

impl<M: Machine> Display for Report<M> {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(out, "starts: {}", self.counts.starts)?;
        writeln!(out, "reached: {}", self.counts.reached)?;
        writeln!(out, "emitted: {}", self.counts.emitted)?;
        writeln!(out, "evaluated: {}", self.counts.evaluated)?;
        writeln!(out, "refused: {}", self.counts.refused)?;
        writeln!(out, "depth: {}", self.counts.depth)?;
        writeln!(out, "cut: {}", self.counts.cut)?;
        writeln!(out, "outcome: {:?}", self.outcome())?;
        for claim in &self.claims {
            writeln!(out, "claim {}: {:?}", claim.name, claim.outcome)?;
            for hole in &claim.holes {
                writeln!(out, "hole {}: {hole}", claim.name)?;
            }
            if let Some(failure) = claim.path.as_ref().and_then(|path| path.failure.as_ref()) {
                writeln!(out, "failure {}: {failure}", claim.name)?;
            }
        }
        if self.warnings.is_empty() {
            writeln!(out, "vacuity: none")?;
        }
        for warning in &self.warnings {
            writeln!(out, "vacuity: {warning}")?;
        }
        Ok(())
    }
}
