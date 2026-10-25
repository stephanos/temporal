//! Constraint-independent wire mutations and black-box classification.

use crate::{Expectation, GapEntry, GapError, GapOutcome, GenerationError, ProbeResult, Violation};

/// Physical field carrier, independent of its semantic catalog classification.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum WireKind {
    /// UTF-8 string.
    String,
    /// Raw byte string.
    Bytes,
    /// Signed integer.
    Signed,
    /// Unsigned integer.
    Unsigned,
    /// Floating-point number.
    Float,
    /// Boolean.
    Bool,
    /// Descriptor-declared enum values.
    Enum {
        /// All reviewed wire discriminants.
        defined: Vec<i32>,
    },
    /// Nested message, including oneof alternatives.
    Message,
    /// Repeated values.
    Repeated,
    /// Map entries.
    Map,
}

/// One field in a bounded descriptor traversal, including ignored catalog fields.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WireField {
    /// Canonical traversal from the root request.
    pub path: String,
    /// Actual declaring protobuf field identity.
    pub declaration: String,
    /// Physical carrier.
    pub kind: WireKind,
    /// Whether wire presence can be represented explicitly.
    pub optional: bool,
    /// Explicit cycle, traversal bound or unsupported-shape refusal.
    pub refusal: Option<String>,
}

/// A generic mutation that carries no request data or catalog constraints.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum WireMutation {
    /// Remove wire presence or restore an implicit scalar default.
    Absent,
    /// Supply an empty container, string or message.
    Empty,
    /// Supply one more than an explicit, externally chosen probe bound.
    LimitPlusOne {
        /// Bound independent of the catalog field's constraints.
        limit: u64,
    },
    /// Supply a negative number.
    Negative,
    /// Supply an integer absent from the descriptor enum.
    UndefinedEnum,
}

impl WireMutation {
    /// Stable operation identity for reports and transport.
    pub fn name(&self) -> &'static str {
        match self {
            Self::Absent => "absent",
            Self::Empty => "empty",
            Self::LimitPlusOne { .. } => "limit_plus_one",
            Self::Negative => "negative",
            Self::UndefinedEnum => "undefined_enum",
        }
    }
}

/// Whether a descriptor-driven attempt can produce an actual request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum MutationApplicability {
    /// This carrier admits the operation.
    Applicable,
    /// The operation has no meaning for this carrier.
    NotApplicable(String),
    /// Traversal or construction explicitly refused.
    Refused(String),
    /// The mutation produced exactly the original wire request.
    Unchanged,
}

/// Maps a mutation's local applicability and refusal onto the annotation protocol.
pub fn probe_applicability(
    value: &MutationApplicability,
) -> (umpire_runtime::annotations::ProbeApplicability, String) {
    use umpire_runtime::annotations::ProbeApplicability;
    match value {
        MutationApplicability::Applicable => (ProbeApplicability::Applicable, String::new()),
        MutationApplicability::NotApplicable(reason) => {
            (ProbeApplicability::NotApplicable, reason.clone())
        }
        MutationApplicability::Refused(reason) => (ProbeApplicability::Refused, reason.clone()),
        MutationApplicability::Unchanged => {
            (ProbeApplicability::Unchanged, "mutation leaves the baseline unchanged".into())
        }
    }
}

/// One attributed generic mutation, retained even when construction is unavailable.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WireCase {
    /// Descriptor field independent of catalog coverage.
    pub field: WireField,
    /// Planned mutation.
    pub mutation: WireMutation,
    /// Honest applicability or refusal.
    pub applicability: MutationApplicability,
}

impl WireCase {
    /// Stable value-free identity distinguishing bounds on the same field.
    pub fn id(&self) -> String {
        format!("{}:{}", self.field.path, mutation_constraint(&self.mutation))
    }
}

fn mutation_constraint(mutation: &WireMutation) -> String {
    match mutation {
        WireMutation::LimitPlusOne { limit } => format!("mutation.{}({limit})", mutation.name()),
        _ => format!("mutation.{}", mutation.name()),
    }
}

