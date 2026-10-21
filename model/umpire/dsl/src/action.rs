use std::{fmt::Display, hash::Hash};

/// An actor's complete finite action domain, independent of a requested path.
pub trait Actor: Clone + Eq + Hash + Display {
    /// Enumerates every declared input assignment in declaration order.
    fn actions() -> Vec<Self>;

    /// Identifies environment-independent actions; this does not imply fairness.
    fn internal(&self) -> bool;

    /// Identifies actions belonging to an explicitly declared fault actor.
    fn fault(&self) -> bool;
}

/// An action value's identity and classification, independent of how its domain is
/// enumerated. Every static [`Actor`] has it; a World-dependent actor has only this.
pub trait ActorValue: Clone + Eq + Hash + Display {
    /// Identifies environment-independent actions; this does not imply fairness.
    fn internal(&self) -> bool;

    /// Identifies actions belonging to an explicitly declared fault actor.
    fn fault(&self) -> bool;
}

impl<T: Actor> ActorValue for T {
    fn internal(&self) -> bool {
        Actor::internal(self)
    }

    fn fault(&self) -> bool {
        Actor::fault(self)
    }
}

/// Borrows one actor's action from the machine's typed action carrier.
pub trait ActorProjection<T> {
    /// Returns this actor's action, or `None` for another actor.
    fn project(&self) -> Option<&T>;
}

/// Associates a locally declared actor with its defining machine's state.
/// Explicit state-prefixed paths may still use this actor in another machine.
pub trait RegisteredActor<S: crate::Associated>: ActorValue {}
