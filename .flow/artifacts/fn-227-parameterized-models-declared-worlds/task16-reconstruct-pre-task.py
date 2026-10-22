from pathlib import Path
import subprocess,tomllib,re,json,hashlib
base=Path('.flow/tmp/base_commit').read_text().strip()
root=Path('.flow/tmp/fn-227.16/pre-task-source')
roots=['model/umpire/'+n for n in ['ir','ir-derive','front','lift']]
paths=subprocess.check_output(['git','ls-tree','-r','--name-only',base,*roots],text=True).splitlines()
paths=[p for p in paths if '/src/' in p or p.endswith('/Cargo.toml')]
paths+=['model/umpire/dsl/src/closable.rs','model/umpire/dsl/src/phased.rs','model/umpire/lift/tests/fixtures/vector.rs']
hashes={}
for name in paths:
    data=subprocess.check_output(['git','show',base+':'+name]);p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(data);hashes[name]=hashlib.sha256(data).hexdigest()
cargo=tomllib.loads(subprocess.check_output(['git','show',base+':model/Cargo.toml'],text=True))
manifest='[workspace]\nresolver = "3"\nmembers = ["probe", "umpire/ir", "umpire/ir-derive", "umpire/front", "umpire/lift"]\n\n[workspace.package]\n'
for key,value in cargo['workspace']['package'].items():manifest+=key+' = '+json.dumps(value)+'\n'
manifest+='\n[workspace.dependencies]\n'
for name in ['ir','ir-derive','front','lift']:manifest+='umpire-'+name+' = { path = "umpire/'+name+'" }\n'
for name in ['quote','syn','proc-macro2','prettyplease']:
    value=cargo['workspace']['dependencies'][name]
    if isinstance(value,str):manifest+=name+' = '+json.dumps(value)+'\n'
    else:manifest+=name+' = { '+', '.join(k+' = '+json.dumps(v) for k,v in value.items())+' }\n'
(root/'model/Cargo.toml').write_text(manifest)
(root/'model/Cargo.lock').write_bytes(subprocess.check_output(['git','show',base+':model/Cargo.lock']))
for name in roots:
    p=root/name/'Cargo.toml';parts=re.split(r'(?m)^(\[[^\n]+\]\n)',p.read_text());out=parts[0]
    for header,body in zip(parts[1::2],parts[2::2]):
        if not header.startswith(('[dev-dependencies]','[lints]')):out+=header+body
    p.write_text(out)
probe=root/'model/probe';(probe/'src').mkdir(parents=True,exist_ok=True)
(probe/'Cargo.toml').write_text('[package]\nname = "task16-pre-task-probe"\nversion = "0.0.1"\nedition = "2024"\n[dependencies]\numpire-lift = { path = "../umpire/lift" }\n')
s=Path('.flow/tmp/fn-227.16/conditional-length-probe.rs').read_text().replace('include_str!("/Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire/lift/tests/fixtures/vector.rs")','include_str!("../../umpire/lift/tests/fixtures/vector.rs")')
s=s.replace('println!("{:?}", umpire_lift::Lift::parse("conditional_length.rs", &source).map(|_| ()));','let result = umpire_lift::Lift::parse("conditional_length.rs", &source).map(|_| ()); println!("{result:?}"); assert!(result.is_ok());')
(probe/'src/main.rs').write_text(s)
(root/'SOURCE_MANIFEST.json').write_text(json.dumps({'base':base,'source_sha256':hashes,'harness_only_manifest_changes':'Minimal production dependency closure; dev dependencies and workspace lint inheritance omitted. Probe only checks admission, not a replacement baseline/gate.'},indent=2)+'\n')
print('Reconstructed',len(paths),'pinned source/manifest files from',base)
