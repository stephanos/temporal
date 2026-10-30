import pathlib,json,hashlib,subprocess,shlex,re,os,tomllib
root=pathlib.Path.cwd();out=root/'.flow/tmp/fn25420'
source=['mise.toml','mise.lock','model/umpire/mise.toml','model/umpire/mise.lock','model/.cargo/config.toml','model/umpire/.cargo/config.toml','model/umpire/scripts/mold-linker.sh','model/umpire/dev/guest-driver/tests/main.rs','model/umpire/dev/guest-driver/tests/linker.rs','model/GUEST_BUILDS.md','model/design/gate-speed.md']
def sha(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
(out/'source-paths.json').write_text(json.dumps(source,indent=2)+'\n')
inputs=[]
for arg in json.load(open(out/'link-args.json')):
 p=pathlib.Path(arg)
 if p.is_file(): inputs.append({'path':str(p),'bytes':p.stat().st_size,'sha256':sha(p)})
(out/'link-input-manifest.json').write_text(json.dumps(inputs,indent=2)+'\n')
namespace='c23437f5d2c039aa7d835f6b231931a23ce618b36fa8838e1092795b0ca48e88'
wrapper=pathlib.Path('/home/agent/.cache/codex-build/fn25420-mold/guest-wrappers')/namespace/'guest-rustc'
compiler=subprocess.check_output(['mise','exec','--','rustc','-V'],cwd=root/'model/umpire',text=True).strip()
key=[sha(wrapper),sha(root/'model/umpire/Cargo.lock'),compiler]
assert hashlib.sha256(json.dumps(key,separators=(',',':')).encode()).hexdigest()==namespace
invocations={}
for name in ['stock-baseline-build.log','stock-restored-build.log','native-profile.log','driver-build.log','temporal-tool-build.log']:
 rows=[]
 for line in (out/name).read_text().splitlines():
  m=re.search(r'Running `(.+)`',line)
  if m and '--crate-name' in m[1]: rows.append(shlex.split(m[1]))
 invocations[name]=rows
restored=invocations['stock-restored-build.log'];wasi=[r for r in restored if 'wasm32-wasip1' in r]
host=[r for r in restored if any('linker=' in a for a in r)]
assert wasi and host
assert all(not any('mold-linker' in a or 'fuse-ld' in a for a in r) for r in wasi)
assert all(any('mold-linker.sh' in a for a in r) for r in host)
baseline=[r for r in json.load(open(out/'host-linked-baseline.json')) if '/'+namespace+'/' in r['path']]
restored_outputs=json.load(open(out/'host-linked-restored.json'))
assert len(baseline)==len(restored_outputs)==20
assert not any(r['mold'] for r in baseline) and all(r['mold'] for r in restored_outputs)
sidecars=json.load(open(out/'host-sidecars-baseline.json'))
assert sidecars==json.load(open(out/'host-sidecars-mold.json'))
final={p.name:json.loads(p.read_text()) for p in (pathlib.Path('/home/agent/.cache/codex-build/fn25420-mold/guest-cache')/namespace).glob('guest/deps/*.umpire-id')}
assert final==sidecars
assert sha(out/'stock-baseline.wasm')==sha(out/'stock-restored.wasm')
for f,path in [('model/.cargo/config.toml','umpire/scripts/mold-linker.sh'),('model/umpire/.cargo/config.toml','scripts/mold-linker.sh')]:
 c=tomllib.loads((root/f).read_text());assert set(c['target'])=={'aarch64-unknown-linux-gnu','x86_64-unknown-linux-gnu'}
 assert all(t=={'linker':path} for t in c['target'].values())
proof={'decision':'adopt mold 3.0.0 on aarch64/x86_64 Linux GNU targets; measured aarch64 only','source':{s:sha(root/s) for s in source},'measurement':json.load(open(out/'measurement.json')),'link_input_manifest_sha256':sha(out/'link-input-manifest.json'),'guest_namespace':{'key':namespace,'inputs':key,'same_key_verified':True,'baseline_host_outputs':baseline,'restored_host_outputs':restored_outputs,'portable_sidecars_unchanged':True,'host_variants_unchanged':True,'sidecar_count':len(final),'wasm_sha256':sha(out/'stock-restored.wasm'),'wasm_bytes':(out/'stock-restored.wasm').stat().st_size},'effective_invocations':invocations,'wasi_rustc_count':len(wasi),'host_linker_rustc_count':len(host),'tests':{'unit':5,'integration':34,'Go':['TestGuestExports','TestCurrentContractNativeWasmPublishedResponsesAgree'],'clippy':'passed','format':'passed','bootstrap':'passed','temporal_tool_build':'passed','standalone_config_build_and_execution':'passed'},'limitations':['Linux/aarch64 execution only; x86_64 and non-Linux controls are configuration scope, not execution','Focused link measurement; full gate/build improvement not claimed','Task .8 owns full current-input standalone/combined gates and committed Temporal guest freshness','Unrelated host activity not controlled; sibling builds paused for measured relinks','First guest build preceded a concurrent lockfile change; only baseline/restored same-key pair supports namespace comparison','Experimental failures red-linker, red-missing-asset, first mise which/backend selector and capture attempts are retained; excluded from passing proof'],'log_hashes':{p.name:sha(p) for p in out.iterdir() if p.is_file() and p.suffix in ['.log','.stderr','.comment']},'cache_lease':'/home/agent/.cache/codex-build/fn25420-mold; retained with proof artifacts; no cache cleaned'}
(out/'proof.json').write_text(json.dumps(proof,indent=2)+'\n');print('verified proof; WASI invocations',len(wasi),'host linker invocations',len(host),'inputs',len(inputs))
