use crate::{Machine, Path, Registered};

/// Normalizes a legacy state path or an already typed path for its registered world.
/// `Kind` distinguishes conversions without requiring a conversion from unit actions.
pub trait IntoWorldPath<S: Registered, Kind> {
    /// Preserves the pattern while selecting the world's action and outcome types.
    fn into_world_path(
        self,
    ) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>;
}

#[doc(hidden)]
pub mod kind {
    pub struct Legacy;
    pub struct Typed;
}

impl<S: Registered> IntoWorldPath<S, kind::Legacy> for Path<S> {
    fn into_world_path(
        self,
    ) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        self.into_world::<S::World>()
    }
}

impl<S: Registered> IntoWorldPath<S, kind::Typed>
    for Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>
{
    fn into_world_path(
        self,
    ) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        self
    }
}
