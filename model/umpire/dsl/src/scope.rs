/// Action admission and explicit weak fairness for a progress search.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Assumption<A> {
    /// The selected actions, or all catalog actions when absent.
    pub only: Option<Vec<A>>,
    /// Actions omitted entirely from the scoped transition graph.
    pub excluded: Vec<A>,
    /// Actions subject to weak fairness; membership does not admit an action.
    pub fair: Vec<A>,
}

impl<A: Clone + PartialEq> Assumption<A> {
    /// Builds an assumption excluding the supplied actions.
    pub fn without(actions: &[A]) -> Self {
        Self::default().without(actions)
    }

    /// Builds an assumption admitting only the supplied actions.
    pub fn only<B: Clone + Into<A>>(actions: &[B]) -> Self {
        Self::default().only(actions)
    }

    /// Builds an assumption with explicit weak fairness for the supplied actions.
    pub fn fair<B: Clone + Into<A>>(actions: &[B]) -> Self {
        Self::default().fair(actions)
    }

    /// Reports whether an action is retained by this assumption.
    pub fn allows(&self, action: &A) -> bool {
        self.only.as_ref().is_none_or(|selected| selected.contains(action))
            && !self.excluded.contains(action)
    }

    /// Reports explicit fair membership independently of admission.
    pub fn is_fair(&self, action: &A) -> bool {
        self.fair.contains(action)
    }
}

impl<A> Default for Assumption<A> {
    fn default() -> Self {
        Self { only: None, excluded: Vec::new(), fair: Vec::new() }
    }
}

/// Compatibility name for manual trait Models using the original progress builder.
pub type Scope<A> = Assumption<A>;

/// Receiver chaining compatible with the original static assumption constructors.
pub trait AssumptionExt<A>: Sized {
    /// Replaces the selected actions, retaining exclusions and fairness.
    #[must_use]
    fn only<B: Clone + Into<A>>(self, actions: impl AsRef<[B]>) -> Self;

    /// Accumulates excluded actions in first-occurrence order.
    #[must_use]
    fn without<B: Clone + Into<A>>(self, actions: impl AsRef<[B]>) -> Self;

    /// Accumulates explicitly fair actions in first-occurrence order.
    #[must_use]
    fn fair<B: Clone + Into<A>>(self, actions: impl AsRef<[B]>) -> Self;
}

impl<A: Clone + PartialEq> AssumptionExt<A> for Assumption<A> {
    fn only<B: Clone + Into<A>>(mut self, actions: impl AsRef<[B]>) -> Self {
        let mut selected = Vec::new();
        append(&mut selected, actions.as_ref());
        self.only = Some(selected);
        self
    }

    fn without<B: Clone + Into<A>>(mut self, actions: impl AsRef<[B]>) -> Self {
        append(&mut self.excluded, actions.as_ref());
        self
    }

    fn fair<B: Clone + Into<A>>(mut self, actions: impl AsRef<[B]>) -> Self {
        append(&mut self.fair, actions.as_ref());
        self
    }
}

fn append<A: PartialEq, B: Clone + Into<A>>(values: &mut Vec<A>, actions: &[B]) {
    for action in actions {
        let action = action.clone().into();
        if !values.contains(&action) {
            values.push(action);
        }
    }
}
