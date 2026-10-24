//! Exhaustive Kettle wire mirrors, validation intents and Model bindings.
#![allow(dead_code, reason = "Catalog mirrors describe wire fields without being instantiated")]

use crate::wire::umpire::reference::kettle::v1 as k;
use crate::{KettleServiceMethod, config, kettle};
use umpire_catalog::{
    Catalog,
    kinds::{Counter, Enum, Flag, Ignored, Map, Message, OneOf, RawDuration, Text},
};

type ModelAction = <kettle::World as umpire_dsl::Machine>::Action;

/// Raw protobuf duration fields retain their signed carrier values.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "google.protobuf.Duration")]
pub struct ProtoDuration {
    seconds: Counter,
    nanos: Counter,
}

/// The complete projection observed by Judge.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.State")]
pub struct State {
    #[rule(gte = -40, lte = 100, unobservable)]
    temperature: Counter,
    heating_on: Flag,
    beeping: Flag,
}

/// Optional fill settings are independently validated by the reference service.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.FillRequest.FillOptions", type = k::fill_request::FillOptions)]
#[rule(any_of(tap, cartridge), rejects = "water source must match water_kind")]
pub struct FillOptions {
    #[mirror(via = water_source)]
    #[rule(rejects = "water_kind must be TAP or FILTERED|water source must match water_kind")]
    water_kind: Enum<k::fill_request::fill_options::WaterKind>,
    #[mirror(oneof(Tap = "tap", Cartridge = "cartridge"))]
    source: OneOf<k::fill_request::fill_options::Source>,
    #[mirror(via = labels)]
    #[rule(min_len = 1, rejects = "labels require nonempty keys and values")]
    labels: Map<Text, Text>,
    #[rule(max_items = config::MAX_TAGS, rejects = "tags exceed per-kettle scoped limit")]
    tags: Vec<Text>,
    #[rule(
        gt = 0,
        lte = 60_000_000_000,
        rejects = "timeout must be a valid positive duration at most 60 seconds"
    )]
    timeout: RawDuration,
    #[mirror(
        ignored = "Deprecated note is retained for wire compatibility; the service assigns it no semantics"
    )]
    legacy_note: Ignored,
}

/// Creation may fill an omitted idempotency key without replacing a supplied key.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.FillRequest")]
pub struct FillRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
    options: Message<k::fill_request::FillOptions>,
    #[mirror(idempotency_key, also(uuid, accepts))]
    idempotency_key: Text,
}

/// Creation returns the selected initial projection.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.FillResponse")]
pub struct FillResponse {
    state: Message<k::State>,
}

/// The button flag selects the user action committed after success.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.HeatRequest")]
pub struct HeatRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
    enabled: Flag,
}

/// Button acknowledgement returns the committed projection.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.HeatResponse")]
pub struct HeatResponse {
    state: Message<k::State>,
}

/// Dispensing names the current reference kettle.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.PourRequest")]
pub struct PourRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
}

/// Dispensing acknowledges the unchanged projection.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.PourResponse")]
pub struct PourResponse {
    state: Message<k::State>,
}

/// A read names the current reference kettle.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.DescribeRequest")]
pub struct DescribeRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
}

/// Reads reveal only the current projection.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.DescribeResponse")]
pub struct DescribeResponse {
    state: Message<k::State>,
}

/// The heater asks for work on the current kettle.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.PollHeatingRequest")]
pub struct PollHeatingRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
}

/// Polling returns proposed work without committing a heater action.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.PollHeatingResponse")]
pub struct PollHeatingResponse {
    state: Message<k::State>,
    action: Enum<k::HeatingAction>,
    kettle_id: Text,
}

/// The heater reports one action to apply.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, request = "umpire.reference.kettle.v1.RespondHeatingRequest")]
pub struct RespondHeatingRequest {
    #[rule(min_bytes = 1, max_bytes = config::MAX_KETTLE_ID_BYTES, rejects = "kettle_id must contain 1..32 bytes")]
    kettle_id: Text,
    #[mirror(via = heating_action)]
    #[rule(rejects = "heating action must be HEAT or FAILOVER")]
    action: Enum<k::HeatingAction>,
}

/// The receipt identifies the applied action and resulting projection.
#[derive(Catalog)]
#[mirror(api = crate::wire, context = config::Limits, response = "umpire.reference.kettle.v1.RespondHeatingResponse")]
pub struct RespondHeatingResponse {
    state: Message<k::State>,
    action: Enum<k::HeatingAction>,
}

