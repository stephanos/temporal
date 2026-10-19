/// The actions excluded from a progress search.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Assumption<A> {
    /// Actions omitted entirely from the scoped transition graph.
    pub excluded: Vec<A>,
}

impl<A: Clone + PartialEq> Assumption<A> {
    /// Builds an assumption excluding the supplied actions.
    pub fn without(actions: &[A]) -> Self {
        Self { excluded: actions.to_vec() }
    }

    /// Reports whether an action is retained by this assumption.
    pub fn allows(&self, action: &A) -> bool {
        !self.excluded.contains(action)
    }
}

impl<A> Default for Assumption<A> {
    fn default() -> Self {
        Self { excluded: Vec::new() }
    }
}

/// Compatibility name for manual trait Models using the original progress builder.
pub type Scope<A> = Assumption<A>;
