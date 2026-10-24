use crate::{Associated, Machine, Path};

/// Normalizes a legacy state path or an already typed path for its registered world.
/// `Kind` distinguishes conversions without requiring a conversion from unit actions.
pub trait IntoWorldPath<S: Associated, Kind> {
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

impl<S: Associated> IntoWorldPath<S, kind::Legacy> for Path<S> {
    fn into_world_path(
        self,
    ) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        self.into_world::<S::World>()
    }
}

impl<S: Associated> IntoWorldPath<S, kind::Typed>
    for Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome>
{
    fn into_world_path(
        self,
    ) -> Path<S, <S::World as Machine>::Action, <S::World as Machine>::Outcome> {
        self
    }
}
