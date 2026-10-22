use crate::evidence::symbolize::{self, Site, Symbol};
use gomad_wasmhost::linked;
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct SiteScope {
    pub patterns: Vec<String>,
    pub resolved_sha256: String,
    pub resolved_count: u64,
}

impl SiteScope {
    pub(crate) fn validate(&self) -> Result<(), String> {
        if parse_patterns(&self.patterns.join(","))? != self.patterns
            || !super::valid_hash(&self.resolved_sha256)
        {
            return Err("invalid exploration plan site scope".into());
        }
        Ok(())
    }
}

pub(crate) fn parse_patterns(value: &str) -> Result<Vec<String>, String> {
    let mut patterns = Vec::new();
    for pattern in value.split(',') {
        let path = pattern.strip_suffix("/...").unwrap_or(pattern);
        if path.is_empty()
            || path.split('/').any(|part| {
                part.is_empty()
                    || matches!(part, "." | "..")
                    || part.starts_with('.')
                    || part.ends_with('.')
                    || !part.bytes().all(|byte| {
                        byte.is_ascii_alphanumeric()
                            || matches!(byte, b'-' | b'.' | b'_' | b'~' | b'+')
                    })
            })
        {
            return Err(format!("invalid --site-scope package pattern {pattern:?}"));
        }
        patterns.push(pattern.into());
    }
    Ok(patterns)
}

fn matches_package(patterns: &[String], package: &str) -> bool {
    patterns.iter().any(|pattern| {
        pattern.strip_suffix("/...").map_or_else(
            || package == pattern,
            |prefix| {
                package == prefix
                    || package
                        .strip_prefix(prefix)
                        .is_some_and(|tail| tail.starts_with('/'))
            },
        )
    })
}

