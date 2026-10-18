use std::hash::Hash;

use indexmap::IndexSet;
use rustc_hash::FxBuildHasher;

/// A state's insertion index in the explored graph.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub struct StateId(pub usize);

/// State identity and storage used by the explorer.
pub trait Visited<S> {
    /// Inserts a state, returning its existing id when already present.
    fn insert(&mut self, state: S) -> StateId;
    /// Looks up a state by its stable id.
    fn get(&self, id: StateId) -> Option<&S>;
    /// Finds the id of an equal state.
    fn find(&self, state: &S) -> Option<StateId>;
    /// The number of distinct stored states.
    fn len(&self) -> usize;
    /// Whether no states have been stored.
    fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

/// Full-state identity with seedless hashing and discovery-order storage.
pub struct FullStates<S> {
    states: IndexSet<S, FxBuildHasher>,
}

impl<S> Default for FullStates<S> {
    fn default() -> Self {
        Self { states: IndexSet::with_hasher(FxBuildHasher) }
    }
}

impl<S> FullStates<S> {
    pub(crate) fn iter(&self) -> impl ExactSizeIterator<Item = (StateId, &S)> {
        self.states.iter().enumerate().map(|(id, state)| (StateId(id), state))
    }
}

impl<S: Eq + Hash> Visited<S> for FullStates<S> {
    fn insert(&mut self, state: S) -> StateId {
        StateId(self.states.insert_full(state).0)
    }
    fn get(&self, id: StateId) -> Option<&S> {
        self.states.get_index(id.0)
    }
    fn find(&self, state: &S) -> Option<StateId> {
        self.states.get_index_of(state).map(StateId)
    }
    fn len(&self) -> usize {
        self.states.len()
    }
}
