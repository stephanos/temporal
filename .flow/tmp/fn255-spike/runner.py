"""Bounded source differential spike. No production gate registration."""
import argparse
import hashlib
import json
from pathlib import Path
import random
import shutil
import os
import signal
import subprocess
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
FRAMEWORK = ROOT / "model/umpire"


def model(seed, upper, start):
    # Parameters only choose syntax; no expected state/transition semantics are computed here.
    return f'''use umpire::dsl::machine;
#[machine]
pub mod generated {{
    use umpire::dsl::{{Action, Finite, State, check, reject, Rejection}};
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Hash, Finite)]
    pub enum Mode {{ {"#[default]" if start == "Off" else ""} Off, {"#[default]" if start == "On" else ""} On }}
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Token(#[inputs(0..=1)] pub u8);
    #[derive(Debug)]
    pub enum Fact {{ Changed }}
    #[derive(State)]
    pub struct Model {{
        pub left: Mode,
        pub right: Mode,
        #[starts(0..={seed % 2})]
        pub count: u8,
        pub token: Option<Token>,
    }}
    #[derive(Action, Copy)]
    pub enum User {{ Advance, Toggle, Clear, Refuse }}
    impl Model {{
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {{
            match action {{
                User::Advance if self.count < {upper} => {{
                    self.count += 1;
                    self.token = Some(Token({seed % 2}));
                    Some(Fact::Changed)
                }}
                User::Toggle => {{
                    self.left = if self.left == Mode::Off {{ Mode::On }} else {{ Mode::Off }};
                    self.right = if self.right == Mode::Off {{ Mode::On }} else {{ Mode::Off }};
                    Some(Fact::Changed)
                }}
                User::Clear if self.token.is_some() => {{ self.token = None; Some(Fact::Changed) }}
                User::Refuse => {{ self.refuse(); None }}
                _ => None,
            }}
        }}
        #[effect]
        fn refuse(&mut self) {{ reject!(Rejection::NotFound); }}
        #[invariant]
        fn bounded(&self) {{ check!(self.count <= {upper}); }}
    }}
}}
'''


def prepare(destination, seed, count, shapes_override=None, batch_tlc=False):
    destination.mkdir(parents=True, exist_ok=False)
    src = destination / "src"
    src.mkdir()
    shutil.copyfile(HERE / "adapters.rs", src / "adapters.rs")
    rng = random.Random(seed)
    shapes = shapes_override or [
        (rng.randrange(1 << 32), rng.randint(1, 2), rng.choice(["Off", "On"]))
        for _ in range(count)
    ]
    calls = []
    batch_calls = []
    modules = []
    for index, shape in enumerate(shapes):
        filename = f"case_{index}.rs"
        (src / filename).write_text(model(*shape))
        modules.append(f"mod case_{index};")
        batch_calls.append(f'{{ let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse("case_{index}.rs", include_str!("case_{index}.rs"))?)?; modules.push(module); }}')
        calls.append(f'''{{
        use case_{index}::generated;
        let world = generated::world;
        let identity = |a: &<generated::World as Machine>::Action| ActionValues::<generated::World>::action_value(a).0.into();
        let interpreted = Interpreted::new(world())?;
        let compiled = adapters::agree(adapters::native(&world(), identity), adapters::native(&interpreted, identity))?;
        std::fs::write(std::path::Path::new(&std::env::var("FN255_EVIDENCE")?).join("native-{index}.txt"), format!("{{compiled:#?}}"))?;
        snapshots.push(compiled.clone());
        if !{str(batch_tlc).lower()} && std::env::var_os("FN255_CONTROLS_ONLY").is_none() {{
        let source = include_str!("{filename}");
        let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse("{filename}", source)?)?;
        let checked = adapters::tlc(std::path::Path::new({json.dumps(str(HERE / "evidence/tools"))}), &module)?;
        std::fs::write(std::path::Path::new(&std::env::var("FN255_EVIDENCE")?).join("tlc-{index}.txt"), format!("{{checked:#?}}"))?;
        adapters::agree(Ok(compiled), Ok(checked))?;
        }}
        adapters::lowering_controls(world)?;
        println!("case={index} seed={shape[0]} complete");
    }}''')
    (src / "main.rs").write_text('''#![forbid(unsafe_code)]
use umpire::dsl::{Machine, ActionValues};
use umpire_checker::Interpreted;
mod adapters;
''' + "\n".join(modules) + '\nfn main() -> Result<(), Box<dyn std::error::Error>> {\n let mut snapshots = Vec::new();\n' +
        "\n".join(calls) + ('\n let mut modules=Vec::new();\n' + "\n".join(batch_calls) + f'\nadapters::batch(std::path::Path::new({json.dumps(str(HERE / "evidence/tools"))}), &modules, &snapshots)?;\n' if batch_tlc else "") + '\nOk(())\n}\n')
    dependencies = {
        "umpire": "author/umpire", "umpire-checker": "backends/native",
        "umpire-tla": "backends/tla", "umpire-tlc": "backends/tlc",
    }
    manifest = '[package]\nname="fn255-spike"\nversion="0.0.0"\nedition="2024"\n[workspace]\n[dependencies]\n'
    for name, path in dependencies.items():
        manifest += f'{name}={{path={json.dumps(str(FRAMEWORK / path))}}}\n'
    (destination / "Cargo.toml").write_text(manifest)
    (destination / "source-hashes.json").write_text(json.dumps({
        str(p.relative_to(destination)): hashlib.sha256(p.read_bytes()).hexdigest()
        for p in sorted(src.glob("*.rs"))}, indent=2))
    (destination / "inputs.json").write_text(json.dumps({"seed": seed, "shapes": shapes}, indent=2))


