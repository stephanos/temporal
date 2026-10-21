//! Catalog-derived negative witnesses and complete probe coverage.

use super::{Catalog, Expectation, Presence, Violation, validate};
use crate::{GenerationError, Rng};
use std::{collections::BTreeSet, fmt};

/// Stable identity of a catalog expectation and its negative witness variant.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProbeCase {
    /// Author-written message and field documentation.
    pub description: String,
    /// The presence expectation shares a field's rejection pattern.
    pub inherited_pattern: bool,
    /// Stable value-free identity, independent of declaration ordering.
    pub id: String,
    /// Declared finding that the generated request must actually exhibit.
    pub violation: Violation,
    /// Named field attributed by a required or any-of probe.
    pub omitted_field: Option<String>,
}

/// A typed candidate checked against its catalog before transport.
#[derive(Clone)]
pub struct Probe<M> {
    /// Candidate message; callers must not include it in reports.
    pub message: M,
    /// Actual validation finding from the candidate.
    pub violation: Violation,
}

/// Missing, unexpected, or repeated case identities in a probe run.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ProbeCoverageError {
    /// Expected identities without an observation.
    pub missing: Vec<String>,
    /// Observed identities absent from the catalog.
    pub unexpected: Vec<String>,
    /// Repeated expected or observed identities.
    pub duplicated: Vec<String>,
}

impl fmt::Display for ProbeCoverageError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{self:?}")
    }
}
impl std::error::Error for ProbeCoverageError {}

/// Requires exactly one observation for every stable expected case identity.
pub fn verify_probe_coverage(
    cases: &[ProbeCase],
    observed: &[String],
) -> Result<(), ProbeCoverageError> {
    let mut duplicated = BTreeSet::new();
    let mut expected = BTreeSet::new();
    for case in cases {
        if !expected.insert(case.id.clone()) {
            duplicated.insert(case.id.clone());
        }
    }
    let mut actual = BTreeSet::new();
    for id in observed {
        if !actual.insert(id.clone()) {
            duplicated.insert(id.clone());
        }
    }
    let error = ProbeCoverageError {
        missing: expected.difference(&actual).cloned().collect(),
        unexpected: actual.difference(&expected).cloned().collect(),
        duplicated: duplicated.into_iter().collect(),
    };
    if error.missing.is_empty() && error.unexpected.is_empty() && error.duplicated.is_empty() {
        Ok(())
    } else {
        Err(error)
    }
}

/// Value-free refusal to construct a negative witness.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum ProbeError {
    /// The caller supplied an invalid baseline.
    InvalidBaseline(Vec<Violation>),
    /// The case is absent from this catalog or its metadata was changed.
    UnknownCase,
    /// Explicitly unobservable expectations cannot yield request probes.
    Unobservable,
    /// A typed mutation could not be constructed.
    Generation(GenerationError),
    /// The accessor did not produce exactly the declared violation.
    InvalidWitness(Vec<Violation>),
}

impl fmt::Display for ProbeError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{self:?}")
    }
}
impl std::error::Error for ProbeError {}

impl<M: 'static> Catalog<M> {
    /// Enumerates each declared expectation, including each required-field omission.
    pub fn probe_cases(&self) -> Vec<ProbeCase> {
        let mut cases = Vec::new();
        for field in self.fields() {
            for (constraint, expected) in &field.expectations {
                cases.push(ProbeCase {
                    description: [self.description.as_str(), field.description.as_str()]
                        .into_iter()
                        .filter(|s| !s.is_empty())
                        .collect::<Vec<_>>()
                        .join("; "),
                    inherited_pattern: false,
                    id: format!("{}:{constraint}", field.path),
                    violation: Violation {
                        path: field.path.clone(),
                        constraint: constraint.clone(),
                        fields: Vec::new(),
                        expected: Some(expected.clone()),
                    },
                    omitted_field: None,
                });
            }
        }
        for rule in &self.rules {
            let Some(expected) = &rule.expected else { continue };
            let fields = self.rule_fields(rule);
            let violation = Violation {
                path: self.message.clone(),
                constraint: rule.presence.name().into(),
                fields: fields.clone(),
                expected: Some(expected.clone()),
            };
            let id = format!("{}:{}({})", self.message, rule.presence.name(), fields.join(","));
            if matches!(rule.presence, Presence::Required(_) | Presence::AnyOf(_)) {
                for field in fields {
                    cases.push(ProbeCase {
                        description: self.description.clone(),
                        inherited_pattern: self.inherited_patterns.contains(&rule.presence),
                        id: if matches!(rule.presence, Presence::AnyOf(_)) {
                            format!("{id}:all_absent:missing={field}")
                        } else {
                            format!("{id}:omit={field}")
                        },
                        violation: violation.clone(),
                        omitted_field: Some(field),
                    });
                }
            } else {
                cases.push(ProbeCase {
                    id,
                    violation,
                    omitted_field: None,
                    description: self.description.clone(),
                    inherited_pattern: self.inherited_patterns.contains(&rule.presence),
                });
            }
        }
        cases.sort_by(|a, b| a.id.cmp(&b.id));
        cases
    }
}

