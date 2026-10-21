use super::ledger::Execution;
use std::collections::BTreeSet;

#[derive(Default)]
pub(super) struct Retention {
    pub count: u64,
    pub bytes: u64,
    pub choices: BTreeSet<String>,
    pub probes: BTreeSet<String>,
}

impl Retention {
    pub fn retained(&mut self, execution: &Execution) -> Result<(), String> {
        if execution.record.outcome.class == "success" && execution.artifact.is_some() {
            self.count = self.count.checked_add(1).ok_or("success count overflow")?;
            self.bytes = self
                .bytes
                .checked_add(execution.stored_bytes)
                .ok_or("success bytes overflow")?;
        }
        self.choices
            .extend(execution.choice_features.iter().cloned());
        self.probes
            .extend(execution.semantic_probes.iter().cloned());
        Ok(())
    }

    pub fn keep(&self, policy: &str, execution: &Execution) -> bool {
        policy == "all"
            || policy == "novel"
                && (execution
                    .choice_features
                    .iter()
                    .any(|feature| !self.choices.contains(feature))
                    || execution
                        .semantic_probes
                        .iter()
                        .any(|probe| !self.probes.contains(probe)))
    }

    pub fn remaining(&self, count: u64, bytes: u64) -> Result<u64, String> {
        if self.count >= count || self.bytes >= bytes {
            return Err("capacity: successful-execution retention capacity is exhausted".into());
        }
        Ok(bytes - self.bytes)
    }
}
