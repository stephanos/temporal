//! Shared classification of a machine's lifecycle phases.

use std::{fmt::Debug, hash::Hash};

/// A lifecycle phase with an entity-defined terminal classification.
pub trait Phase: Copy + Eq + Hash + Debug {
    /// Whether this phase is closed.
    fn closed(&self) -> bool;
}

/// State whose lifecycle is described by one phase.
pub trait Phased {
    /// The state's lifecycle phase type.
    type Phase: Phase;

    /// The state's current phase.
    fn phase(&self) -> Self::Phase;

    /// Whether the state's current phase is closed.
    fn closed(&self) -> bool {
        self.phase().closed()
    }
}
