//! Constraint vocabulary and annotation consequences.

use std::time::Duration;

/// Unicode scalar count, UTF-8 byte count and literal string constraints.
#[derive(Clone, Default, PartialEq, Eq)]
pub struct StringRules {
    /// Inclusive minimum character count.
    pub min_len: Option<usize>,
    /// Inclusive maximum character count.
    pub max_len: Option<usize>,
    /// Inclusive minimum UTF-8 byte count.
    pub min_bytes: Option<usize>,
    /// Inclusive maximum UTF-8 byte count.
    pub max_bytes: Option<usize>,
    /// Required literal prefix.
    pub prefix: Option<String>,
    /// Required literal suffix.
    pub suffix: Option<String>,
    /// Required literal substring.
    pub contains: Option<String>,
}

impl std::fmt::Debug for StringRules {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let mut fields = f.debug_struct("StringRules");
        fields.field("min_len", &self.min_len).field("max_len", &self.max_len);
        if self.min_bytes.is_some() {
            fields.field("min_bytes", &self.min_bytes);
        }
        if self.max_bytes.is_some() {
            fields.field("max_bytes", &self.max_bytes);
        }
        fields
            .field("prefix", &self.prefix)
            .field("suffix", &self.suffix)
            .field("contains", &self.contains)
            .finish()
    }
}

/// Byte count constraints, also used for generated random data.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct BytesRules {
    /// Inclusive minimum byte count.
    pub min_len: Option<usize>,
    /// Inclusive maximum byte count.
    pub max_len: Option<usize>,
}

/// Integral bounds; durations and timestamps use total nanoseconds.
/// A lower bound greater than the upper bound admits values outside the range,
/// following protovalidate. Equal bounds retain their declared inclusivity.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct NumberRules {
    /// Exclusive lower bound.
    pub gt: Option<i128>,
    /// Inclusive lower bound.
    pub gte: Option<i128>,
    /// Exclusive upper bound.
    pub lt: Option<i128>,
    /// Inclusive upper bound.
    pub lte: Option<i128>,
}

/// Integer discriminants declared by the message's exhaustive enum mapping.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct EnumRules {
    /// Reject integer values not present in `values`.
    pub defined_only: bool,
    /// Exhaustively declared discriminants, in author order.
    pub values: Vec<i32>,
}

/// Typed enum validity bridge, identified by its concrete Rust type.
#[derive(Clone)]
pub struct EnumValidator {
    /// Fully qualified Rust enum type.
    pub name: &'static str,
    /// Valid default discriminant for generation.
    pub default: i32,
    /// Concrete enum's validity check.
    pub is_valid: fn(i32) -> bool,
}

impl std::fmt::Debug for EnumValidator {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("EnumValidator")
            .field("name", &self.name)
            .field("default", &self.default)
            .finish_non_exhaustive()
    }
}
impl PartialEq for EnumValidator {
    fn eq(&self, other: &Self) -> bool {
        self.name == other.name && self.default == other.default
    }
}
impl Eq for EnumValidator {}

/// Exactly one semantic kind per field.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Kind {
    /// Boolean flag.
    Flag,
    /// Finite floating-point number with integral bounds.
    Number(NumberRules),
    /// Enum validated through its concrete generated type.
    DefinedEnum(EnumValidator),
    /// Canonical, hyphenated UUID string.
    Uuid,
    /// Nonempty identifier plus string constraints.
    Identifier(StringRules),
    /// A valid timestamp, constrained in nanoseconds from the epoch.
    Timestamp(NumberRules),
    /// A valid signed duration, constrained in nanoseconds.
    Duration(NumberRules),
    /// Integral numeric constraints.
    Counter(NumberRules),
    /// Seeded random string or byte data.
    Random(BytesRules),
    /// Externally supplied data, always redacted in reports.
    Secret,
    /// Enum discriminants.
    Enum(EnumRules),
    /// Unicode text constraints.
    Text(StringRules),
    /// Opaque bytes and byte count constraints.
    Bytes(BytesRules),
}

