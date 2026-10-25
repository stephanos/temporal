//! Descriptor-driven probe orchestration with system-owned wire and catalog policy.

use crate::{
    Expectation, GapEntry, GapReport, MutationApplicability, ProbeResult, UNCHANGED_REFUSAL,
    Violation, WireCase, WireField, mutation_entries, probe_applicability, wire_cases,
};
use std::{collections::BTreeSet, fmt::Write, sync::OnceLock};
use umpire_runtime::annotations::{
    ProbeApplicability, ProbeCase, ProbeExpectation, ProbeObservation, ProbeResponse,
};

/// Probe admission or construction failure.
pub type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;
/// Maximum number of cases or observations in one inventory.
pub const MAX_CASES: usize = 32_768;
/// Maximum predictions attributed to one mutation.
pub const MAX_PREDICTIONS: usize = 256;

/// System policy and typed wire operations used by the shared engine.
pub trait ProbeSystem {
    /// Descriptor method identity.
    type Method: Copy;
    /// Current execution prerequisites, independent of canonical metadata.
    type Context;
    /// In-scope methods and their declared prerequisites.
    fn methods(&self) -> Vec<(Self::Method, &'static str)>;
    /// Bare descriptor method name.
    fn name(&self, method: Self::Method) -> &'static str;
    /// Fully qualified RPC path.
    fn rpc_path(&self, method: Self::Method) -> String;
    /// Descriptor fields, including ignored catalog fields.
    fn fields(&self, method: Self::Method) -> Vec<WireField>;
    /// Explicit externally chosen mutation bounds.
    fn bounds(&self) -> Vec<u64>;
    /// Valid slots used only to establish value-free canonical metadata.
    fn canonical_context(&self) -> Self::Context;
    /// Constructs a request using the current prerequisites.
    fn baseline(&self, method: Self::Method, context: &Self::Context) -> Result<Vec<u8>>;
    /// Applies one typed descriptor mutation.
    fn mutate(&self, method: Self::Method, case: &WireCase, bytes: &[u8]) -> Result<Vec<u8>>;
    /// Validates the actual encoded request against its catalog.
    fn findings(&self, method: Self::Method, bytes: &[u8]) -> Result<Vec<Violation>>;
    /// Resolves a finding against any system-owned strict claim registry.
    fn prediction(&self, _method: Self::Method, finding: Violation) -> Result<Violation> {
        Ok(finding)
    }
    /// Identity suffix for a resolved prediction, independent of request values.
    fn prediction_key(&self, finding: &Violation, index: usize) -> String {
        format!("{index}:{}:{}", finding.path, finding.constraint)
    }
    /// Refuses a mutation that would invalidate a stateful execution prerequisite.
    fn mutation_refusal(
        &self,
        _case: &WireCase,
        _prerequisite: &str,
        _findings: &[Violation],
    ) -> Option<String> {
        None
    }
    /// Whether an unpredicted mutation can change between Applicable and Unchanged with slots.
    fn contextual_unchanged(&self) -> bool {
        false
    }
    /// Whether a rejection status is conclusive evidence for this system.
    fn conclusive_status(&self, code: i32) -> bool {
        (1..=16).contains(&code)
    }
    /// Additional system-specific strict cases.
    fn additional_cases(&self) -> Result<Vec<ProbeCase>> {
        Ok(Vec::new())
    }
    /// Prepares a system-specific strict witness or an explicit baseline selector.
    fn prepare_additional(
        &self,
        _id: &str,
        _baseline: bool,
        _context: &Self::Context,
    ) -> Result<ProbeResponse> {
        Err("unknown probe case".into())
    }
    /// Admits source dispositions against the complete identity universe.
    fn validate_inventory(&self, _ids: &[String]) -> Result<()> {
        Ok(())
    }
    /// Classifies the system-specific strict observations after complete-set admission.
    fn classify_additional(&self, _observations: &[ProbeObservation]) -> Result<ProbeResponse> {
        Ok(ProbeResponse::default())
    }
    /// Applies an admitted source disposition to a generic finding.
    fn apply_disposition(&self, _id: &str, _row: &mut GapEntry) {}
    /// Completes system-specific reporting after generic classification.
    fn finish_report(&self, _response: &mut ProbeResponse) -> Result<()> {
        Ok(())
    }
}

/// One normalized probe operation; wire request admission stays with the adapter.
pub enum Request<'a, C> {
    /// Declare all strict and generic cases.
    Inventory,
    /// Materialize a baseline or mutation with current prerequisites.
    Prepare {
        /// Declared identity or system baseline selector.
        id: &'a str,
        /// Return the valid baseline instead of mutating it.
        baseline: bool,
        /// Current execution prerequisites.
        context: &'a C,
    },
    /// Classify exactly one actual observation per declared case.
    Classify(&'a [ProbeObservation]),
}

struct Entry<M> {
    method: M,
    prerequisite: &'static str,
    case: WireCase,
    canonical: OnceLock<Prepared>,
}
struct Prepared {
    metadata: ProbeCase,
    findings: Vec<Violation>,
}

/// One inventory, preparation and classification engine for a typed system adapter.
pub struct Engine<S: ProbeSystem> {
    system: S,
    entries: Vec<Entry<S::Method>>,
    additional: Vec<ProbeCase>,
}

impl<S: ProbeSystem> Engine<S> {
    /// Admits identities without materializing canonical requests or caching request bytes.
    ///
    /// # Errors
    /// Refuses duplicate identities, oversized inventories or invalid source dispositions.
    pub fn new(system: S) -> Result<Self> {
        let bounds = system.bounds();
        let mut entries = Vec::new();
        for (method, prerequisite) in system.methods() {
            for case in wire_cases(&system.fields(method), &bounds) {
                entries.push(Entry { method, prerequisite, case, canonical: OnceLock::new() });
                if entries.len() > MAX_CASES {
                    return Err("generic probe inventory exceeds metadata budget".into());
                }
            }
        }
        entries.sort_by_key(|entry| entry.id(&system));
        let additional = system.additional_cases()?;
        let ids: Vec<_> = additional
            .iter()
            .map(|case| case.id.clone())
            .chain(entries.iter().map(|entry| entry.id(&system)))
            .collect();
        if ids.len() > MAX_CASES || ids.iter().any(|id| id.len() > 4096) {
            return Err("probe inventory exceeds metadata budget".into());
        }
        if ids.iter().collect::<BTreeSet<_>>().len() != ids.len() {
            return Err("duplicate probe identity".into());
        }
        system.validate_inventory(&ids)?;
        Ok(Self { system, entries, additional })
    }

