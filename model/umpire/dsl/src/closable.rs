//! Shared closure laws for delegating postconditions evaluated in a claim frame.

use crate::{Outcome, Phased, Rejection, check};

/// Checks that a step from a closed phase preserves that exact phase.
///
/// This promises neither request rejection nor whole-state immutability. It says nothing
/// about repeated request IDs, status reporting, or recording closure exactly once.
pub fn terminal_remains<S: Phased + Eq>(after: &S, before: &S) {
    if before.closed() {
        check!(after.phase() == before.phase());
    }
}

/// Checks that a step from closure preserves the whole state and rejects as expected.
///
/// The expected rejection is entity-specific. This law does not model repeated-request-ID
/// deduplication, which may return the original request's answer.
pub fn closed_rejects<S: Phased + Eq>(
    after: &S,
    before: &S,
    outcome: &Outcome,
    expected: Rejection,
) {
    if before.closed() {
        check!(after == before);
        check!(*outcome == Outcome::Rejected(expected));
    }
}
