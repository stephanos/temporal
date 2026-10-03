# fn-110.3 crypto/syscall relocation: local developmental evidence

Host: linux/arm64 (development only; not a qualified Gomad platform). Both trees were
built with an uncommitted descriptor shim adding linux/arm64 to supported_platforms
(and its shim-dependent regenerated outputs). The shim is never committed.

- Baseline: task 3 start tree (HEAD 331b75bb6) + shim, toolchain key 464561b56f99e7b17b6a5b442f9898c79c4ed16804f6383dcfd86d65ba1147d9 (359 s build).
- Candidate: task 3 tree + shim, toolchain key 6879442cb531988eed05718d31e5e6c0482264c36b95c66d00e7e1dc0072bda0 (225 s build; archive overlay collision check passed).
- Patch -U3: 39,837 B / 1,169 lines (sha256 11450b94…ea3ae) -> 38,362 B / 1,112 lines (sha256 86def26a…76ea5c).
- New overlay files: crypto/rand/gomad.go 469 B/19 L; syscall/gomad_env_unix.go 549 B/18 L; syscall/gomad_unix.go 502 B/18 L.

## Baseline vs candidate (byte-identical outputs)

| Probe | sha256 (both trees) |
| --- | --- |
| profile io_entropy + environment, seeds 1/11/17/999 (runner execution.Run with I/O transcript) | fe8d14b82a16ef58ff0c4cae19e175b9b372d6e858df3eb86b1fe8d9ff0b435d |
| seeded (GOMADSEED 1, 17) and disabled env / Write(9) / stdout+stderr / crypto probes | f049e9635aea4d112c58f590d09a609a095f27648f93974b162f2641ed0eeff5 |
| gotest TestSeedReachesTestBinary seed 17 + TestDisabledCompatibility | 8ba3dc3883b4a0664c4de94364e7df20ce415263ca6eb35e438b717887629244 |
| disabled-mode upstream crypto/rand + syscall per-test dispositions (378 cases) | identical; TestPrlimitFileLimit FAIL on both (Linux-only; existing rlimit.go hunk) |

## Candidate-only focused tests

- TestProfileEntropyIsIndependentOfScheduleSeed PASS; TestToolchainLeavesFD5ForProcessesWithoutIOProfile PASS; TestIOProfileFailureArtifactReplaysExactly PASS.
- target/internal/livecap TestPinnedToolchain* SKIP (darwin/arm64 only) -> native gate incomplete.
- go list syscall source selection: gomad_env_unix.go wherever env_unix.go builds (incl. js/wasm, wasip1, plan9); gomad_unix.go wherever syscall_unix.go builds (unix); neither on windows.

## Probe outputs (candidate; baseline identical)

