//! Native typed values and deterministic generation.

use crate::{BytesRules, Kind, NumberRules, RepeatedRules, StringRules, kind::number_name};

pub use umpire_checker::SplitMix64 as Rng;

/// Refusal to construct a conforming typed value.
#[derive(Clone, Debug, PartialEq, Eq, thiserror::Error)]
#[error("{self:?}")]
pub enum GenerationError {
    /// This wire carrier cannot represent the requested mutation.
    InapplicableMutation,
    /// A projected message was not a valid protobuf encoding.
    InvalidEncoding,
    /// Secrets must come from the caller.
    ExternalSecret,
    /// The kind does not apply to this Rust type.
    UnsupportedType,
    /// No value satisfies the declared constraints.
    ImpossibleConstraints,
    /// Construction exceeds the bounded generator's size or search limit.
    SizeLimit,
}

/// Direct typed operations on a Rust field. Adapters may implement this for local message types.
pub trait FieldValue: Clone + Default + PartialEq + 'static {
    /// Constructs a negative candidate; the catalog verifies its actual finding.
    fn negative(
        &self,
        kind: &Kind,
        _repeated: Option<&RepeatedRules>,
        constraint: &str,
        rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        Self::generate(&crate::negative::negative_kind(kind, constraint)?, rng)
    }
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
    /// Constructs a present candidate for the message's presence-rule search.
    fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        let value = Self::generate(kind, rng)?;
        if value.present() { Ok(value) } else { Err(GenerationError::ImpossibleConstraints) }
    }
    /// Whether this field supports repeated constraints.
    fn is_repeated() -> bool {
        false
    }
    /// Violated repeated constraints.
    fn repeated_violations(&self, _rules: &RepeatedRules) -> Vec<&'static str> {
        Vec::new()
    }
    /// Generates a repeated field; scalar fields refuse.
    /// Exhausted bounded sampling returns `SizeLimit`, not a proof of impossible constraints.
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
    if rules.min_bytes.is_some_and(|min| value.len() < min) {
        result.push("string.min_bytes");
    }
    if rules.max_bytes.is_some_and(|max| value.len() > max) {
        result.push("string.max_bytes");
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

fn sampled_len(min: usize, max: Option<usize>, rng: &mut Rng) -> Result<usize, GenerationError> {
    generated_len(min, max)?;
    let max = max.unwrap_or_else(|| min.saturating_add(16)).min(MAX_GENERATED_LEN);
    let span = u64::try_from(max - min + 1).map_err(|_| GenerationError::SizeLimit)?;
    Ok(min + usize::try_from(rng.next_u64() % span).map_err(|_| GenerationError::SizeLimit)?)
}

fn random_character(max_bytes: usize, rng: &mut Rng) -> char {
    let count = match max_bytes {
        1 => 0x80,
        2 => 0x800,
        3 => 0xf800,
        _ => 0x10_f800,
    };
    let index = u32::try_from(rng.next_u64() % count).unwrap_or(0);
    let scalar = if index >= 0xd800 { index + 0x800 } else { index };
    char::from_u32(scalar).unwrap_or('\0')
}

fn uuid(value: &str) -> bool {
    value.len() == 36
        && value.bytes().enumerate().all(|(i, byte)| {
            if matches!(i, 8 | 13 | 18 | 23) { byte == b'-' } else { byte.is_ascii_hexdigit() }
        })
}

fn place_anchor(
    slots: &mut [Option<char>],
    offset: usize,
    anchor: &[char],
    budget: &mut usize,
) -> Result<bool, GenerationError> {
    let slots = &mut slots[offset..offset + anchor.len()];
    for (slot, value) in slots.iter().zip(anchor) {
        if *budget == 0 {
            return Err(GenerationError::SizeLimit);
        }
        *budget -= 1;
        if slot.is_some_and(|fixed| fixed != *value) {
            return Ok(false);
        }
    }
    for (slot, value) in slots.iter_mut().zip(anchor) {
        *slot = Some(*value);
    }
    Ok(true)
}

fn text_candidate(
    len: usize,
    anchors: &[Vec<char>; 3],
    rules: &StringRules,
    rng: &mut Rng,
    budget: &mut usize,
) -> Result<Option<String>, GenerationError> {
    let [prefix, suffix, contains] = anchors;
    let mut slots = vec![None; len];
    if !place_anchor(&mut slots, 0, prefix, budget)?
        || !place_anchor(&mut slots, len - suffix.len(), suffix, budget)?
    {
        return Ok(None);
    }
    let positions = len - contains.len() + 1;
    let first = usize::try_from(
        rng.next_u64() % u64::try_from(positions).map_err(|_| GenerationError::SizeLimit)?,
    )
    .map_err(|_| GenerationError::SizeLimit)?;
    for step in 0..positions {
        let position = (first + step) % positions;
        let mut candidate = slots.clone();
        if place_anchor(&mut candidate, position, contains, budget)?
            && let Some(value) = complete_text(candidate, rules, rng)?
        {
            return Ok(Some(value));
        }
    }
    Ok(None)
}

fn complete_text(
    slots: Vec<Option<char>>,
    rules: &StringRules,
    rng: &mut Rng,
) -> Result<Option<String>, GenerationError> {
    if rules.min_bytes.is_none() && rules.max_bytes.is_none() {
        return Ok(Some(
            slots
                .into_iter()
                .map(|slot| slot.unwrap_or_else(|| random_character(4, rng)))
                .collect(),
        ));
    }
    let mut remaining = slots.iter().filter(|slot| slot.is_none()).count();
    let fixed: usize = slots.iter().flatten().map(|value| value.len_utf8()).sum();
    let min = rules.min_bytes.unwrap_or(0).max(fixed + remaining);
    let max = rules.max_bytes.unwrap_or(usize::MAX).min(fixed + 4 * remaining);
    if min > max {
        return Ok(None);
    }
    let span = u64::try_from(max - min + 1).map_err(|_| GenerationError::SizeLimit)?;
    let mut bytes = min
        + usize::try_from(rng.next_u64() % span).map_err(|_| GenerationError::SizeLimit)?
        - fixed;
    let mut value = String::new();
    for slot in slots {
        if let Some(character) = slot {
            value.push(character);
        } else {
            remaining -= 1;
            let min_width = bytes.saturating_sub(4 * remaining).max(1);
            let max_width = (bytes - remaining).min(4);
            let width = min_width
                + usize::try_from(
                    rng.next_u64()
                        % u64::try_from(max_width - min_width + 1)
                            .map_err(|_| GenerationError::SizeLimit)?,
                )
                .map_err(|_| GenerationError::SizeLimit)?;
            let (start, count) = match width {
                1 => (0, 0x80),
                2 => (0x80, 0x780),
                3 => (0x800, 0xf000),
                _ => (0x1_0000, 0x10_0000),
            };
            let scalar = start
                + u32::try_from(rng.next_u64() % count).map_err(|_| GenerationError::SizeLimit)?;
            let scalar = if width == 3 && scalar >= 0xd800 { scalar + 0x800 } else { scalar };
            value.push(char::from_u32(scalar).ok_or(GenerationError::ImpossibleConstraints)?);
            bytes -= width;
        }
    }
    Ok(Some(value))
}

fn generate_text(
    rules: &StringRules,
    identifier: bool,
    rng: &mut Rng,
) -> Result<String, GenerationError> {
    if rules.min_bytes.is_some_and(|min| rules.max_bytes.is_some_and(|max| min > max)) {
        return Err(GenerationError::ImpossibleConstraints);
    }
    let anchors = [&rules.prefix, &rules.suffix, &rules.contains].map(|anchor| {
        anchor
            .as_deref()
            .unwrap_or_default()
            .chars()
            .take(MAX_GENERATED_LEN + 1)
            .collect::<Vec<_>>()
    });
    let min = rules
        .min_len
        .unwrap_or(0)
        .max(usize::from(identifier))
        .max(rules.min_bytes.unwrap_or(0).div_ceil(4))
        .max(anchors.iter().map(Vec::len).max().unwrap_or(0));
    let max_len = rules.max_len.into_iter().chain(rules.max_bytes).min();
    generated_len(min, max_len)?;
    let max = rules
        .max_len
        .unwrap_or_else(|| {
            (min + anchors.iter().map(Vec::len).sum::<usize>() + 16)
                .max(rules.min_bytes.unwrap_or(0))
        })
        .min(rules.max_bytes.unwrap_or(usize::MAX))
        .min(MAX_GENERATED_LEN);
    let first = sampled_len(min, Some(max), rng)?;
    let mut budget = MAX_GENERATED_LEN * MAX_GENERATED_LEN;
    for len in std::iter::once(first).chain((min..=max).filter(|len| *len != first)) {
        if let Some(value) = text_candidate(len, &anchors, rules, rng, &mut budget)? {
            return Ok(value);
        }
    }
    if max_len.is_none_or(|max| max > MAX_GENERATED_LEN) {
        Err(GenerationError::SizeLimit)
    } else {
        Err(GenerationError::ImpossibleConstraints)
    }
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
                let len = sampled_len(
                    rules.min_len.unwrap_or(16).min(rules.max_len.unwrap_or(usize::MAX)),
                    rules.max_len,
                    rng,
                )?;
                if rules.min_len.is_some_and(|min| len < min) {
                    return Err(GenerationError::ImpossibleConstraints);
                }
                let mut value = String::with_capacity(len);
                while value.len() < len {
                    value.push(random_character(len - value.len(), rng));
                }
                Ok(value)
            }
            Kind::Identifier(rules) | Kind::Text(rules) => {
                generate_text(rules, matches!(kind, Kind::Identifier(_)), rng)
            }
            _ => Err(GenerationError::UnsupportedType),
        }
    }
    fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        let mut kind = kind.clone();
        match &mut kind {
            Kind::Text(rules) | Kind::Identifier(rules) => {
                rules.min_len = Some(rules.min_len.unwrap_or(0).max(1));
            }
            Kind::Random(rules) => rules.min_len = Some(rules.min_len.unwrap_or(0).max(1)),
            _ => {}
        }
        Self::generate(&kind, rng)
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
                let len = sampled_len(min, rules.max_len, rng)?;
                Ok((0..len).map(|_| rng.next_u64().to_le_bytes()[0]).collect())
            }
            _ => Err(GenerationError::UnsupportedType),
        }
    }
    fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        let mut kind = kind.clone();
        if let Kind::Bytes(rules) | Kind::Random(rules) = &mut kind {
            rules.min_len = Some(rules.min_len.unwrap_or(0).max(1));
        }
        Self::generate(&kind, rng)
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
    if reversed_range(rules) {
        let lower_passes = rules.gt.is_none_or(|bound| value > bound)
            && rules.gte.is_none_or(|bound| value >= bound);
        let upper_passes = rules.lt.is_none_or(|bound| value < bound)
            && rules.lte.is_none_or(|bound| value <= bound);
        if lower_passes || upper_passes {
            result.clear();
        }
    }
    result
}

