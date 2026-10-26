"""Unexecuted, bounded source-level differential spike. No production gate registration."""
import argparse
import json
from pathlib import Path
import random
import shutil
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
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub enum Mode {{ Off, On }}
    #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, Finite)]
    pub struct Token(#[inputs(0..=1)] pub u8);
    #[derive(Debug)]
    pub enum Fact {{ Changed }}
    #[derive(State)]
    pub struct Model {{
        pub left: Mode,
        pub right: Mode,
        pub count: u8,
        pub token: Option<Token>,
    }}
    #[derive(Action, Copy)]
    pub enum User {{ Advance, Toggle, Clear, Refuse }}
    impl Model {{
        #[starts]
        fn starts() -> impl IntoIterator<Item = Self> {{
            [Self {{ left: Mode::{start}, right: Mode::Off, count: 0, token: None }}]
        }}
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {{
            match action {{
                User::Advance if self.count < {upper} => {{
                    self.count += 1;
                    self.token = Some(Token({seed % 2}));
                    Some(Fact::Changed)
                }}
                User::Toggle => {{
                    self.left = match self.left {{ Mode::Off => Mode::On, Mode::On => Mode::Off }};
                    self.right = match self.right {{ Mode::Off => Mode::On, Mode::On => Mode::Off }};
                    Some(Fact::Changed)
                }}
                User::Clear if self.token.is_some() => {{ self.token = None; Some(Fact::Changed) }}
                User::Refuse => {{ reject!(Rejection::NotFound); None }}
                _ => None,
            }}
        }}
        #[invariant]
        fn bounded(&self) {{ check!(self.count <= {upper}); }}
    }}
}}
'''


def prepare(destination, seed, count):
    destination.mkdir(parents=True, exist_ok=False)
    src = destination / "src"
    src.mkdir()
    shutil.copyfile(HERE / "adapters.rs", src / "adapters.rs")
    rng = random.Random(seed)
    shapes = [(rng.randrange(1 << 32), rng.randint(1, 2), rng.choice(["Off", "On"]))
              for _ in range(count)]
    calls = []
    modules = []
    for index, shape in enumerate(shapes):
        filename = f"case_{index}.rs"
        (src / filename).write_text(model(*shape))
        modules.append(f"mod case_{index};")
        calls.append(f'''{{
        use case_{index}::generated;
        let world = generated::world;
        let identity = |a: &<generated::World as Machine>::Action| ActionValues::<generated::World>::action_value(a).0.into();
        let interpreted = Interpreted::new(world())?;
        let compiled = adapters::agree(adapters::native(&world(), identity), adapters::native(&interpreted, identity))?;
        let source = include_str!("{filename}");
        let module = umpire_tla::Tla::emit(&umpire_tla::Lift::parse("{filename}", source)?)?;
        let checked = adapters::tlc(std::path::Path::new({json.dumps(str(FRAMEWORK))}), &module)?;
        adapters::agree(Ok(compiled), Ok(checked))?;
        adapters::lowering_controls(world)?;
        println!("case={index} seed={shape[0]} complete");
    }}''')
    (src / "main.rs").write_text('''#![forbid(unsafe_code)]
use umpire::dsl::{Machine, ActionValues};
use umpire_checker::Interpreted;
mod adapters;
''' + "\n".join(modules) + '\nfn main() -> Result<(), Box<dyn std::error::Error>> {\n' +
        "\n".join(calls) + '\nOk(())\n}\n')
    dependencies = {
        "umpire": "author/umpire", "umpire-checker": "backends/native",
        "umpire-tla": "backends/tla", "umpire-tlc": "backends/tlc",
    }
    manifest = '[package]\nname="fn255-spike"\nversion="0.0.0"\nedition="2024"\n[workspace]\n[dependencies]\n'
    for name, path in dependencies.items():
        manifest += f'{name}={{path={json.dumps(str(FRAMEWORK / path))}}}\n'
    (destination / "Cargo.toml").write_text(manifest)
    (destination / "inputs.json").write_text(json.dumps({"seed": seed, "shapes": shapes}, indent=2))


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("destination", type=Path)
    parser.add_argument("--seed", type=int, default=255001)
    parser.add_argument("--cases", type=int, choices=range(1, 5), default=1)
    parser.add_argument("--execute", action="store_true", help="explicitly compile/run; omit for source preparation")
    args = parser.parse_args()
    destination = args.destination.resolve()
    prepare(destination, args.seed, args.cases)
    if not args.execute:
        print(f"prepared unvalidated source: {destination}")
        return
    import os
    environment = dict(os.environ, MISE_AUTO_INSTALL="false", MISE_EXEC_AUTO_INSTALL="false",
                       UMPIRE_QUICK_CACHE="0", UMPIRE_CACHE_DISABLE="1", CARGO_BUILD_JOBS="2")
    environment.pop("BASH_ENV", None)
    started = time.monotonic()
    subprocess.run(["mise", "exec", "--", "cargo", "build", "--offline", "--manifest-path",
                    str(destination / "Cargo.toml"), "--target-dir", str(destination / "target")],
                   cwd=FRAMEWORK, env=environment, check=True, timeout=120)
    compile_seconds = time.monotonic() - started
    started = time.monotonic()
    subprocess.run([str(destination / "target/debug/fn255-spike")], cwd=FRAMEWORK,
                   env=environment, check=True, timeout=45 * args.cases)
    print(json.dumps({"status": "completed", "cases": args.cases,
                      "compile_seconds": compile_seconds, "backend_seconds": time.monotonic() - started}))


if __name__ == "__main__":
    main()
