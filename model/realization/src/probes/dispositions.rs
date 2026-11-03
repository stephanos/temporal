use std::collections::{BTreeMap, BTreeSet};
use umpire_catalog::{AcceptedGap, GapEntry, GapOutcome, ProbeResult};

#[path = "../../../feature/activity/probe_gaps.rs"]
mod known;

pub(super) fn unannotated() -> Vec<(String, AcceptedGap)> {
    known::unannotated()
}

pub(super) struct Dispositions(BTreeMap<String, (AcceptedGap, bool)>);

impl Dispositions {
    pub(super) fn new<'a>(
        case_ids: impl IntoIterator<Item = &'a str>,
        gaps: Vec<(String, AcceptedGap)>,
    ) -> Result<Self, &'static str> {
        let case_ids: BTreeSet<_> = case_ids.into_iter().collect();
        let mut registry = BTreeMap::new();
        for (id, gap) in gaps {
            if !gap.is_valid() {
                return Err("invalid probe disposition date or reason");
            }
            if !case_ids.contains(id.as_str()) {
                return Err("unknown probe disposition identity");
            }
            if registry.insert(id, (gap, false)).is_some() {
                return Err("duplicate probe disposition identity");
            }
        }
        Ok(Self(registry))
    }

    pub(super) fn with_unannotated<'a>(
        case_ids: impl IntoIterator<Item = &'a str>,
        general: Vec<(String, AcceptedGap)>,
        unannotated: Vec<(String, AcceptedGap)>,
    ) -> Result<Self, &'static str> {
        let case_ids: Vec<_> = case_ids.into_iter().collect();
        let mut registry = Self::new(case_ids.iter().copied(), general)?;
        let known = Self::new(case_ids.iter().copied(), unannotated)?;
        for (id, (gap, _)) in known.0 {
            if registry.0.insert(id, (gap, true)).is_some() {
                return Err("duplicate probe disposition identity");
            }
        }
        Ok(registry)
    }

    pub(super) fn apply(&self, case_id: &str, row: &mut GapEntry) {
        row.accepted_gap = self.0.get(case_id).and_then(|(gap, unannotated_only)| {
            (!unannotated_only
                || matches!(
                    row.outcome,
                    GapOutcome::Unannotated(ProbeResult::Rejected { code: 3, .. })
                ))
            .then(|| gap.clone())
        });
    }
}