pub(crate) enum RpcRole {
    Observed(&'static str),
    OutOfScope(&'static str),
}

pub(crate) fn classify(method: KettleServiceMethod) -> RpcRole {
    match method {
        KettleServiceMethod::Fill => {
            RpcRole::Observed("creates the reference kettle at the selected initial state")
        }
        KettleServiceMethod::Heat => RpcRole::Observed(
            "enabled selects User::PressButton or User::DepressButton after success",
        ),
        KettleServiceMethod::Pour => {
            RpcRole::Observed("dispensing acknowledgement preserves the current projection")
        }
        KettleServiceMethod::Describe => RpcRole::Observed("zero-step current state projection"),
        KettleServiceMethod::PollHeating => {
            RpcRole::Observed("heater callback chooses an action without committing it")
        }
        KettleServiceMethod::RespondHeating => {
            RpcRole::Observed("receipt commits exactly one Heater action")
        }
        KettleServiceMethod::ListKettles => {
            RpcRole::OutOfScope("enumerating resources is outside the single-kettle Model")
        }
        KettleServiceMethod::ResetKettle => {
            RpcRole::OutOfScope("server reset is outside the bounded live Kettle run")
        }
    }
}

mod water_source {
    use super::k;
    pub(super) fn view(message: &k::fill_request::FillOptions) -> i32 {
        use k::fill_request::fill_options::{Source, WaterKind};
        match (&message.source, WaterKind::try_from(message.water_kind)) {
            (Some(Source::Tap(tap)), Ok(WaterKind::Tap)) if !tap.trim().is_empty() => {
                message.water_kind
            }
            (Some(Source::Cartridge(cartridge)), Ok(WaterKind::Filtered))
                if !cartridge.is_empty() =>
            {
                message.water_kind
            }
            _ => -1,
        }
    }
    #[allow(
        clippy::unnecessary_wraps,
        reason = "Typed projection setters share the fallible View contract"
    )]
    pub(super) fn set(
        message: &mut k::fill_request::FillOptions,
        value: i32,
    ) -> Result<(), umpire_catalog::GenerationError> {
        message.water_kind = value;
        Ok(())
    }
}

mod heating_action {
    use super::k;
    pub(super) fn view(message: &k::RespondHeatingRequest) -> i32 {
        if matches!(
            k::HeatingAction::try_from(message.action),
            Ok(k::HeatingAction::Heat | k::HeatingAction::Failover)
        ) {
            message.action
        } else {
            -1
        }
    }
    #[allow(
        clippy::unnecessary_wraps,
        reason = "Typed projection setters share the fallible View contract"
    )]
    pub(super) fn set(
        message: &mut k::RespondHeatingRequest,
        value: i32,
    ) -> Result<(), umpire_catalog::GenerationError> {
        message.action = value;
        Ok(())
    }
}

mod labels {
    use super::k;
    use umpire_catalog::{
        View,
        kinds::{Map, Text},
    };
    pub(super) fn view(message: &k::fill_request::FillOptions) -> Vec<Vec<u8>> {
        use prost::Message;
        let mut entries: Vec<_> = message.labels.iter().collect();
        entries.sort_by(|a, b| a.0.cmp(b.0));
        entries
            .into_iter()
            .map(|(key, value)| {
                if key.is_empty() || value.is_empty() {
                    Vec::new()
                } else {
                    umpire_catalog::views::TextEntry { key: key.clone(), value: value.clone() }
                        .encode_to_vec()
                }
            })
            .collect()
    }
    pub(super) fn set(
        message: &mut k::fill_request::FillOptions,
        value: Vec<Vec<u8>>,
    ) -> Result<(), umpire_catalog::GenerationError> {
        <_ as View<Map<Text, Text>>>::set(&mut message.labels, value)
    }
}

/// Executable factories and exhaustive runtime Model bindings.
pub mod catalog {
    use super::{
        DescribeRequest, DescribeResponse, FillOptions, FillRequest, FillResponse, HeatRequest,
        HeatResponse, ModelAction, PollHeatingRequest, PollHeatingResponse, PourRequest,
        PourResponse, ProtoDuration, RespondHeatingRequest, RespondHeatingResponse, State, config,
        k, kettle,
    };
    use std::fmt::Write;
    use umpire_catalog::{AdmissionErrors, ProbeCase};

