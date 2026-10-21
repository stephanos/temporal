//! Located refusal controls for declarative realization handlers.

use umpire_temporal_annotate_syntax::{realization::Schema, realize::expand};

#[test]
fn handler_admission_rejects_ambiguous_or_unclassified_handlers()
-> Result<(), Box<dyn std::error::Error>> {
    let schema = Schema::new(&[], &[])?;
    for (body, expected) in [
        (
            "#[observe(WorkflowService::Describe)] fn first(response: Response) -> Result<Projection<Action>, Refusal> { todo!() } #[observe(WorkflowService::Describe)] fn second(response: Response) -> Result<Projection<Action>, Refusal> { todo!() }",
            "duplicate observation handler",
        ),
        (
            "#[observe(WorkflowService::Describe)] fn first(response: Response) -> Result<usize, Refusal> { todo!() }",
            "expected Projection, Step, or a state value with project",
        ),
        ("#[propose] fn first() {}", "propose is reserved"),
        (
            "#[answer(Task, ignores(input = \" \"))] fn first(task: Task) -> Result<Reply<Action>, Refusal> { todo!() }",
            "ignore reason must not be empty",
        ),
        (
            "#[answer(Task)] fn first(task: Task) -> Result<Reply<Action>, Refusal> { todo!() }",
            "answer requires at least one produces request context",
        ),
    ] {
        let item = syn::parse_str(&format!("impl Realization {{ {body} }}"))?;
        let error = expand(&syn::parse_str("activity")?, item, &schema)
            .err()
            .ok_or("expected handler admission refusal")?;
        assert!(error.to_string().contains(expected), "{error}");
    }
    Ok(())
}

#[test]
fn generated_enum_wildcards_are_refused_without_rejecting_byte_policy_fallbacks()
-> Result<(), Box<dyn std::error::Error>> {
    let mut schema = Schema::default();
    schema.enums.insert(
        "test.Status".into(),
        umpire_temporal_annotate_syntax::realization::Enumeration {
            rust_type: "generated :: Status".into(),
            variants: vec!["generated :: Status :: Ready".into()],
        },
    );
    for body in [
        "fn decode(value: generated::Status) { match value { generated::Status::Ready => (), _ => () } }",
        "fn decode(value: generated::Status) { match value { _ => () } }",
        "fn decode(value: generated::Status) { match &value { _ => () } }",
        "fn decode(value: (generated::Status, generated::Status)) { match value { (generated::Status::Ready, generated::Status::Ready) => (), (generated::Status::Ready, _) => () } }",
    ] {
        let item = syn::parse_str(&format!("impl Realization {{ {body} }}"))?;
        let error = expand(&syn::parse_str("activity")?, item, &schema)
            .err()
            .ok_or("expected generated enum wildcard refusal")?;
        assert!(error.to_string().contains("wildcard"), "{error}");
    }
    let bytes = syn::parse_str(
        "impl Realization { fn policy(value: &[u8]) { match value { b\"Complete\" => (), _ => () } } }",
    )?;
    assert!(expand(&syn::parse_str("activity")?, bytes, &schema).is_ok());
    Ok(())
}
