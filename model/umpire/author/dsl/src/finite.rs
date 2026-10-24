use crate::InputDomain;

/// A type whose complete value domain is finite and enumerated in a stable order.
///
/// Enumeration is not construction validation: a bounded newtype may still be built raw.
#[diagnostic::on_unimplemented(
    message = "`{Self}` has no finite action-input domain",
    label = "not `Finite`",
    note = "derive `Finite`, or give the field `#[inputs(..)]` or a World-bound `#[domain(path)]`"
)]
pub trait Finite: Sized {
    /// Enumerates every value once, in declaration order.
    fn domain() -> impl Iterator<Item = Self>;

    #[doc(hidden)]
    fn __umpire_input_domain() -> InputDomain<Self>
    where
        Self: 'static,
    {
        InputDomain::enumerated(Self::domain)
    }
}

impl Finite for bool {
    fn domain() -> impl Iterator<Item = Self> {
        [false, true].into_iter()
    }
}