    /// Every Model action has an explicit service binding.
    pub fn bind(action: &ModelAction) -> umpire_catalog::Binding {
        match action {
            ModelAction::User(kettle::User::PressButton | kettle::User::DepressButton) => {
                umpire_catalog::Binding::Rpc("umpire.reference.kettle.v1.KettleService.Heat")
            }
            ModelAction::Heater(kettle::Heater::Heat | kettle::Heater::Failover) => {
                umpire_catalog::Binding::Rpc(
                    "umpire.reference.kettle.v1.KettleService.RespondHeating",
                )
            }
        }
    }

    fn append_coverage<M: 'static>(output: &mut String, catalog: &umpire_catalog::Catalog<M>) {
        writeln!(
            output,
            "{} {}",
            catalog.message(),
            if catalog.is_request() { "request" } else { "response" }
        )
        .unwrap_or_else(|_| unreachable!());
        for field in catalog.fields() {
            writeln!(
                output,
                "  {} {:?} tags={:?} expectations={:?}",
                field.path, field.kind, field.tags, field.expectations
            )
            .unwrap_or_else(|_| unreachable!());
        }
        for rule in catalog.rules() {
            writeln!(output, "  {rule:?}").unwrap_or_else(|_| unreachable!());
        }
    }

    macro_rules! inventory {
        ($($mirror:ident),* $(,)?) => {
            /// All primary and supplemental catalog expectations, sorted by identity.
            pub fn expected_probe_cases(limits: &config::Limits) -> Result<Vec<ProbeCase>, AdmissionErrors> {
                let mut cases = Vec::new();
                $(for catalog in $mirror::catalogs(limits)? { cases.extend(catalog.probe_cases()); })*
                cases.sort_by(|a, b| a.id.cmp(&b.id));
                Ok(cases)
            }
            /// Stable field and rule inventory for every in-scope wire message.
            pub fn coverage(limits: &config::Limits) -> Result<String, AdmissionErrors> {
                let mut output = String::new();
                $(for catalog in $mirror::catalogs(limits)? { append_coverage(&mut output, &catalog); })*
                append_coverage(&mut output, &FillOptions::__labels_entry_catalog(limits)?);
                Ok(output)
            }
        };
    }
    inventory! {
        ProtoDuration, State, FillOptions, FillRequest, FillResponse, HeatRequest, HeatResponse,
        PourRequest, PourResponse, DescribeRequest, DescribeResponse, PollHeatingRequest,
        PollHeatingResponse, RespondHeatingRequest, RespondHeatingResponse,
    }

    umpire_catalog::annotate! {
        pub catalog enum_inventory;
        enum water_kind: k::fill_request::fill_options::WaterKind = "umpire.reference.kettle.v1.FillRequest.FillOptions.WaterKind" {
            k::fill_request::fill_options::WaterKind::Unspecified = "WATER_KIND_UNSPECIFIED",
            k::fill_request::fill_options::WaterKind::Tap = "WATER_KIND_TAP",
            k::fill_request::fill_options::WaterKind::Filtered = "WATER_KIND_FILTERED",
        }
        enum heating_action: k::HeatingAction = "umpire.reference.kettle.v1.HeatingAction" {
            k::HeatingAction::Unspecified = "HEATING_ACTION_UNSPECIFIED",
            k::HeatingAction::Heat = "HEATING_ACTION_HEAT",
            k::HeatingAction::Failover = "HEATING_ACTION_FAILOVER",
        }
    }
}

/// Raw catalog predictions for an actual request, including accepted and unobservable intents.
pub fn request_findings(
    method: KettleServiceMethod,
    bytes: &[u8],
    kettle_id: &str,
) -> Result<Vec<umpire_catalog::Violation>, Box<dyn std::error::Error>> {
    use prost::Message;
    use umpire_catalog::Inspect;
    let limits = config::Limits::for_kettle(kettle_id);
    dispatch_kettle_annotations!(
        method,
        false,
        Wire,
        {
            let message = Wire::decode(bytes)?;
            let mut findings = Vec::new();
            message.inspect(&limits, false, &mut findings)?;
            Ok(findings)
        },
        Err("request is outside the Kettle catalog".into())
    )
}
