use super::proposals::{Decision, Ordering};
use crate::{
    campaign::exploration::{
        choice::hex,
        simulation::Dimension,
        site_scope::{self, Resolution, SiteScope},
    },
    evidence::{
        canonical,
        inspection::Details,
        search_context,
        store::Opened,
        symbolize::{self, Site},
    },
};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct SourceBinding {
    payload_sha256: String,
    plan_sha256: String,
    round: u64,
    scope: SiteScope,
}

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Key {
    schema: String,
    selection: String,
    patterns: Vec<String>,
    source: Option<SourceBinding>,
    parent_record_sha256: String,
    parent_module_sha256: String,
    parent_trace_sha256: String,
    identity: linked::ChoiceIdentity,
    scope: SiteScope,
    classifications_sha256: String,
    decisions: u64,
}

pub(crate) struct Admission {
    pub ordering: Option<Ordering>,
    pub scope: Option<SiteScope>,
}

pub(crate) fn prepare(opened: &Opened, explicit: Option<Vec<String>>) -> Result<Admission, String> {
    let context = search_context::read(opened)?;
    let selection = if explicit.is_some() {
        "explicit"
    } else {
        "inherited"
    };
    let patterns = explicit.or_else(|| context.as_ref().map(|context| context.patterns.clone()));
    let Some(patterns) = patterns else {
        return Ok(Admission {
            ordering: None,
            scope: None,
        });
    };
    if site_scope::parse_patterns(&patterns.join(","))? != patterns {
        return Err("invalid normalized minimizer site scope".into());
    }
    let details = Details::read(opened)?.ok_or("minimization choice evidence is missing")?;
    let choices = details
        .choices
        .as_ref()
        .ok_or("minimization choice evidence is missing")?;
    choices.inspect(opened)?;
    let raw = opened.payload(&choices.trace.file, choices.trace.limit)?;
    let module = opened.payload("target/module.wasm", 512 << 20)?;
    site_scope::require_symbols(&module)?;
    let trace = linked::inspect_choices(&raw, &choices.identity)?;
    let plan = linked::project_choices(&raw, &choices.identity)?;
    let symbols = symbolize::resolve_bytes(
        &module,
        trace.sites.iter().map(|site| Site {
            value: site.site_offset,
            missing: site.site_missing,
        }),
    );
    let mut resolution = Resolution::default();
    resolution.observe_symbols(&raw, &choices.identity, &patterns, &symbols)?;
    let scope = resolution.scope(&patterns);
    let mut in_scope = BTreeSet::new();
    let mut classifications = Vec::with_capacity(plan.decisions.len());
    for decision in &plan.decisions {
        let key = Decision {
            dimension: Dimension::Runtime,
            ordinal: decision.ordinal(),
            identity: format!("sha256:{}", hex(&decision.selected_identity())),
        };
        let allowed = resolution.allows(&patterns, decision);
        classifications.push((key.clone(), allowed));
        if allowed {
            in_scope.insert(key);
        }
    }
    let source = context.map(|context| SourceBinding {
        payload_sha256: opened.manifest().files[search_context::FILE].sha256.clone(),
        plan_sha256: context.plan_sha256,
        round: context.round,
        scope: context.scope,
    });
    let key = Key {
        schema: "gomad.minimizer-site-ordering/v1".into(),
        selection: selection.into(),
        patterns,
        source,
        parent_record_sha256: opened.manifest().record_sha256.clone(),
        parent_module_sha256: opened.manifest().record.identity.target.clone(),
        parent_trace_sha256: canonical::digest(&raw),
        identity: choices.identity.clone(),
        scope: scope.clone(),
        classifications_sha256: canonical::digest(&canonical::encode(&classifications)?),
        decisions: plan.decisions.len() as u64,
    };
    let key = serde_json::to_value(key).map_err(|error| error.to_string())?;
    if canonical::encode(&key)?.len() > 16 << 20
        || canonical::encode(&classifications)?.len() > 16 << 20
    {
        return Err("minimizer site classification exceeds byte bound".into());
    }
    Ok(Admission {
        ordering: Some(Ordering { key, in_scope }),
        scope: Some(scope),
    })
}