pub(crate) fn require_symbols(module: &[u8]) -> Result<(), String> {
    match symbolize::require_symbols(module) {
        Err(symbolize::Reason::NoSymbols) => {
            Err("--site-scope requires module symbol information: no_symbols".into())
        }
        _ => Ok(()),
    }
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Resolution {
    packages: BTreeMap<u64, Option<String>>,
    fingerprints: BTreeSet<String>,
}

impl Resolution {
    pub(crate) fn scope(&self, patterns: &[String]) -> SiteScope {
        SiteScope {
            patterns: patterns.to_vec(),
            resolved_sha256: super::domain_hash(
                "gomad-explore-site-scope/v1",
                self.fingerprints
                    .iter()
                    .map(String::as_str)
                    .collect::<String>()
                    .as_bytes(),
            ),
            resolved_count: self.fingerprints.len() as u64,
        }
    }

    pub(crate) fn observe(
        &mut self,
        module: &[u8],
        raw: &[u8],
        identity: &linked::ChoiceIdentity,
        patterns: &[String],
    ) -> Result<(), String> {
        let trace = linked::inspect_choices(raw, identity)?;
        let symbols = symbolize::resolve_bytes(
            module,
            trace
                .sites
                .iter()
                .filter(|site| !site.site_missing)
                .map(|site| Site {
                    value: site.site_offset,
                    missing: false,
                }),
        );
        self.observe_symbols(raw, identity, patterns, &symbols)
    }

    pub(crate) fn observe_symbols(
        &mut self,
        raw: &[u8],
        identity: &linked::ChoiceIdentity,
        patterns: &[String],
        symbols: &symbolize::Symbols,
    ) -> Result<(), String> {
        let trace = linked::inspect_choices(raw, identity)?;
        for site in &trace.sites {
            if !site.site_missing && !self.packages.contains_key(&site.site_offset) {
                let package = match symbols.get(Site {
                    value: site.site_offset,
                    missing: false,
                }) {
                    Symbol::Resolved { frame, .. } => Some(frame.package.clone()),
                    Symbol::Unavailable { .. } => None,
                };
                self.packages.insert(site.site_offset, package);
            }
        }
        for site in trace.sites {
            if !site.site_missing && self.matches(patterns, site.site_offset) {
                self.fingerprints.insert(site.fingerprint);
            }
        }
        Ok(())
    }

    pub(crate) fn validate(&self, module: &[u8]) -> Result<(), String> {
        let bytes = crate::evidence::canonical::encode(self)?;
        if bytes.len() as u64 > crate::evidence::search_context::MAX_BYTES
            || self.packages.len() > (crate::evidence::search_context::MAX_BYTES / 16) as usize
            || self
                .fingerprints
                .iter()
                .any(|value| !crate::evidence::canonical::valid_digest(value))
        {
            return Err("invalid search context resolution".into());
        }
        let symbols = symbolize::resolve_bytes(
            module,
            self.packages.keys().map(|offset| Site {
                value: *offset,
                missing: false,
            }),
        );
        for (offset, observed) in &self.packages {
            let expected = match symbols.get(Site {
                value: *offset,
                missing: false,
            }) {
                Symbol::Resolved { frame, .. } => Some(frame.package.clone()),
                Symbol::Unavailable { .. } => None,
            };
            if *observed != expected {
                return Err("search context package observation mismatch".into());
            }
        }
        Ok(())
    }

    pub(crate) fn validate_fingerprints(
        &self,
        identity: &linked::ChoiceIdentity,
        patterns: &[String],
    ) -> Result<(), String> {
        use sha2::Digest as _;
        let mut expected = BTreeSet::new();
        for offset in self
            .packages
            .keys()
            .filter(|offset| self.matches(patterns, **offset))
        {
            for kind in 1_u8..=4 {
                let mut hash = sha2::Sha256::new();
                hash.update(identity.target_sha256);
                hash.update([kind]);
                hash.update(offset.to_be_bytes());
                expected.insert(format!("{:x}", hash.finalize()));
            }
        }
        if !self.fingerprints.is_subset(&expected) {
            return Err("search context fingerprint has no matching package observation".into());
        }
        Ok(())
    }

    fn matches(&self, patterns: &[String], offset: u64) -> bool {
        self.packages
            .get(&offset)
            .and_then(Option::as_ref)
            .is_some_and(|package| matches_package(patterns, package))
    }

    pub(crate) fn allows(&self, patterns: &[String], decision: &linked::Decision) -> bool {
        if decision.kind() == 1 {
            decision
                .creation_sites()
                .iter()
                .any(|site| !site.site_missing && self.matches(patterns, site.site_offset))
        } else {
            let (offset, missing) = decision.site();
            !missing && self.matches(patterns, offset)
        }
    }
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct RoundCounts {
    pub forced_in_scope: u64,
    pub skipped_out_of_scope: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_module() -> Vec<u8> {
        std::fs::read(
            std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                .join("../../testdata/parity/symbolize/guest.wasm"),
        )
        .unwrap()
    }

    fn identity() -> linked::ChoiceIdentity {
        linked::ChoiceIdentity {
            target_sha256: [1; 32],
            toolchain_build_key: "02".repeat(32),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [3; 32],
        }
    }

    fn runnable_trace(sites: [Option<u64>; 2]) -> Vec<u8> {
        use sha2::Digest as _;
        let mut trace = vec![0; 288];
        for (index, site) in sites.into_iter().enumerate() {
            let start = index * 96;
            trace[start..start + 8].copy_from_slice(&u64::try_from(index).unwrap().to_be_bytes());
            trace[start + 8] = 4;
            trace[start + 9] = if site.is_some() { 0 } else { 4 };
            trace[start + 12..start + 16].copy_from_slice(&2_u32.to_be_bytes());
            trace[start + 16..start + 20]
                .copy_from_slice(&u32::try_from(index).unwrap().to_be_bytes());
            trace[start + 24..start + 32].copy_from_slice(&site.unwrap_or(0).to_be_bytes());
            trace[start + 32..start + 64].fill(u8::try_from(index + 1).unwrap());
        }
        trace[192..200].copy_from_slice(&2_u64.to_be_bytes());
        trace[200] = 1;
        trace[201] = 5;
        trace[204..208].copy_from_slice(&2_u32.to_be_bytes());
        trace[208..212].copy_from_slice(&1_u32.to_be_bytes());
        trace[224..256].fill(2);
        let mut hash = sha2::Sha256::new();
        hash.update(b"gomad-choice-alternative-set/v1\0");
        hash.update(2_u64.to_be_bytes());
        hash.update([1; 32]);
        hash.update([2; 32]);
        trace[256..288].copy_from_slice(&hash.finalize());
        trace
    }

    #[test]
    fn site_scope_runnable_matches_any_alternative_and_legacy_has_no_creation_site() {
        let module = fixture_module();
        let patterns = vec!["main".into()];
        let identity = identity();
        let raw = runnable_trace([Some(5383), None]);
        let mut resolution = Resolution::default();
        resolution
            .observe(&module, &raw, &identity, &patterns)
            .unwrap();
        let projected = linked::project_choices(&raw, &identity).unwrap();
        assert!(resolution.allows(&patterns, &projected.decisions[0]));
        assert_eq!(resolution.scope(&patterns).resolved_count, 1);
        let unknown = runnable_trace([Some(u64::MAX), None]);
        resolution
            .observe(&module, &unknown, &identity, &patterns)
            .unwrap();
        let projected = linked::project_choices(&unknown, &identity).unwrap();
        assert!(!resolution.allows(&patterns, &projected.decisions[0]));
        let mut legacy = raw[192..].to_vec();
        legacy[..8].fill(0);
        let projected = linked::project_choices(&legacy, &identity).unwrap();
        assert_eq!(projected.decisions[0].creation_sites(), []);
        assert!(!resolution.allows(&patterns, &projected.decisions[0]));
    }

    #[test]
    fn site_scope_new_sites_extend_digest_once_independent_of_observation_order() {
        let module = fixture_module();
        let patterns = vec!["main".into()];
        let identity = identity();
        let first = runnable_trace([Some(5383), Some(5383)]);
        let second = runnable_trace([Some(5384), Some(5384)]);
        let mut resolution = Resolution::default();
        resolution
            .observe(&module, &first, &identity, &patterns)
            .unwrap();
        let initial = resolution.scope(&patterns);
        assert_eq!(initial.resolved_count, 1);
        resolution
            .observe(&module, &second, &identity, &patterns)
            .unwrap();
        let extended = resolution.scope(&patterns);
        assert_eq!(extended.resolved_count, 2);
        assert_ne!(extended.resolved_sha256, initial.resolved_sha256);
        resolution
            .observe(&module, &first, &identity, &patterns)
            .unwrap();
        assert_eq!(resolution.scope(&patterns), extended);
        let mut reversed = Resolution::default();
        reversed
            .observe(&module, &second, &identity, &patterns)
            .unwrap();
        reversed
            .observe(&module, &first, &identity, &patterns)
            .unwrap();
        assert_eq!(reversed.scope(&patterns), extended);
    }

    #[test]
    fn site_scope_snapshot_rejects_noncanonical_fingerprints_and_false_packages() {
        let module = fixture_module();
        for value in [
            serde_json::json!({"packages":{},"fingerprints":["invalid"]}),
            serde_json::json!({"packages":{"5383":"wrong"},"fingerprints":[]}),
        ] {
            let resolution: Resolution = serde_json::from_value(value).unwrap();
            assert!(resolution.validate(&module).is_err());
        }
    }

    #[test]
    fn site_scope_shared_symbols_match_convenience_observation() {
        let module = fixture_module();
        let patterns = vec!["main".into()];
        let raw = runnable_trace([Some(5383), None]);
        let symbols = symbolize::resolve_bytes(
            &module,
            [Site {
                value: 5383,
                missing: false,
            }],
        );
        let mut shared = Resolution::default();
        shared
            .observe_symbols(&raw, &identity(), &patterns, &symbols)
            .unwrap();
        let mut convenience = Resolution::default();
        convenience
            .observe(&module, &raw, &identity(), &patterns)
            .unwrap();
        assert_eq!(shared, convenience);
    }

    #[test]
    fn site_scope_patterns_reject_invalid_import_paths() {
        for value in [
            "",
            ",",
            "main,",
            " main",
            "main/.../other",
            "/main",
            "main//child",
            "main/..",
            "main@v1",
            "main\\child",
        ] {
            assert!(parse_patterns(value).is_err(), "{value:?}");
        }
        assert_eq!(
            parse_patterns("main,go.temporal.io/server/service/history/...").unwrap(),
            ["main", "go.temporal.io/server/service/history/..."]
        );
    }

    #[test]
    fn site_scope_plan_admission_refuses_malformed_scope_blocks() {
        for value in [
            serde_json::json!({"patterns":[],"resolved_sha256":format!("sha256:{}", "0".repeat(64)),"resolved_count":0}),
            serde_json::json!({"patterns":["main"],"resolved_sha256":"invalid","resolved_count":0}),
            serde_json::json!({"patterns":["main,"],"resolved_sha256":format!("sha256:{}", "0".repeat(64)),"resolved_count":0}),
            serde_json::json!({"patterns":["main"],"resolved_sha256":format!("sha256:{}", "0".repeat(64)),"resolved_count":0,"other":true}),
        ] {
            let admitted = serde_json::from_value::<SiteScope>(value)
                .map_err(|error| error.to_string())
                .and_then(|scope| scope.validate());
            assert!(admitted.is_err());
        }
    }

    #[test]
    fn site_scope_prefix_matches_package_boundary() {
        let patterns = vec!["example.org/server/...".into(), "main".into()];
        for (package, expected) in [
            ("main", true),
            ("main/sub", false),
            ("example.org/server", true),
            ("example.org/server/history", true),
            ("example.org/serverless", false),
        ] {
            assert_eq!(matches_package(&patterns, package), expected, "{package}");
        }
    }

    #[test]
    fn site_scope_stripped_module_refuses_before_execution() {
        for name in ["guest-stripped.wasm", "no-metadata.wasm"] {
            let module = std::fs::read(
                std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
                    .join("../../testdata/parity/symbolize")
                    .join(name),
            )
            .unwrap();
            assert!(require_symbols(&module).unwrap_err().contains("no_symbols"));
        }
    }
}
