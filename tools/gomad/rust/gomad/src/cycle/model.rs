use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use std::str::FromStr;

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Mode {
    Serial,
    Parallel,
}

impl FromStr for Mode {
    type Err = String;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "serial" => Ok(Self::Serial),
            "parallel" => Ok(Self::Parallel),
            _ => Err("unknown mode".into()),
        }
    }
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Isolation {
    Normal,
    Budgeted,
    Exclusive,
    Solo,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Flag {
    Bg,
    Bgok,
    Cleantree,
    Heavy,
    Memguard,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "kebab-case")]
pub(crate) enum Adapter {
    CheckedCommand,
    Primary,
    Runtime,
    RuntimeAggregate,
    Qualify,
    Corpus,
    SqliteSelection,
    SqliteCatalogue,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct RuntimeBounds {
    pub instances: u32,
    pub replays: u32,
    pub seed: u64,
    pub shard_wall_seconds: u64,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Bounds {
    pub command_default_timeout_s: Option<u64>,
    pub primary_timeout_s: Option<u64>,
    pub static_timeout_s: Option<u64>,
    pub runtime: Option<RuntimeBounds>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Semantic {
    pub label: String,
    pub required_subgates: Vec<String>,
    pub repetitions: u32,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct StageContract {
    pub label: String,
    pub dependencies: Vec<String>,
    pub isolation_class: Isolation,
    pub cores: f64,
    pub mem_gib: f64,
    pub estimated_seconds: f64,
    pub locks: Vec<String>,
    pub scratch: Vec<String>,
    pub flags: Vec<Flag>,
    pub tier: u32,
    pub adapter: Adapter,
    pub bounds: Bounds,
    pub required_subgates: Vec<String>,
    pub repetitions: u32,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Stage {
    pub contract: StageContract,
    pub dependencies: Vec<String>,
    pub isolation_class: Isolation,
    pub argv: Vec<String>,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Plan {
    pub mode: Mode,
    pub stages: Vec<Stage>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct PlanConfig {
    pub binary: PathBuf,
    pub config_path: PathBuf,
    pub config_sha256: String,
}
