/// An invalid request to the checker, distinct from a Model's verdict.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
pub enum Refusal {
    /// The supplied state budget cannot admit any state.
    #[error("state budget must be at least one (given: {given})")]
    Budget {
        /// The state budget supplied by the caller.
        given: usize,
    },
    /// The supplied state id was not admitted by this exploration.
    #[error("unknown state id: {given}")]
    State {
        /// The state id supplied by the caller.
        given: usize,
    },
}
