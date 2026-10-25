//! Typed filler inputs and required capability compiler controls.
#![forbid(unsafe_code)]

#[allow(dead_code, reason = "shared controls include a runtime unused by the typed filler test")]
mod support;

use umpire_runtime::{
    AnyMessage as _,
    capability::{Fill, Identity, Request},
    prost_types::Any,
};

struct Binding;

impl Identity for Binding {
    type Id = u8;
    fn identify(&self, value: &Any) -> Result<u8, umpire_runtime::protocol::Refusal> {
        support::payload(value)
    }
}

impl Fill<support::Action> for Binding {
    type World = support::World;
    type Inputs = u8;
    type Request = umpire_runtime::protocol::ResetRequest;

    fn fill(
        &self,
        action: &support::Action,
        world: &support::World,
        inputs: &u8,
        identity: &u8,
    ) -> Result<Self::Request, umpire_runtime::protocol::Refusal> {
        if !matches!(action, support::Action::Deliver) || world.starts != 1 || identity != inputs {
            return Err(umpire_runtime::protocol::Refusal::malformed("crossed filler identity"));
        }
        Ok(Self::Request { configuration: Some(support::any(*inputs)) })
    }
}

#[test]
fn filler_uses_declared_world_inputs_and_admitted_identity()
-> Result<(), Box<dyn std::error::Error>> {
    let world = support::World { starts: 1 };
    let binding = Binding;
    let id = binding.identify(&support::any(7)).map_err(|e| e.message)?;
    let request =
        binding.fill(&support::Action::Deliver, &world, &7, &id).map_err(|e| e.message)?;
    assert_eq!(request.configuration, Some(support::any(7)));
    let packed = Request::pack_request(&request);
    assert_eq!(
        packed.unpack::<umpire_runtime::protocol::ResetRequest>().map_err(|e| e.message)?,
        request
    );
    assert_eq!(
        binding
            .fill(&support::Action::Deliver, &world, &8, &id)
            .err()
            .ok_or("expected refusal")?
            .message,
        "crossed filler identity"
    );
    Ok(())
}

#[test]
fn assembly_requires_each_named_capability() -> Result<(), Box<dyn std::error::Error>> {
    use std::{fs, process::Command};
    let runtime = std::path::Path::new(env!("CARGO_MANIFEST_DIR"));
    let model = runtime.ancestors().nth(3).ok_or("model workspace")?;
    let cache = std::env::var_os("CARGO_TARGET_DIR")
        .map_or_else(|| model.join("target"), std::path::PathBuf::from)
        .join("tests/runtime-capability-controls");
    let directory = cache.join(format!("fixture-{}", std::process::id()));
    fs::create_dir_all(directory.join("src"))?;
    let manifest = format!(
        r#"[package]
name = "capability-control"
version = "0.0.0"
edition = "2024"
[workspace]
[dependencies]
umpire-runtime = {{ path = {:?} }}
umpire-dsl = {{ path = {:?} }}
"#,
        runtime,
        model.join("umpire/author/dsl")
    );
    fs::write(directory.join("Cargo.toml"), manifest)?;
    fs::copy(model.join("Cargo.lock"), directory.join("Cargo.lock"))?;
    let declarations = r"
use umpire_dsl::{Machine, Steps};
use umpire_runtime::{capability::*, evidence::Rpc, NoTarget, Projection, Realize, prost_types::Any, protocol::Refusal};
struct World;
impl Machine for World {
 type State=u8; type Action=u8; type Outcome=(); type Fact=();
 fn starts(&self)->Vec<u8>{vec![0]} fn actions(&self)->Vec<u8>{vec![]}
 fn internal(&self,_:&u8)->bool{false} fn step(&self,_:&u8,_:u8)->Steps<u8,(),()>{Steps::new()}
}
struct Binding;
";
    let implementations = [
        (
            "Identity",
            "impl Identity for Binding { type Id=(); fn identify(&self,_:&Any)->Result<(),Refusal>{Ok(())} }",
        ),
        (
            "Fill",
            "impl Fill<u8> for Binding { type World=World; type Inputs=(); type Request=NoRequest; }",
        ),
        (
            "Snapshot",
            "impl Snapshot<World> for Binding { type Target=NoTarget; fn project(&self,_:&World,_:&Any,_:&u8)->Result<Projection<u8,(),NoTarget>,Refusal>{Ok(Projection::Current)} }",
        ),
        ("Evidence", "impl Evidence<Rpc> for Binding { type World=World; type View=NoEvidence; }"),
        ("ReplyPolicy", "impl ReplyPolicy<World> for Binding {}"),
    ];
    for omitted in std::iter::once(None).chain(implementations.iter().map(|(name, _)| Some(*name)))
    {
        let mut source = declarations.to_owned();
        for (name, implementation) in implementations {
            if Some(name) != omitted {
                source.push_str(implementation);
            }
        }
        source.push_str("impl Realize<World> for Binding {}\n");
        fs::write(directory.join("src/lib.rs"), source)?;
        let mut command = Command::new(env!("CARGO"));
        command
            .current_dir(&directory)
            .env("CARGO_TARGET_DIR", cache.join("target"))
            .args(["check", "--offline"]);
        if omitted.is_some() {
            command.arg("--locked");
        }
        let output = command.output()?;
        let stderr = String::from_utf8(output.stderr)?;
        if let Some(capability) = omitted {
            assert!(!output.status.success(), "missing {capability} compiled");
            assert!(stderr.contains("error[E0277]") && stderr.contains(capability), "{stderr}");
        } else {
            assert!(output.status.success(), "positive assembly failed: {stderr}");
        }
        fs::write(directory.join(format!("{}.stderr", omitted.unwrap_or("complete"))), stderr)?;
    }
    Ok(())
}
