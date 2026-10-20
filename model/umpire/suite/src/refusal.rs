/// A request the suite cannot answer as written, distinct from a test's own failure.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
pub enum Refusal {
    /// The test's stated case count disagrees with the authored finite space.
    #[error("pattern has {actual} alternatives, test states Cases({expected})\n{alternatives}")]
    Cases {
        /// Enumerated mathematical count.
        actual: usize,
        /// Count written by the test.
        expected: usize,
        /// Admitted concrete alternatives, or the resource refusal preventing expansion.
        alternatives: String,
    },
    /// An empty alternative set cannot establish coverage.
    #[error("pattern has no alternatives; empty coverage is refused")]
    Empty,
    /// A concrete story cannot select among alternatives.
    #[error("a story has one path, but pattern has {count} alternatives; use cover")]
    StoryAlternatives {
        /// Authored alternatives.
        count: usize,
    },
    /// A finite pattern exceeded its admission limits.
    #[error(transparent)]
    Pattern(#[from] umpire_dsl::PatternError),
    /// A coverage or sampling graph is incomplete.
    #[error("exploration cut at the Model-state budget of {budget}; complete graph required")]
    Cut {
        /// Distinct-Model-state capacity.
        budget: usize,
    },
    /// One authored alternative could not be witnessed.
    #[error("alternative {index} [{choices}]: {failure}")]
    Alternative {
        /// Zero-based authored alternative index.
        index: usize,
        /// Concrete authored choices.
        choices: String,
        /// The precise witness refusal.
        failure: Box<Refusal>,
    },
    /// A selected alternative cannot finish within the finite experiment.
    #[error("selected alternative cannot finish within walk bound {bound} steps")]
    WalkBound {
        /// Maximum Model edges in one walk.
        bound: usize,
    },
    /// The retained pattern product exceeds its independent storage ceiling.
    #[error("pattern search exceeds storage limit {limit}")]
    SearchLimit {
        /// Maximum product records.
        limit: usize,
    },
    /// Declared coverage metadata or a callback could not be interpreted honestly.
    #[error("coverage metadata: {detail}")]
    Metadata {
        /// Metadata defect or callback panic.
        detail: String,
    },
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