```
fixture=./io_entropy seed=1 exit=0 term=exit records=5 transcript_sha256=73a5b28675608e7228cc90bcd77c1ba0f5322bfd5fadaad91470357cde3751bc
stdout:
63eb21de14e58516f41d6bbf3a874d170506d831aa110cc8c572973591aa567c
57CEBV4OX435FJRQ5N2HJQRZ2M
c97d6f05ee0dcd98e99e2b77e996a74b091396dd1620041fdba1555ecc1370e4
stderr:
--
fixture=./io_entropy seed=11 exit=0 term=exit records=5 transcript_sha256=73a5b28675608e7228cc90bcd77c1ba0f5322bfd5fadaad91470357cde3751bc
stdout:
63eb21de14e58516f41d6bbf3a874d170506d831aa110cc8c572973591aa567c
57CEBV4OX435FJRQ5N2HJQRZ2M
c97d6f05ee0dcd98e99e2b77e996a74b091396dd1620041fdba1555ecc1370e4
stderr:
--
fixture=./io_entropy seed=17 exit=0 term=exit records=5 transcript_sha256=73a5b28675608e7228cc90bcd77c1ba0f5322bfd5fadaad91470357cde3751bc
stdout:
63eb21de14e58516f41d6bbf3a874d170506d831aa110cc8c572973591aa567c
57CEBV4OX435FJRQ5N2HJQRZ2M
c97d6f05ee0dcd98e99e2b77e996a74b091396dd1620041fdba1555ecc1370e4
stderr:
--
fixture=./io_entropy seed=999 exit=0 term=exit records=5 transcript_sha256=73a5b28675608e7228cc90bcd77c1ba0f5322bfd5fadaad91470357cde3751bc
stdout:
63eb21de14e58516f41d6bbf3a874d170506d831aa110cc8c572973591aa567c
57CEBV4OX435FJRQ5N2HJQRZ2M
c97d6f05ee0dcd98e99e2b77e996a74b091396dd1620041fdba1555ecc1370e4
stderr:
--
fixture=./environment seed=1 exit=0 term=exit records=2 transcript_sha256=48f3c83cc8e6c8377f0bc3083a7630637b3dbc501073eac6d0443348635b2023
stdout:
init=["TZ=UTC" "FN110_HOST_SECRET=leak"] main=["TZ=UTC" "FN110_HOST_SECRET=leak"]
stderr:
--
fixture=./environment seed=11 exit=0 term=exit records=2 transcript_sha256=48f3c83cc8e6c8377f0bc3083a7630637b3dbc501073eac6d0443348635b2023
stdout:
init=["TZ=UTC" "FN110_HOST_SECRET=leak"] main=["TZ=UTC" "FN110_HOST_SECRET=leak"]
stderr:
--
fixture=./environment seed=17 exit=0 term=exit records=2 transcript_sha256=48f3c83cc8e6c8377f0bc3083a7630637b3dbc501073eac6d0443348635b2023
stdout:
init=["TZ=UTC" "FN110_HOST_SECRET=leak"] main=["TZ=UTC" "FN110_HOST_SECRET=leak"]
stderr:
--
fixture=./environment seed=999 exit=0 term=exit records=2 transcript_sha256=48f3c83cc8e6c8377f0bc3083a7630637b3dbc501073eac6d0443348635b2023
stdout:
init=["TZ=UTC" "FN110_HOST_SECRET=leak"] main=["TZ=UTC" "FN110_HOST_SECRET=leak"]
stderr:
--

mode=disabled probe=env rc=0
"host" true "" false "modeled" true false environ=<host>
mode=disabled probe=write rc=0
after -1 bad file descriptor
mode=disabled probe=out rc=0
stdout-direct
stdout 14 <nil>
stderr-direct
stderr 14 <nil>
rand ok true true true
mode=1 probe=env rc=0
"" false "" false "modeled" true false environ=<host>
mode=1 probe=write rc=2
fatal error: GOMAD_CAPABILITY_DENIED
runtime.throw({?, ?})
runtime.gomadCapabilityGuard()
syscall.Write(0x9, {0x1d0ca11360b8, 0x6, 0x6})
main.main()
runtime.main()
mode=1 probe=out rc=0
stdout-direct
stdout 14 <nil>
stderr-direct
stderr 14 <nil>
rand ok true true true
mode=17 probe=env rc=0
"" false "" false "modeled" true false environ=<host>
mode=17 probe=write rc=2
fatal error: GOMAD_CAPABILITY_DENIED
runtime.throw({?, ?})
runtime.gomadCapabilityGuard()
syscall.Write(0x9, {0x35055f07c0b8, 0x6, 0x6})
main.main()
runtime.main()
mode=17 probe=out rc=0
stdout-direct
stdout 14 <nil>
stderr-direct
stderr 14 <nil>
rand ok true true true

## gotest seed 17
=== RUN   TestSeedReachesTestBinary
seeded order=[4 5 2 0 1 3]
--- PASS: TestSeedReachesTestBinary (0.00s)
## gotest disabled
=== RUN   TestDisabledCompatibility
GOMAD3_COMPAT gomaxprocs=12 keys=[alpha bravo charlie delta]
--- PASS: TestDisabledCompatibility (0.00s)
```