fn reversed_range(rules: &NumberRules) -> bool {
    let lower = rules.gt.into_iter().chain(rules.gte).max();
    let upper = rules.lt.into_iter().chain(rules.lte).min();
    matches!((lower, upper), (Some(lo), Some(hi)) if lo > hi)
}

fn number_sample(
    rules: &NumberRules,
    min: i128,
    max: i128,
    rng: &mut Rng,
    nonzero: bool,
) -> Result<i128, GenerationError> {
    let lo = match rules.gt {
        Some(n) => n.checked_add(1),
        None => Some(min),
    }
    .map(|n| n.max(rules.gte.unwrap_or(min)));
    let hi = match rules.lt {
        Some(n) => n.checked_sub(1),
        None => Some(max),
    }
    .map(|n| n.min(rules.lte.unwrap_or(max)));
    let mut ranges = Vec::new();
    let mut add = |lo: Option<i128>, hi: Option<i128>| {
        if let (Some(lo), Some(hi)) = (lo, hi) {
            let (lo, hi) = (lo.max(min), hi.min(max));
            if lo <= hi {
                if nonzero && lo <= 0 && hi >= 0 {
                    if lo < 0 {
                        ranges.push((lo, -1));
                    }
                    if hi > 0 {
                        ranges.push((1, hi));
                    }
                } else {
                    ranges.push((lo, hi));
                }
            }
        }
    };
    if reversed_range(rules) {
        add(Some(min), hi);
        add(lo, Some(max));
    } else {
        add(lo, hi);
    }
    if ranges.is_empty() {
        return Err(GenerationError::ImpossibleConstraints);
    }
    let index = usize::try_from(
        rng.next_u64() % u64::try_from(ranges.len()).map_err(|_| GenerationError::SizeLimit)?,
    )
    .map_err(|_| GenerationError::SizeLimit)?;
    let (lo, hi) = ranges[index];
    let span = u128::try_from(hi - lo + 1).map_err(|_| GenerationError::ImpossibleConstraints)?;
    let bits = (u128::from(rng.next_u64()) << 64) | u128::from(rng.next_u64());
    let offset = i128::try_from(bits % span).map_err(|_| GenerationError::ImpossibleConstraints)?;
    Ok(lo + offset)
}

