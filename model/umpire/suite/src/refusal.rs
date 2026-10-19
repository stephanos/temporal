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
    /// The checker refused the request.
    #[error(transparent)]
    Checker(#[from] umpire_checker::Refusal),
}
