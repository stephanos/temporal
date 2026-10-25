//! The admitted Model tree: what `umpire-front` reads from a `#[machine]` module, and what the
//! macro embeds in the compiled Model for the lifted back ends. Internal, with no stability
//! promise; it is plain data, so compiled Models can carry it.
#![forbid(unsafe_code)]

#[doc(hidden)]
pub mod expr;
mod literal;
#[doc(hidden)]
pub mod print;
#[doc(hidden)]
pub mod tree;
#[doc(hidden)]
pub mod value;
pub use literal::{Binding, Concrete, Literal, Mismatch};
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
    /// A source-local integer wrapper with one authored finite range.
    Bounded(Bounded),
    /// An absent or present value of one bounded nominal type.
    Optional(Bounded),
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
/// A local one-field integer wrapper, preserving its authored identity and Rust backing type.
#[derive(Clone, Debug, PartialEq, Eq, Hash, umpire_ir_derive::Print)]
#[print_path(::umpire_dsl::__lifted::Bounded)]
pub struct Bounded {
    /// Authored wrapper name.
    pub name: String,
    /// Named member, or tuple member zero.
    pub field: Option<String>,
    /// Primitive backing type spelling.
    pub backing: String,
    /// Backing type's inclusive minimum.
    pub min: i128,
    /// Backing type's inclusive maximum.
    pub max: i128,
    /// Declared inclusive lower bound.
    pub low: i128,
    /// Declared inclusive upper bound.
    pub high: i128,
}
impl Bounded {
    /// Whether this descriptor names an admitted Rust backing type and a nonempty finite range.
    #[must_use]
    pub fn valid(&self) -> bool {
        let bounds = match self.backing.as_str() {
            "u8" => (0, i128::from(u8::MAX)),
            "u16" => (0, i128::from(u16::MAX)),
            "i8" => (i128::from(i8::MIN), i128::from(i8::MAX)),
            "i16" => (i128::from(i16::MIN), i128::from(i16::MAX)),
            "i32" => (i128::from(i32::MIN), i128::from(i32::MAX)),
            _ => return false,
        };
        !self.name.is_empty()
            && self.field.as_ref().is_none_or(|field| !field.is_empty())
            && (self.min, self.max) == bounds
            && self.min <= self.low
            && self.low <= self.high
            && self.high <= self.max
    }
    /// Whether an optional value has an exact nonnegative signed-32-bit ordinal encoding.
    #[must_use]
    pub fn optional_valid(&self) -> bool {
        self.valid() && self.low >= 0 && self.high < i128::from(i32::MAX)
    }
    /// Membership in the authored finite domain.
    #[must_use]
    pub fn contains(&self, raw: i128) -> bool {
        self.valid() && self.low <= raw && raw <= self.high
    }
    /// Encode a present valid payload, leaving zero for absence.
    #[must_use]
    pub fn encode(&self, raw: i128) -> Option<i128> {
        (self.optional_valid() && self.contains(raw)).then(|| raw + 1)
    }
    /// Decode an exact optional code; the outer absence denotes an invalid code.
    #[must_use]
    pub fn decode(&self, code: i128) -> Option<Option<i128>> {
        if !self.optional_valid() {
            return None;
        }
        if code == 0 {
            return Some(None);
        }
        let raw = code.checked_sub(1)?;
        self.contains(raw).then_some(Some(raw))
    }
    /// The checked Rust integer type of the payload.
    #[must_use]
    pub fn backing_type(&self) -> Type {
        Type::Int { min: self.min, max: self.max }
    }
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

#[cfg(test)]
mod bounded_controls {
    use super::Bounded;

    #[test]
    fn exact_optional_codes_preserve_absence_and_exhaustion() {
        let ticks = Bounded {
            name: "Ticks".into(),
            field: None,
            backing: "u8".into(),
            min: 0,
            max: 255,
            low: 1,
            high: 3,
        };
        let mut remaining = ticks.clone();
        remaining.name = "Remaining".into();
        remaining.field = Some("ticks".into());
        remaining.low = 0;
        assert_ne!(ticks, remaining);
        for code in -1..=5 {
            assert_eq!(ticks.decode(code).is_some(), [0, 2, 3, 4].contains(&code));
            assert_eq!(remaining.decode(code).is_some(), [0, 1, 2, 3, 4].contains(&code));
        }
        assert_eq!(ticks.decode(0), Some(None));
        assert_eq!(remaining.decode(1), Some(Some(0)));
        assert_eq!(ticks.encode(0), None);
        assert_eq!(ticks.encode(3), Some(4));
        for (backing, min, max, low, high) in [
            ("u8", 0, 255, -1, 3),
            ("u8", 0, 255, 3, 1),
            ("u8", 0, 256, 1, 3),
            ("u32", 0, 255, 1, 3),
            ("i32", -2_147_483_648, 2_147_483_647, 0, 2_147_483_647),
        ] {
            let descriptor =
                Bounded { backing: backing.into(), min, max, low, high, ..ticks.clone() };
            assert!(!descriptor.optional_valid());
            assert_eq!(descriptor.decode(0), None);
        }
    }
}
