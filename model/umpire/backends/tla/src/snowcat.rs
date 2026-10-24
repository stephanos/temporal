use crate::Type;

pub(super) fn ty(value: &Type) -> String {
    match value {
        Type::Bool => "Bool".into(),
        Type::Int { .. } | Type::Bounded(_) | Type::Optional(_) => "Int".into(),
        Type::Text | Type::Enum { .. } => "Str".into(),
        Type::Unit => "Seq(Int)".into(),
        Type::Seq(element) => format!("Seq({})", ty(element)),
        Type::Tuple(elements) => tuple(elements.iter().map(ty)),
    }
}
pub(super) fn tuple(types: impl IntoIterator<Item = String>) -> String {
    format!("<<{}>>", types.into_iter().collect::<Vec<_>>().join(", "))
}
