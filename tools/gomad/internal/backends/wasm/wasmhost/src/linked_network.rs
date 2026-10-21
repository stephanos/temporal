#[cfg(not(feature = "conformance"))]
use crate::{abi, cluster::network::Endpoint, engine, linked_cluster::NetworkModel, model};
use crate::{cluster::network::Failure, world};
use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde_json::{Value, json};
#[cfg(not(feature = "conformance"))]
use wasmtime::{Error, Result};

#[cfg(not(feature = "conformance"))]
pub(crate) fn prepare(configuration: NetworkModel, model: &model::Model) -> Result<Endpoint> {
    Endpoint::standalone(configuration, model.epoch_nanos(), model.transcript_limit())
        .map_err(error)
}

#[cfg(not(feature = "conformance"))]
pub(crate) fn error(failure: Failure) -> Error {
    match failure {
        Failure::Invalid(dimension) => Error::new(crate::namespace::Boundary {
            kind: "invalid",
            dimension,
        }),
        Failure::Capacity(dimension) => Error::new(crate::namespace::Boundary {
            kind: "capacity",
            dimension,
        }),
        Failure::Boundary(kind, dimension) => {
            Error::new(crate::namespace::Boundary { kind, dimension })
        }
        failure => Error::msg(failure.message()),
    }
}

#[cfg(not(feature = "conformance"))]
impl engine::State {
    pub(crate) fn network_call(
        &mut self,
        operation: abi::Operation,
        mut data: Vec<u8>,
    ) -> Result<Vec<u8>> {
        self.inspection_snapshot(engine::Snapshot::Call);
        if self.network.is_none() {
            self.model_operation(|_, _, _| {
                let call = crate::network::stream_abi::decode(&data)
                    .map_err(|invalid| error(Failure::Invalid(invalid.0)))?;
                if call.op != 0 {
                    return Err(error(Failure::Invalid("network v2 operation")));
                }
                crate::network::stream_abi::query(&call, data.len())
                    .map_err(|invalid| error(Failure::Invalid(invalid.0)))?;
                crate::network::stream_abi::set_u32(&mut data, 40, 2);
                Ok(())
            })?;
            return Ok(data);
        }
        let mut network = self.network.take().unwrap();
        let result = self.model_operation(|model, _, _| {
            let served = if operation == abi::Operation::RuntimeIdle {
                if !model.runtime_active() {
                    return Err(Error::msg(
                        "WASI invalid: runtime idle outside active lifetime",
                    ));
                }
                network.idle(&mut data, &mut model.now, &mut model.advances)
            } else {
                network.serve(&mut data, model.now, model.runtime_active())
            };
            match served {
                Ok(()) => Ok(None),
                Err(Failure::Infrastructure(message)) => Ok(Some(message)),
                Err(failure) => Err(error(failure)),
            }
        });
        self.network = Some(network);
        if let Some(message) = result? {
            return Err(self.fail("infrastructure", message));
        }
        Ok(data)
    }
}

pub(crate) fn json_bytes(value: &Value) -> std::result::Result<Vec<u8>, Failure> {
    let text =
        serde_json::to_string(value).map_err(|error| Failure::Infrastructure(error.to_string()))?;
    Ok(text
        .replace('&', "\\u0026")
        .replace('<', "\\u003c")
        .replace('>', "\\u003e")
        .replace('\u{2028}', "\\u2028")
        .replace('\u{2029}', "\\u2029")
        .into_bytes())
}

fn request(value: &world::Request) -> Value {
    let mut output = json!({"kind":value.kind,"resource":{"adapter":value.resource.adapter,"kind":value.resource.kind,"key":value.resource.key},"priority":value.priority});
    if !value.payload.is_empty() {
        output["payload"] = STANDARD.encode(&value.payload).into();
    }
    output
}

fn readiness(value: &world::Readiness) -> Value {
    let mut output = json!({"requestId":value.request_id.to_string(),"at":value.at.to_string(),"kind":value.kind});
    if !value.payload.is_empty() {
        output["payload"] = STANDARD.encode(&value.payload).into();
    }
    if !value.equivalence_class.is_empty() {
        output["equivalenceClass"] = value.equivalence_class.clone().into();
    }
    output
}

pub(crate) fn world_snapshot(model: &world::Model) -> Value {
    let config = model.config();
    let limits = &config.limits;
    let (next_request, next_event, next_transition) = model.next_ids();
    let requests: Vec<_> = model.requests().map(|(id, entry)| {
        let mut output = json!({"id":id.to_string(),"request":request(&entry.request),"state":entry.state.as_str()});
        if entry.event_id != 0 { output["eventId"] = entry.event_id.to_string().into(); }
        output
    }).collect();
    let events: Vec<_> = model.events().map(|(id, event)| json!({"id":id.to_string(),"readiness":readiness(&event.readiness),"state":event.state.as_str()})).collect();
    let transitions: Vec<_> = model.transitions().iter().map(|transition| {
        let mut output = json!({"sequence":transition.sequence.to_string(),"kind":transition.body.kind(),"previousDigest":world::digest::hex(&transition.previous_digest),"digest":world::digest::hex(&transition.digest)});
        match &transition.body {
            world::TransitionBody::Register { request: value, request_id } => output["register"] = json!({"request":request(value),"requestId":request_id.to_string()}),
            world::TransitionBody::Ready { readiness: value, event_id } => output["ready"] = json!({"readiness":readiness(value),"eventId":event_id.to_string()}),
            world::TransitionBody::Cancel { request_id, cancellation } => {
                let mut value = json!({"requestId":cancellation.request_id.to_string(),"status":cancellation.status.as_str()});
                if cancellation.event_id != 0 { value["eventId"] = cancellation.event_id.to_string().into(); }
                output["cancel"] = json!({"requestId":request_id.to_string(),"cancellation":value});
            },
            world::TransitionBody::Quiesce { result } => {
                let (bytes, _) = crate::cluster::network::evidence::quiescence_entry(result);
                let value: Value = serde_json::from_slice(&bytes).expect("helper encodes valid quiescence JSON");
                output["quiesce"] = json!({"result":value});
            },
        }
        output
    }).collect();
    json!({"schemaVersion":world::SCHEMA_VERSION,"config":{"seed":config.seed.to_string(),"limits":{"maxRequests":limits.max_requests,"maxEvents":limits.max_events,"maxQueuedEvents":limits.max_queued_events,"maxTransitions":limits.max_transitions,"maxPayloadBytes":limits.max_payload_bytes,"maxStringBytes":limits.max_string_bytes}},"now":model.now().to_string(),"nextRequestId":next_request.to_string(),"nextEventId":next_event.to_string(),"nextTransition":next_transition.to_string(),"payloadBytes":model.payload_bytes(),"requests":requests,"events":events,"transitions":transitions,"transcriptDigest":world::digest::hex(model.transcript_digest()),"replay":{"cursor":0,"expected":0},"stateDigest":world::digest::hex(&model.state_digest())})
}
