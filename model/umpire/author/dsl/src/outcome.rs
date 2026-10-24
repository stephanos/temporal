//! Scoped recording of a handler's request outcome.

use std::cell::Cell;

/// A named declared outcome kind, classified without cloning the edge's outcome.
pub struct OutcomeClass<O> {
    /// The stable display spelling of the declared kind.
    pub name: &'static str,
    predicate: fn(&O) -> bool,
}

impl<O> OutcomeClass<O> {
    /// Declares an outcome kind with a typed borrowed classifier.
    pub const fn new(name: &'static str, predicate: fn(&O) -> bool) -> Self {
        Self { name, predicate }
    }

    /// Whether the edge's outcome belongs to this declared kind.
    pub fn matches(&self, outcome: &O) -> bool {
        (self.predicate)(outcome)
    }
}

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

impl Outcome {
    /// Declares every stock outcome kind with borrowed classification.
    pub fn catalog() -> Vec<OutcomeClass<Self>> {
        vec![
            OutcomeClass::new("Accepted", |outcome| matches!(outcome, Self::Accepted)),
            OutcomeClass::new("Rejected(NotFound)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::NotFound))
            }),
            OutcomeClass::new("Rejected(AlreadyExists)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::AlreadyExists))
            }),
            OutcomeClass::new("Rejected(FailedPrecondition)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::FailedPrecondition))
            }),
            OutcomeClass::new("Rejected(InvalidArgument)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::InvalidArgument))
            }),
            OutcomeClass::new("Rejected(PermissionDenied)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::PermissionDenied))
            }),
            OutcomeClass::new("Rejected(Unavailable)", |outcome| {
                matches!(outcome, Self::Rejected(Rejection::Unavailable))
            }),
        ]
    }
}

#[derive(Clone, Copy)]
struct Frame {
    outcome: Outcome,
    stayed: bool,
}

thread_local! {
    static FRAME: Cell<Option<Frame>> = const { Cell::new(None) };
}

struct Restore(Option<Frame>);

impl Drop for Restore {
    fn drop(&mut self) {
        FRAME.set(self.0);
    }
}

/// Runs a unit-returning handler in an isolated frame, restoring its parent even on unwind.
pub fn evaluate(handler: impl FnOnce()) -> Outcome {
    evaluate_staying(handler).0
}

/// Runs a handler like [`evaluate`] and also reports whether it marked a self-loop with `stay!()`.
pub fn evaluate_staying(handler: impl FnOnce()) -> (Outcome, bool) {
    let _restore =
        Restore(FRAME.replace(Some(Frame { outcome: Outcome::Accepted, stayed: false })));
    handler();
    FRAME.get().map_or((Outcome::Accepted, false), |frame| (frame.outcome, frame.stayed))
}

/// Records the first rejection; calls outside a handler evaluation have no effect.
pub fn reject(rejection: Rejection) {
    if let Some(frame) = FRAME.get()
        && frame.outcome == Outcome::Accepted
    {
        FRAME.set(Some(Frame { outcome: Outcome::Rejected(rejection), ..frame }));
    }
}

/// Marks the running handler's step enabled even if it leaves the state unchanged.
#[doc(hidden)]
pub fn stay() {
    if let Some(frame) = FRAME.get() {
        FRAME.set(Some(Frame { stayed: true, ..frame }));
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

/// Marks an enabled self-loop: the handler's step is emitted even if the state is unchanged.
/// It does not return, and a rejection in the same step wins. `#[machine]` admits it only in
/// `#[on]` handler bodies; anywhere else it is a compile error.
#[macro_export]
macro_rules! stay {
    () => {
        ::core::compile_error!("stay!() marks a self-loop only inside an #[on] handler")
    };
    (__umpire_on_handler) => {
        $crate::outcome::stay()
    };
}