/// Enumerates operations from physical field shapes, never semantic constraints.
pub fn wire_cases(fields: &[WireField], limits: &[u64]) -> Vec<WireCase> {
    let mut limits = limits.to_vec();
    limits.sort_unstable();
    limits.dedup();
    let mut cases = Vec::new();
    for field in fields {
        let mut mutations = vec![
            WireMutation::Absent,
            WireMutation::Empty,
            WireMutation::Negative,
            WireMutation::UndefinedEnum,
        ];
        mutations.extend(limits.iter().map(|limit| WireMutation::LimitPlusOne { limit: *limit }));
        for mutation in mutations {
            let applicable = match mutation {
                WireMutation::Absent => true,
                WireMutation::Empty => matches!(
                    field.kind,
                    WireKind::String
                        | WireKind::Bytes
                        | WireKind::Message
                        | WireKind::Repeated
                        | WireKind::Map
                ),
                WireMutation::LimitPlusOne { .. } => matches!(
                    field.kind,
                    WireKind::String
                        | WireKind::Bytes
                        | WireKind::Signed
                        | WireKind::Unsigned
                        | WireKind::Float
                        | WireKind::Repeated
                        | WireKind::Map
                ),
                WireMutation::Negative => matches!(field.kind, WireKind::Signed | WireKind::Float),
                WireMutation::UndefinedEnum => matches!(field.kind, WireKind::Enum { .. }),
            };
            let applicability = if let Some(reason) = &field.refusal {
                MutationApplicability::Refused(reason.clone())
            } else if applicable {
                MutationApplicability::Applicable
            } else {
                MutationApplicability::NotApplicable(format!(
                    "{} is not applicable to {:?}",
                    mutation.name(),
                    field.kind
                ))
            };
            cases.push(WireCase { field: field.clone(), mutation, applicability });
        }
    }
    cases.sort_by_key(WireCase::id);
    cases
}

/// Applies generic scalar mutations without reading semantic catalog kinds.
pub trait WireValue {
    /// Mutates atomically or returns a value-free refusal.
    fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError>;
}

fn length(limit: u64) -> Result<usize, GenerationError> {
    let length = usize::try_from(limit.checked_add(1).ok_or(GenerationError::SizeLimit)?)
        .map_err(|_| GenerationError::SizeLimit)?;
    if length > 16 * 1024 * 1024 {
        return Err(GenerationError::SizeLimit);
    }
    Ok(length)
}

impl WireValue for String {
    fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> {
        if !matches!(kind, WireKind::String) {
            return Err(GenerationError::InapplicableMutation);
        }
        match mutation {
            WireMutation::Absent | WireMutation::Empty => self.clear(),
            WireMutation::LimitPlusOne { limit } => {
                let length = length(*limit)?;
                let mut candidate = String::new();
                candidate.try_reserve_exact(length).map_err(|_| GenerationError::SizeLimit)?;
                candidate.extend(std::iter::repeat_n('x', length));
                *self = candidate;
            }
            _ => return Err(GenerationError::InapplicableMutation),
        }
        Ok(())
    }
}

impl WireValue for Vec<u8> {
    fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> {
        if !matches!(kind, WireKind::Bytes) {
            return Err(GenerationError::InapplicableMutation);
        }
        match mutation {
            WireMutation::Absent | WireMutation::Empty => self.clear(),
            WireMutation::LimitPlusOne { limit } => {
                let length = length(*limit)?;
                let mut candidate = Vec::new();
                candidate.try_reserve_exact(length).map_err(|_| GenerationError::SizeLimit)?;
                candidate.resize(length, b'x');
                *self = candidate;
            }
            _ => return Err(GenerationError::InapplicableMutation),
        }
        Ok(())
    }
}

impl WireValue for bool {
    fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> {
        if matches!((kind, mutation), (WireKind::Bool, WireMutation::Absent)) {
            *self = false;
            Ok(())
        } else {
            Err(GenerationError::InapplicableMutation)
        }
    }
}

macro_rules! integers {
    ($($ty:ty => $kind:ident),*) => {$ (
        impl WireValue for $ty {
            fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> {
                if !matches!(kind, WireKind::$kind) && !(stringify!($ty) == "i32" && matches!(kind, WireKind::Enum { .. })) { return Err(GenerationError::InapplicableMutation); }
                let candidate = match mutation {
                    WireMutation::Absent => 0,
                    WireMutation::Negative if matches!(kind, WireKind::Signed) => <$ty>::try_from(-1_i32).map_err(|_| GenerationError::InapplicableMutation)?,
                    WireMutation::LimitPlusOne { limit } if !matches!(kind, WireKind::Enum { .. }) => <$ty>::try_from(limit.checked_add(1).ok_or(GenerationError::SizeLimit)?).map_err(|_| GenerationError::SizeLimit)?,
                    WireMutation::UndefinedEnum => {
                        let WireKind::Enum { defined } = kind else { return Err(GenerationError::InapplicableMutation); };
                        let mut candidate = i32::MAX;
                        while defined.contains(&candidate) { candidate = candidate.checked_sub(1).ok_or(GenerationError::SizeLimit)?; }
                        <$ty>::try_from(candidate).map_err(|_| GenerationError::SizeLimit)?
                    }
                    _ => return Err(GenerationError::InapplicableMutation),
                };
                *self = candidate;
                Ok(())
            }
        }
    )*};
}
integers!(i32 => Signed, i64 => Signed, u32 => Unsigned, u64 => Unsigned);

