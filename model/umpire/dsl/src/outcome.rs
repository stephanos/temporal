//! Scoped recording of a handler's request outcome.

use std::cell::Cell;

/// A generic status a machine can return for a rejected request.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Rejection {
    /// The requested entity does not exist.
    NotFound,
    /// The requested entity already exists.
    AlreadyExists,
    /// The request's preconditions are not satisfied.
    FailedPrecondition,
    /// A request argument is invalid.
    InvalidArgument,
    /// The requester lacks permission.
    PermissionDenied,
    /// The system is temporarily unavailable.
    Unavailable,
}

/// The result of an enabled machine step.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum Outcome {
    /// The request was accepted.
    Accepted,
    /// The request was answered with a rejection.
    Rejected(Rejection),
}

thread_local! {
    static FRAME: Cell<Option<Outcome>> = const { Cell::new(None) };
}

struct Restore(Option<Outcome>);

impl Drop for Restore {
    fn drop(&mut self) {
        FRAME.set(self.0);
    }
}

/// Runs a unit-returning handler in an isolated frame, restoring its parent even on unwind.
pub fn evaluate(handler: impl FnOnce()) -> Outcome {
    let _restore = Restore(FRAME.replace(Some(Outcome::Accepted)));
    handler();
    FRAME.get().unwrap_or(Outcome::Accepted)
}

/// Records the first rejection; calls outside a handler evaluation have no effect.
pub fn reject(rejection: Rejection) {
    if FRAME.get() == Some(Outcome::Accepted) {
        FRAME.set(Some(Outcome::Rejected(rejection)));
    }
}

/// Records a rejection and returns from the enclosing unit-returning function.
/// In a helper this returns only from that helper, leaving its caller to continue.
#[macro_export]
macro_rules! reject {
    ($rejection:expr $(,)?) => {{
        $crate::outcome::reject($rejection);
        return;
    }};
}