    /// Dispatches inventory, preparation or complete actual-observation classification.
    ///
    /// # Errors
    /// Refuses invalid baselines, unknown cases, context drift and fabricated or incomplete evidence.
    pub fn run(&self, request: &Request<'_, S::Context>) -> Result<ProbeResponse> {
        match *request {
            Request::Inventory => {
                let mut cases = self.additional.clone();
                for entry in &self.entries {
                    let canonical = self.canonical(entry)?;
                    let mut metadata = canonical.metadata.clone();
                    if self.system.contextual_unchanged()
                        && metadata.applicability == ProbeApplicability::Unchanged as i32
                    {
                        metadata = self.metadata(entry)?;
                    }
                    cases.push(metadata);
                }
                Ok(ProbeResponse { cases, ..Default::default() })
            }
            Request::Prepare { id, baseline, context } => {
                let Some(entry) = self.entries.iter().find(|entry| entry.id(&self.system) == id)
                else {
                    return self.system.prepare_additional(id, baseline, context);
                };
                let (prepared, bytes) = self.materialize(entry, context, baseline)?;
                if !baseline {
                    let canonical = self.canonical(entry)?;
                    if !self.compatible_state(
                        prepared.metadata.applicability,
                        &prepared.metadata.refusal,
                        &canonical.metadata,
                    ) || prepared.metadata.predictions != canonical.metadata.predictions
                    {
                        return Err(
                            "generic prediction context diverges from canonical preparation".into(),
                        );
                    }
                }
                let metadata = prepared.metadata;
                let mut response = ProbeResponse {
                    method: metadata.method.clone(),
                    cases: vec![metadata],
                    ..Default::default()
                };
                if bytes.len() > 512 * 1024 {
                    response.recipe = Some(umpire_runtime::probe_recipe(&bytes)?);
                } else {
                    response.message = bytes;
                }
                Ok(response)
            }
            Request::Classify(observations) => self.classify(observations),
        }
    }

    fn metadata(&self, entry: &Entry<S::Method>) -> Result<ProbeCase> {
        let (applicability, mut refusal) = probe_applicability(&entry.case.applicability);
        if matches!(entry.case.applicability, MutationApplicability::NotApplicable(_)) {
            refusal =
                format!("{} is not applicable to this wire carrier", entry.case.mutation.name());
        }
        let metadata = ProbeCase {
            id: entry.id(&self.system),
            method: self.system.rpc_path(entry.method),
            prerequisite: entry.prerequisite.into(),
            generic: true,
            applicability: applicability as i32,
            refusal,
            ..Default::default()
        };
        if metadata.id.len() > 4096 || metadata.refusal.len() > 256 {
            return Err("generic probe identity exceeds metadata budget".into());
        }
        Ok(metadata)
    }

