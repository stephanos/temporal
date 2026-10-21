use std::collections::{BTreeMap, BTreeSet};
use umpire_annotations::{AcceptedGap, GapEntry};

pub(super) struct Dispositions(BTreeMap<String, AcceptedGap>);

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
            if registry.insert(id, gap).is_some() {
                return Err("duplicate probe disposition identity");
            }
        }
        Ok(Self(registry))
    }

    pub(super) fn apply(&self, case_id: &str, row: &mut GapEntry) {
        row.accepted_gap = self.0.get(case_id).cloned();
    }
}
