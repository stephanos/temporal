//! Request-local simulation decisions over the existing executable model owners.
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::{collections::{BTreeMap, BTreeSet}, sync::{Arc, Mutex}};

pub(crate) type Shared = Arc<Mutex<Controller>>;

#[derive(Clone, Copy, Debug, Deserialize, Eq, Ord, PartialEq, PartialOrd, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum Dimension { Runtime, Scenario, Network, Storage, Fault, Crash }

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Override {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub site_sha256: String,
    pub alternatives: u32,
    pub alternative_set_sha256: String,
    pub selected: u32,
    pub selected_sha256: String,
    pub identity: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Plan {
    pub schema: String,
    pub execution_sha256: String,
    pub controller_sha256: String,
    pub base_seed: u64,
    pub overrides: Option<Vec<Override>>,
    pub candidate_sha256: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct Decision {
    pub dimension: Dimension,
    pub ordinal: u64,
    pub site_sha256: String,
    pub alternatives: Vec<String>,
    pub alternative_set_sha256: String,
    pub selected: u32,
    pub identity: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct Divergence {
    pub dimension: &'static str,
    pub ordinal: u64,
    pub expected_sha256: String,
    pub actual_sha256: String,
    pub expected_exploration_override: Override,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub actual_exploration: Option<Decision>,
}

#[derive(Debug)]
pub struct Controller {
    plan: Plan,
    maximum: u64,
    decisions: Vec<Decision>,
    ordinals: BTreeMap<Dimension, u64>,
    consumed: BTreeSet<(Dimension, u64)>,
    divergence: Option<Divergence>,
}

fn valid_sha(value: &str) -> bool {
    value.len() == 71 && value.starts_with("sha256:")
        && value.as_bytes()[7..].iter().all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(b))
}

/// # Errors
/// Serialization errors are returned before a model transition can be selected.
pub fn domain_hash(domain: &str, value: &Value) -> Result<String, String> {
    let bytes = serde_json::to_vec(value).map_err(|error| error.to_string())?;
    let mut hash = Sha256::new();
    hash.update(domain.as_bytes());
    hash.update([0]);
    hash.update(bytes);
    Ok(format!("sha256:{:x}", hash.finalize()))
}

impl Override {
    fn identity(&self) -> Result<String, String> {
        domain_hash("gomad-simulation-exploration-forced-decision/v1", &json!({
            "dimension":self.dimension,"ordinal":self.ordinal,"site_sha256":self.site_sha256,
            "alternatives":self.alternatives,"alternative_set_sha256":self.alternative_set_sha256,
            "selected":self.selected,"selected_sha256":self.selected_sha256,"identity":""
        }))
    }

    /// # Errors
    /// Refuses a rank outside the observed live alternative set.
    pub fn from_decision(decision: &Decision, selected: u32) -> Result<Self, String> {
        let selected_sha256 = decision.alternatives.get(selected as usize)
            .ok_or("exploration override selected rank is invalid")?.clone();
        let mut value = Self { dimension:decision.dimension,ordinal:decision.ordinal,
            site_sha256:decision.site_sha256.clone(),alternatives:u32::try_from(decision.alternatives.len())
                .map_err(|_| "exploration alternatives capacity")?,
            alternative_set_sha256:decision.alternative_set_sha256.clone(),selected,selected_sha256,identity:String::new() };
        value.identity = value.identity()?;
        Ok(value)
    }
}

impl Plan {
    fn identity(&self) -> Result<String, String> {
        domain_hash("gomad-simulation-exploration-candidate/v1", &json!({
            "execution_sha256":self.execution_sha256,"controller_sha256":self.controller_sha256,
            "base_seed":self.base_seed,"overrides":self.overrides
        }))
    }

    fn validate(&self) -> Result<(), String> {
        if self.schema != "gomad.simulation-exploration-plan/v1"
            || !valid_sha(&self.execution_sha256) || !valid_sha(&self.controller_sha256)
            || !valid_sha(&self.candidate_sha256) || self.identity()? != self.candidate_sha256 {
            return Err("simulation exploration plan identity is invalid".into());
        }
        let overrides = self.overrides.as_deref().unwrap_or_default();
        if overrides.len() > 65536 { return Err("simulation exploration overrides capacity".into()); }
        let mut previous = None;
        for value in overrides {
            let key = (value.dimension,value.ordinal);
            if previous.is_some_and(|previous| previous >= key)
                || value.alternatives < 2 || value.selected >= value.alternatives
                || [&value.site_sha256,&value.alternative_set_sha256,&value.selected_sha256,&value.identity]
                    .iter().any(|identity| !valid_sha(identity)) || value.identity()? != value.identity {
                return Err("simulation exploration override is invalid".into());
            }
            previous = Some(key);
        }
        Ok(())
    }

    /// # Errors
    /// Refuses malformed, noncanonical or oversized plans before guest admission.
    pub fn decode(raw: &[u8]) -> Result<Self, String> {
        if raw.is_empty() || raw.len() > 16 << 20 { return Err("simulation exploration plan bytes capacity".into()); }
        let plan: Self = serde_json::from_slice(raw).map_err(|error| error.to_string())?;
        plan.validate()?;
        if plan.encode()? != raw { return Err("simulation exploration plan is not canonical JSON".into()); }
        Ok(plan)
    }

    /// # Errors
    /// The plan must retain its validated candidate identity and bound.
    pub fn encode(&self) -> Result<Vec<u8>, String> {
        self.validate()?;
        let raw = serde_json::to_vec(self).map_err(|error| error.to_string())?;
        if raw.len() > 16 << 20 { return Err("simulation exploration plan bytes capacity".into()); }
        Ok(raw)
    }

    /// # Errors
    /// Refuses invalid identities, duplicate forced keys and invalid ranks.
    pub fn new(execution: String, controller: String, seed: u64, mut overrides: Vec<Override>) -> Result<Self, String> {
        overrides.sort_by_key(|value| (value.dimension,value.ordinal));
        let mut plan = Self {schema:"gomad.simulation-exploration-plan/v1".into(),
            execution_sha256:execution,controller_sha256:controller,base_seed:seed,
            overrides:(!overrides.is_empty()).then_some(overrides),candidate_sha256:String::new()};
        plan.candidate_sha256=plan.identity()?;
        plan.validate()?;
        Ok(plan)
    }
}

impl Decision {
    fn identity(&self) -> Result<String, String> {
        domain_hash("gomad-simulation-exploration-decision/v1", &json!({
            "dimension":self.dimension,"ordinal":self.ordinal,"site_sha256":self.site_sha256,
            "alternatives":self.alternatives,"alternative_set_sha256":self.alternative_set_sha256,
            "selected":self.selected,"identity":""
        }))
    }
}

impl Controller {
    /// # Errors
    /// Rejects invalid plans and decision budgets before attaching a model owner.
    pub fn new(plan: Plan, maximum: u64) -> Result<Self, String> {
        plan.validate()?;
        if maximum == 0 || maximum > 65536 { return Err("simulation exploration decisions capacity".into()); }
        Ok(Self { plan,maximum,decisions:Vec::new(),ordinals:BTreeMap::new(),consumed:BTreeSet::new(),divergence:None })
    }

    pub fn decisions(&self) -> &[Decision] { &self.decisions }
    pub fn divergence(&self) -> Option<&Divergence> { self.divergence.as_ref() }

    fn diverge(&mut self, expected: Override, actual: Option<Decision>) -> Result<u32,String> {
        let actual_sha256 = match &actual { Some(actual)=>actual.identity.clone(),None=>domain_hash(
            "gomad-missing-exploration-decision/v1",&json!({"dimension":expected.dimension,"ordinal":expected.ordinal}))? };
        self.divergence=Some(Divergence {dimension:"exploration",ordinal:expected.ordinal,
            expected_sha256:expected.identity.clone(),actual_sha256,expected_exploration_override:expected,actual_exploration:actual});
        Err("simulation exploration replay divergence".into())
    }

    /// # Errors
    /// Validates forced identity and live alternatives before the caller mutates its model.
    pub fn choose(&mut self,dimension:Dimension,site:&str,alternatives:Vec<String>,selected:u32)->Result<u32,String> {
        if self.divergence.is_some() { return Err("simulation exploration replay divergence".into()); }
        if self.decisions.len() as u64 >= self.maximum { return Err("simulation exploration decisions capacity".into()); }
        if !valid_sha(site) || alternatives.len()<2 || alternatives.len()>u32::MAX as usize
            || selected as usize>=alternatives.len() || alternatives.iter().any(|value| !valid_sha(value))
            || alternatives.iter().collect::<BTreeSet<_>>().len()!=alternatives.len() {
            return Err("simulation exploration live alternatives are invalid".into());
        }
        let ordinal=*self.ordinals.get(&dimension).unwrap_or(&0);
        let alternative_set_sha256=domain_hash("gomad-simulation-exploration-alternative-set/v1",&json!({
            "dimension":dimension,"ordinal":ordinal,"site_sha256":site,"alternatives":alternatives
        }))?;
        let mut decision=Decision {dimension,ordinal,site_sha256:site.into(),alternatives,alternative_set_sha256,selected,identity:String::new()};
        decision.identity=decision.identity()?;
        let forced=self.plan.overrides.iter().flatten().find(|value| value.dimension==dimension && !self.consumed.contains(&(dimension,value.ordinal))).cloned();
        if let Some(forced)=forced {
            if forced.ordinal<ordinal { return self.diverge(forced,Some(decision)); }
            if forced.ordinal==ordinal {
                if forced.site_sha256!=decision.site_sha256 || forced.alternatives as usize!=decision.alternatives.len()
                    || forced.alternative_set_sha256!=decision.alternative_set_sha256
                    || decision.alternatives[forced.selected as usize]!=forced.selected_sha256 {
                    return self.diverge(forced,Some(decision));
                }
                decision.selected=forced.selected;
                decision.identity=decision.identity()?;
                self.consumed.insert((dimension,ordinal));
            }
        }
        let selected=decision.selected;
        self.decisions.push(decision);
        self.ordinals.insert(dimension,ordinal+1);
        Ok(selected)
    }

    /// # Errors
    /// Any forced non-runtime decision not reached by the actual execution diverges.
    pub fn finish(&mut self)->Result<(),String> {
        if self.divergence.is_some() { return Err("simulation exploration replay divergence".into()); }
        let missing=self.plan.overrides.iter().flatten().find(|value| value.dimension!=Dimension::Runtime && !self.consumed.contains(&(value.dimension,value.ordinal))).cloned();
        if let Some(missing)=missing { self.diverge(missing,None)?; }
        Ok(())
    }
}
