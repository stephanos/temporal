use super::*;

fn fixture() -> helper::Evidence {
    helper::Evidence::decode(include_bytes!(
        "../../../wasmhost/tests/testdata/inspection/source-five-decisions.json"
    ))
    .unwrap()
}

fn request() -> InspectionRequest {
    let mut options = Options::default();
    options.set("check", "healthy").unwrap();
    options.set("choice-range", "0:4").unwrap();
    options.set("search-attempts", "10").unwrap();
    options.set("search-timeout", "1s").unwrap();
    options
        .request(Path::new("/retained-source-fixture"))
        .unwrap()
        .unwrap()
}

#[test]
fn search_admission_refuses_all_original_requests_before_evaluation() {
    for variant in [
        "unknown-check",
        "invalid-range",
        "zero-attempts",
        "zero-wall",
        "unreachable-choice",
        "nonmonotonic",
    ] {
        let mut evidence = fixture();
        let mut request = request();
        let search = request.search.as_mut().unwrap();
        match variant {
            "unknown-check" => search.check = "absent".into(),
            "invalid-range" => search.first = 5,
            "zero-attempts" => search.attempt_budget = 0,
            "zero-wall" => search.wall_time = 0,
            "unreachable-choice" => search.last = 5,
            "nonmonotonic" => {
                search.mode = "bisect".into();
                evidence.incarnations[0].registry.entries[0].monotonic = false;
            }
            _ => unreachable!(),
        }
        let error = search_admission(&evidence, &request, request.search.as_ref().unwrap())
            .err()
            .expect(variant);
        println!(
            "{}",
            json!({"schema":"gomad.native-inspection-predicate/v1",
                "original_root":"TestInspectionSearchRejectsRequestsBeforeEvaluation",
                "variant":variant,"observed":{"error":error.message,"search_entered":false}
            })
        );
    }
}

#[test]
fn retained_resolve_refuses_observation_coordinate_before_creating_control() {
    let mut evidence = fixture();
    let mut request = request();
    request.search = None;
    request.stop = Stop {
        kind: "choice".into(),
        value: 2,
    };
    let valid =
        resolve(&evidence, &request, None).unwrap_or_else(|error| panic!("{}", error.message));
    assert_eq!(valid.target.incarnation, 2);
    assert_eq!(valid.expected.trace, 2);
    evidence.boundaries[2].trace = 1;
    evidence.boundaries[2].cuts[0].choice_records = 1;
    evidence.boundaries[2].cuts[0].choice_bytes = 96;
    let error = resolve(&evidence, &request, None)
        .err()
        .expect("invalid choice mapping admitted");
    assert!(error.message.contains("decision/trace mapping"));
    println!(
        "{}",
        json!({"schema":"gomad.native-inspection-predicate/v1",
            "original_root":"TestInspectionChoiceMappingUsesDecisionsRatherThanObservations",
            "variant":"production-retained-resolve-gate","observed":{"error":error.message}
        })
    );
}
