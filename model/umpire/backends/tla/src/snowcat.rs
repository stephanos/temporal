//! Apalache's Snowcat type annotations, as emitted and as ITF `varTypes` declare them.
use crate::Type;

/// The Snowcat type of an admitted type.
pub fn ty(value: &Type) -> String {
    match value {
        Type::Bool => "Bool".into(),
        Type::Int { .. } | Type::Bounded(_) | Type::Optional(_) => "Int".into(),
        Type::Text | Type::Enum { .. } => "Str".into(),
        Type::Unit => "Seq(Int)".into(),
        Type::Seq(element) => format!("Seq({})", ty(element)),
        Type::Tuple(elements) => tuple(elements.iter().map(ty)),
    }
}
pub(crate) fn tuple(types: impl IntoIterator<Item = String>) -> String {
    format!("<<{}>>", types.into_iter().collect::<Vec<_>>().join(", "))
}
