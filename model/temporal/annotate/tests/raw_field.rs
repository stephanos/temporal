//! Descriptor and runtime paths keep protobuf spelling for Rust keyword fields.

#[derive(Clone, Default)]
struct Message {
    r#type: String,
}

umpire_temporal_annotate::annotate! {
    catalog controls;
    message response message: Message = "example.Message" {
        r#type: Text;
        rules { required(r#type); }
    }
}

#[test]
fn keyword_field_uses_the_source_path() -> Result<(), Box<dyn std::error::Error>> {
    let catalog = controls::message()?;
    assert_eq!(catalog.fields().next().ok_or("missing field")?.path, "example.Message.type");
    assert_eq!(
        umpire_catalog::validate(&Message::default(), &catalog)[0].fields,
        ["example.Message.type"]
    );
    Ok(())
}
