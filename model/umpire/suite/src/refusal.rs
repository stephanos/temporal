/// A request the suite cannot answer as written, distinct from a test's own failure.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
pub enum Refusal {
    /// The action emits no step from the state.
    #[error("{action} is disabled at {state}")]
    Disabled {
        /// The action's spelling.
        action: String,
        /// The state's spelling.
        state: String,
    },
    /// The action emits several steps, and an example takes exactly one.
    #[error("{action} emits {choices} steps at {state}; an example takes exactly one")]
    Ambiguous {
        /// The action's spelling.
        action: String,
        /// How many steps it emits.
        choices: usize,
        /// The state's spelling.
        state: String,
    },
    /// `Run::along` needs a concrete state to start from.
    #[error("Run::along starts at a state, not a predicate: make waypoint 0 a state")]
    Unanchored,
    /// No admitted path bridges the gap before a waypoint.
    #[error("waypoint {waypoint} is not reached from {state}{}", if *cut { " within the state budget" } else { "" })]
    Unbridged {
        /// The unreached waypoint's index in the path, the first being 0.
        waypoint: usize,
        /// The last reached state's spelling.
        state: String,
        /// The search stopped at its state budget, so a longer search might reach it.
        cut: bool,
    },
    /// No reached state refutes the predicate.
    #[error("no counterexample: the predicate holds at all {reached} reached states")]
    Holds {
        /// How many states the exploration reached.
        reached: usize,
    },
    /// The exploration was cut before it found a counterexample.
    #[error(
        "unknown: no counterexample within the state budget of {budget}, which cut the exploration"
    )]
    Unknown {
        /// The space's state budget.
        budget: usize,
    },
    /// A waypoint predicate panicked while judging a reached state.
    #[error("waypoint {waypoint} panicked at {state}: {failure}")]
    Predicate {
        /// The waypoint's zero-based index.
        waypoint: usize,
        /// The reached state's spelling.
        state: String,
        /// The panic payload, when it is a string.
        failure: String,
    },
    /// A step panicked during the exploration, so an answer would rest on the edge it skipped.
    #[error("at {state}, {failure}, so the exploration skipped that step")]
    Skipped {
        /// The checker's failure line.
        failure: String,
        /// The spelling of the state the step was taken from.
        state: String,
    },
    /// The checker refused the request.
    #[error(transparent)]
    Checker(#[from] umpire_checker::Refusal),
}
