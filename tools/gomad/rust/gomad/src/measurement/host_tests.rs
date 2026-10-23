use super::*;
const ARM: &str = "processor: 0\nFeatures: fp asimd aes crc32\nCPU architecture: 8\n";
const X86: &str = "processor: 0\nvendor_id: GenuineIntel\nflags: fpu sse sse2\n";
fn witness(os: &str, machine: &str, cpu: &str) -> HostWitness {
    HostWitness {
        kernel_os: os.into(),
        machine: machine.into(),
        kernel_version: "actual-kernel-witness".into(),
        cpuinfo: cpu.into(),
        translation: String::new(),
    }
}
#[test]
fn diagnostic_performance_host_any_native_policy() {
    for (label, os, arch, cpu, want) in [
        ("Linux arm64", "linux", "arm64", ARM.to_owned(), true),
        ("Linux amd64", "linux", "amd64", X86.to_owned(), true),
        (
            "raw Darwin arm64 unsupported shape",
            "darwin",
            "arm64",
            ARM.to_owned(),
            false,
        ),
        (
            "emulated arm64 on x86",
            "linux",
            "arm64",
            X86.to_owned(),
            false,
        ),
        (
            "missing host witness",
            "linux",
            "arm64",
            String::new(),
            false,
        ),
        (
            "32-bit ARM",
            "linux",
            "arm64",
            "Features: fp neon\nCPU architecture: 8\n".into(),
            false,
        ),
        (
            "oversized witness",
            "linux",
            "arm64",
            format!(
                "{ARM}{}",
                "\n".repeat(usize::try_from(CPUINFO_LIMIT).unwrap())
            ),
            false,
        ),
    ] {
        assert_eq!(performance_host(os, arch, &cpu), want, "{label}");
    }
    for (os, arch, machine, cpu) in [
        ("linux", "aarch64", "aarch64", ARM),
        ("linux", "x86_64", "x86_64", X86),
        (
            "darwin",
            "arm64",
            "arm64",
            "hw.optional.arm64: 1\nmachdep.cpu.brand_string: Apple M1\n",
        ),
        (
            "macos",
            "x86_64",
            "x86_64",
            "hw.optional.arm64: 0\nmachdep.cpu.brand_string: Intel\n",
        ),
    ] {
        let original = witness(os, machine, cpu);
        assert!(host_check(os, arch, &original));
        assert!(performance_host(
            os,
            arch,
            &serde_json::to_string(&original).unwrap()
        ));
        for mutation in 0..5 {
            let mut changed = original.clone();
            match mutation {
                0 => changed.machine = "unrelated-architecture".into(),
                1 => changed.kernel_os = "unrelated-os".into(),
                2 => changed.kernel_version.clear(),
                3 => changed.translation = "1".into(),
                _ => changed.cpuinfo.clear(),
            }
            assert!(
                !host_check(os, arch, &changed),
                "{os}/{arch} mutation{mutation}"
            );
        }
    }
}
#[test]
fn diagnostic_performance_host_evidence_bounds() {
    let root = std::env::temp_dir().join(format!(
        "gomad-host-bounds-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    fs::create_dir(&root).unwrap();
    let path = root.join("cpuinfo");
    for size in [CPUINFO_LIMIT, CPUINFO_LIMIT + 1] {
        let bytes = format!(
            "{ARM}{}",
            "\n".repeat(usize::try_from(size).unwrap() - ARM.len())
        );
        fs::write(&path, &bytes).unwrap();
        let (observed, admitted) = performance_host_evidence(&path, "linux", "arm64");
        if size == CPUINFO_LIMIT {
            assert_eq!(observed, bytes);
            assert!(admitted);
        } else {
            assert_eq!(observed, "");
            assert!(!admitted);
        }
    }
    fs::remove_dir_all(root).unwrap();
}
#[test]
fn actual_current_host_evidence_binds_engine_architecture() {
    if !matches!(std::env::consts::OS, "linux" | "macos") {
        return;
    }
    let root = std::env::temp_dir().join(format!(
        "gomad-host-current-{}-{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos()
    ));
    let environ = std::env::vars()
        .map(|(key, value)| format!("{key}={value}"))
        .collect::<Vec<_>>();
    let result = current(&root, &environ).unwrap();
    assert_ne!(result.boot_id, "");
    assert!(host_check(
        std::env::consts::OS,
        std::env::consts::ARCH,
        &result.witness
    ));
    assert!(!host_check(
        std::env::consts::OS,
        "unrelated-architecture",
        &result.witness
    ));
    for (path, digest) in result.inputs {
        assert_eq!(canonical::digest(&fs::read(path).unwrap()), digest);
    }
    fs::remove_dir_all(root).unwrap();
}

#[test]
fn cold_preparation_memory_witnesses_are_unique_bounded_and_overflow_checked() {
    assert_eq!(
        linux_available_memory("MemAvailable: 8388608 kB\n").unwrap(),
        8_u64 << 30
    );
    for raw in [
        "",
        "MemAvailable: 1 MB\n",
        "MemAvailable: 1 kB\nMemAvailable: 2 kB\n",
        "MemAvailable: 18446744073709551615 kB\n",
    ] {
        assert!(linux_available_memory(raw).is_err());
    }
    assert_eq!(darwin_available_memory("Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages inactive: 262144.\n").unwrap(),8_u64<<30);
    for raw in [
        "",
        "Mach Virtual Memory Statistics: (page size of 0 bytes)\nPages free: 1.\nPages inactive: 1.\n",
        "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 1.\nPages free: 1.\nPages inactive: 1.\n",
        "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 18446744073709551615.\nPages inactive: 1.\n",
    ] {
        assert!(darwin_available_memory(raw).is_err());
    }
}

#[test]
fn matched_memory_admission_rejects_malformed_witnesses_before_both_owners() {
    let valid = "MemAvailable: 8388608 kB\n";
    for (label, raw) in [
        ("empty", String::new()),
        ("missing", "MemTotal: 100 kB\n".into()),
        ("duplicate", format!("{valid}{valid}")),
        ("lowercase", "memavailable: 8388608 kB\n".into()),
        ("negative", "MemAvailable: -1 kB\n".into()),
        ("positive-sign", "MemAvailable: +1 kB\n".into()),
        ("wrong-unit", "MemAvailable: 8388608 MB\n".into()),
        ("extra-field", "MemAvailable: 8388608 kB extra\n".into()),
        ("missing-unit", "MemAvailable: 8388608\n".into()),
        (
            "spaced-duplicate",
            format!("{valid}MemAvailable : 8388608 kB\n"),
        ),
        (
            "joined-duplicate",
            format!("{valid}MemAvailable:8388608 kB\n"),
        ),
        (
            "decimal-overflow",
            "MemAvailable: 18446744073709551616 kB\n".into(),
        ),
        (
            "byte-overflow",
            "MemAvailable: 18014398509481984 kB\n".into(),
        ),
        (
            "oversized-whitespace",
            format!("{valid}{}", " ".repeat(8 << 10)),
        ),
    ] {
        for owner in ["original", "candidate"] {
            let mut dispatched = false;
            let result = matched_preparation_admitted(
                "linux",
                "1",
                || Ok(raw.clone()),
                || {
                    dispatched = true;
                    Ok(owner)
                },
            );
            assert!(result.is_err(), "{label}/{owner}: {result:?}");
            assert!(!dispatched, "{label}/{owner}: malformed witness dispatched");
        }
    }
}

#[test]
fn matched_memory_admission_preserves_explicit_mode_platform_and_dispatch_result() {
    for (platform, mode, admitted) in [
        ("darwin", "", true),
        ("linux", "", true),
        ("darwin", "1", false),
        ("linux", "1", false),
        ("linux", "0", false),
        ("linux", "true", false),
        ("linux", " 1", false),
    ] {
        let mut dispatched = false;
        let result = matched_preparation_admitted(
            platform,
            mode,
            || Err("unreadable".into()),
            || {
                dispatched = true;
                Ok("dispatched")
            },
        );
        assert_eq!(result.is_ok(), admitted, "{platform}/{mode:?}: {result:?}");
        assert_eq!(dispatched, admitted, "{platform}/{mode:?}");
    }
    for owner in ["original", "candidate"] {
        for (kilobytes, admitted) in [(8_388_607, false), (8_388_608, true), (8_388_609, true)] {
            let mut dispatched = false;
            let result = matched_preparation_admitted(
                "linux",
                "1",
                || Ok(format!("MemAvailable: {kilobytes} kB\n")),
                || {
                    dispatched = true;
                    Ok(owner)
                },
            );
            assert_eq!(result.is_ok(), admitted, "{owner}/{kilobytes}: {result:?}");
            assert_eq!(dispatched, admitted, "{owner}/{kilobytes}");
            if admitted {
                assert_eq!(result.unwrap(), owner);
            }
        }
    }
    let valid = "MemAvailable: 8388608 kB\n";
    for (size, admitted) in [(8 << 10, true), ((8 << 10) + 1, false)] {
        let raw = format!("{valid}{}", " ".repeat(size - valid.len()));
        let result = matched_preparation_admitted("linux", "1", || Ok(raw), || Ok("prepared"));
        assert_eq!(result.is_ok(), admitted, "witness size {size}");
    }
}

#[test]
fn warm_darwin_memory_admission_gates_the_actual_dispatch_callback() {
    for (label, raw, admitted) in [
        (
            "at-bound",
            "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages inactive: 262144.\n",
            true,
        ),
        (
            "below-bound",
            "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262143.\nPages inactive: 262144.\n",
            false,
        ),
        (
            "missing-inactive",
            "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 524288.\n",
            false,
        ),
        (
            "duplicate-free",
            "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages free: 262144.\nPages inactive: 262144.\n",
            false,
        ),
        (
            "malformed-header",
            "16384\nPages free: 524288.\nPages inactive: 0.\n",
            false,
        ),
        (
            "invalid-page-size",
            "Mach Virtual Memory Statistics: (page size of 0 bytes)\nPages free: 524288.\nPages inactive: 0.\n",
            false,
        ),
        (
            "overflow",
            "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 18446744073709551615.\nPages inactive: 1.\n",
            false,
        ),
    ] {
        let mut dispatched = false;
        let result = preparation_admitted(
            "macos",
            || Ok(raw.into()),
            || {
                dispatched = true;
                Ok("exit")
            },
        );
        assert_eq!(result.is_ok(), admitted, "{label}: {result:?}");
        assert_eq!(dispatched, admitted, "{label}");
        if admitted {
            assert_eq!(result.unwrap(), "exit");
        }
    }
    let valid = "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages inactive: 262144.\n";
    for (size, admitted) in [(1 << 20, true), ((1 << 20) + 1, false)] {
        let raw = format!("{valid}{}", " ".repeat(size - valid.len()));
        let mut dispatched = false;
        let result = preparation_admitted(
            "macos",
            || Ok(raw),
            || {
                dispatched = true;
                Ok("exit")
            },
        );
        assert_eq!(result.is_ok(), admitted, "witness size {size}");
        assert_eq!(dispatched, admitted, "witness size {size}");
    }
    let mut dispatched = false;
    let malformed_tail = format!("{valid}Pages free\n");
    let result = preparation_admitted(
        "macos",
        || Ok(malformed_tail),
        || {
            dispatched = true;
            Ok("exit")
        },
    );
    assert!(result.is_err());
    assert!(!dispatched);
    let result: Result<(), String> = preparation_admitted(
        "macos",
        || Ok(valid.into()),
        || Err("dispatch failed".into()),
    );
    assert_eq!(result.unwrap_err(), "dispatch failed");
}
