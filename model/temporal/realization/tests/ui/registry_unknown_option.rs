umpire_temporal::registry! {
    model { specimen }
    actors { enum Client { #[metadata(temporal(event = "event", evidence = event, unknown = value))] First } }
    body { #[registry(inputs = ())] impl Binding {} }
}
fn main() {}
