use super::*;

/// Runs the fixed retained allocation fixture in the standalone test executable.
///
/// # Panics
/// Panics if the fixed publication cannot be constructed or opened.
pub fn run<T>(measure: impl FnOnce(&Opened, &Payload) -> T) -> T {
    let bytes = vec![0xa5; 8 << 20];
    let descriptor = Payload {
        file: "evidence/world".into(),
        bytes: bytes.len() as u64,
        sha256: canonical::digest(&bytes),
        limit: 8 << 20,
    };
    let mut publication = metadata_publication();
    let mut inputs: Inputs = serde_json::from_slice(&publication.payloads[FILE]).unwrap();
    inputs.retained[0]
        .evidence
        .insert("world".into(), descriptor.clone());
    publication
        .payloads
        .insert(FILE.into(), canonical::encode(&inputs).unwrap());
    publication.payloads.insert("evidence/world".into(), bytes);
    let fixture = ReaderFixture::new(publication);
    let opened = fixture.open();
    measure(&opened, &descriptor)
}