def observed_java(root_pid):
    processes = {}
    for folder in Path("/proc").glob("[0-9]*"):
        try:
            stat = (folder / "stat").read_text().rsplit(") ", 1)[1].split()
            command = (folder / "cmdline").read_bytes().split(b"\0")
            processes[int(folder.name)] = (int(stat[1]), [a.decode(errors="replace") for a in command if a])
        except (OSError, ValueError, IndexError):
            continue
    descendants = {root_pid}
    for _ in range(12):
        found = {pid for pid, (parent, _) in processes.items() if parent in descendants}
        if found <= descendants:
            break
        descendants |= found
    return {pid: args for pid, (_, args) in processes.items()
            if pid in descendants and args and Path(args[0]).name == "java"}


def bounded(command, timeout, environment, evidence):
    started = time.monotonic()
    observed = {}
    peak = 0
    with evidence.with_suffix(".log").open("w") as output:
        process = subprocess.Popen(command, cwd=FRAMEWORK, env=environment,
                                   start_new_session=True, stdout=output, stderr=subprocess.STDOUT)
        try:
            while process.poll() is None:
                if time.monotonic() - started > timeout:
                    raise subprocess.TimeoutExpired(command, timeout)
                if shutil.disk_usage(HERE).free < 30 * 1024**3:
                    raise RuntimeError("fn255 spike disk floor below 30 GiB")
                live = observed_java(process.pid)
                peak = max(peak, len(live))
                for pid, argv in live.items():
                    observed.setdefault(pid, {"pid": pid, "argv": argv,
                                             "first_seconds": time.monotonic() - started})
                time.sleep(0.01)
            code = process.wait()
            if code:
                raise subprocess.CalledProcessError(code, command)
        finally:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            evidence.with_suffix(".json").write_text(json.dumps({
                "command": command, "exit": process.returncode,
                "seconds": time.monotonic() - started,
                "java_sample_interval_seconds": 0.01,
                "observed_java": list(observed.values()), "peak_java": peak,
            }, indent=2))
    return time.monotonic() - started


def execute(destination, cases, environment):
    environment = dict(environment, FN255_EVIDENCE=str(destination))
    compile_seconds = bounded(["mise", "exec", "--", "cargo", "build", "--offline", "--manifest-path",
             str(destination / "Cargo.toml"), "--target-dir", str(HERE / "evidence/target")],
            120, environment, destination / "compile")
    backend_seconds = bounded([str(HERE / "evidence/target/debug/fn255-spike")], 45 * cases,
                              environment, destination / "runtime")
    result = {"status": "completed", "cases": cases, "compile_seconds": compile_seconds,
              "backend_seconds": backend_seconds}
    (destination / "measurement.json").write_text(json.dumps(result, indent=2))
    return result


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("destination", type=Path)
    parser.add_argument("--seed", type=int, default=255001)
    parser.add_argument("--cases", type=int, choices=range(1, 3), default=1)
    parser.add_argument("--tlc-jar", type=Path, default=FRAMEWORK / "build/tlc/tla2tools-1.7.4.jar")
    parser.add_argument("--batch-tlc", action="store_true")
    parser.add_argument("--execute", action="store_true", help="explicitly compile/run; omit for source preparation")
    parser.add_argument("--shrink-controls", action="store_true",
                        help="after explicit execution, try at most three source reductions on case zero")
    args = parser.parse_args()
    if args.shrink_controls and not args.execute:
        parser.error("--shrink-controls requires explicit --execute")
    if args.execute and not args.tlc_jar.is_file():
        parser.error("--tlc-jar must name the pinned TLC 1.7.4 jar")
    destination = args.destination.resolve()
    prepare(destination, args.seed, args.cases, batch_tlc=args.batch_tlc)
    if not args.execute:
        print(f"prepared unvalidated source: {destination}")
        return
    tools = HERE / "evidence/tools"
    (tools / "build/tlc").mkdir(parents=True, exist_ok=True)
    cached_jar = tools / "build/tlc/tla2tools-1.7.4.jar"
    if args.tlc_jar.resolve() != cached_jar.resolve():
        shutil.copyfile(args.tlc_jar, cached_jar)
    shutil.copyfile(FRAMEWORK / "mise.toml", tools / "mise.toml")
    environment = dict(os.environ, MISE_AUTO_INSTALL="false", MISE_EXEC_AUTO_INSTALL="false",
                       UMPIRE_QUICK_CACHE="0", UMPIRE_CACHE_DISABLE="1", CARGO_BUILD_JOBS="2",
                       MISE_TRUSTED_CONFIG_PATHS=str(HERE / "evidence/tools"))
    environment.pop("BASH_ENV", None)
    environment.pop("FN255_CONTROLS_ONLY", None)
    print(json.dumps(execute(destination, args.cases, environment)))
    if args.shrink_controls:
        current = json.loads((destination / "inputs.json").read_text())["shapes"][0]
        history = []
        for iteration in range(3):
            candidate = list(current)
            candidate[iteration] = [0, 1, "Off"][iteration]
            if candidate == current:
                continue
            scratch = destination / f"shrink-{iteration}"
            prepare(scratch, args.seed, 1, [candidate])
            try:
                result = execute(scratch, 1, dict(environment, FN255_CONTROLS_ONLY="1"))
            except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as error:
                # A refusal/build failure is inconclusive, never retained as a successful shrink.
                history.append({"candidate": candidate, "retained": False, "error": str(error)})
            else:
                current = candidate
                history.append({"candidate": candidate, "retained": True, "timing": result})
        (destination / "shrunk-controls.json").write_text(json.dumps(
            {"shape": current, "history": history, "max_iterations": 3}, indent=2))


if __name__ == "__main__":
    main()