macro_rules! floats {
    ($($ty:ty),*) => {$ (
        impl WireValue for $ty {
            fn mutate(&mut self, kind: &WireKind, mutation: &WireMutation) -> Result<(), GenerationError> {
                if !matches!(kind, WireKind::Float) { return Err(GenerationError::InapplicableMutation); }
                let candidate = match mutation {
                    WireMutation::Absent => 0.0,
                    WireMutation::Negative => -1.0,
                    WireMutation::LimitPlusOne { limit } => limit.checked_add(1).ok_or(GenerationError::SizeLimit)?.to_string().parse().map_err(|_| GenerationError::SizeLimit)?,
                    _ => return Err(GenerationError::InapplicableMutation),
                };
                *self = candidate;
                Ok(())
            }
        }
    )*};
}
floats!(f32, f64);

/// Classifies one generic case using any matching rejection explanation.
/// All supplied predictions are admitted; strict rule probes retain their individual semantics.
pub fn mutation_entries(
    case: &WireCase,
    predictions: &[(Violation, ProbeResult)],
    otherwise: ProbeResult,
) -> Result<Vec<GapEntry>, GapError> {
    let finding = || Violation {
        path: case.field.path.clone(),
        constraint: mutation_constraint(&case.mutation),
        fields: Vec::new(),
        expected: None,
    };
    if !matches!(case.applicability, MutationApplicability::Applicable) {
        if !predictions.is_empty() || !matches!(otherwise, ProbeResult::Unverifiable) {
            return Err(GapError);
        }
        let mut entry = GapEntry::new(finding(), ProbeResult::Unverifiable);
        entry.description = match &case.applicability {
            MutationApplicability::NotApplicable(reason)
            | MutationApplicability::Refused(reason) => reason.clone(),
            MutationApplicability::Unchanged => "mutation produced the baseline request".into(),
            MutationApplicability::Applicable => String::new(),
        };
        return Ok(vec![entry]);
    }
    if matches!(otherwise, ProbeResult::Rejected { code: 0, .. }) {
        return Err(GapError);
    }
    let entries: Vec<_> = predictions
        .iter()
        .map(|(finding, actual)| {
            let consistent = match (actual, &otherwise) {
                (ProbeResult::Accepted, ProbeResult::Accepted)
                | (ProbeResult::Unverifiable, ProbeResult::Unverifiable) => true,
                (ProbeResult::Rejected { code: a, .. }, ProbeResult::Rejected { code: b, .. }) => {
                    a == b
                }
                _ => false,
            };
            if !consistent {
                return Err(GapError);
            }
            GapEntry::checked(finding.clone(), actual.clone())
        })
        .collect::<Result<_, _>>()?;
    let rejects = |entry: &&GapEntry| {
        entry.violation.expected.as_ref().is_some_and(|e| {
            matches!(
                e.expectation,
                Expectation::Rejects { .. }
                    | Expectation::RejectsCode(_)
                    | Expectation::RejectsAnyStatus
            )
        })
    };
    let accepts = |entry: &&GapEntry| {
        entry
            .violation
            .expected
            .as_ref()
            .is_some_and(|e| matches!(e.expectation, Expectation::Accepts))
    };
    let selected = match &otherwise {
        ProbeResult::Rejected { .. } => entries
            .iter()
            .find(|entry| rejects(entry) && entry.outcome == GapOutcome::Confirmed)
            .or_else(|| entries.iter().find(rejects))
            .or_else(|| entries.iter().find(accepts)),
        ProbeResult::Accepted => {
            entries.iter().find(rejects).or_else(|| entries.iter().find(accepts))
        }
        ProbeResult::Unverifiable => None,
    };
    if let Some(entry) = selected {
        return Ok(vec![entry.clone()]);
    }
    let mut entry = GapEntry::new(finding(), otherwise.clone());
    entry.outcome = match otherwise {
        ProbeResult::Rejected { .. } => GapOutcome::Unannotated(otherwise),
        ProbeResult::Accepted => GapOutcome::Confirmed,
        ProbeResult::Unverifiable => GapOutcome::Unverifiable,
    };
    Ok(vec![entry])
}

#[cfg(test)]
mod applicability_tests {
    use super::{MutationApplicability, probe_applicability};
    use umpire_runtime::annotations::ProbeApplicability;

    #[test]
    fn maps_all_variants_and_wire_values_stay_closed() {
        for (value, expected, refusal) in [
            (MutationApplicability::Applicable, ProbeApplicability::Applicable, ""),
            (
                MutationApplicability::NotApplicable("not applicable".into()),
                ProbeApplicability::NotApplicable,
                "not applicable",
            ),
            (
                MutationApplicability::Refused("refused".into()),
                ProbeApplicability::Refused,
                "refused",
            ),
            (
                MutationApplicability::Unchanged,
                ProbeApplicability::Unchanged,
                "mutation leaves the baseline unchanged",
            ),
        ] {
            let (actual, reason) = probe_applicability(&value);
            assert_eq!(actual, expected);
            assert_eq!(reason, refusal);
            assert_eq!(ProbeApplicability::try_from(actual as i32), Ok(expected));
        }
        for unknown in [-1, 4, i32::MAX] {
            assert!(ProbeApplicability::try_from(unknown).is_err());
        }
    }
}
