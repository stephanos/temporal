#[path = "../../../../umpire/author/dependency_roots.rs"]
mod dependency_roots;

pub(crate) fn resolve(tokens: proc_macro2::TokenStream) -> proc_macro2::TokenStream {
    dependency_roots::resolve(
        tokens,
        &[
            ("umpire-runtime", "runtime"),
            ("umpire-dsl", "dsl"),
            ("umpire-temporal", "temporal"),
            ("umpire-temporal-api", "api"),
        ],
    )
}
