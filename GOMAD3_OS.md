# Gomad v3 and system-level determinism

Research date: 2026-08-15

This report compares Gomad v3 with Hermit and Antithesis, evaluates Apple's
Linux-container path on macOS, and proposes bounded lower-level modes for both
patched Go and heterogeneous, unmodified software.
Statements labeled **Inference** or **Estimate** are design judgments, not claims
made by the cited projects.

Source snapshot:

- local Gomad commit `c63bea68e7759206397bc9bbc7f8f9d16d6b1a63`;
- Hermit
  [`c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1`](https://github.com/facebookexperimental/hermit/tree/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1);
- Apple `container`
  [`7d4ffb6cb1aed2c4eba42d5787de09162c82b591`](https://github.com/apple/container/tree/7d4ffb6cb1aed2c4eba42d5787de09162c82b591);
- Apple `containerization`
  [`5427fd21ded4b84034126caef5b3182900b4776d`](https://github.com/apple/containerization/tree/5427fd21ded4b84034126caef5b3182900b4776d);
- QEMU
  [`af06b5df2610fe5de6c02d17c17bced9e9f0d47d`](https://gitlab.com/qemu-project/qemu/-/tree/af06b5df2610fe5de6c02d17c17bced9e9f0d47d);
- rr
  [`7352eb807ed75e3b51be85fa6a27f121235dbfb0`](https://github.com/rr-debugger/rr/tree/7352eb807ed75e3b51be85fa6a27f121235dbfb0).

## Executive conclusion

The apparent difference is real, but there are three distinct levels:

1. **Gomad v3 is Go-runtime and reviewed Go-library level.** It controls Go
   scheduling, `select`, maps, timers, entropy, and selected `os`/`net` behavior
   through a pinned patched toolchain. A direct raw syscall escapes its contract.
2. **Hermit is syscall/process-tree level, not an OS or production
   hypervisor.** Linux seccomp and `ptrace` stop selected events; a user-space
   policy serializes threads, virtualizes results, and records/replays inputs,
   while Linux still implements most operations.
3. **Antithesis is machine/hypervisor level.** Its reproducibility unit is the
   complete interconnected experiment inside one deterministic VM, including
   its guest OS and simulated network/storage environment.

Apple's `container` feature adds a convenient Linux VM envelope on macOS, not
determinism. There are two viable products with different foundations:

1. For patched Go, build a pinned **Linux/arm64 Gomad platform bundle** and add
   a **syscall escape firewall** below the runtime. Keep the Go runtime as the
   scheduler and logical-time owner; use seccomp plus `ptrace` only to audit
   and deny raw escapes.
2. For **arbitrary Linux software**, do not generalize that tracer into a
   scheduler. Put the whole experiment in one single-vCPU QEMU TCG VM, use
   QEMU record/replay for exact execution replay, and add a deterministic
   fault broker for network, storage, entropy, clock, and lifecycle actions.
   Go and JVM hooks are optional exploration and observability lenses, not the
   correctness boundary.

A syscall broker alone is insufficient for the second contract. It cannot see
vDSO time reads, nondeterministic CPU instructions, or shared-memory races
between syscalls; JIT and self-modifying code also defeat static instruction
allowlists. Exact replay of genuinely unmodified, potentially racy software
needs deterministic CPU execution plus deterministic devices, or a recorder
such as `rr` that also controls scheduling and CPU events. QEMU TCG is the more
language- and guest-wide bounded substrate.

This fits the existing roadmap rather than replacing it: COMPAT-7 already
proposes a Linux/arm64 platform bundle, and SIM-5 already proposes a
process-backed fidelity tier
([compatibility](GOMAD3_NEXT_COMPATIBILITY.md#compat-7-platform-bundles),
[simulation](GOMAD3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier)). The
syscall layer belongs beneath the Go execution backends as defense in depth;
it does not replace Gomad's semantic network, storage, World, or fault models.
The heterogeneous backend is a separate machine-level seam that may reuse
Gomad's campaign, artifact, oracle, and fault-plan concepts, but it must not
pretend that a machine event log is a Go choice tape.

A Go firewall go/no-go spike is two to four weeks and a supported restricted
mode is roughly six to twelve person-months. A heterogeneous replay spike is
four to eight weeks; a useful, intentionally narrow system DST is approximately
eight to fourteen person-months, and broad product hardening eighteen to thirty.
Hermit-like arbitrary-binary scheduling and an Antithesis-class exploration
platform remain multi-year programs.

Do not start by writing a hypervisor. Prototype the heterogeneous contract on
QEMU TCG record/replay and accept its single-vCPU performance cost before
building higher-level fault and exploration machinery.

## Boundary comparison

| Dimension | Gomad v3 | Hermit | Antithesis |
| --- | --- | --- | --- |
| Boundary | Go runtime/compiler and inventoried standard-library entries | Linux syscalls and selected CPU events in a user-space tracer | Virtual CPU, guest machine, and deterministic devices |
| Target | Rebuilt with patched Go | Unmodified x86-64 Linux process tree | Containerized x86-64 deployment |
| Scheduling | Patched Go runtime, one P | Detcore serializes Linux threads/processes; PMU preemption when available | Hypervisor plus controlled guest-OS scheduling |
| Time | Go runtime `faketime` and native Go timers | Detcore logical time and selected time emulation | Hypervisor-derived virtual time; host sources hidden |
| I/O | In-memory `os`/`net`, transcript, World; raw syscalls escape | Selected syscalls emulated/transformed/ordered; Linux still does most work | Simulated/fault-injected network, storage, and hypervisor I/O |
| Replay unit | One prepared target process plus choice/I/O/World evidence | Complete traced process tree | Complete interconnected experiment in one VM |
| Exploration | Seeds, exact choice tapes/frontier, semantic guidance | Seeded chaos plus experimental record/replay/analysis | Guided branching multiverse, faults, properties, time travel |
| Platform | Qualified only on `darwin/arm64` | `x86-64 Linux` | Hosted simulated x86-64 environment |

Thus Hermit is **below the language runtime but above the kernel**; Antithesis
is **below the guest OS**; Gomad is **inside the Go runtime plus selected
standard-library boundaries**.

## Gomad v3 today

Gomad promises repeatability of runtime-controlled choices for an unchanged
toolchain, target, architecture, deterministic inputs, and seed; it does not
claim deterministic arbitrary host I/O
([architecture](tools/gomadv3/ARCHITECTURE.md#system-boundary)). Activation
forces one P, disables asynchronous preemption and the system monitor, seeds
runtime choices, and starts a virtual clock
([activation](tools/gomadv3/ARCHITECTURE.md#activation)). The runtime patch
implements seeded run-queue and `select` choices, deterministic runtime
randomness, and `faketime`
([patch](tools/gomadv3/toolchain/runtime/go1.26.4.patch)).

Time advances to the earliest native timer only when no goroutine is runnable.
A busy loop or unsupported blocking host I/O prevents logical advancement and
is bounded by Runner's wall watchdog
([quiescence](tools/gomadv3/ARCHITECTURE.md#quiescence-and-native-timers)).
This gives Gomad direct knowledge of goroutines, channels, `select`, and Go
timers that a syscall tracer has to infer indirectly.

The compiler inserts typed prologues into reviewed `os` and `net` definitions.
The current manifest inventories 129 entries: 62 modeled, 64 denied, and three
delegated. Entries bind source declarations, operations, probes,
hooks/dispositions, adapters, and fixtures
([manifest](tools/gomadv3/deterministicio/boundary/manifest.json)). Modeled
implementations provide process-local filesystem, loopback TCP, hostname, and
entropy behavior; a pinned `modernc.org/libc` adapter maps selected calls into
the same boundary
([deterministic I/O](tools/gomadv3/README.md#deterministic-io)).

The limit is explicit: this is not an OS sandbox. Raw syscalls can bypass it,
and DNS, non-loopback sockets, subprocesses, cgo, plugins, external linking,
and unrecognized native I/O are unsupported
([deterministic I/O](tools/gomadv3/README.md#deterministic-io)).

Runner already provides the control plane a lower backend needs: fresh process
and directory per seed, an empty environment, process-group termination,
bounded output, immutable identity, atomic artifacts, replay validation, World,
and ordered campaign commits
([Runner](tools/gomadv3/ARCHITECTURE.md#runner-and-process-containment)).

The current descriptor and I/O manifest support only `darwin/arm64`
([descriptor](tools/gomadv3/toolchain/version/version.json)). Replay requires
the artifact host and target platform to match the current process
([replay check](tools/gomadv3/runner/replay_operation.go)). Linux support is
therefore a new qualified platform bundle and artifact identity, not
cross-platform replay of existing Darwin artifacts.

## Hermit: syscall-level, not OS-level

Hermit runs an unmodified x86-64 Linux guest under Reverie and controls thread
scheduling, time, random data, CPUID, and selected metadata. Its README also
says it is in maintenance mode, uncommon-syscall and complex replay coverage is
incomplete, it is not a security boundary, and changing files/network responses
remain guest inputs
([README](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/README.md#L2-L25)).

```text
Hermit CLI
  -> namespaces, mounts, environment, process tree
  -> Reverie: seccomp-BPF + ptrace mechanism
  -> Detcore: scheduling, logical time, policy, record/replay
  -> ordinary Linux kernel implements most operations
```

Reverie owns tracing, registers, memory, syscall injection, and event delivery;
Detcore decides whether an event is emulated, transformed, serialized, recorded,
replayed, or passed to Linux
([architecture](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L8-L49)).
For subscribed syscalls, seccomp stops the tracee before execution. The tracer
reads registers, applies policy, then emulates, injects-and-normalizes, or
suppresses the original call
([syscall pipeline](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L173-L266)).

Hermit sees kernel entry regardless of source language, and it schedules the
whole traced process tree. The latter is much harder than interception: syscall
boundaries cannot preempt a spinning thread, so Hermit combines event check-ins
with retired-branch PMU timeslices
([scheduler](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L329-L405)).
It also handles signals, futexes, lifecycle, FDs/inodes, procfs, CPUID, and
RDTSC, while documenting remaining external inputs and instruction escapes
([instructions](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L275-L328),
[files](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L465-L490)).

Its ptrace mode has a 3–6x native wall-time planning range
([performance](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/README.md#L193-L202)).
DBI and KVM backends remain research/exploratory, and Hermit stresses that
interception without complete policy is not determinism
([backends](https://github.com/facebookexperimental/hermit/blob/c0766281cbe2fb0439e373e4bb5fd3cc3f09d7d1/docs/ARCHITECTURE.md#L50-L88)).

The lesson is not "adopt ptrace everywhere." Moving down one layer trades Go
source-inventory work for ABI, kernel-object, instruction, signal, and scheduler
work.

## rr: record one realized userspace schedule

rr is the closest open-source model for the approved “record internal
scheduling; explore external plans” contract at process scope. It records and
replays a Linux process tree, runs one application thread at a time, uses a
hardware performance counter to place asynchronous events, and records syscall
effects and other nondeterministic inputs. Serial execution lets replay
reconstruct shared-memory data races within the recorded process tree
([technical report](https://arxiv.org/html/1705.05937#S2.SS2)).

That is still not a machine boundary: the Linux kernel, devices, unrelated
processes, and uncontrolled shared-memory writers are outside the trace. rr also
requires supported CPUs/performance counters; its current README lists selected
AArch64 systems including Apple M-series, but VM use depends on virtualized
hardware counters
([pinned README](https://github.com/rr-debugger/rr/blob/7352eb807ed75e3b51be85fa6a27f121235dbfb0/README.md#system-requirements)).

**Inference:** rr is an excellent comparison and possible single-node debugger,
but qualifying its PMU assumptions inside Apple's VM would still leave network,
storage, kernel, and multi-container state outside one replay unit. QEMU is the
smaller integration risk for the heterogeneous whole-system contract, despite
its higher execution cost.

## Antithesis: whole-machine determinism

Antithesis's "Determinator" runs containers representing one or more computers
inside one VM. The reproducibility unit is the entire interconnected workload,
not one process
([hypervisor account](https://antithesis.com/blog/deterministic_hypervisor/)).
Everything inside sees one linear history; the external explorer sees a tree of
paths it can revisit and branch.

The public account describes a deterministic fork of FreeBSD `bhyve`, Intel VMX
and performance counters, virtualized guest time sources, one physical core per
VM, controlled guest-OS scheduling, and x86 `VMCALL` input/output. Network and
storage are simulated inside the same boundary
([same source](https://antithesis.com/blog/deterministic_hypervisor/)).

This required iterative characterization of CPU behavior, rare PMU and
interrupt effects, and about 50 GiB of diagnostic output in a 20-minute run.
Snapshots, time-travel debugging, guided exploration, properties, faults, and
probability analysis are additional platform components. The hosted environment
offers only a simulated x86-64 CPU and currently disallows nested hardware
virtualization
([environment](https://antithesis.com/docs/configuration/the_antithesis_environment/)).
Its overview adds intelligent guidance and user-defined properties
([overview](https://antithesis.com/docs/introduction/how_antithesis_works/));
its DST guide calls such environments complex/resource-intensive and notes
that external dependencies may still require mocks
([DST guide](https://antithesis.com/docs/resources/deterministic_simulation_testing/)).

**Inference:** recreating this on Apple arm64 is a new deterministic CPU/VMM,
device model, snapshot engine, fault framework, and search product—not a Gomad
port. That is the ocean.

## Apple's Linux VM: useful envelope, not engine

Apple `container` runs each OCI container in its own lightweight Linux VM via
Virtualization.framework
([technical overview](https://github.com/apple/container/blob/7d4ffb6cb1aed2c4eba42d5787de09162c82b591/docs/technical-overview.md#L17-L36)).
It requires Apple silicon and supports macOS 26
([README](https://github.com/apple/container/blob/7d4ffb6cb1aed2c4eba42d5787de09162c82b591/README.md)).
Custom Linux kernels are supported
([Containerization](https://github.com/apple/containerization/blob/5427fd21ded4b84034126caef5b3182900b4776d/README.md#linux-kernel)).

Nothing there promises deterministic CPU execution, clocks, scheduling, disks,
or vmnet. Treat it as packaging and isolation.

Apple supports configurable Linux capabilities; `CAP_SYS_PTRACE` is absent
from the default set
([capabilities](https://github.com/apple/container/blob/7d4ffb6cb1aed2c4eba42d5787de09162c82b591/docs/runtime-configuration.md#L4-L43)).
Its reference arm64 kernel config enables namespaces, perf events, seccomp, and
KVM
([kernel config](https://github.com/apple/containerization/blob/5427fd21ded4b84034126caef5b3182900b4776d/kernel/config-arm64)).
Those settings are not proof that a host exposes suitable PMU/ptrace semantics;
qualification must test them.

On M3+ Apple silicon, `--virtualization` exposes nested arm64 KVM with a suitable
kernel
([nested virtualization](https://github.com/apple/container/blob/7d4ffb6cb1aed2c4eba42d5787de09162c82b591/docs/runtime-configuration.md#L77-L99)).
Assume that hardware is available. It removes a KVM availability blocker, but
ordinary KVM still does not provide deterministic time, interrupts, devices,
or scheduling. The recommended Go ptrace/seccomp path does not need nested KVM.

Apple can execute amd64 image programs through Rosetta
([multiplatform images](https://github.com/apple/container/blob/7d4ffb6cb1aed2c4eba42d5787de09162c82b591/docs/multiplatform-images.md)).
Apple is precise that this runs x86-64 applications in an **arm64 Linux
distribution** through a registered interpreter, not an Intel Linux distribution
([Intel binaries in Linux VMs](https://developer.apple.com/documentation/virtualization/running-intel-binaries-in-linux-vms)).

Hermit expects x86 registers/syscalls, CPUID/RDTSC faulting, and x86 PMU
semantics. **Inference:** Rosetta is not a credible supported Hermit substrate.
Toy ptrace behavior would not establish deterministic scheduling; a native
arm64 port is substantial.

## Bounded options

### A. External Hermit adapter on native x86 Linux

Wrap Hermit execution/evidence in Gomad artifacts without forking it. This is a
useful comparison baseline, but not the Apple-silicon product path. Hermit's
choice/time model does not map cleanly to Gomad's choice tape, and enabling both
schedulers creates dual ownership.

**Estimate:** four to eight weeks for an experimental x86-only adapter. A
supported Hermit fork or arm64 port is at least one to two engineer-years.

### B. Linux/arm64 Gomad plus syscall firewall — recommended

```text
macOS
  -> one pinned Apple Linux VM per campaign
     -> Linux/arm64 Gomad Runner
        -> seccomp + ptrace syscall boundary
           -> patched Go target
              -> Gomad runtime owns scheduling/time
              -> deterministic I/O owns filesystem/network/entropy
              -> World owns modeled external events
```

Ownership must remain strict:

- Go runtime: only scheduler and logical-time authority;
- deterministic-I/O/World: domain semantics;
- syscall backend: escape detection/denial, lifecycle containment, evidence;
- Runner: campaign policy, watchdogs, output, artifacts, replay.

Calls modeled by the overlay should not reach Linux. A direct `openat`, socket,
`io_uring`, BPF, perf, device, process, or similar escape becomes a stable
unsupported result before effect.

A deep `systemboundary` module should expose only host qualification,
launch/stop, typed syscall policy, and a bounded semantic transcript. It should
privately own Linux ABI decoding, tracee memory, thread lifecycle, seccomp
programs, and ptrace stops. Runner must not understand registers or stop states.

### C. QEMU TCG system replay — recommended for heterogeneous mode

QEMU record/replay replays full VM execution, including memory, device state,
clocks, interrupts, network input, and block operations. It supports Arm and
AArch64 as well as x86-64, replay-associated snapshots, and reverse debugging.
Every block device must use `blkreplay`, and every network backend must have a
replay filter
([pinned replay documentation](https://gitlab.com/qemu-project/qemu/-/blob/af06b5df2610fe5de6c02d17c17bced9e9f0d47d/docs/system/replay.rst)).
Instruction counting is essential to replay and incompatible with
multi-threaded TCG
([pinned `icount` design](https://gitlab.com/qemu-project/qemu/-/blob/af06b5df2610fe5de6c02d17c17bced9e9f0d47d/docs/devel/tcg-icount.rst)).

**Inference:** one vCPU under single-threaded TCG, inside one Apple Linux VM,
is the most credible whole-system substrate without writing a hypervisor. It
does not require nested KVM and can host either AArch64 or, more slowly, x86
Linux without Rosetta's tracing mismatch. It reproduces one realized execution;
it does not supply Antithesis's fault language, properties, guided branching,
or throughput.

## Heterogeneous system DST

### Exact contract

The bounded contract should be:

- run unmodified Linux userspace—Go, JVM, C/C++, Rust, Python, Node, databases,
  proxies, clients, and their native libraries—inside one qualified machine;
- control and vary external network, DNS, storage, entropy, clock, process, and
  resource faults through typed plans;
- retain enough machine evidence to replay the exact realized run from the
  same initial state;
- make no initial promise to explore every internal thread schedule, reproduce
  multicore weak-memory behavior, or branch into a new recording from an
  arbitrary replay point; and
- bind replay to an exact platform bundle. A seed alone is not the replay
  artifact; the QEMU event log and initial machine state are authoritative.

“Arbitrary” therefore means unmodified userspace compatible with the qualified
Linux architecture, kernel, and software-only device profile. It does not mean
arbitrary kernels, kernel modules, GPUs, RDMA, physical devices, enclaves, or
nested hypervisors.

### Architecture and ownership

```text
macOS 26 on Apple silicon
  -> one pinned Apple container VM per campaign
     -> Linux/arm64 Gomad system runner
        -> machine backend: QMP, snapshots, logs, artifacts
        -> QEMU system emulation
           - TCG, one vCPU, single TCG thread, icount record/replay
           - immutable base image + per-run qcow2 overlay + blkreplay
           - replay-filtered NIC and replayed serial control channel
           -> one pinned Linux guest
              - all application containers/processes
              - network namespaces, router, and DNS
              - per-service volumes and resource controls
              - privileged fault/result agent
```

The entire distributed experiment must be inside the **inner QEMU VM**. Putting
each service in a separate Apple container VM would leave macOS scheduling and
vmnet ordering outside the replay unit. The Apple VM is a packaging and Linux
portability envelope; QEMU is the exactness boundary.

Ownership must be singular:

- **QEMU** owns architectural instruction order, interrupts, virtual time,
  device completion order, input recording, and exact replay.
- **Machine backend** owns the pinned command line and device allowlist, QMP
  lifecycle, start-state snapshots, replay validation, and artifact assembly.
- **Guest fault agent** owns process/container lifecycle, network/DNS policy,
  resource controls, and supported storage faults inside the replayed guest.
- **Scenario broker** turns a seed and typed plan into fault commands and
  records semantic intent plus realized acknowledgements.
- **Host runner** owns wall-time safety and resource cleanup only. A host
  watchdog may abort a run, but the abort is an external event that must be
  recorded if the result is retained.
- **Language hooks** may contribute choices, coverage, and properties. They do
  not own machine time or establish replay correctness.

This should be a separate `machinebackend`-style deep module rather than an
extension of the ptrace implementation. Its public seam needs prepare, record,
replay, stop, snapshot, and artifact operations; QMP details, QEMU command
construction, disk graphs, replay positions, and device qualification remain
private.

### Boundary mediation

| Source | Required treatment in exact mode |
| --- | --- |
| CPU and topology | Pin QEMU build, machine type, CPU model/features, TCG single-thread mode, and one vCPU. Disable hotplug, PMU, host CPU passthrough, hardware RNG, and unqualified extensions. |
| Time | Derive guest virtual time from QEMU instruction count and record host-clock-derived device reads. Expose no live host clock through a shared device. |
| Randomness | Use a qualified virtual RNG with seeded or recorded bytes. Mask unqualified CPU RNG instructions and host entropy paths. |
| Network | Put all services and the router in the inner guest. Apply a replay filter to every NIC backend; disable live backends during replay. |
| DNS | Run resolver and authoritative fixtures in the guest. Inject replies, delay, failure, and cache state there. |
| Storage | Use content-addressed immutable bases, per-run qcow2 overlays, and `blkreplay` on every writable device. Forbid writable host mounts and shared filesystems. |
| Signals and lifecycle | Deliver kill, pause, restart, and crash actions through the guest agent over a qualified replayed serial channel. |
| Resources | Pin RAM and device topology. Apply cgroup, PID, FD, memory-pressure, CPU-throttle, and prepared ENOSPC/I/O faults in the guest; never use incidental host exhaustion. |
| External services | Prefer an in-guest replica or deterministic proxy. Live ingress may be recorded through a filtered NIC, but is unavailable during replay. |

The syscall firewall is not part of this correctness path. `io_uring`, futexes,
vDSO calls, JIT code, and shared-memory synchronization execute normally inside
the guest. They become replayable only because CPU execution and every relevant
device completion are replayed. A device or host channel that bypasses QEMU's
replay contract must fail qualification.

### Fault and replay protocol

Recording and replay must deliberately differ:

1. The recorder restores a quiescent initial snapshot and a fresh disk overlay.
2. It sends each typed fault command once over the replayed control channel.
   Triggers use scenario ordinals or guest events, never host wall time.
3. QEMU records the actual input at its realized instruction count. The guest
   agent applies it and emits a stable acknowledgement and observation event.
4. The artifact retains both layers: semantic plan/target/parameters and the
   realized QEMU position, acknowledgement, and outcome.
5. Replay restores the same state and reads the immutable QEMU log. The host
   must **not** send fault commands again; QEMU injects the recorded bytes.
6. Replay requires complete log consumption, matching acknowledgement and event
   streams, terminal state, output hashes, and selected disk/state digests.
7. A mismatch is a replay divergence, not a flaky test retry.

The artifact identity must include:

- QEMU executable hash, source revision, replay format, complete command line,
  machine/CPU model, accelerator, device graph, firmware, and vCPU/RAM values;
- guest architecture, kernel, initrd, base disk and snapshot identities, OCI
  image digests, configuration, initial volumes, and topology;
- machine backend, scenario broker, fault agent, oracle, and policy versions;
- seed, semantic fault tape, QEMU replay log, overlays, guest event stream,
  output, bounds, and terminal digests; and
- proof that no writable host mount, passthrough device, unfiltered NIC/block
  backend, or live replay input was attached.

QEMU changes the replay-log version when its format changes, and its device
contract requires state-changing callbacks to be deterministic or synchronized
with the replay log
([developer replay design](https://gitlab.com/qemu-project/qemu/-/blob/af06b5df2610fe5de6c02d17c17bced9e9f0d47d/docs/devel/replay.rst)).
Exactness means identical architectural guest execution within this qualified
configuration, not matching host wall time or physical microarchitecture.

### Snapshots and exploration

Keep three concepts separate:

1. **Baseline snapshot:** a quiescent CPU/RAM/device/disk state cloned into a
   new overlay to start independent recordings with different plans.
2. **Replay checkpoint:** a position-specific snapshot that accelerates seek or
   reverse debugging within one immutable replay log.
3. **Branch:** restoring a mid-run state and starting a different future.

QEMU documents the first two. Branching from the middle of a replay into a new
recording is not established by the cited interface and is a research spike,
not an MVP promise. Initially replay from the start and parallelize independent
runs. QEMU snapshots also require a strict device allowlist because some device
models do not completely support VM snapshots
([snapshot documentation](https://www.qemu.org/docs/master/system/images.html#vm-snapshots)).

Apple Virtualization.framework save/restore is not a substitute. The saved VM
state and disk must remain coordinated, not every configuration is savable,
and the `container` CLI does not expose a whole-VM checkpoint command. Treat an
outer Apple snapshot only as a future boot optimization, never as replay
evidence.

Exploration v1 varies external plans: packet loss/delay/partitions, resolver
behavior, process kill/pause/restart, resource pressure, whole-machine clock
jumps, prepared disk-full/error states, and workload inputs. It does not vary
the Linux scheduler directly. Single-vCPU TCG reconstructs the guest kernel's
one realized schedule from deterministic execution and replayed inputs; a
guest scheduler trace is useful diagnostics, not authoritative replay data.

### Go, Java, and other runtime lenses

Unmodified binaries are the baseline. A normal JDK/JRE image, JVM threads,
JIT-generated code, garbage collector, JNI libraries, and application all sit
inside the machine boundary and require no Java agent for replay. Representative
HotSpot/OpenJDK, other JIT, database, and native-library workloads still need
qualification against the pinned CPU/device profile.

Optional integrations can make exploration better:

- patched Go can export goroutine/select/timer choices and semantic coverage;
- a JVM agent/JVMTI integration can expose monitor, executor, safepoint, task,
  and coverage events;
- native compiler instrumentation can expose edge coverage and sanitizer
  properties; and
- any application can emit stable structured properties through a tiny guest
  protocol.

These are lenses, not compatibility requirements. The current Gomad Go logical
clock and deterministic-I/O overlay must not simply be enabled unchanged for a
service that talks to arbitrary peers: that would introduce a second time/I/O
universe. A later system-mode Go adapter may contribute controlled runtime
choices while Linux TCP/filesystem effects remain inside QEMU, with QEMU still
the sole machine time and replay owner.

### macOS and Apple silicon

M3+ is required only for Apple's optional nested virtualization. Exact replay
deliberately uses QEMU TCG and therefore needs neither M3+ nor KVM. KVM does not
provide QEMU's instruction-count execution model, and recording under KVM then
replaying under TCG would change the execution being reproduced. Nested KVM is
useful only for image building and explicitly non-replay smoke tests.

Running TCG inside an Apple Linux VM is double virtualization and will be slow,
but gives the same Linux runner/backend on macOS and native Linux CI. A direct
macOS QEMU backend may later remove the outer VM if it passes the same artifact
and replay contract. Newer Macs with more CPU and memory help by running many
independent single-vCPU campaigns in parallel; they do not make one replay fast
or deterministic by themselves.

Platform order should be **AArch64 first, amd64 second**. `amd64` is x86-64;
32-bit `x86` is not a useful target. AArch64 matches the Apple-silicon host,
Apple's native Linux/container path, and current Go/JVM ecosystems, and avoids
cross-ISA TCG
translation. Add a separate amd64 platform bundle only when x86-only images or
native dependencies justify its lower throughput. Artifacts remain
architecture-bound; recording on AArch64 and replaying on amd64 is unsupported.

### Deliberate exclusions

- Multiple vCPUs and multi-threaded TCG: no initial exact-replay claim; weak
  memory and true simultaneous races are underexplored.
- GPU, RDMA, VFIO, vhost-user, USB, host DMA, PMU, nested guests, CPU/device
  hotplug, and host CPU passthrough: unsupported until separately qualified.
- `virtiofs`, live bind mounts, mutable base images, external NFS/iSCSI, and
  arbitrary physical storage: forbidden in exact mode.
- `io_uring`: disabled initially unless its block/network completion paths pass
  the full device and replay qualification corpus.
- Per-process clock skew: later work. Whole-machine virtual-time faults are
  simpler; containers in one guest share a kernel clock.
- Internet and real cloud dependencies: bring them into the guest, proxy and
  record them, or declare the run non-replayable.
- Malicious targets: the nested VM improves containment, but this is a trusted
  testing system, not a hardened hostile-code service.

## Staged Go recommendation

### Stage 0: capability spike — 2–4 weeks

Use a disposable Apple-container image; add no product API. Prove:

1. native Linux/arm64 ptrace of clone/fork/exec/exit and seccomp events with
   minimum capabilities;
2. `PTRACE_O_EXITKILL`, tracee-memory access, pidfds/process groups, and clean
   cancellation;
3. which PMU events the available Apple-silicon host exposes, without making
   the MVP depend on them;
4. the Go runtime patch builds and neutral fixtures pass on Linux/arm64 with the
   deterministic-I/O overlay initially disabled;
5. raw `openat`, `connect`, `getrandom`, and `io_uring_setup` stop before effect;
6. fixed target/seed runs produce identical semantic classifications.

Exit only if a pinned image/kernel recipe works on macOS 26 and native
Linux/arm64 CI, tracer death reaps the tree, and unsupported behavior fails
closed rather than hanging or passing through.

### Stage 1: qualify Gomad on Linux/arm64 — 2–4 person-months

- Add a Linux platform bundle to the version descriptor.
- Regenerate/review Linux standard-library declaration hashes/dispositions.
- Implement Linux publication, locking, supervision, and host-clock escape
  audits.
- Split Darwin assumptions in deterministic-I/O and the modernc adapter.
- Keep artifacts platform-bound.
- Pass current conformance, core qualification, choices, World, I/O, resume,
  and artifact gates before enabling tracing.
- Pin container image, kernel, and toolchain by digest.

### Stage 2: containment-only Go mode — 2–4 person-months

- Add the typed backend and a versioned Linux/arm64 syscall manifest.
- Allow only a reviewed pure-Go runtime syscall closure.
- Observe lifecycle events; deny every syscall outside that closure before
  execution.
- Initially deny direct time, entropy, files, network, devices, process
  creation, `io_uring`, BPF, perf, keyring, and namespace escapes.
- Record stable classifications, never PIDs, FDs, addresses, timestamps, or
  ptrace arrival order.
- Bind syscall policy, tracer, kernel config/image, and container image to
  artifacts.

Keep current pure-Go, internal-linking, one-P, cgo, plugin, subprocess, signal,
and external-network restrictions. The release's value is closing accidental
raw escapes, not expanding compatibility.

### Stage 3: model only justified syscall families — +4–12 person-months

Promote a denied syscall only for a real supported workload. Reuse existing
filesystem/network/World semantics. Each promotion needs complete ABI and
tracee-memory validation, deterministic object identity, replay ordinals and
bounds, interruption/partial-result/errno semantics, fixtures, and an audit of
alternative APIs. Do not add Linux thread scheduling or PMU preemption here.

### Stage 4: stop before a generic ptrace scheduler

Do not evolve the Go syscall firewall into the arbitrary-software backend.
Doing so requires deterministic thread/process scheduling, futex and signal
control, CPU-only preemption, vDSO/instruction interception, shared-memory
handling, `/proc`, FD/object models, and broad syscall coverage. Hermit maps
this work and also demonstrates its long tail.

**Estimate:** two to four senior engineer-years for a credible restricted
Hermit-like mode, before distributed fault exploration. Use the machine replay
path instead when unmodified heterogeneous software is the requirement.

## Staged heterogeneous recommendation

These stages are independent of the Linux/arm64 Go port except for reusable
campaign, artifact, and reporting code.

### SYS-0: replay proof — 4–8 engineer-weeks

On an Apple-silicon Mac running macOS 26, run QEMU TCG inside a pinned Apple
container VM and prove:

1. one AArch64 Linux guest containing a Go service, a JVM service, and a native
   client boots from an immutable image and completes a TCP/storage scenario;
2. one-vCPU, single-threaded TCG recording and replay produce matching guest
   events, terminal output, and selected disk/state digests;
3. every block device uses `blkreplay`, every NIC uses a replay filter, replay
   has no live backend input, and an intentional missing filter fails the gate;
4. serial fault commands, network input, entropy, clock reads, process kill,
   and clean shutdown replay at the same instruction positions;
5. a replay snapshot can seek within the log, while an attempted unsupported
   device or mutable mount is rejected before boot; and
6. the same artifact replays on a second qualified Apple-silicon host and native
   Linux/arm64 CI with the identical pinned QEMU build.

Exit with measured slowdown, artifact growth, supported QEMU device inventory,
and a written explanation for every divergence. Do not add product API or fork
QEMU during the spike.

### SYS-1: machine/replay kernel — 3–5 person-months

- Implement the machine-backend seam, pinned QEMU builder, QMP supervision,
  replayed result channel, and typed failure taxonomy.
- Add content-addressed images, qcow2 overlay lifecycle, baseline snapshots,
  replay-log publication, complete platform identity, and fail-before-boot
  validation.
- Ship one guest architecture, one machine/CPU model, one vCPU, and the
  smallest headless network/block/serial device allowlist.
- Qualify artifact corruption, truncation, disk-full, kill, cancellation,
  backend death, incomplete log consumption, and terminal-digest mismatch.

### SYS-2: controlled external faults — +4–7 person-months

- Add a minimal guest init/fault agent and import a strict subset of OCI or
  Compose configuration into namespaces inside the one guest.
- Implement stable process kill/pause/restart, network partition/drop/delay,
  DNS response/failure, cgroup pressure, and prepared volume-full/error faults.
- Record plan intent separately from realized delivery and acknowledgement;
  replay validates both streams without reinjecting commands.
- Add in-guest structured properties and output collection with hard bounds.
- Keep whole-machine clock changes coarse; defer per-process clock skew and
  low-level block corruption until their semantics are explicit.

### SYS-3: compatibility and useful exploration — +6–12 person-months

- Qualify representative OpenJDK/HotSpot, Go, native, Node/V8, Python, database,
  proxy, multiprocess, signal-heavy, JIT, and high-I/O workloads.
- Decide `io_uring` support from device-completion evidence; otherwise retain a
  pinned guest profile that disables it.
- Add seed/fault-plan generation, property and semantic coverage guidance,
  snapshot fan-out from quiescent baselines, corpus retention, minimization,
  and replay-first failure triage.
- Measure failures found per CPU-hour. Scale by independent one-vCPU VMs, not
  by adding vCPUs to a replay unit.

**Estimate:** eight to fourteen person-months for a useful qualified MVP and
eighteen to thirty person-months for a supported broader product. These are
adoption, qualification, and fault-product estimates; a maintained QEMU fork
would add roughly one to two engineers continuously and should require a proven
upstream replay defect.

## Effort summary

Order-of-magnitude estimates assuming senior Go-runtime/Linux-tracing and QEMU
systems engineers:

| Outcome | Estimate | macOS path |
| --- | --- | --- |
| Capability answer | 2–4 engineer-weeks | Apple silicon/macOS 26; no KVM needed |
| Linux/arm64 Gomad | 2–4 person-months | Yes |
| Go port + firewall | 6–12 person-months total | One pinned VM per campaign |
| Selected raw I/O replay | +4–12 person-months | If kernel APIs qualify |
| Hermit experimental wrapper | 1–2 person-months | Native x86 Linux, not reliable Rosetta |
| Hermit-like arbitrary binaries | 2–4 senior engineer-years | New arm64 design or slow x86 emulation |
| Heterogeneous replay proof | 4–8 engineer-weeks | TCG inside one Apple VM; no KVM |
| Heterogeneous machine backend | 3–5 person-months | One inner AArch64 VM |
| Useful system DST MVP | 8–14 person-months total | Exact replay plus bounded external faults |
| Supported heterogeneous product | 18–30 person-months | Broader userspace/device qualification |
| Antithesis-class platform | Specialist team, multiple years | Apple KVM does not supply determinism |

## Main risks, 10x behavior, and verification

- **Dual controllers:** in Go mode, ptrace must not schedule threads or
  virtualize time while the Go runtime owns them. In system replay mode, QEMU
  is the only machine schedule/time owner; ptrace scheduling is inactive and
  language hooks contribute evidence or nested choices only.
- **Escape aliases:** `openat2`, `io_uring`, BPF, device ioctls, vDSO, and CPU
  instructions can bypass a naive allowlist. Generate an explicit disposition
  for every pinned Linux syscall and keep negative escape probes.
- **Allowed-call provenance:** seccomp sees the syscall ABI, not the Go caller.
  Calls needed by the runtime, such as `mmap` and `futex`, cannot be denied by
  number when user code reaches the same syscall path. Preserve static
  capability-closure review and treat the firewall as defense in depth, not
  proof that every raw call is impossible. Exact call-site restrictions would
  be a separate identity-bound research project.
- **Tracer failure:** use `PTRACE_O_EXITKILL`, existing supervision, liveness
  ownership, bounded cancellation, reaping, and a final empty-tree check.
- **Host input:** never let live macOS mounts, vmnet, DNS, or host time enter a
  retained decision. Import immutable inputs, use Gomad captured mounts in Go
  mode, or route input through a replay-qualified virtual device.
- **Drift:** bind kernel/config, container images, tracer, toolchain, syscall
  manifest, architecture, Apple-container version, QEMU build and command line,
  CPU/machine model, device graph, and start state as applicable. Fail before
  replay on mismatch.
- **Security:** remain explicit that this is for trusted tests. The outer VM,
  inner VM, and firewall improve containment but do not make the tracer, guest
  agent, or QEMU control plane an adversarial sandbox.
- **Device completeness:** QEMU exactness is conditional on every state-changing
  device path satisfying its replay/save contract. Use a small allowlist,
  negative attachment tests, and corruption/kill probes; do not infer support
  from one successful run.
- **SMP coverage:** one vCPU removes true parallelism and weak-memory effects.
  It still permits many preemptive thread interleavings but may miss a class of
  races. Treat deterministic multi-vCPU execution as separate research.
- **Performance:** seccomp should trap only relevant syscalls in Go mode. TCG
  system replay will be materially slower than native execution; measure real
  mixed-language workloads and optimize failures found per CPU-hour rather
  than headline request throughput.
- **Artifact volume:** VM replay logs, snapshots, and overlays are much larger
  than choice tapes. Bound runs, chunk and content-address immutable data,
  retain semantic summaries for passes, and preserve full machine evidence only
  for selected failures or corpus entries.
- **10x seeds:** reuse one outer Apple VM per campaign only if it can launch
  fresh inner QEMU machines with isolated state. Scale system mode horizontally
  across one-vCPU recordings. Likely bottlenecks are TCG CPU, logs, snapshot I/O,
  VM RSS, and artifact publication.
- **10x services:** separate Apple VMs put vmnet/host scheduling outside one
  reproducibility unit. Whole-system experiments must put all services and the
  modeled network inside one inner replay VM. At 10x, boot time, guest memory,
  single-vCPU contention, event volume, and fault-agent backpressure need hard
  bounds and diagnostics.

Go/firewall qualification should run on native Linux/arm64 CI and macOS 26 on
Apple silicon and cover:

- namespaces, seccomp, ptrace options, tracee memory, pidfds, and process groups;
- tracer/Runner death positive controls and minimum capability checks;
- current goroutine/channel/select/timer/map/GC conformance;
- direct raw-syscall probes for every denied capability;
- proof that supported `os`/`net` calls remain in the in-memory boundary;
- clone/exec/exit races; futex/mmap/epoll/signal/interruption paths;
- entropy, clocks, arm64 counters, `/proc`, `/sys`, devices, BPF, perf, keyring,
  namespaces, and `io_uring` escapes;
- transcript/process/FD/event/output capacity overflow;
- kernel/image/manifest/tracer/toolchain/platform identity mismatch; and
- kill/cancel/disk-full/corruption at every artifact publication phase.

Every fixture should repeat with a fixed seed, vary seeds where choices exist,
and exact-replay retained successes/failures. Compare semantic outcomes, not raw
ptrace order or host identifiers.

System-replay qualification should separately cover:

- empty, CPU-only, timer/entropy, multiprocess, TCP, DNS, block, signal, and JIT
  workloads under record and exact replay;
- a representative Go service, JVM service with JNI, native client, database,
  and proxy communicating within one guest;
- an explicit disposition for every QEMU device and backend option, with
  negative tests for missing `blkreplay`, missing network filters, host mounts,
  passthrough, extra vCPUs, KVM, and multi-threaded TCG;
- snapshot/overlay/log identity, complete log consumption, seek/checkpoint,
  cross-host replay, and rejection under a changed QEMU build or machine model;
- each typed fault at start, during I/O, during acknowledgement, and at process
  or VM exit, followed by replay without host reinjection;
- guest-agent, QEMU, outer-VM, host-runner, and artifact-publication death at
  every ownership transition;
- nondeterministic positive controls that deliberately bypass a filter or
  mutate start state and must cause qualification or replay failure; and
- wall time, TCG CPU, guest instruction count, RSS, log/overlay/snapshot bytes,
  boot amortization, and failures found per CPU-hour at 1x and 10x workload.

Benchmark native Gomad, Linux Gomad without tracing, firewall mode, trap-all
diagnostics, native/KVM non-replay smoke mode, QEMU TCG record, and QEMU TCG
replay separately. Never publish native/KVM timings or outcomes as exact replay
artifacts.

## Final recommendation

Keep the patched-Go runtime path as the fast, semantically rich DST tier. Fund
its Linux/firewall spike only if closing Go raw-syscall escapes is independently
valuable; it is not the route to arbitrary software.

For the heterogeneous requirement, fund **SYS-0 next** on an Apple-silicon Mac
running macOS 26. M3+ is preferred for campaign throughput, not required. Put a
small Go + JVM + native distributed workload inside one AArch64 QEMU guest, run
single-vCPU TCG record/replay inside a pinned Apple container VM, and require
cross-host exact replay plus negative device/filter controls. If that proof
passes with tolerable throughput and artifact cost, build SYS-1 and SYS-2. This
yields a credible exact-replay and external-fault MVP in roughly eight to
fourteen person-months without writing a hypervisor.

Do not fork Hermit, port its x86 scheduler, or fork QEMU initially. Do not use
Rosetta or KVM for exact artifacts, and do not split services across Apple VMs.
M3+ makes nested KVM available; it does not make execution deterministic.

The explicit stopping line is single-vCPU replay of one realized execution plus
controlled external fault-plan exploration. Deterministic SMP, mid-run
branching into new futures, a large qualified device ecosystem, coverage-guided
multiverse search, and time-travel product UX are Antithesis-class follow-ons.
Adopt an existing platform or authorize a separate specialist-team, multi-year
program before crossing that line.
