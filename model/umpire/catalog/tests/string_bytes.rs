//! Byte limits retain named request expectations through macro expansion.

#[derive(Clone, Default)]
struct Request {
    text: String,
}

umpire_catalog::annotate! {
    catalog controls;
    message request request: Request = "example.Request" {
        text: Text[min_bytes = 2, max_bytes = 4] => Accepts;
    }
}

#[test]
fn string_byte_rules_expand_into_executable_request_metadata()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::request()?;
    let field = catalog.fields().next().ok_or("missing field")?;
    assert_eq!(field.constraints(), ["string.min_bytes", "string.max_bytes"]);
    assert_eq!(field.expectations.len(), 2);
    assert_eq!(umpire_catalog::validate(&Request { text: "🙂".into() }, &catalog), []);
    assert_eq!(
        umpire_catalog::validate(&Request { text: "é🙂".into() }, &catalog)[0].constraint,
        "string.max_bytes"
    );
    let mut message = Request::default();
    umpire_catalog::fill(&mut message, &catalog, &mut umpire_catalog::Rng::new(7))?;
    assert!((2..=4).contains(&message.text.len()));
    Ok(())
}
