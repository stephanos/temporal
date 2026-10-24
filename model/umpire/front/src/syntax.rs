pub(crate) fn has(attrs: &[syn::Attribute], name: &str) -> bool {
    attr(attrs, name).is_some()
}
pub(crate) fn attr<'a>(attrs: &'a [syn::Attribute], name: &str) -> Option<&'a syn::Attribute> {
    attrs.iter().find(|a| a.path().is_ident(name))
}
pub(crate) fn path_name(name: &str) -> String {
    name.replace(' ', "")
}

/// A DSL member through its defining crate or the author facade.
pub(crate) fn dsl_path(path: &str) -> Option<&str> {
    ["umpire_dsl::", "umpire::dsl::", "umpire::"]
        .into_iter()
        .find_map(|prefix| path.strip_prefix(prefix))
}