impl Kind {
    /// Stable constraint names carried into validation and probe metadata.
    pub(crate) fn constraints(&self) -> Vec<&'static str> {
        let mut names = Vec::new();
        match self {
            Self::Uuid => names.push("string.uuid"),
            Self::Identifier(rules) | Self::Text(rules) => {
                if rules.min_len.is_some() || matches!(self, Self::Identifier(_)) {
                    names.push("string.min_len");
                }
                if rules.max_len.is_some() {
                    names.push("string.max_len");
                }
                if rules.min_bytes.is_some() {
                    names.push("string.min_bytes");
                }
                if rules.max_bytes.is_some() {
                    names.push("string.max_bytes");
                }
                if rules.prefix.is_some() {
                    names.push("string.prefix");
                }
                if rules.suffix.is_some() {
                    names.push("string.suffix");
                }
                if rules.contains.is_some() {
                    names.push("string.contains");
                }
            }
            Self::Bytes(rules) | Self::Random(rules) => {
                if rules.min_len.is_some() {
                    names.push("bytes.min_len");
                }
                if rules.max_len.is_some() {
                    names.push("bytes.max_len");
                }
            }
            Self::Counter(rules)
            | Self::Number(rules)
            | Self::Timestamp(rules)
            | Self::Duration(rules) => {
                let prefix = self.number_prefix();
                if matches!(self, Self::Number(_)) {
                    names.push("number.finite");
                }
                if matches!(self, Self::Timestamp(_) | Self::Duration(_)) {
                    names.push(match self {
                        Self::Timestamp(_) => "timestamp.valid",
                        _ => "duration.valid",
                    });
                }
                if rules.gt.is_some() {
                    names.push(number_name(prefix, "gt"));
                }
                if rules.gte.is_some() {
                    names.push(number_name(prefix, "gte"));
                }
                if rules.lt.is_some() {
                    names.push(number_name(prefix, "lt"));
                }
                if rules.lte.is_some() {
                    names.push(number_name(prefix, "lte"));
                }
            }
            Self::Enum(rules) if rules.defined_only => names.push("enum.defined_only"),
            Self::DefinedEnum(_) => names.push("enum.defined_only"),
            Self::Flag | Self::Enum(_) | Self::Secret => {}
        }
        names
    }

    pub(crate) fn number_prefix(&self) -> &'static str {
        match self {
            Self::Timestamp(_) => "timestamp",
            Self::Duration(_) => "duration",
            _ => "number",
        }
    }
}

pub(crate) fn number_name(prefix: &str, bound: &str) -> &'static str {
    match (prefix, bound) {
        ("i32", "gt") => "int32.gt",
        ("i32", "gte") => "int32.gte",
        ("i32", "lt") => "int32.lt",
        ("i32", _) => "int32.lte",
        ("i64", "gt") => "int64.gt",
        ("i64", "gte") => "int64.gte",
        ("i64", "lt") => "int64.lt",
        ("i64", _) => "int64.lte",
        ("u32", "gt") => "uint32.gt",
        ("u32", "gte") => "uint32.gte",
        ("u32", "lt") => "uint32.lt",
        ("u32", _) => "uint32.lte",
        ("u64", "gt") => "uint64.gt",
        ("u64", "gte") => "uint64.gte",
        ("u64", "lt") => "uint64.lt",
        ("u64", _) => "uint64.lte",
        ("timestamp", "gt") => "timestamp.gt",
        ("timestamp", "gte") => "timestamp.gte",
        ("timestamp", "lt") => "timestamp.lt",
        ("timestamp", _) => "timestamp.lte",
        ("duration", "gt") => "duration.gt",
        ("duration", "gte") => "duration.gte",
        ("duration", "lt") => "duration.lt",
        ("duration", _) => "duration.lte",
        (_, "gt") => "number.gt",
        (_, "gte") => "number.gte",
        (_, "lt") => "number.lt",
        _ => "number.lte",
    }
}

/// Constraints on repeated typed fields.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct RepeatedRules {
    /// Inclusive minimum item count.
    pub min_items: Option<usize>,
    /// Inclusive maximum item count.
    pub max_items: Option<usize>,
    /// Reject equal items.
    pub unique: bool,
}

/// A concrete action available to the engine or the Model.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Consequence {
    /// Values never decrease.
    Monotone,
    /// Values cannot change after creation.
    Immutable,
    /// Observations may be cached for this duration.
    Cached {
        /// Maximum cache age.
        ttl: Duration,
    },
    /// Observations may trail a write for this many polls.
    EventuallyConsistent {
        /// Poll bound.
        polls: u32,
    },
    /// Repeated operations with the same value represent one operation.
    IdempotencyKey,
    /// The Model supplies the value; filling leaves it untouched.
    Decided,
    /// Supplied by the system; filling leaves it untouched.
    Observed,
    /// Explicitly outside the Model's judgment.
    Ignored(String),
}

/// Extensible tags must declare a consequence and compatible kinds.
pub trait Tag {
    /// Stable tag name used in admission errors.
    fn name(&self) -> &str;
    /// The engine or Model action represented by this tag.
    fn consequence(&self) -> Consequence;
    /// Whether this tag applies to this semantic kind.
    fn applies_to(&self, kind: &Kind) -> bool;
}

impl Tag for Consequence {
    fn name(&self) -> &str {
        match self {
            Self::Monotone => "Monotone",
            Self::Immutable => "Immutable",
            Self::Cached { .. } => "Cached",
            Self::EventuallyConsistent { .. } => "EventuallyConsistent",
            Self::IdempotencyKey => "IdempotencyKey",
            Self::Decided => "Decided",
            Self::Observed => "Observed",
            Self::Ignored(_) => "Ignored",
        }
    }
    fn consequence(&self) -> Consequence {
        self.clone()
    }
    fn applies_to(&self, kind: &Kind) -> bool {
        match self {
            Self::Monotone => {
                matches!(kind, Kind::Counter(_) | Kind::Timestamp(_) | Kind::Duration(_))
            }
            Self::IdempotencyKey => {
                matches!(kind, Kind::Uuid | Kind::Identifier(_) | Kind::Text(_) | Kind::Bytes(_))
            }
            _ => true,
        }
    }
}
