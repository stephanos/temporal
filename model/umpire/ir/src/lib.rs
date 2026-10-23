//! The admitted Model tree: what `umpire-front` reads from a `#[machine]` module, and what the
//! macro embeds in the compiled Model for the lifted back ends. Internal, with no stability
//! promise; it is plain data, so compiled Models can carry it.
#[doc(hidden)]
pub mod expr;
#[doc(hidden)]
pub mod print;
#[doc(hidden)]
pub mod tree;
#[doc(hidden)]
pub mod value;
use std::collections::BTreeMap;
pub use tree::Lifted;

/// Admitted value types for typed backend decoding.
#[derive(Clone, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Type)]
pub enum Type {
    /// Boolean.
    Bool,
    /// Checked Rust integer.
    Int {
        /// Inclusive minimum.
        min: i128,
        /// Inclusive maximum.
        max: i128,
    },
    /// Fieldless enum encoded with qualified variant names.
    Enum {
        /// Authored type name.
        name: String,
        /// Qualified variant names in declaration order.
        variants: Vec<String>,
    },
    /// String.
    Text,
    /// Unit.
    Unit,
    /// Tuple.
    Tuple(Vec<Type>),
    /// Sequence.
    Seq(Box<Type>),
}
/// The stock `umpire_dsl::Rejection` variants, in declaration order.
pub const REJECTIONS: [&str; 6] = [
    "NotFound",
    "AlreadyExists",
    "FailedPrecondition",
    "InvalidArgument",
    "PermissionDenied",
    "Unavailable",
];
impl Type {
    /// The stock `umpire_dsl::Rejection`, a fieldless enum.
    #[must_use]
    pub fn rejection() -> Self {
        Self::Enum {
            name: "Rejection".into(),
            variants: REJECTIONS.iter().map(|variant| format!("Rejection::{variant}")).collect(),
        }
    }
    /// The stock `umpire_dsl::Outcome`, encoded as its finite values: `Outcome::Accepted` and
    /// each `Outcome::Rejected(variant)`, spelled as their `Debug` text after the prefix.
    #[must_use]
    pub fn outcome() -> Self {
        let rejected = REJECTIONS.iter().map(|variant| format!("Outcome::Rejected({variant})"));
        Self::Enum {
            name: "Outcome".into(),
            variants: std::iter::once("Outcome::Accepted".into()).chain(rejected).collect(),
        }
    }
}
/// A located refusal of unsupported source behavior; the consumer names the file.
#[derive(Clone, Debug, PartialEq, Eq, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Diagnostic)]
pub struct Diagnostic {
    /// One-based line.
    pub line: usize,
    /// One-based column.
    pub column: usize,
    /// Admission failure.
    pub message: String,
}
/// The names a `#[machine]` module spells, which generated back-end names keep apart from.
#[derive(Clone, Debug, Default, PartialEq, Eq, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Source)]
pub struct Source {
    /// Every identifier.
    pub idents: Vec<String>,
    /// Every enum variant, as its enum and variant name.
    pub variants: Vec<(String, String)>,
    /// Every method name.
    pub methods: Vec<String>,
    /// Where each identifier or string literal value first occurs, as line and column.
    pub first: BTreeMap<String, (usize, usize)>,
    /// The first string literal carrying a control character other than a line break or tab.
    pub control: Option<Diagnostic>,
}
/// What the `#[machine]` macro embeds in a compiled Model.
#[derive(Clone, Debug, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Embedded)]
pub struct Embedded {
    /// The canonical path of the source file the module is written in.
    pub file: String,
    /// The admitted Model, or the located subset refusal; native checking holds either way.
    pub model: Result<Lifted, Diagnostic>,
    /// The module's names.
    pub source: Source,
}

#[cfg(test)]
mod printing_controls {
    use crate::print::Print;

    #[derive(umpire_ir_derive::Print)]
    #[print_path(::umpire_dsl::__lifted::Control)]
    struct Control {
        original: String,
        added: Choice,
    }

    #[derive(umpire_ir_derive::Print)]
    #[print_path(::umpire_dsl::__lifted::Choice)]
    enum Choice {
        Unit,
        Tuple(i128, bool),
        Added { extra: Option<u64> },
    }

    #[test]
    fn new_fields_and_variants_print_without_registration() -> Result<(), Box<dyn std::error::Error>>
    {
        for (added, expected) in [
            (Choice::Unit, "::umpire_dsl::__lifted::Choice::Unit"),
            (
                Choice::Tuple(i128::MIN, true),
                "::umpire_dsl::__lifted::Choice::Tuple(::core::primitive::i128::MIN, true)",
            ),
            (
                Choice::Added { extra: Some(7) },
                "::umpire_dsl::__lifted::Choice::Added { extra: ::core::option::Option::Some(7u64) }",
            ),
        ] {
            let value = Control { original: "kept".into(), added };
            let expected: proc_macro2::TokenStream = format!(
                "::umpire_dsl::__lifted::Control {{ original: ::std::string::String::from(\"kept\"), added: {expected} }}"
            ).parse()?;
            assert_eq!(value.print().to_string(), expected.to_string());
        }
        Ok(())
    }
}
