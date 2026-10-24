//! Projected oneofs expose source alternatives and their actual presence.

#[derive(Clone)]
enum Choice {
    Result(Vec<u8>),
    Failure(Vec<u8>),
}

#[derive(Clone, Default)]
struct Response {
    value: Option<Choice>,
}

umpire_temporal_annotate::annotate! {
    catalog controls;
    message response response: Response = "example.Response" {
        value: Bytes[max_len = 1] + Decided via Option<Vec<u8>> (
            |m: &Response| m.value.as_ref().map(|value| match value {
                Choice::Result(bytes) | Choice::Failure(bytes) => bytes.clone(),
            }),
            |m: &mut Response, bytes: Option<Vec<u8>>| {
                if let Some(bytes) = bytes {
                    match m.value.as_mut() {
                        Some(Choice::Result(value) | Choice::Failure(value)) => *value = bytes,
                        None => (),
                    }
                } else {
                    m.value = None;
                }
            }
        ) variants {
            Choice::Result = "result",
            Choice::Failure = "failure",
        };
        rules { one_of(result, failure); }
    }
}

#[test]
fn projected_oneof_tracks_source_paths_and_selected_empty_payloads()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::response()?;
    assert_eq!(
        catalog.fields().map(|field| field.path.as_str()).collect::<Vec<_>>(),
        ["example.Response.failure", "example.Response.result"]
    );
    for choice in [Choice::Result(vec![]), Choice::Failure(vec![])] {
        assert_eq!(umpire_catalog::validate(&Response { value: Some(choice) }, &catalog), []);
    }
    let missing = umpire_catalog::validate(&Response::default(), &catalog);
    assert_eq!(missing.len(), 1);
    assert_eq!(missing[0].constraint, "one_of");
    assert_eq!(missing[0].fields, ["example.Response.result", "example.Response.failure"]);
    for (choice, path) in [
        (Choice::Result(vec![1, 2]), "example.Response.result"),
        (Choice::Failure(vec![1, 2]), "example.Response.failure"),
    ] {
        let findings = umpire_catalog::validate(&Response { value: Some(choice) }, &catalog);
        assert_eq!(findings.len(), 1);
        assert_eq!(findings[0].path, path);
        assert_eq!(findings[0].constraint, "bytes.max_len");
    }
    Ok(())
}
