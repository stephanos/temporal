//! The values an interpreter of the tree computes, which a compiled Model's generated accessors
//! read from and write into its typed state.

/// One value of an admitted [`Type`](crate::Type).
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub enum Value {
    /// Boolean.
    Bool(bool),
    /// Integer of any admitted width.
    Int(i128),
    /// Fieldless enum variant, by its position in its type's variants.
    Enum(usize),
    /// String.
    Text(String),
    /// Unit.
    Unit,
    /// Tuple.
    Tuple(Vec<Value>),
    /// Sequence.
    Seq(Vec<Value>),
}