impl<M: Clone + 'static> Catalog<M> {
    /// Mutates one constraint from a valid caller baseline, then verifies the finding.
    pub fn probe(
        &self,
        baseline: &M,
        case: &ProbeCase,
        rng: &mut Rng,
    ) -> Result<Probe<M>, ProbeError> {
        let mut stream = rng.clone();
        let probe = self.probe_with(baseline, case, |candidate| {
            if case.violation.fields.is_empty() {
                self.fields[&case.violation.path].negative(
                    candidate,
                    &case.violation.constraint,
                    &mut stream,
                )
            } else {
                let rule = self
                    .rules
                    .iter()
                    .find(|rule| {
                        rule.presence.name() == case.violation.constraint
                            && self.rule_fields(rule) == case.violation.fields
                    })
                    .ok_or(GenerationError::UnsupportedType)?;
                let absent: Vec<String> = match &rule.presence {
                    Presence::Required(_) => {
                        vec![case.omitted_field.clone().ok_or(GenerationError::UnsupportedType)?]
                    }
                    Presence::AnyOf(_) | Presence::OneOf(_) => case.violation.fields.clone(),
                    Presence::Requires(a, b) => {
                        self.fields[&self.qualified(a)].fill(candidate, &mut stream)?;
                        vec![self.qualified(b)]
                    }
                    Presence::Excludes(a, b) => {
                        self.fields[&self.qualified(a)].fill(candidate, &mut stream)?;
                        self.fields[&self.qualified(b)].fill(candidate, &mut stream)?;
                        Vec::new()
                    }
                };
                for field in absent {
                    self.fields[&field].clear(candidate)?;
                }
                Ok(())
            }
        })?;
        *rng = stream;
        Ok(probe)
    }

    /// Checks a caller's typed projection mutation using the same catalog contract.
    pub fn probe_with(
        &self,
        baseline: &M,
        case: &ProbeCase,
        mutate: impl FnOnce(&mut M) -> Result<(), GenerationError>,
    ) -> Result<Probe<M>, ProbeError> {
        if !self.probe_cases().contains(case) {
            return Err(ProbeError::UnknownCase);
        }
        if matches!(
            case.violation.expected.as_ref().map(|expected| &expected.expectation),
            Some(Expectation::Unobservable)
        ) {
            return Err(ProbeError::Unobservable);
        }
        let baseline_violations = validate(baseline, self);
        if !baseline_violations.is_empty() {
            return Err(ProbeError::InvalidBaseline(baseline_violations));
        }
        let mut candidate = baseline.clone();
        mutate(&mut candidate).map_err(ProbeError::Generation)?;
        let actual = validate(&candidate, self);
        if case.violation.constraint == "required"
            && let Some(omitted) = &case.omitted_field
            && (self.fields[omitted].present(&candidate)
                || case
                    .violation
                    .fields
                    .iter()
                    .any(|field| field != omitted && !self.fields[field].present(&candidate)))
        {
            return Err(ProbeError::InvalidWitness(actual));
        }
        let mut target = case.violation.clone();
        if target.constraint != "requires" {
            target.fields.sort();
        }
        let matches = actual.len() == 1 && {
            let mut finding = actual[0].clone();
            if finding.constraint != "requires" {
                finding.fields.sort();
            }
            finding == target
        };
        if !matches {
            return Err(ProbeError::InvalidWitness(actual));
        }
        Ok(Probe {
            message: candidate,
            violation: actual.into_iter().next().ok_or(ProbeError::UnknownCase)?,
        })
    }
}
