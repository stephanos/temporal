use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) enum Environment {
    Exact(BTreeMap<String, String>),
    Clean(BTreeMap<String, String>),
}
impl Environment {
    pub fn clean(
        inherited: BTreeMap<String, String>,
        overrides: &BTreeMap<String, String>,
        temporary: &BTreeMap<String, String>,
    ) -> Self {
        let mut values: BTreeMap<_, _> = inherited
            .into_iter()
            .filter(|(key, _)| {
                !key.starts_with("GOMAD")
                    && !key.starts_with("QUALDRV")
                    && ![
                        "RUSTFLAGS",
                        "CARGO_ENCODED_RUSTFLAGS",
                        "GOROOT",
                        "GOEXPERIMENT",
                        "GOOS",
                        "GOARCH",
                    ]
                    .contains(&key.as_str())
            })
            .collect();
        values.extend(overrides.clone());
        values.extend(temporary.clone());
        for (key, value) in [
            ("LC_ALL", "C"),
            ("GOENV", "off"),
            ("GOTOOLCHAIN", "local"),
            ("GOWORK", "off"),
            ("CARGO_INCREMENTAL", "0"),
            ("RUSTC_WRAPPER", ""),
            ("RUSTC_WORKSPACE_WRAPPER", ""),
        ] {
            values.insert(key.into(), value.into());
        }
        values.retain(|key, _| {
            !key.starts_with("GIT_") && !key.starts_with("SANDBOX_") && key != "GOROOT"
        });
        Self::Clean(values)
    }
    pub fn values(&self) -> &BTreeMap<String, String> {
        match self {
            Self::Exact(values) | Self::Clean(values) => values,
        }
    }
    pub fn validate(&self) -> Result<(), String> {
        if self
            .values()
            .iter()
            .any(|(key, value)| key.is_empty() || key.contains(['=', '\0']) || value.contains('\0'))
        {
            return Err("invalid exact command environment".into());
        }
        Ok(())
    }
}
