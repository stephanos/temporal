import hashlib,json,pathlib,subprocess,tomllib
root=pathlib.Path.cwd()
out=root/'.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task1'
base=(root/'.flow/tmp/fn2471/base_commit').read_text().strip()
for old,new in [('model/umpire/backends/tla/src/process.rs','model/umpire/backends/process/src/lib.rs'),('model/umpire/backends/tla/src/process.py','model/umpire/backends/process/src/process.py')]:
    assert subprocess.check_output(['git','show',f'{base}:{old}'])==(root/new).read_bytes(),new
hashes=json.loads((out/'baseline-artifacts.json').read_text())
for p,expected in hashes.items():
    assert hashlib.sha256((root/p).read_bytes()).hexdigest()==expected['sha256'],p
before=json.loads((out/'baseline-packages.json').read_text())
after=tomllib.loads((root/'model/Cargo.lock').read_text())['package']
old={p['name']+'@'+p['version']:p for p in before}
new={p['name']+'@'+p['version']:p for p in after}
assert set(new)-set(old)=={'umpire-backend-process@0.0.1'}
assert not set(old)-set(new)
for key,p in old.items():
    q=new[key].copy()
    if p['name'] in ['umpire-tla','umpire-tlc','umpire-apalache']:
        assert set(q['dependencies'])-set(p['dependencies'])=={'umpire-backend-process'}
        q['dependencies']=[d for d in q['dependencies'] if d!='umpire-backend-process']
    assert p==q,key
assert new['umpire-backend-process@0.0.1']=={'name':'umpire-backend-process','version':'0.0.1','dependencies':['thiserror']}
features={}
for phase in ['baseline','post']:
    data=json.loads((root/f'.flow/tmp/fn2471/{phase}-metadata.json').read_text())
    features[phase]={n['id']:n['features'] for n in data['resolve']['nodes']}
for package,feature_set in features['baseline'].items():
    assert features['post'][package]==feature_set,package
(out/'package-features.json').write_text(json.dumps(features,sort_keys=True,indent=2)+'\n')
(out/'post-packages.json').write_text(json.dumps(after,sort_keys=True,indent=2)+'\n')
result={'moved_sources':'byte-identical to pre-edit base','artifacts_unchanged':len(hashes),'third_party_packages_versions_checksums_features':'unchanged','local_graph_delta':'process crate depends on existing thiserror; TLA/TLC/Apalache each add process edge','guest_sha256':hashes['model/temporal/hosts/go/umpire_temporal_guest.wasm']['sha256']}
(out/'equivalence.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
