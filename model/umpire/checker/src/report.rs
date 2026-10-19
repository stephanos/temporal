use std::fmt::Write;

use umpire_dsl::{Machine, Outcome, Spell};

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

impl<S: Spell, A: Spell> Spell for Path<S, A> {
    fn spell(&self, out: &mut String) {
        if self.cycle_start.is_some() {
            out.push_str("prefix:\n");
        }
        out.push_str("start: ");
        self.start.spell(out);
        out.push('\n');
        for (index, step) in self.steps.iter().enumerate() {
            if self.cycle_start == Some(index) {
                out.push_str("cycle:\n");
            }
            write!(out, "{}: ", index + 1).ok();
            step.action.spell(out);
            write!(out, " choice={} -> ", step.choice).ok();
            step.after.spell(out);
            out.push('\n');
        }
        if self.cycle_start.is_some() {
            out.push_str("repeat cycle\n");
        }
        if let Some(failure) = &self.failure {
            writeln!(out, "failure: {failure}").ok();
        }
    }
}

/// A logical progress claim's temporal classification.
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

impl Spell for ProgressVerdict {
    fn spell(&self, out: &mut String) {
        match self {
            Self::Deadlock => out.push_str("Deadlock"),
            Self::Cycle => out.push_str("Cycle"),
            Self::MissedDeadline => out.push_str("MissedDeadline"),
            Self::Verified { longest } => {
                out.push_str("Verified { longest: ");
                longest.spell(out);
                out.push_str(" }");
            }
            Self::Unknown => out.push_str("Unknown"),
        }
    }
}

/// A named claim's aggregate verdict and first counterexample.
#[derive(Clone)]
pub struct ClaimResult<S, A> {
    /// The model's declaration name, or an effect failure's discovery identity.
    pub name: String,
    /// Whether the claim holds, fails, or remains undecided.
    pub outcome: Outcome,
    /// Temporal classification for a logical progress result; absent for safety or defects.
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

impl<M: Machine> Report<M> {
    /// Panics for refuted or unknown verification, printing the first refutation's path.
    pub fn assert_verified(&self) {
        if let Some(claim) = self.claims.iter().find(|claim| claim.outcome == Outcome::Refuted) {
            let mut path = String::new();
            if let Some(counterexample) = &claim.path {
                counterexample.spell(&mut path);
            }
            panic!("claim {} refuted:\n{path}", claim.name);
        }
        if self.outcome() != Outcome::Verified {
            let mut report = String::new();
            self.spell(&mut report);
            panic!("verification unknown:\n{report}");
        }
    }
}

impl<M: Machine> Spell for Report<M> {
    fn spell(&self, out: &mut String) {
        writeln!(out, "starts: {}", self.counts.starts).ok();
        writeln!(out, "reached: {}", self.counts.reached).ok();
        writeln!(out, "emitted: {}", self.counts.emitted).ok();
        writeln!(out, "evaluated: {}", self.counts.evaluated).ok();
        writeln!(out, "refused: {}", self.counts.refused).ok();
        writeln!(out, "depth: {}", self.counts.depth).ok();
        writeln!(out, "cut: {}", self.counts.cut).ok();
        out.push_str("outcome: ");
        self.outcome().spell(out);
        out.push('\n');
        for claim in &self.claims {
            write!(out, "claim {}: ", claim.name).ok();
            claim.outcome.spell(out);
            out.push('\n');
            for hole in &claim.holes {
                writeln!(out, "hole {}: {hole}", claim.name).ok();
            }
            if let Some(failure) = claim.path.as_ref().and_then(|path| path.failure.as_ref()) {
                writeln!(out, "failure {}: {failure}", claim.name).ok();
            }
        }
        if self.warnings.is_empty() {
            writeln!(out, "vacuity: none").ok();
        }
        for warning in &self.warnings {
            writeln!(out, "vacuity: {warning}").ok();
        }
    }
}
