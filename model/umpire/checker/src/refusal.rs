/// An invalid request to the checker, distinct from a Model's verdict.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
pub enum Refusal {
    /// A Model step was invalid and produced no admitted edge.
    #[error("invalid action {action} from {before}: {error}")]
    Step {
        /// The action whose handler was invalid.
        action: String,
        /// The source state's canonical spelling.
        before: String,
        /// The invalid step's cause.
        error: umpire_dsl::StepError,
    },
    /// The requested name was not registered during exploration.
    #[error("unknown registered progress claim: {claim}")]
    ProgressRegistrationMissing {
        /// The requested registration name.
        claim: String,
    },
    /// More than one original progress registration has the requested name.
    #[error("ambiguous registered progress claim: {claim}")]
    ProgressRegistrationAmbiguous {
        /// The ambiguous registration name.
        claim: String,
    },
    /// Completed-action columns require the original ordered action catalog.
    #[error("progress query action catalog differs from exploration")]
    ProgressCatalogChanged,
    /// A progress bound must admit at least one transition.
    #[error("progress claim {claim} bound must be at least one (given: {given})")]
    ProgressBound {
        /// The claim whose bound is invalid.
        claim: String,
        /// The numeric bound supplied by the author or caller.
        given: u32,
    },
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
    /// The supplied scope action is not in the Model's action catalog.
    #[error("unknown scope action: {action}")]
    ScopeAction {
        /// The unknown action's canonical spelling.
        action: String,
    },
    /// An explicit action restriction retains no catalog action.
    #[error("assumption admits no actions")]
    EmptyAssumption,
    /// The action catalog declares more concrete actions than admitted; nothing was explored.
    #[error("action catalog has {count} actions, over limit {limit}")]
    CatalogLimit {
        /// Exact concrete action count.
        count: usize,
        /// Admitted maximum.
        limit: usize,
    },
    /// The action catalog's count overflows `usize`; nothing was explored.
    #[error("action catalog count overflows usize (limit {limit})")]
    CatalogOverflow {
        /// Admitted maximum.
        limit: usize,
    },
}

impl From<umpire_dsl::CatalogError> for Refusal {
    fn from(error: umpire_dsl::CatalogError) -> Self {
        match error {
            umpire_dsl::CatalogError::TooLarge { count, limit } => {
                Self::CatalogLimit { count, limit }
            }
            umpire_dsl::CatalogError::Overflow { limit } => Self::CatalogOverflow { limit },
        }
    }
}
