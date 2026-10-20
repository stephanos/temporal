//! Native typed values and deterministic generation.

use crate::{BytesRules, Kind, NumberRules, RepeatedRules, StringRules, kind::number_name};
use std::fmt;

/// Stable `SplitMix64` stream; no operating-system randomness is consulted.
#[derive(Clone, Debug)]
pub struct Rng(u64);

impl Rng {
    /// Starts a deterministic stream.
    pub const fn new(seed: u64) -> Self {
        Self(seed)
    }
    /// Draws the next integer.
    pub fn next_u64(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut value = self.0;
        value = (value ^ (value >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        value = (value ^ (value >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        value ^ (value >> 31)
    }
}

/// Refusal to construct a conforming typed value.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GenerationError {
    /// Secrets must come from the caller.
    ExternalSecret,
    /// The kind does not apply to this Rust type.
    UnsupportedType,
    /// No value satisfies the declared constraints.
    ImpossibleConstraints,
    /// Construction exceeds the bounded generator's size or presence-search limit.
    SizeLimit,
}

impl fmt::Display for GenerationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{self:?}")
    }
}
impl std::error::Error for GenerationError {}

/// Direct typed operations on a Rust field. Adapters may implement this for local message types.
pub trait FieldValue: Clone + Default + PartialEq + 'static {
    /// Whether the field's Rust type can carry this kind.
    fn supports(kind: &Kind) -> bool;
    /// Constraint names specialized to the field's numeric carrier.
    fn constraints(kind: &Kind) -> Vec<&'static str> {
        kind.constraints()
    }
    /// JSON-Schema-style presence: nondefault scalar, nonempty sequence, or present optional.
    fn present(&self) -> bool;
    /// Constraint names violated by this value, without its contents.
    fn violations(&self, kind: &Kind) -> Vec<&'static str>;
    /// Constructs a value or explicitly refuses.
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError>;
    /// Whether this field supports repeated constraints.
    fn is_repeated() -> bool {
        false
    }
    /// Violated repeated constraints.
    fn repeated_violations(&self, _rules: &RepeatedRules) -> Vec<&'static str> {
        Vec::new()
    }
    /// Generates a repeated field; scalar fields refuse.
    fn generate_repeated(
        _kind: &Kind,
        _rules: &RepeatedRules,
        _rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        Err(GenerationError::UnsupportedType)
    }
}

const MAX_GENERATED_LEN: usize = 4096;

fn text_violations(value: &str, rules: &StringRules, identifier: bool) -> Vec<&'static str> {
    let mut result = Vec::new();
    let min = rules.min_len.unwrap_or(0).max(usize::from(identifier));
    let len = value.chars().count();
    if len < min {
        result.push("string.min_len");
    }
    if rules.max_len.is_some_and(|max| len > max) {
        result.push("string.max_len");
    }
    if rules.prefix.as_ref().is_some_and(|prefix| !value.starts_with(prefix)) {
        result.push("string.prefix");
    }
    if rules.suffix.as_ref().is_some_and(|suffix| !value.ends_with(suffix)) {
        result.push("string.suffix");
    }
    if rules.contains.as_ref().is_some_and(|part| !value.contains(part)) {
        result.push("string.contains");
    }
    result
}

fn bytes_violations(len: usize, rules: &BytesRules) -> Vec<&'static str> {
    let mut result = Vec::new();
    if rules.min_len.is_some_and(|min| len < min) {
        result.push("bytes.min_len");
    }
    if rules.max_len.is_some_and(|max| len > max) {
        result.push("bytes.max_len");
    }
    result
}

fn generated_len(min: usize, max: Option<usize>) -> Result<usize, GenerationError> {
    if max.is_some_and(|max| min > max) {
        return Err(GenerationError::ImpossibleConstraints);
    }
    if min > MAX_GENERATED_LEN {
        return Err(GenerationError::SizeLimit);
    }
    Ok(min)
}

fn uuid(value: &str) -> bool {
    value.len() == 36
        && value.bytes().enumerate().all(|(i, byte)| {
            if matches!(i, 8 | 13 | 18 | 23) { byte == b'-' } else { byte.is_ascii_hexdigit() }
        })
}

fn append_overlapping(value: &mut String, suffix: &str) {
    let overlap = suffix
        .char_indices()
        .map(|(i, _)| i)
        .chain(std::iter::once(suffix.len()))
        .filter(|&i| value.ends_with(&suffix[..i]))
        .max()
        .unwrap_or(0);
    value.push_str(&suffix[overlap..]);
}

