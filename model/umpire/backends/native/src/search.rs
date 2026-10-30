//! Budgeted label ancestry and zero/one-cost traversal shared by search callers.

use std::collections::VecDeque;

/// A bound or resource refusal before admitting a search label.
#[derive(Clone, Copy, Debug, PartialEq, Eq, thiserror::Error)]
pub enum SearchLimit {
    /// An emitted edge would exceed the admitted depth.
    #[error("search depth exceeds limit {limit}")]
    Depth {
        /// Maximum emitted-edge distance from a root.
        limit: usize,
    },
    /// Label admission or caller evaluation would exceed admitted work.
    #[error("search work exceeds limit {limit}")]
    Work {
        /// Maximum admitted labels plus explicitly charged evaluations.
        limit: usize,
    },
    /// Storage for the next label could not be reserved.
    #[error("search label allocation failed")]
    Allocation,
}

/// One immutable arrival, preserving the caller's complete position and emitted edge.
pub struct Label<P, E> {
    /// Caller-owned state, evidence cursor and any history required for future admission.
    pub position: P,
    /// Number of emitted edges from this label's root; epsilon moves cost zero.
    pub distance: usize,
    /// Parent label id and its emitted edge, absent for an epsilon move.
    pub parent: Option<(usize, Option<E>)>,
}

/// Zero/one-cost traversal with immutable ancestry and explicit depth/work bounds.
///
/// Callers supply successors in canonical order and own all identity/dominance decisions.
/// No states, cursors or histories are merged here. Each admitted root or successor spends
/// one unit of work; callers can additionally charge predicate or transition evaluations.
/// Epsilon successors run before queued emitted edges, as in a zero/one breadth-first search.
/// A refusal leaves admitted labels and the pending queue intact.
pub struct Traversal<P, E> {
    labels: Vec<Label<P, E>>,
    queue: VecDeque<usize>,
    max_depth: usize,
    max_work: usize,
    work: usize,
}

impl<P, E> Traversal<P, E> {
    /// Creates an empty traversal under the admitted emitted-edge depth and work ceilings.
    pub fn new(max_depth: usize, max_work: usize) -> Self {
        Self { labels: Vec::new(), queue: VecDeque::new(), max_depth, max_work, work: 0 }
    }

    /// Admits a root in caller-supplied order.
    ///
    /// # Errors
    /// Refuses exhausted work or allocation before changing the label set or pending queue.
    pub fn root(&mut self, position: P) -> Result<usize, SearchLimit> {
        self.admit(Label { position, distance: 0, parent: None }, false)
    }

    /// Admits a successor of an existing label, retaining its exact emitted edge.
    ///
    /// # Panics
    /// Panics if `parent` was not returned by this traversal.
    ///
    /// # Errors
    /// Refuses depth, work or allocation before changing the label set or pending queue.
    pub fn push(
        &mut self,
        parent: usize,
        position: P,
        edge: Option<E>,
    ) -> Result<usize, SearchLimit> {
        let distance = self.labels[parent]
            .distance
            .checked_add(usize::from(edge.is_some()))
            .filter(|distance| *distance <= self.max_depth)
            .ok_or(SearchLimit::Depth { limit: self.max_depth })?;
        let epsilon = edge.is_none();
        self.admit(Label { position, distance, parent: Some((parent, edge)) }, epsilon)
    }

    /// Charges one caller evaluation independently of label admission.
    ///
    /// # Errors
    /// Refuses before incrementing work when the admitted ceiling is exhausted.
    pub fn charge(&mut self) -> Result<(), SearchLimit> {
        self.check_work()?;
        self.work += 1;
        Ok(())
    }

    /// Reads an admitted label without requiring its position or emission to be cloneable.
    ///
    /// # Panics
    /// Panics if `id` was not returned by this traversal.
    pub fn label(&self, id: usize) -> &Label<P, E> {
        &self.labels[id]
    }

    /// Reads the immutable ancestry arena in admission order.
    pub fn labels(&self) -> &[Label<P, E>] {
        &self.labels
    }

    /// Number of admitted labels, including labels already visited.
    pub fn len(&self) -> usize {
        self.labels.len()
    }

    /// Whether no root or successor has been admitted.
    pub fn is_empty(&self) -> bool {
        self.labels.is_empty()
    }

    /// Copies queued positions in their exact visitation order without copying emissions.
    ///
    /// # Errors
    /// Refuses allocation or a caller's fallible position copy.
    pub fn pending<T, X>(&self, mut copy: impl FnMut(&P) -> Result<T, X>) -> Result<Vec<T>, X>
    where
        X: From<SearchLimit>,
    {
        let mut positions = Vec::new();
        positions.try_reserve(self.queue.len()).map_err(|_| SearchLimit::Allocation)?;
        for id in &self.queue {
            positions.push(copy(&self.labels[*id].position)?);
        }
        Ok(positions)
    }

    fn check_work(&self) -> Result<(), SearchLimit> {
        if self.work >= self.max_work {
            Err(SearchLimit::Work { limit: self.max_work })
        } else {
            Ok(())
        }
    }

    fn admit(&mut self, label: Label<P, E>, epsilon: bool) -> Result<usize, SearchLimit> {
        self.check_work()?;
        self.labels.try_reserve(1).map_err(|_| SearchLimit::Allocation)?;
        self.queue.try_reserve(1).map_err(|_| SearchLimit::Allocation)?;
        let id = self.labels.len();
        self.labels.push(label);
        if epsilon {
            self.queue.push_front(id);
        } else {
            self.queue.push_back(id);
        }
        self.work += 1;
        Ok(id)
    }
}

impl<P, E> Iterator for Traversal<P, E> {
    type Item = usize;

    fn next(&mut self) -> Option<Self::Item> {
        self.queue.pop_front()
    }
}