    fn canonical<'a>(&self, entry: &'a Entry<S::Method>) -> Result<&'a Prepared> {
        if let Some(value) = entry.canonical.get() {
            return Ok(value);
        }
        let (prepared, _) = self.materialize(entry, &self.system.canonical_context(), false)?;
        Ok(entry.canonical.get_or_init(|| prepared))
    }

    fn materialize(
        &self,
        entry: &Entry<S::Method>,
        context: &S::Context,
        baseline: bool,
    ) -> Result<(Prepared, Vec<u8>)> {
        let mut prepared = Prepared { metadata: self.metadata(entry)?, findings: Vec::new() };
        if !matches!(entry.case.applicability, MutationApplicability::Applicable) {
            return Ok((prepared, Vec::new()));
        }
        let original = self.system.baseline(entry.method, context)?;
        if !self.system.findings(entry.method, &original)?.is_empty() {
            return Err("invalid generic probe baseline".into());
        }
        if baseline {
            return Ok((prepared, original));
        }
        let Ok(bytes) = self.system.mutate(entry.method, &entry.case, &original) else {
            prepared.metadata.applicability = ProbeApplicability::Refused as i32;
            prepared.metadata.refusal = "typed mutation construction refused".into();
            return Ok((prepared, Vec::new()));
        };
        if bytes == original {
            prepared.metadata.applicability = ProbeApplicability::Unchanged as i32;
            prepared.metadata.refusal = UNCHANGED_REFUSAL.into();
            return Ok((prepared, Vec::new()));
        }
        let mut findings = self.system.findings(entry.method, &bytes)?;
        findings.sort_by_key(|finding| {
            (finding.path.clone(), finding.constraint.clone(), finding.fields.clone())
        });
        findings.dedup();
        findings.retain(|finding| {
            finding
                .expected
                .as_ref()
                .is_some_and(|expected| !matches!(expected.expectation, Expectation::Unobservable))
        });
        if let Some(refusal) =
            self.system.mutation_refusal(&entry.case, entry.prerequisite, &findings)
        {
            prepared.metadata.applicability = ProbeApplicability::Refused as i32;
            prepared.metadata.refusal = refusal;
            return Ok((prepared, Vec::new()));
        }
        for (index, finding) in findings.into_iter().enumerate() {
            let finding = self.system.prediction(entry.method, finding)?;
            let id = format!(
                "{}::prediction:{}",
                entry.id(&self.system),
                self.system.prediction_key(&finding, index)
            );
            prepared.metadata.predictions.push(expectation(
                id,
                &finding.expected.as_ref().ok_or("missing prediction expectation")?.expectation,
            )?);
            prepared.findings.push(finding);
        }
        let predictions = &prepared.metadata.predictions;
        if predictions.len() > MAX_PREDICTIONS
            || predictions.iter().any(|prediction| prediction.id.len() > 4096)
            || predictions.iter().map(|prediction| &prediction.id).collect::<BTreeSet<_>>().len()
                != predictions.len()
            || prepared.metadata.refusal.len() > 256
        {
            return Err("generic predictions exceed metadata budget".into());
        }
        Ok((prepared, bytes))
    }

    fn compatible_state(&self, actual: i32, refusal: &str, expected: &ProbeCase) -> bool {
        (actual == expected.applicability && refusal == expected.refusal)
            || (self.system.contextual_unchanged()
                && expected.predictions.is_empty()
                && matches!(
                    ProbeApplicability::try_from(expected.applicability),
                    Ok(ProbeApplicability::Applicable | ProbeApplicability::Unchanged)
                )
                && ((actual == ProbeApplicability::Applicable as i32 && refusal.is_empty())
                    || (actual == ProbeApplicability::Unchanged as i32
                        && refusal == UNCHANGED_REFUSAL)))
    }

    fn classify(&self, observations: &[ProbeObservation]) -> Result<ProbeResponse> {
        let expected: BTreeSet<_> = self
            .additional
            .iter()
            .map(|case| case.id.clone())
            .chain(self.entries.iter().map(|entry| entry.id(&self.system)))
            .collect();
        let actual: BTreeSet<_> = observations.iter().map(|value| value.case_id.as_str()).collect();
        if observations.len() > MAX_CASES
            || actual.len() != observations.len()
            || expected.len() != actual.len()
            || !expected.iter().all(|id| actual.contains(id.as_str()))
        {
            return Err("incomplete, duplicate or unknown probe observations".into());
        }
        let mut response = self.system.classify_additional(observations)?;
        let mut tested = 0;
        for entry in &self.entries {
            let id = entry.id(&self.system);
            let observation = observations
                .iter()
                .find(|value| value.case_id == id)
                .ok_or("missing generic observation")?;
            let mut rows = self.classify_entry(entry, observation)?;
            for row in &mut rows {
                self.system.apply_disposition(&id, row);
                response.fails |= row.fails();
            }
            tested +=
                usize::from(observation.applicability == ProbeApplicability::Applicable as i32);
            writeln!(response.report, "Generic case: {id}")?;
            for prediction in &observation.predictions {
                writeln!(response.report, "Prediction: {}", prediction.prediction_id)?;
            }
            write!(response.report, "{}", GapReport { entries: rows })?;
        }
        writeln!(
            response.report,
            "Generic summary: tested={tested} local_not_tested={}",
            self.entries.len() - tested
        )?;
        self.system.finish_report(&mut response)?;
        Ok(response)
    }

    fn classify_entry(
        &self,
        entry: &Entry<S::Method>,
        observation: &ProbeObservation,
    ) -> Result<Vec<GapEntry>> {
        let state = ProbeApplicability::try_from(observation.applicability)?;
        let expected = self.canonical(entry)?;
        if observation.predictions.len() > MAX_PREDICTIONS
            || observation.predictions.len() != expected.metadata.predictions.len()
            || observation.prediction_count as usize != observation.predictions.len()
            || observation.refusal.len() > 256
            || observation.message_matches
            || !observation.details_type.is_empty()
        {
            return Err("invalid generic observation metadata".into());
        }
        if !self.compatible_state(
            observation.applicability,
            &observation.refusal,
            &expected.metadata,
        ) {
            return Err("generic observation diverges from canonical preparation".into());
        }
        let mut case = entry.case.clone();
        if state != ProbeApplicability::Applicable {
            if observation.accepted || observation.code != 0 || !observation.predictions.is_empty()
            {
                return Err("invalid local mutation observation".into());
            }
            case.applicability = match state {
                ProbeApplicability::NotApplicable => {
                    MutationApplicability::NotApplicable(observation.refusal.clone())
                }
                ProbeApplicability::Refused => {
                    MutationApplicability::Refused(observation.refusal.clone())
                }
                ProbeApplicability::Unchanged => MutationApplicability::Unchanged,
                ProbeApplicability::Applicable => unreachable!(),
            };
            return Ok(mutation_entries(&case, &[], ProbeResult::Unverifiable)?);
        }
        if !matches!(case.applicability, MutationApplicability::Applicable)
            || !observation.refusal.is_empty()
            || (observation.accepted && observation.code != 0)
            || (!observation.accepted && !self.system.conclusive_status(observation.code))
        {
            return Err("invalid dispatched mutation observation".into());
        }
        let mut indices = BTreeSet::new();
        let mut ids = BTreeSet::new();
        let mut predictions = Vec::new();
        for prediction in &observation.predictions {
            let index = prediction.index as usize;
            if expected
                .metadata
                .predictions
                .get(index)
                .is_none_or(|declared| declared.id != prediction.prediction_id)
                || !indices.insert(index)
                || prediction.prediction_id.len() > 4096
                || prediction.details_type.len() > 512
                || !ids.insert(&prediction.prediction_id)
            {
                return Err("invalid generic prediction correlation".into());
            }
            predictions.push((
                expected.findings[index].clone(),
                result(
                    observation.accepted,
                    observation.code,
                    prediction.message_matches,
                    &prediction.details_type,
                ),
            ));
        }
        Ok(mutation_entries(
            &case,
            &predictions,
            result(observation.accepted, observation.code, false, ""),
        )?)
    }
}

