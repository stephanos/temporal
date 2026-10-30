import pathlib,json,hashlib,re,subprocess
root=pathlib.Path.cwd();out=root/'.flow/tmp/fn25420'
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
paths=['Makefile','model/dev/xtask/tests/make_paths.rs','model/design/gate-speed.md']
(out/'make-source-paths.json').write_text(json.dumps(paths,indent=2)+'\n')
steps={}
for phase in ['before','after']:
 steps[phase]={}
 for step in ['first','nested','repeat']:
  file=out/f'make-{phase}-{step}.log';s=file.read_text()
  rows=[x for x in s.splitlines() if 'Running `' in x and '--crate-name' in x]
  item={'log':file.name,'sha256':sha(file),'config_dirty_count':s.count('config settings changed'),'rustc_invocation_count':len(rows),'mold_rustc_invocation_count':sum('mold-linker.sh' in x for x in rows),'cargo_finished':[x.strip() for x in s.splitlines() if 'Finished `' in x]}
  steps[phase][step]=item
  assert item['config_dirty_count']==(43 if phase=='before' else 0)
  assert item['rustc_invocation_count']==(43 if phase=='before' else 0)
  if phase=='before':assert item['mold_rustc_invocation_count']==(43 if step=='nested' else 0)
exe=pathlib.Path('/home/agent/.cache/codex-build/fn25420-mold/debug/temporal-xtask')
comment=subprocess.check_output(['readelf','-p','.comment',str(exe)],text=True)
assert 'mold 3.0.0' in comment
(out/'make-final-executable.comment').write_text(comment)
assert 'test result: ok. 5 passed' in (out/'make-final-tests.log').read_text()
assert 'Finished `' in (out/'make-test-clippy.log').read_text() and not re.search(r'^error(?:\[|:)', (out/'make-test-clippy.log').read_text(), re.M)
proof={'source':{p:sha(root/p) for p in paths},'decision':'RUST_MODEL_CARGO explicitly loads model/.cargo/config.toml without changing caller cwd, Cargo home/target or inline environment assignment semantics','regression':{'red':'make-red.log: omitted explicit config, exact intended command mismatch','green':'make-green.log','system_configuration':'both supported target triples have only linker=umpire/scripts/mold-linker.sh; guarded by permanent Temporal test','caller_semantics':'regression executes extracted shipping Make definition/quick recipe in a path with spaces, preserving relative CARGO_HOME/CARGO_TARGET_DIR and cwd and disabling tool auto installation'},'actual_commands':['root: make --no-print-directory model-quick','nested: cd model && mise exec -- cargo build --locked --profile dev -p temporal-xtask --bin temporal-xtask -vv','root: make --no-print-directory model-quick'],'observer':'PATH mise observer records original argv/cwd, adds only global Cargo -vv and --select-only after quick, then executes real /home/agent/.local/bin/mise; shipped Make recipe/definition and real bridge/bootstrap/selection execute','observer_sha256':sha(out/'make-bin/mise'),'cache':'same isolated /home/agent/.cache/codex-build/fn25420-mold; sccache and Rust wrappers absent','steps':steps,'original_mise_calls':{phase:[json.loads(x) for x in (out/f'make-{phase}-calls.jsonl').read_text().splitlines()] for phase in ['before','after']},'final_executable':{'path':str(exe),'sha256':sha(exe),'comment':comment},'verification':{'temporal_integration_tests':5,'changed_test_clippy':'cargo clippy --locked -p temporal-xtask --test main -- -D warnings: pass','format':'cargo fmt --check -p temporal-xtask: pass','diff_check':'git diff --check: pass','broadened_clippy':'cargo clippy --locked -p temporal-xtask --all-targets -- -D warnings: fails unchanged gate.rs:244 clippy::too_many_lines (129/100); no unrelated fix'},'limitations':['quick --select-only covers real Make/bridge/bootstrap/config/selection, not selected validation gates','This correction does not renew full current-input or committed Temporal guest freshness evidence','After first step starts with mold units warmed by green regression build; all three after steps remain fresh','Original task20 proof/source hashes retained unchanged as historical reviewed candidate'],'log_hashes':{p.name:sha(p) for p in out.glob('make-*') if p.is_file() and p.suffix in ['.log','.jsonl','.comment']}}
(out/'make-entry-proof.json').write_text(json.dumps(proof,indent=2)+'\n');print(json.dumps({p:{s:(v['config_dirty_count'],v['rustc_invocation_count']) for s,v in rows.items()} for p,rows in steps.items()}))
