//! Shared upstream message catalogs, independent of activity state.
#![allow(dead_code, reason = "Mirror fields describe the API and are never instantiated")]
use super::{dc, views};
use umpire_catalog::{
    Catalog,
    kinds::{
        Counter, Duration, Enum, Flag, Identifier, Map, Message, OneOf, RawDuration, Secret, Text,
    },
};
use umpire_temporal_api::temporal::api::{
    common::v1 as common, enums::v1 as enums, failure::v1 as failure,
};
umpire_temporal_api::workflow_foundation_catalog!();
