use crate::evidence::{
    inspection::{self, Details},
    store::Opened,
    symbolize::{Site, Symbol, Symbols},
};
use gomad_wasmhost::linked;
use serde::Serialize;
use serde_json::Value;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Serialize)]
struct Symbolized<'a> {
    #[serde(flatten)]
    value: &'a inspection::Decision,
    symbol: &'a Symbol,
}

#[derive(Serialize)]
struct Reduction<'a> {
    kind: &'a str,
    before_sha256: &'a str,
    after_sha256: &'a str,
    removed: Vec<Symbolized<'a>>,
}

#[derive(Serialize)]
pub(crate) struct Report<'a> {
    #[serde(flatten)]
    fields: BTreeMap<String, Value>,
    accepted: Vec<Reduction<'a>>,
    final_forced_by_symbol: Vec<Value>,
}

pub(crate) struct Projection {
    decisions: BTreeMap<(u64, String), Site>,
    final_sites: Vec<Site>,
    unmapped_final: u64,
    requested: BTreeSet<Site>,
}

fn key(decision: &linked::Decision) -> (u64, String) {
    (
        decision.ordinal(),
        crate::campaign::exploration::choice::hex(&decision.selected_identity()),
    )
}

fn site(decision: &linked::Decision) -> Site {
    let (value, missing) = decision.site();
    Site { value, missing }
}

impl Projection {
    pub(crate) fn read(
        opened: &Opened,
        metadata: &inspection::Minimization,
    ) -> Result<Self, String> {
        let retained = super::source::read(opened)?;
        let details = Details::read(opened)?.ok_or("minimization inspection missing")?;
        let choices = details.choices.as_ref();
        let has_source = retained.is_some();
        let (raw, identity) = if let Some(source) = retained {
            (source.raw, Some(source.identity))
        } else if let Some(choices) = choices {
            choices.inspect(opened)?;
            (
                opened.payload(&choices.trace.file, choices.trace.limit)?,
                Some(choices.identity.clone()),
            )
        } else {
            (Vec::new(), None)
        };
        let decisions = identity
            .as_ref()
            .map(|identity| linked::project_choices(&raw, identity))
            .transpose()?
            .map(|plan| plan.decisions)
            .unwrap_or_default();
        let removed = metadata
            .accepted
            .iter()
            .flat_map(|reduction| &reduction.removed)
            .filter(|decision| decision.dimension == "runtime")
            .map(|decision| (decision.ordinal, decision.identity.clone()))
            .collect::<BTreeSet<_>>();
        let final_sites = if has_source {
            decisions
                .iter()
                .filter(|decision| !removed.contains(&key(decision)))
                .map(site)
                .collect::<Vec<_>>()
        } else {
            decisions
                .iter()
                .take(
                    usize::try_from(metadata.final_forced_decisions)
                        .map_err(|_| "minimization count exceeds host capacity")?,
                )
                .map(site)
                .collect::<Vec<_>>()
        };
        if has_source && final_sites.len() as u64 != metadata.final_forced_decisions {
            return Err("minimization source removals differ from final forced count".into());
        }
        let unmapped_final = metadata.final_forced_decisions - final_sites.len() as u64;
        let decisions = decisions
            .iter()
            .map(|decision| (key(decision), site(decision)))
            .collect::<BTreeMap<_, _>>();
        let requested = decisions.values().chain(&final_sites).copied().collect();
        Ok(Self {
            decisions,
            final_sites,
            unmapped_final,
            requested,
        })
    }

    pub(crate) fn sites(&self) -> impl Iterator<Item = Site> + '_ {
        self.requested.iter().copied()
    }

    pub(crate) fn report<'a>(
        &self,
        metadata: &'a inspection::Minimization,
        symbols: &'a Symbols,
    ) -> Result<Report<'a>, String> {
        let Value::Object(mut fields) =
            serde_json::to_value(metadata).map_err(|error| error.to_string())?
        else {
            return Err("minimization report is not an object".into());
        };
        fields.remove("accepted");
        let accepted = metadata
            .accepted
            .iter()
            .map(|reduction| Reduction {
                kind: &reduction.kind,
                before_sha256: &reduction.before_sha256,
                after_sha256: &reduction.after_sha256,
                removed: reduction
                    .removed
                    .iter()
                    .map(|decision| {
                        let joined = (decision.dimension == "runtime")
                            .then(|| {
                                self.decisions
                                    .get(&(decision.ordinal, decision.identity.clone()))
                            })
                            .flatten();
                        Symbolized {
                            value: decision,
                            symbol: symbols.get(joined.copied().unwrap_or(Site {
                                value: 0,
                                missing: true,
                            })),
                        }
                    })
                    .collect(),
            })
            .collect();
        let mut groups = BTreeMap::<(bool, &str, &str, String), u64>::new();
        for site in &self.final_sites {
            let group = match symbols.get(*site) {
                Symbol::Resolved { frame, .. } => (
                    false,
                    frame.package.as_str(),
                    frame.function.as_str(),
                    String::new(),
                ),
                Symbol::Unavailable { unavailable } => (
                    true,
                    "",
                    "",
                    serde_json::to_value(unavailable)
                        .map_err(|error| error.to_string())?
                        .as_str()
                        .ok_or("invalid unavailable reason")?
                        .into(),
                ),
            };
            *groups.entry(group).or_default() += 1;
        }
        if self.unmapped_final != 0 {
            *groups
                .entry((true, "", "", "site_missing".into()))
                .or_default() += self.unmapped_final;
        }
        let final_forced_by_symbol = groups
            .into_iter()
            .map(|((_, package, function, unavailable), count)| {
                if unavailable.is_empty() {
                    serde_json::json!({"package":package,"function":function,"count":count})
                } else {
                    serde_json::json!({"unavailable":unavailable,"count":count})
                }
            })
            .collect();
        Ok(Report {
            fields: fields.into_iter().collect(),
            accepted,
            final_forced_by_symbol,
        })
    }
}