impl FieldValue for String {
    fn supports(kind: &Kind) -> bool {
        matches!(
            kind,
            Kind::Uuid | Kind::Identifier(_) | Kind::Text(_) | Kind::Random(_) | Kind::Secret
        )
    }
    fn present(&self) -> bool {
        !self.is_empty()
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        match kind {
            Kind::Uuid if !uuid(self) => vec!["string.uuid"],
            Kind::Identifier(rules) | Kind::Text(rules) => {
                text_violations(self, rules, matches!(kind, Kind::Identifier(_)))
            }
            Kind::Random(rules) => bytes_violations(self.len(), rules),
            _ => Vec::new(),
        }
    }
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        match kind {
            Kind::Secret => Err(GenerationError::ExternalSecret),
            Kind::Uuid => {
                let a = rng.next_u64();
                let b = rng.next_u64();
                Ok(format!(
                    "{:08x}-{:04x}-{:04x}-{:04x}-{:012x}",
                    a >> 32,
                    (a >> 16) & 0xffff,
                    (a & 0x0fff) | 0x4000,
                    ((b >> 48) & 0x3fff) | 0x8000,
                    b & 0x0000_ffff_ffff_ffff
                ))
            }
            Kind::Random(rules) => {
                let len = generated_len(
                    rules.min_len.unwrap_or(16).min(rules.max_len.unwrap_or(usize::MAX)),
                    rules.max_len,
                )?;
                if rules.min_len.is_some_and(|min| len < min) {
                    return Err(GenerationError::ImpossibleConstraints);
                }
                Ok((0..len)
                    .map(|_| char::from(b'a' + u8::try_from(rng.next_u64() % 26).unwrap_or(0)))
                    .collect())
            }
            Kind::Identifier(rules) | Kind::Text(rules) => {
                let mut value = rules.prefix.clone().unwrap_or_default();
                if let Some(part) = &rules.contains
                    && !value.contains(part)
                {
                    append_overlapping(&mut value, part);
                }
                let suffix = rules.suffix.as_deref().unwrap_or_default();
                if !value.ends_with(suffix) {
                    append_overlapping(&mut value, suffix);
                }
                let min = rules
                    .min_len
                    .unwrap_or(0)
                    .max(usize::from(matches!(kind, Kind::Identifier(_))));
                generated_len(min.max(value.chars().count()), rules.max_len)?;
                if value.chars().count() < min {
                    let padding: String = (value.chars().count()..min)
                        .map(|_| char::from(b'a' + u8::try_from(rng.next_u64() % 26).unwrap_or(0)))
                        .collect();
                    for insertion in
                        value.char_indices().map(|(i, _)| i).chain(std::iter::once(value.len()))
                    {
                        let mut padded = value.clone();
                        padded.insert_str(insertion, &padding);
                        if padded.violations(kind).is_empty() {
                            return Ok(padded);
                        }
                    }
                    value.push_str(suffix);
                    let len = value.chars().count();
                    generated_len(len.max(min), rules.max_len)?;
                    if len < min {
                        value.insert_str(value.len() - suffix.len(), &"a".repeat(min - len));
                    }
                }
                if value.violations(kind).is_empty() {
                    Ok(value)
                } else {
                    Err(GenerationError::ImpossibleConstraints)
                }
            }
            _ => Err(GenerationError::UnsupportedType),
        }
    }
}

impl FieldValue for Vec<u8> {
    fn supports(kind: &Kind) -> bool {
        matches!(kind, Kind::Bytes(_) | Kind::Random(_) | Kind::Secret)
    }
    fn present(&self) -> bool {
        !self.is_empty()
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        match kind {
            Kind::Bytes(rules) | Kind::Random(rules) => bytes_violations(self.len(), rules),
            _ => Vec::new(),
        }
    }
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        match kind {
            Kind::Secret => Err(GenerationError::ExternalSecret),
            Kind::Bytes(rules) | Kind::Random(rules) => {
                let min = rules.min_len.unwrap_or_else(|| rules.max_len.unwrap_or(1).min(1));
                let len = generated_len(min, rules.max_len)?;
                Ok((0..len).map(|_| rng.next_u64().to_le_bytes()[0]).collect())
            }
            _ => Err(GenerationError::UnsupportedType),
        }
    }
}