macro_rules! integral {
    ($($ty:ty),*) => {$ (
        impl FieldValue for $ty {
            fn supports(kind: &Kind) -> bool { matches!(kind, Kind::Counter(_)) || (stringify!($ty) == "i32" && matches!(kind, Kind::Enum(_) | Kind::DefinedEnum(_))) }
            fn constraints(kind: &Kind) -> Vec<&'static str> {
                kind.constraints().into_iter().map(|name| name.strip_prefix("number.").map_or(name, |bound| number_name(stringify!($ty), bound))).collect()
            }
            fn present(&self) -> bool { *self != 0 }
            fn violations(&self, kind: &Kind) -> Vec<&'static str> {
                match kind {
                    Kind::Counter(rules) => number_violations(i128::from(*self), rules, stringify!($ty)),
                    Kind::DefinedEnum(rule) if i32::try_from(*self).ok().is_none_or(|n| !(rule.is_valid)(n)) => vec!["enum.defined_only"],
                    Kind::Enum(rules) if rules.defined_only && !rules.values.iter().any(|value| i128::from(*value) == i128::from(*self)) => vec!["enum.defined_only"],
                    _ => Vec::new(),
                }
            }
            fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
                match kind {
                    Kind::DefinedEnum(rule) => <$ty>::try_from(rule.default).map_err(|_| GenerationError::ImpossibleConstraints),
                    Kind::Counter(rules) => {
                        <$ty>::try_from(number_sample(rules, i128::from(<$ty>::MIN), i128::from(<$ty>::MAX), rng, false)?).map_err(|_| GenerationError::ImpossibleConstraints)
                    }
                    Kind::Enum(rules) => {
                        if !rules.defined_only {
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
            fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
                match kind {
                    Kind::DefinedEnum(rule) => {
                        let value = (1..=4096).find(|n| (rule.is_valid)(*n)).ok_or(GenerationError::SizeLimit)?;
                        <$ty>::try_from(value).map_err(|_| GenerationError::ImpossibleConstraints)
                    }
                    Kind::Counter(rules) => <$ty>::try_from(number_sample(rules, i128::from(<$ty>::MIN), i128::from(<$ty>::MAX), rng, true)?).map_err(|_| GenerationError::ImpossibleConstraints),
                    Kind::Enum(rules) => {
                        if !rules.defined_only { return Ok(1); }
                        let mut rules = rules.clone(); rules.values.retain(|n| *n != 0);
                        Self::generate(&Kind::Enum(rules), rng)
                    }
                    _ => Err(GenerationError::UnsupportedType),
                }
            }
        }
    )*};
}
integral!(i32, i64, u32, u64);