impl<M: Copy> Entry<M> {
    fn id<S: ProbeSystem<Method = M>>(&self, system: &S) -> String {
        format!("generic::{}::{}", system.name(self.method), self.case.id())
    }
}

/// Maps a declared expectation to shared probe metadata.
///
/// # Errors
/// Unobservable claims cannot become request predictions.
pub fn expectation(id: String, declared: &Expectation) -> Result<ProbeExpectation> {
    let mut expected = ProbeExpectation { id, ..Default::default() };
    match declared {
        Expectation::Accepts => expected.accepts = true,
        Expectation::RejectsCode(code) => expected.code = *code as i32,
        Expectation::RejectsAnyStatus => {}
        Expectation::Rejects { code, message_pattern, details_type } => {
            expected.code = *code;
            expected.message_pattern.clone_from(message_pattern);
            expected.details_type = details_type.clone().unwrap_or_default();
        }
        Expectation::Unobservable => {
            return Err("observable request case declared unobservable".into());
        }
    }
    Ok(expected)
}

/// Converts an actual strict observation into catalog evidence.
pub fn observation_result(observation: &ProbeObservation) -> ProbeResult {
    result(
        observation.accepted,
        observation.code,
        observation.message_matches,
        &observation.details_type,
    )
}
fn result(accepted: bool, code: i32, message_matches: bool, details_type: &str) -> ProbeResult {
    if accepted {
        ProbeResult::Accepted
    } else {
        ProbeResult::Rejected {
            code,
            message_matches,
            details_type: (!details_type.is_empty()).then(|| details_type.into()),
        }
    }
}