fn number_violations(value: i128, rules: &NumberRules, prefix: &str) -> Vec<&'static str> {
    let mut result = Vec::new();
    for (violated, name) in [
        (rules.gt.is_some_and(|bound| value <= bound), "gt"),
        (rules.gte.is_some_and(|bound| value < bound), "gte"),
        (rules.lt.is_some_and(|bound| value >= bound), "lt"),
        (rules.lte.is_some_and(|bound| value > bound), "lte"),
    ] {
        if violated {
            result.push(number_name(prefix, name));
        }
    }
    result
}

fn number_bounds(
    rules: &NumberRules,
    min: i128,
    max: i128,
) -> Result<(i128, i128), GenerationError> {
    let lo = min.max(rules.gte.unwrap_or(min)).max(match rules.gt {
        Some(n) => n.checked_add(1).ok_or(GenerationError::ImpossibleConstraints)?,
        None => min,
    });
    let hi = max.min(rules.lte.unwrap_or(max)).min(match rules.lt {
        Some(n) => n.checked_sub(1).ok_or(GenerationError::ImpossibleConstraints)?,
        None => max,
    });
    if lo > hi { Err(GenerationError::ImpossibleConstraints) } else { Ok((lo, hi)) }
}

macro_rules! integral {
    ($($ty:ty),*) => {$ (
        impl FieldValue for $ty {
            fn supports(kind: &Kind) -> bool { matches!(kind, Kind::Counter(_)) || (stringify!($ty) == "i32" && matches!(kind, Kind::Enum(_))) }
            fn constraints(kind: &Kind) -> Vec<&'static str> {
                kind.constraints().into_iter().map(|name| name.strip_prefix("number.").map_or(name, |bound| number_name(stringify!($ty), bound))).collect()
            }
            fn present(&self) -> bool { *self != 0 }
            fn violations(&self, kind: &Kind) -> Vec<&'static str> {
                match kind {
                    Kind::Counter(rules) => number_violations(i128::from(*self), rules, stringify!($ty)),
                    Kind::Enum(rules) if rules.defined_only && !rules.values.iter().any(|value| i128::from(*value) == i128::from(*self)) => vec!["enum.defined_only"],
                    _ => Vec::new(),
                }
            }
            fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
                match kind {
                    Kind::Counter(rules) => {
                        let (lo, hi) = number_bounds(rules, i128::from(<$ty>::MIN), i128::from(<$ty>::MAX))?;
                        let span = u128::try_from(hi - lo + 1).map_err(|_| GenerationError::ImpossibleConstraints)?;
                        let offset = u128::from(rng.next_u64()) % span;
                        <$ty>::try_from(lo + i128::try_from(offset).map_err(|_| GenerationError::ImpossibleConstraints)?).map_err(|_| GenerationError::ImpossibleConstraints)
                    }
                    Kind::Enum(rules) => {
                        if !rules.defined_only && rules.values.is_empty() {
                            let [a, b, c, d, ..] = rng.next_u64().to_le_bytes();
                            return <$ty>::try_from(i32::from_le_bytes([a, b, c, d])).map_err(|_| GenerationError::ImpossibleConstraints);
                        }
                        if rules.values.is_empty() { return Err(GenerationError::ImpossibleConstraints); }
                        let index = usize::try_from(rng.next_u64() % u64::try_from(rules.values.len()).map_err(|_| GenerationError::SizeLimit)?).map_err(|_| GenerationError::SizeLimit)?;
                        <$ty>::try_from(rules.values[index]).map_err(|_| GenerationError::ImpossibleConstraints)
                    }
                    _ => Err(GenerationError::UnsupportedType),
                }
            }
        }
    )*};
}
integral!(i32, i64, u32, u64);

/// A protobuf-compatible timestamp; projected accessors can adapt generated types.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Timestamp {
    /// Whole seconds since the epoch.
    pub seconds: i64,
    /// Nonnegative fractional nanoseconds.
    pub nanos: i32,
}

/// A protobuf-compatible signed duration.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Duration {
    /// Whole signed seconds.
    pub seconds: i64,
    /// Fractional nanoseconds with the same sign as nonzero seconds.
    pub nanos: i32,
}

