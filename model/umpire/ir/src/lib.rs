//! The admitted Model tree: what `umpire-front` reads from a `#[machine]` module, and what the
//! macro embeds in the compiled Model for the lifted back ends. Internal, with no stability
//! promise; it is plain data, so compiled Models can carry it.
#[doc(hidden)]
pub mod expr;
#[doc(hidden)]
pub mod tree;
use std::collections::BTreeMap;
pub use tree::Lifted;

/// Admitted value types for typed backend decoding.
#[derive(Clone, Debug, PartialEq, Eq)]
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
/// A located refusal of unsupported source behavior; the consumer names the file.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Diagnostic {
    /// One-based line.
    pub line: usize,
    /// One-based column.
    pub column: usize,
    /// Admission failure.
    pub message: String,
}
/// The names a `#[machine]` module spells, which generated back-end names keep apart from.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
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
#[derive(Clone, Debug)]
pub struct Embedded {
    /// The canonical path of the source file the module is written in.
    pub file: String,
    /// The admitted Model, or the located subset refusal; native checking holds either way.
    pub model: Result<Lifted, Diagnostic>,
    /// The module's names.
    pub source: Source,
}
