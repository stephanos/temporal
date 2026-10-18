/// The actions excluded from a progress search.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Scope<A> {
    /// Actions omitted entirely from the scoped transition graph.
    pub excluded: Vec<A>,
}

impl<A: Copy + PartialEq> Scope<A> {
    /// Builds a scope excluding the supplied actions.
    pub fn without(actions: &[A]) -> Self {
        Self { excluded: actions.to_vec() }
    }

    /// Reports whether an action is retained by this scope.
    pub fn allows(&self, action: A) -> bool {
        !self.excluded.contains(&action)
    }
}

impl<A> Default for Scope<A> {
    fn default() -> Self {
        Self { excluded: Vec::new() }
    }
}