macro_rules! time_value {
    ($ty:ident, $min:expr, $max:expr, $valid:expr) => {
        impl FieldValue for $ty {
            fn supports(kind: &Kind) -> bool {
                matches!(kind, Kind::$ty(_))
            }
            fn present(&self) -> bool {
                self.seconds != 0 || self.nanos != 0
            }
            fn violations(&self, kind: &Kind) -> Vec<&'static str> {
                let mut result = Vec::new();
                if !($valid)(self) {
                    result.push(concat!(stringify!($ty), ".invalid"));
                }
                if let Kind::$ty(rules) = kind {
                    if !result.is_empty() {
                        result[0] = match kind {
                            Kind::Timestamp(_) => "timestamp.valid",
                            _ => "duration.valid",
                        };
                    }
                    result.extend(number_violations(
                        i128::from(self.seconds) * 1_000_000_000 + i128::from(self.nanos),
                        rules,
                        kind.number_prefix(),
                    ));
                }
                result
            }
            fn generate(kind: &Kind, _rng: &mut Rng) -> Result<Self, GenerationError> {
                let Kind::$ty(rules) = kind else {
                    return Err(GenerationError::UnsupportedType);
                };
                let (lo, _) = number_bounds(rules, $min, $max)?;
                let seconds = lo.div_euclid(1_000_000_000);
                let nanos = lo.rem_euclid(1_000_000_000);
                let (seconds, nanos) = if stringify!($ty) == "Duration" && seconds < 0 && nanos > 0
                {
                    (seconds + 1, nanos - 1_000_000_000)
                } else {
                    (seconds, nanos)
                };
                Ok(Self {
                    seconds: i64::try_from(seconds)
                        .map_err(|_| GenerationError::ImpossibleConstraints)?,
                    nanos: i32::try_from(nanos)
                        .map_err(|_| GenerationError::ImpossibleConstraints)?,
                })
            }
        }
    };
}
time_value!(
    Timestamp,
    -62_135_596_800_i128 * 1_000_000_000,
    253_402_300_799_i128 * 1_000_000_000 + 999_999_999,
    |v: &Timestamp| (-62_135_596_800..=253_402_300_799).contains(&v.seconds)
        && (0..1_000_000_000).contains(&v.nanos)
);
time_value!(
    Duration,
    -315_576_000_000_i128 * 1_000_000_000 - 999_999_999,
    315_576_000_000_i128 * 1_000_000_000 + 999_999_999,
    |v: &Duration| (-315_576_000_000..=315_576_000_000).contains(&v.seconds)
        && (-999_999_999..=999_999_999).contains(&v.nanos)
        && (v.seconds == 0 || v.nanos == 0 || (v.seconds > 0) == (v.nanos > 0))
);

impl<T: FieldValue> FieldValue for Option<T> {
    fn constraints(kind: &Kind) -> Vec<&'static str> {
        T::constraints(kind)
    }
    fn supports(kind: &Kind) -> bool {
        T::supports(kind)
    }
    fn present(&self) -> bool {
        self.is_some()
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        self.as_ref().map_or_else(Vec::new, |value| value.violations(kind))
    }
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        T::generate(kind, rng).map(Some)
    }
}

impl<T: FieldValue> FieldValue for Vec<T> {
    fn constraints(kind: &Kind) -> Vec<&'static str> {
        T::constraints(kind)
    }
    fn supports(kind: &Kind) -> bool {
        T::supports(kind)
    }
    fn present(&self) -> bool {
        !self.is_empty()
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        self.iter().flat_map(|value| value.violations(kind)).collect()
    }
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        Ok(vec![T::generate(kind, rng)?])
    }
    fn is_repeated() -> bool {
        true
    }
    fn repeated_violations(&self, rules: &RepeatedRules) -> Vec<&'static str> {
        let mut result = Vec::new();
        if rules.min_items.is_some_and(|min| self.len() < min) {
            result.push("repeated.min_items");
        }
        if rules.max_items.is_some_and(|max| self.len() > max) {
            result.push("repeated.max_items");
        }
        if rules.unique && self.iter().enumerate().any(|(i, value)| self[..i].contains(value)) {
            result.push("repeated.unique");
        }
        result
    }
    fn generate_repeated(
        kind: &Kind,
        rules: &RepeatedRules,
        rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        let len = generated_len(rules.min_items.unwrap_or(1), rules.max_items)?;
        let mut result = Vec::with_capacity(len);
        for _ in 0..len {
            let mut found = None;
            for _ in 0..MAX_GENERATED_LEN {
                let value = T::generate(kind, rng)?;
                if !rules.unique || !result.contains(&value) {
                    found = Some(value);
                    break;
                }
            }
            result.push(found.ok_or(GenerationError::ImpossibleConstraints)?);
        }
        Ok(result)
    }
}
