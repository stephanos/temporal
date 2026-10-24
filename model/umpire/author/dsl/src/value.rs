/// A Model value exported for the source lifter's binding: the selected World's fields, each
/// start's fields and each catalog action's payload. It mirrors the lifter's literals, so a
/// Model crate never depends on the lifter. Composite kinds extend this enum.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Value {
    /// A boolean.
    Bool(bool),
    /// An integer of any admitted width.
    Int(i128),
    /// A fieldless enum's qualified variant, such as `Phase::Open`.
    Enum(&'static str),
    /// Ordered elements.
    Sequence(Vec<Value>),
}

/// Converts one field value into a [`Value`].
#[diagnostic::on_unimplemented(
    message = "`{Self}` has no exported Model value",
    label = "not `IntoValue`",
    note = "a declared World's fields, state fields and action payloads are bool, an integer, or a fieldless enum deriving `Finite`"
)]
pub trait IntoValue {
    /// This value's exported form.
    fn value(&self) -> Value;
}

impl IntoValue for bool {
    fn value(&self) -> Value {
        Value::Bool(*self)
    }
}

macro_rules! integers {
    ($($ty:ty),*) => {$(
        impl IntoValue for $ty {
            fn value(&self) -> Value {
                Value::Int(i128::from(*self))
            }
        }
    )*};
}

integers!(i8, i16, i32, i64, u8, u16, u32, u64);

/// A `#[machine]` World's or state's named field values in declaration order, private fields
/// included. `derive(World)` and `derive(State)` implement it inside
/// `#[machine]`; it exposes what their derived `Debug` already prints.
pub trait Values {
    /// Field names and values in declaration order.
    fn values(&self) -> Vec<(&'static str, Value)>;

    /// Checks one selected integer prefix through native start domains in declaration order.
    /// Returns only the first failed domain, without enumerating other domain combinations.
    fn start_failure(&self, _selected: &[(&str, i128)]) -> Option<StartFailure> {
        None
    }
}

/// An actor's concrete action values and its World-bound input domains.
pub trait ActionValues<W> {
    /// The qualified variant, such as `Worker::Poll`, and its payload fields in declaration
    /// order; tuple fields are named by position.
    fn action_value(&self) -> (&'static str, Vec<(&'static str, Value)>);

    /// Each `#[domain]` field's values for `world` as `(variant, field, values)`, stopping one
    /// value past [`CATALOG_LIMIT`](crate::CATALOG_LIMIT) so an oversized domain is refused
    /// rather than truncated.
    fn domains(world: &W) -> Vec<(&'static str, &'static str, Vec<Value>)>;
}

impl IntoValue for () {
    fn value(&self) -> Value {
        Value::Sequence(Vec::new())
    }
}

impl IntoValue for crate::Outcome {
    fn value(&self) -> Value {
        use crate::{Outcome, Rejection};
        Value::Enum(match self {
            Outcome::Accepted => "Outcome::Accepted",
            Outcome::Rejected(Rejection::NotFound) => "Outcome::Rejected(NotFound)",
            Outcome::Rejected(Rejection::AlreadyExists) => "Outcome::Rejected(AlreadyExists)",
            Outcome::Rejected(Rejection::FailedPrecondition) => {
                "Outcome::Rejected(FailedPrecondition)"
            }
            Outcome::Rejected(Rejection::InvalidArgument) => "Outcome::Rejected(InvalidArgument)",
            Outcome::Rejected(Rejection::PermissionDenied) => "Outcome::Rejected(PermissionDenied)",
            Outcome::Rejected(Rejection::Unavailable) => "Outcome::Rejected(Unavailable)",
        })
    }
}

#[cfg(not(target_arch = "wasm32"))]
impl<T: IntoValue + Clone> IntoValue for imbl::Vector<T> {
    fn value(&self) -> Value {
        Value::Sequence(self.iter().map(IntoValue::value).collect())
    }
}

/// A native start-domain refusal and the fields bound before that domain was evaluated.
pub struct StartFailure {
    /// Concrete fields in declaration order, excluding the failed and subsequent fields.
    pub prefix: Vec<(&'static str, Value)>,
    /// The original native panic payload, kept for arithmetic classification by the caller.
    pub panic: Box<dyn std::any::Any + Send>,
}