impl FieldValue for bool {
    fn supports(kind: &Kind) -> bool {
        matches!(kind, Kind::Flag)
    }
    fn present(&self) -> bool {
        *self
    }
    fn violations(&self, _kind: &Kind) -> Vec<&'static str> {
        Vec::new()
    }
    fn generate(kind: &Kind, _rng: &mut Rng) -> Result<Self, GenerationError> {
        if Self::supports(kind) { Ok(true) } else { Err(GenerationError::UnsupportedType) }
    }
}

impl FieldValue for f64 {
    fn supports(kind: &Kind) -> bool {
        matches!(kind, Kind::Number(_))
    }
    fn present(&self) -> bool {
        *self != 0.0
    }
    fn violations(&self, kind: &Kind) -> Vec<&'static str> {
        if !self.is_finite() {
            return vec!["number.finite"];
        }
        let Kind::Number(rules) = kind else {
            return Vec::new();
        };
        let bound = |n: i128| n.to_string().parse::<f64>().ok();
        let checks = [
            ("number.gt", rules.gt.and_then(bound).is_some_and(|n| *self <= n)),
            ("number.gte", rules.gte.and_then(bound).is_some_and(|n| *self < n)),
            ("number.lt", rules.lt.and_then(bound).is_some_and(|n| *self >= n)),
            ("number.lte", rules.lte.and_then(bound).is_some_and(|n| *self > n)),
        ];
        let inverted =
            rules.gt.or(rules.gte).zip(rules.lt.or(rules.lte)).is_some_and(|(a, b)| a > b);
        if inverted && !(checks[0].1 || checks[1].1) || inverted && !(checks[2].1 || checks[3].1) {
            return Vec::new();
        }
        checks.into_iter().filter_map(|(name, failed)| failed.then_some(name)).collect()
    }
    fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        generate_number(kind, rng, false)
    }
    fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
        generate_number(kind, rng, true)
    }
    fn negative(
        &self,
        kind: &Kind,
        _repeated: Option<&RepeatedRules>,
        constraint: &str,
        rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        if constraint == "number.finite" {
            return Ok(f64::INFINITY);
        }
        Self::generate(&crate::negative::negative_kind(kind, constraint)?, rng)
    }
}

