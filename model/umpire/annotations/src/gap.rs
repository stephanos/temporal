//! Value-free probe outcomes and stable gap rendering.

use crate::{Expectation, Violation};
use std::fmt;

/// Probe evidence excludes raw server messages, which may echo secret inputs.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ProbeResult {
    /// Request succeeded.
    Accepted,
    /// Request failed with a status and a probe-evaluated pattern match.
    Rejected {
        /// Actual protocol status code.
        code: i32,
        /// Whether the server message matched the expected pattern.
        message_matches: bool,
        /// Structured error detail type, when supplied.
        details_type: Option<String>,
    },
    /// No request can observe this constraint.
    Unverifiable,
}

/// Relationship between a catalog expectation and one server observation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GapOutcome {
    /// Actual behavior matches the declared expectation.
    Confirmed,
    /// The server rejected differently, or rejected an expected acceptance.
    Mismatch(ProbeResult),
    /// The server accepted a request expected to be rejected.
    Accepted,
    /// Output-only or explicitly unobservable.
    Unverifiable,
}

/// An explicit, dated acknowledgment of one mismatch.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AcceptedGap {
    /// ISO calendar date, `YYYY-MM-DD`.
    pub date: String,
    /// Explanation reviewed alongside the report.
    pub reason: String,
}

impl AcceptedGap {
    /// Whether the date is a real ISO calendar date and the reason is nonblank.
    pub fn is_valid(&self) -> bool {
        let bytes = self.date.as_bytes();
        let shape = bytes.len() == 10
            && bytes[4] == b'-'
            && bytes[7] == b'-'
            && bytes
                .iter()
                .enumerate()
                .all(|(i, byte)| matches!(i, 4 | 7) || byte.is_ascii_digit())
            && !self.reason.trim().is_empty();
        if !shape {
            return false;
        }
        let Ok(year) = self.date[..4].parse::<u16>() else {
            return false;
        };
        let Ok(month) = self.date[5..7].parse::<u8>() else {
            return false;
        };
        let Ok(day) = self.date[8..].parse::<u8>() else {
            return false;
        };
        let leap =
            year.is_multiple_of(4) && (!year.is_multiple_of(100) || year.is_multiple_of(400));
        let days = match month {
            1 | 3 | 5 | 7 | 8 | 10 | 12 => 31,
            4 | 6 | 9 | 11 => 30,
            2 if leap => 29,
            2 => 28,
            _ => 0,
        };
        year > 0 && day > 0 && day <= days
    }
}

/// One report row derived from an actual validation finding.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GapEntry {
    /// Validation identity and expected behavior.
    pub violation: Violation,
    /// Comparison with the probe result.
    pub outcome: GapOutcome,
    /// Explicit acceptance, if any.
    pub accepted_gap: Option<AcceptedGap>,
}

/// An observable expectation was reported without an observation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GapError;

impl fmt::Display for GapError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("observable expectation requires a probe result")
    }
}
impl std::error::Error for GapError {}

impl GapEntry {
    /// Classifies live evidence, refusing an unverifiable observable expectation.
    pub fn checked(violation: Violation, actual: ProbeResult) -> Result<Self, GapError> {
        if matches!(actual, ProbeResult::Unverifiable)
            && violation
                .expected
                .as_ref()
                .is_some_and(|expected| !matches!(expected.expectation, Expectation::Unobservable))
        {
            return Err(GapError);
        }
        Ok(Self::new(violation, actual))
    }
    /// Classifies a probe result against the validation finding's expectation.
    pub fn new(violation: Violation, actual: ProbeResult) -> Self {
        let expectation = violation.expected.as_ref().map(|expected| &expected.expectation);
        let outcome = match (expectation, &actual) {
            (None | Some(Expectation::Unobservable), _) | (_, ProbeResult::Unverifiable) => {
                GapOutcome::Unverifiable
            }
            (Some(Expectation::Accepts), ProbeResult::Accepted) => GapOutcome::Confirmed,
            (
                Some(Expectation::Rejects {
                    code: expected_code, details_type: expected_type, ..
                }),
                ProbeResult::Rejected { code, message_matches, details_type },
            ) if expected_code == code
                && *message_matches
                && expected_type
                    .as_ref()
                    .is_none_or(|expected| Some(expected) == details_type.as_ref()) =>
            {
                GapOutcome::Confirmed
            }
            (Some(Expectation::Rejects { .. }), ProbeResult::Accepted) => GapOutcome::Accepted,
            _ => GapOutcome::Mismatch(actual),
        };
        Self { violation, outcome, accepted_gap: None }
    }
    /// Whether this row is an unacknowledged server gap.
    pub fn fails(&self) -> bool {
        matches!(self.outcome, GapOutcome::Mismatch(_) | GapOutcome::Accepted)
            && !self.accepted_gap.as_ref().is_some_and(AcceptedGap::is_valid)
    }
}

/// Stable report suitable for per-feature golden files.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GapReport {
    /// Actual probe findings; rendered in canonical order.
    pub entries: Vec<GapEntry>,
    /// Server validators without catalog coverage, reported but not failed.
    pub uncovered_validators: Vec<String>,
}

impl GapReport {
    /// Fails on unacknowledged Accepted or Mismatch rows.
    pub fn fails(&self) -> bool {
        self.entries.iter().any(GapEntry::fails)
    }
}

impl fmt::Display for GapReport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let mut rows: Vec<_> = self.entries.iter().collect();
        rows.sort_by(|a, b| {
            (
                &a.violation.path,
                &a.violation.constraint,
                &a.violation.fields,
                format!("{:?}{:?}{:?}", a.outcome, a.violation.expected, a.accepted_gap),
            )
                .cmp(&(
                    &b.violation.path,
                    &b.violation.constraint,
                    &b.violation.fields,
                    format!("{:?}{:?}{:?}", b.outcome, b.violation.expected, b.accepted_gap),
                ))
        });
        for entry in rows {
            write!(f, "{}: ", entry.violation)?;
            match &entry.outcome {
                GapOutcome::Confirmed => write!(f, "Confirmed")?,
                GapOutcome::Accepted => write!(f, "Accepted")?,
                GapOutcome::Unverifiable => write!(f, "Unverifiable")?,
                GapOutcome::Mismatch(actual) => write!(
                    f,
                    "Mismatch {{ expected: {:?}, actual: {actual:?} }}",
                    entry.violation.expected.as_ref().map(|expected| &expected.expectation)
                )?,
            }
            if let Some(expected) = &entry.violation.expected {
                write!(f, " [covers: {}]", expected.covers)?;
            }
            if let Some(accepted) = &entry.accepted_gap {
                write!(f, " [accepted {}: {}]", accepted.date, accepted.reason)?;
            }
            writeln!(f)?;
        }
        let mut uncovered = self.uncovered_validators.clone();
        uncovered.sort();
        uncovered.dedup();
        for validator in uncovered {
            writeln!(f, "Uncovered: {validator}")?;
        }
        Ok(())
    }
}
