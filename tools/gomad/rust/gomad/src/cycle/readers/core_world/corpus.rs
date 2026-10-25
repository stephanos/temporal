use super::{BrokerConfig, NetworkModel, Value, codec, fields, integer, json, require, sha, world};
use std::collections::{BTreeMap, BTreeSet};

/// Current corpus bytes, independently read and bound to the admitted source inventory.
pub(crate) struct Corpus {
    kind: String,
    documents: BTreeMap<String, Value>,
}

impl Corpus {
    pub(crate) fn admit(
        kind: &str,
        raw: &BTreeMap<String, Vec<u8>>,
        frozen: &BTreeMap<String, String>,
        pin: Option<&str>,
    ) -> Result<Self, String> {
        require(
            !raw.is_empty() && raw.keys().eq(frozen.keys()),
            "corpus file inventory",
        )?;
        let schema = match kind {
            "world" => "gomad.world-conformance/v1",
            "clock" => "gomad.clock-conformance/v1",
            "network" => "gomad.network-cluster-corpus/v1",
            "broker" => "",
            _ => return Err("foreign corpus kind".into()),
        };
        let mut documents = BTreeMap::new();
        for (name, bytes) in raw {
            require(
                !name.is_empty() && !name.contains(['/', '\\', '\0']) && sha(bytes) == frozen[name],
                "corpus current input identity",
            )?;
            let doc = crate::cycle::custody::strict_json(bytes)?;
            require(
                doc["name"] == *name && (schema.is_empty() || doc["schema"] == schema),
                "corpus name/schema",
            )?;
            if let Some(pin) = pin {
                let key = match kind {
                    "world" => "world_implementation_sha256",
                    "clock" => "clock_implementation_sha256",
                    "network" => "network_implementation_sha256",
                    _ => return Err("unexpected broker pin".into()),
                };
                require(doc[key] == pin, "corpus captured implementation header")?;
            }
            documents.insert(name.clone(), doc);
        }
        if matches!(kind, "world" | "clock") {
            let categories = documents
                .values()
                .map(|v| v["source"].as_str().ok_or("corpus source category"))
                .collect::<Result<BTreeSet<_>, _>>()?;
            require(
                categories == BTreeSet::from(["scripted", "generated", "recorded"]),
                "corpus source categories",
            )?;
        }
        Ok(Self {
            kind: kind.into(),
            documents,
        })
    }

    /// Compares complete current corpus execution rows; no historical pin normalization.
    pub(crate) fn receive(&self, stdout: &[u8]) -> Result<(), String> {
        require(
            self.kind != "broker",
            "broker receiving requires reopened network evidence",
        )?;
        self.receive_rows(stdout)
    }

    pub(crate) fn receive_broker(
        &self,
        stdout: &[u8],
        raw: &BTreeMap<String, Vec<u8>>,
        frozen: &BTreeMap<String, String>,
    ) -> Result<(), String> {
        require(self.kind == "broker", "foreign broker corpus")?;
        let mut expected = BTreeSet::new();
        for (name, doc) in &self.documents {
            if doc["network"].is_null() {
                continue;
            }
            for node in doc["nodes"].as_array().ok_or("broker nodes")? {
                let node = node.as_str().ok_or("broker node")?;
                expected.insert(format!("{name}.{node}"));
            }
        }
        require(
            raw.keys().cloned().collect::<BTreeSet<_>>() == expected
                && raw.keys().eq(frozen.keys()),
            "broker network file inventory",
        )?;
        for (name, bytes) in raw {
            require(sha(bytes) == frozen[name], "broker network input identity")?;
        }
        self.receive_rows(stdout)?;
        let text = std::str::from_utf8(stdout).map_err(|e| e.to_string())?;
        for line in text.lines() {
            let Some((_, value)) = line.split_once("GOMAD_CORE_OBSERVATION ") else {
                continue;
            };
            let row = crate::cycle::custody::strict_json(value.as_bytes())?;
            if row["case"] != "network.broker.replay" {
                continue;
            }
            let name = row["inputs"]["name"].as_str().ok_or("broker corpus name")?;
            let doc = self.documents.get(name).ok_or("foreign broker corpus")?;
            let actual = &row["observed"];
            fields(actual, &["outputs", "world_base64", "evidence"])?;
            let evidence = actual["evidence"]
                .as_object()
                .ok_or("broker node evidence")?;
            let nodes = if doc["network"].is_null() {
                Vec::new()
            } else {
                doc["nodes"]
                    .as_array()
                    .ok_or("broker nodes")?
                    .iter()
                    .map(|v| v.as_str().ok_or("broker node"))
                    .collect::<Result<Vec<_>, _>>()?
            };
            require(
                evidence.keys().map(String::as_str).collect::<BTreeSet<_>>()
                    == nodes.iter().copied().collect(),
                "broker node evidence owners",
            )?;
            if nodes.is_empty() {
                continue;
            }
            let model: NetworkModel =
                serde_json::from_value(doc["network"].clone()).map_err(|e| e.to_string())?;
            for node in nodes {
                let encoded = evidence[node].as_str().ok_or("broker evidence encoding")?;
                let wire =
                    crate::qualify::cells::base64(encoded).ok_or("broker evidence base64")?;
                require(
                    wire == raw[&format!("{name}.{node}")],
                    "broker node network bytes",
                )?;
                codec::observe_network_evidence(&wire, 8 << 20, &model, &model)?;
            }
        }
        Ok(())
    }