fn generate_number(kind: &Kind, rng: &mut Rng, present_only: bool) -> Result<f64, GenerationError> {
    let Kind::Number(rules) = kind else {
        return Err(GenerationError::UnsupportedType);
    };
    let mut candidates = vec![0.0, 1.0, -1.0];
    for bound in [rules.gt, rules.gte, rules.lt, rules.lte].into_iter().flatten() {
        let bound: f64 =
            bound.to_string().parse().map_err(|_| GenerationError::ImpossibleConstraints)?;
        candidates.extend([bound, bound.next_down(), bound.next_up()]);
    }
    candidates
        .retain(|value| (!present_only || value.present()) && value.violations(kind).is_empty());
    let count = u64::try_from(candidates.len()).map_err(|_| GenerationError::SizeLimit)?;
    if count == 0 {
        return Err(GenerationError::ImpossibleConstraints);
    }
    let index = usize::try_from(rng.next_u64() % count).map_err(|_| GenerationError::SizeLimit)?;
    Ok(candidates[index])
}

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
        impl $ty {
            fn generated(
                rules: &NumberRules,
                rng: &mut Rng,
                nonzero: bool,
            ) -> Result<Self, GenerationError> {
                let value = number_sample(rules, $min, $max, rng, nonzero)?;
                let seconds = value.div_euclid(1_000_000_000);
                let nanos = value.rem_euclid(1_000_000_000);
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
        impl FieldValue for $ty {
            fn negative(
                &self,
                kind: &Kind,
                _repeated: Option<&RepeatedRules>,
                constraint: &str,
                rng: &mut Rng,
            ) -> Result<Self, GenerationError> {
                if matches!(constraint, "timestamp.valid" | "duration.valid") {
                    let offset = if self.nanos < 0 { -1 } else { 1 };
                    return Ok(Self {
                        seconds: self
                            .seconds
                            .checked_sub(offset)
                            .ok_or(GenerationError::ImpossibleConstraints)?,
                        nanos: self.nanos
                            + i32::try_from(offset)
                                .map_err(|_| GenerationError::ImpossibleConstraints)?
                                * 1_000_000_000,
                    });
                }
                Self::generate(&crate::negative::negative_kind(kind, constraint)?, rng)
            }
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
            fn generate(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
                let Kind::$ty(rules) = kind else {
                    return Err(GenerationError::UnsupportedType);
                };
                Self::generated(rules, rng, false)
            }
            fn generate_present(kind: &Kind, rng: &mut Rng) -> Result<Self, GenerationError> {
                let Kind::$ty(rules) = kind else {
                    return Err(GenerationError::UnsupportedType);
                };
                Self::generated(rules, rng, true)
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
    fn negative(
        &self,
        kind: &Kind,
        repeated: Option<&RepeatedRules>,
        constraint: &str,
        rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        self.clone().unwrap_or_default().negative(kind, repeated, constraint, rng).map(Some)
    }
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
    fn negative(
        &self,
        kind: &Kind,
        repeated: Option<&RepeatedRules>,
        constraint: &str,
        rng: &mut Rng,
    ) -> Result<Self, GenerationError> {
        if let Some(rules) = repeated {
            let len = match constraint {
                "repeated.min_items" => rules.min_items.and_then(|min| min.checked_sub(1)),
                "repeated.max_items" => rules.max_items.and_then(|max| max.checked_add(1)),
                "repeated.unique" => Some(self.len().max(2)),
                _ => None,
            };
            if let Some(len) = len {
                if len > MAX_GENERATED_LEN {
                    return Err(GenerationError::SizeLimit);
                }
                let mut value = self.clone();
                while value.len() < len {
                    value.push(T::generate(kind, rng)?);
                }
                value.truncate(len);
                if constraint == "repeated.unique" {
                    value[1] = value[0].clone();
                }
                return Ok(value);
            }
        }
        let mut value = self.clone();
        let negative =
            value.first().cloned().unwrap_or_default().negative(kind, None, constraint, rng)?;
        if value.is_empty() {
            value.push(negative);
        } else {
            value[0] = negative;
        }
        Ok(value)
    }
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
        let min = rules.min_items.unwrap_or_else(|| rules.max_items.unwrap_or(1).min(1));
        let len = generated_len(min, rules.max_items)?;
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
            result.push(found.ok_or(GenerationError::SizeLimit)?);
        }
        Ok(result)
    }
}
