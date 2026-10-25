//! Typed Kettle wire and prerequisite policy for the shared probe engine.

use crate::{
    KettleServiceMethod as Method, api,
    config::{self, Limits},
    kettle_request_fields, mutate_kettle_request,
    wire::umpire::reference::kettle::v1 as k,
};
use prost::Message;
use std::sync::OnceLock;
use umpire_catalog::{
    Violation, WireCase, WireField,
    generic_probe::{Engine, ProbeSystem, Request, Result},
};
use umpire_runtime::annotations::ProbeResponse;

struct Kettle;
impl ProbeSystem for Kettle {
    type Method = Method;
    type Context = ();
    fn methods(&self) -> Vec<(Method, &'static str)> {
        vec![
            (Method::Fill, "empty"),
            (Method::Heat, "filled"),
            (Method::Pour, "filled"),
            (Method::Describe, "filled"),
            (Method::PollHeating, "heating"),
            (Method::RespondHeating, "heating"),
        ]
    }
    fn name(&self, method: Method) -> &'static str {
        method.name()
    }
    fn rpc_path(&self, method: Method) -> String {
        format!("/umpire.reference.kettle.v1.KettleService/{}", method.name())
    }
    fn fields(&self, method: Method) -> Vec<WireField> {
        kettle_request_fields(method)
    }
    fn bounds(&self) -> Vec<u64> {
        let limits = Limits::default();
        vec![
            Limits::for_kettle("compact").get(&config::MAX_TAGS),
            limits.get(&config::MAX_TAGS),
            limits.get(&config::MAX_KETTLE_ID_BYTES),
            60,
        ]
    }
    fn canonical_context(&self) {}
    fn baseline(&self, method: Method, (): &()) -> Result<Vec<u8>> {
        Ok(baseline_bytes(method))
    }
    fn mutate(&self, method: Method, case: &WireCase, bytes: &[u8]) -> Result<Vec<u8>> {
        Ok(mutate_kettle_request(method, bytes, &case.field.path, &case.mutation)?)
    }
    fn findings(&self, method: Method, bytes: &[u8]) -> Result<Vec<Violation>> {
        api::request_findings(method, bytes, "room")
    }
    fn mutation_refusal(
        &self,
        case: &WireCase,
        prerequisite: &str,
        findings: &[Violation],
    ) -> Option<String> {
        (findings.is_empty()
            && prerequisite != "empty"
            && [".kettle_id", ".enabled", ".action"]
                .iter()
                .any(|suffix| case.field.path.ends_with(suffix)))
        .then(|| "mutation changes a stateful prerequisite".into())
    }
    fn conclusive_status(&self, code: i32) -> bool {
        matches!(code, 3 | 5..=11 | 16)
    }
}

fn engine() -> Result<&'static Engine<Kettle>> {
    static ENGINE: OnceLock<Engine<Kettle>> = OnceLock::new();
    if let Some(engine) = ENGINE.get() {
        return Ok(engine);
    }
    let engine = Engine::new(Kettle)?;
    Ok(ENGINE.get_or_init(|| engine))
}

fn baseline_bytes(method: Method) -> Vec<u8> {
    let id = "room".to_string();
    match method {
        Method::Fill => k::FillRequest {
            kettle_id: id,
            idempotency_key: "12345678-1234-4234-8234-123456789abc".into(),
            options: Some(k::fill_request::FillOptions {
                water_kind: 1,
                source: Some(k::fill_request::fill_options::Source::Tap("tap".into())),
                labels: [("origin".into(), "local".into())].into_iter().collect(),
                tags: vec!["clean".into()],
                timeout: Some(prost_types::Duration { seconds: 1, nanos: 0 }),
                ..Default::default()
            }),
        }
        .encode_to_vec(),
        Method::Heat => k::HeatRequest { kettle_id: id, enabled: true }.encode_to_vec(),
        Method::Pour => k::PourRequest { kettle_id: id }.encode_to_vec(),
        Method::Describe => k::DescribeRequest { kettle_id: id }.encode_to_vec(),
        Method::PollHeating => k::PollHeatingRequest { kettle_id: id }.encode_to_vec(),
        Method::RespondHeating => {
            k::RespondHeatingRequest { kettle_id: id, action: 1 }.encode_to_vec()
        }
        Method::ListKettles | Method::ResetKettle => unreachable!("out-of-scope probe method"),
    }
}

/// Prepare descriptor mutations or classify the complete set of actual server observations.
pub fn run(request: &k::ProbeRequest) -> Result<ProbeResponse> {
    if !request.kettle_id.is_empty() && request.kettle_id != "room" {
        return Err("Kettle probe context must use the canonical room kettle".into());
    }
    let operation = match request.operation {
        0 => Request::Inventory,
        1 => Request::Prepare { id: &request.case_id, baseline: request.baseline, context: &() },
        2 => Request::Classify(&request.observations),
        3 => {
            return Ok(ProbeResponse {
                report: api::catalog::coverage(&Limits::default())?,
                ..Default::default()
            });
        }
        _ => return Err("unknown Kettle probe operation".into()),
    };
    engine()?.run(&operation)
}