    #[expect(
        clippy::too_many_lines,
        reason = "Each current corpus row is checked against its complete independently frozen response and state"
    )]
    fn receive_rows(&self, stdout: &[u8]) -> Result<(), String> {
        let mut required = BTreeMap::new();
        for (name, doc) in &self.documents {
            match self.kind.as_str() {
                "world" | "clock" => {
                    required.insert(
                        (format!("{}.corpus-replay", self.kind), name.clone(), None),
                        1 + usize::from(self.kind == "world" && !doc["final"].is_null()),
                    );
                    if self.kind == "clock" && doc["source"] == "generated" {
                        required.insert(
                            (
                                "generated_clock_repeated_execution".into(),
                                name.clone(),
                                None,
                            ),
                            1,
                        );
                    }
                    if self.kind == "world" && !doc["final"].is_null() {
                        required.insert(
                            (
                                "final_world_wire_owner_round_trip".into(),
                                name.clone(),
                                None,
                            ),
                            1,
                        );
                    }
                }
                "network" => {
                    required.insert(
                        (
                            if doc["creation_error"]
                                .as_str()
                                .ok_or("network corpus creation error")?
                                .is_empty()
                            {
                                "network.corpus.initial"
                            } else {
                                "network.corpus.creation"
                            }
                            .into(),
                            name.clone(),
                            None,
                        ),
                        1,
                    );
                    for index in 0..doc["steps"].as_array().map_or(0, Vec::len) {
                        required.insert(
                            (
                                "network.corpus.step".into(),
                                name.clone(),
                                Some(index as u64),
                            ),
                            1,
                        );
                    }
                }
                "broker" => {
                    required.insert(("network.broker.replay".into(), name.clone(), None), 1);
                }
                _ => return Err("foreign corpus kind".into()),
            }
        }
        let mut seen = BTreeMap::new();
        let text = std::str::from_utf8(stdout).map_err(|e| e.to_string())?;
        for line in text.lines() {
            let Some((_, raw)) = line.split_once("GOMAD_CORE_OBSERVATION ") else {
                continue;
            };
            let row = crate::cycle::custody::strict_json(raw.as_bytes())?;
            let case = row["case"].as_str().ok_or("corpus observation case")?;
            let relevant = match self.kind.as_str() {
                "world" => {
                    case == "world.corpus-replay" || case == "final_world_wire_owner_round_trip"
                }
                "clock" => {
                    case == "clock.corpus-replay" || case == "generated_clock_repeated_execution"
                }
                "network" => case.starts_with("network.corpus."),
                "broker" => case == "network.broker.replay",
                _ => false,
            };
            if !relevant {
                continue;
            }
            fields(&row, &["schema", "case", "inputs", "observed"])?;
            require(
                row["schema"] == "gomad.core-predicate-observation/v1",
                "corpus observation schema",
            )?;
            let input = &row["inputs"];
            let actual = &row["observed"];
            let name = input[if matches!(self.kind.as_str(), "world" | "clock") {
                "corpus"
            } else {
                "name"
            }]
            .as_str()
            .ok_or("corpus observation name")?;
            let index = if case == "network.corpus.step" {
                Some(integer(&input["index"])?)
            } else {
                None
            };
            let key = (case.to_owned(), name.to_owned(), index);
            let count = seen.entry(key.clone()).or_insert(0usize);
            *count += 1;
            require(
                required.get(&key).is_some_and(|n| *count <= *n),
                "unknown/duplicate corpus observation",
            )?;
            let doc = self.documents.get(name).ok_or("foreign corpus document")?;
            match case {
                "world.corpus-replay" | "clock.corpus-replay" => {
                    require(
                        input == &json!({"corpus":name,"document":doc}),
                        "corpus substituted inputs",
                    )?;
                    let steps = doc["steps"]
                        .as_array()
                        .ok_or("corpus steps")?
                        .iter()
                        .map(|s| s["expected"].clone())
                        .collect::<Vec<_>>();
                    let mut expected = json!({"steps":steps});
                    if self.kind == "world" {
                        expected["creation_error"] = doc["creation_error"].clone();
                    }
                    require(actual == &expected, "corpus actual API response/state")?;
                }
                "generated_clock_repeated_execution" => {
                    require(
                        input == &json!({"corpus":name,"clock":doc["clock"],"steps":doc["steps"]}),
                        "clock repeated inputs",
                    )?;
                    let steps = doc["steps"]
                        .as_array()
                        .ok_or("clock steps")?
                        .iter()
                        .map(|s| s["expected"].clone())
                        .collect::<Vec<_>>();
                    require(
                        !steps.is_empty() && actual == &json!({"first":steps,"second":steps}),
                        "clock repeated raw responses",
                    )?;
                }
                "final_world_wire_owner_round_trip" => Self::roundtrip(name, doc, input, actual)?,
                "network.corpus.creation" | "network.corpus.initial" | "network.corpus.step" => {
                    let expected = if case.ends_with("creation") {
                        json!({"error":doc["creation_error"]})
                    } else if case.ends_with("initial") {
                        doc["initial"].clone()
                    } else {
                        doc["steps"]
                            .get(
                                usize::try_from(index.ok_or("network step index")?)
                                    .map_err(|_| "network step index")?,
                            )
                            .ok_or("network step index")?["expected"]
                            .clone()
                    };
                    require(actual == &expected, "network corpus API response/state")?;
                }
                "network.broker.replay" => {
                    require(
                        actual["outputs"] == doc["outputs"]
                            && actual["world_base64"] == doc["world_base64"],
                        "broker ordered outputs/World",
                    )?;
                }
                _ => return Err("foreign corpus case".into()),
            }
        }
        require(seen == required, "corpus harness omitted cases")
    }

    fn roundtrip(name: &str, doc: &Value, input: &Value, actual: &Value) -> Result<(), String> {
        let decode = |key: &str| -> Result<Vec<u8>, String> {
            crate::qualify::cells::base64(actual[key].as_str().ok_or("World encoded observation")?)
                .ok_or("World base64".into())
        };
        let raw = decode("expanded_base64")?;
        let restored = decode("restored_base64")?;
        let envelope = decode("envelope_base64")?;
        require(
            input == &json!({"corpus":name,"expanded_sha256":sha(&raw)})
                && raw == restored
                && doc["final"]["expanded_base64"] == actual["expanded_base64"]
                && doc["final"]["expanded_sha256"] == sha(&raw),
            "World final raw preimage",
        )?;
        let c = &doc["config"];
        let broker: BrokerConfig=serde_json::from_value(json!({"seed":c["seed"],"limits":{
            "max_requests":c["max_requests"],"max_events":c["max_events"],"max_queued_events":c["max_queued_events"],
            "max_transitions":c["max_transitions"],"max_payload_bytes":c["max_payload_bytes"],"max_string_bytes":c["max_string_bytes"]}})).map_err(|e|e.to_string())?;
        let admitted = world(&raw, "", &broker, &sha(&raw))?;
        require(
            world(
                &envelope,
                "gomad/world/compressed-snapshot/v2",
                &broker,
                &sha(&envelope),
            )?
            .summary()
                == admitted.summary(),
            "World compressed admission",
        )?;
        require(
            codec::encode_world(&raw, "gomad/world/compressed-snapshot/v2")? == envelope,
            "World same-candidate encoding",
        )?;
        let summary = admitted.summary();
        let expected = &doc["steps"]
            .as_array()
            .ok_or("World steps")?
            .last()
            .ok_or("World final step")?["expected"]["state"];
        for key in ["now", "payload_bytes", "transcript_digest", "state_digest"] {
            require(summary[key] == expected[key], "World final state")?;
        }
        require(
            summary["next_ids"]
                == json!([
                    expected["next_request_id"],
                    expected["next_event_id"],
                    expected["next_transition"]
                ]),
            "World final next IDs",
        )?;
        require(
            actual
                == &json!({"expanded_base64":actual["expanded_base64"],"envelope_base64":actual["envelope_base64"],"restored_base64":actual["restored_base64"],
            "envelope_sha256":sha(&envelope),"restored_sha256":sha(&restored),"state_digest":summary["state_digest"],"next_ids":summary["next_ids"]}),
            "World observed owner outputs",
        )
    }
}
