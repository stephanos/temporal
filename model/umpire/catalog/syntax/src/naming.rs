//! Rust identifiers for protobuf names, using prost's case and escaping rules.
use heck::{ToSnakeCase, ToUpperCamelCase};

fn sanitize(name: String) -> String {
    if matches!(name.as_str(), "_" | "super" | "self" | "Self" | "extern" | "crate") {
        format!("{name}_")
    } else if name.starts_with(char::is_numeric) {
        format!("_{name}")
    } else if name == "gen" || (!name.is_empty() && syn::parse_str::<syn::Ident>(&name).is_err()) {
        format!("r#{name}")
    } else {
        name
    }
}

/// The Rust field or module identifier emitted for a protobuf name.
pub fn snake(name: &str) -> String {
    sanitize(name.to_snake_case())
}

/// The Rust type or variant identifier emitted for a protobuf name.
pub fn upper_camel(name: &str) -> String {
    sanitize(name.to_upper_camel_case())
}
