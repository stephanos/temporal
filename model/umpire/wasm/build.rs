//! Generate the oracle protocol from the shared schema.

fn main() -> std::io::Result<()> {
    println!("cargo:rerun-if-changed=../proto/umpire/oracle/v1/oracle.proto");
    prost_build::compile_protos(&["../proto/umpire/oracle/v1/oracle.proto"], &["../proto"])
}
