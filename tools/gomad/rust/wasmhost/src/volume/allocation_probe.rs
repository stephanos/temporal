use super::state::{Fs, Handle, OpenFlags, VolumeConfig, VolumeLimits};

pub(crate) fn metadata<T>(measure: impl FnOnce(&Fs, &Handle) -> T) -> T {
    let mut fs = Fs::new(0);
    fs.configure_volumes(
        &[VolumeConfig {
            id: "data".into(),
            mount: "/data".into(),
            capacity_bytes: 1 << 20,
        }],
        VolumeLimits {
            pending_operations: 1000,
            transitions: 1000,
        },
        0,
    )
    .unwrap();
    let file = fs
        .open(
            "/data/large",
            OpenFlags {
                read: true,
                write: true,
                create: true,
                ..OpenFlags::default()
            },
            0o600,
            0,
        )
        .unwrap();
    fs.handle_resize(&file, 1 << 19, 0).unwrap();
    measure(&fs, &file)
}
