import json, pathlib, subprocess, sys
root=pathlib.Path.cwd()
phase=sys.argv[1]
direct=len(sys.argv)>2 and sys.argv[2]=='direct'
rows=[json.loads(l) for l in (root/f'.flow/tmp/fn2471/{phase}-build.jsonl').read_text().splitlines() if l.startswith('{')]
rlib=next(f for r in rows if r.get('reason')=='compiler-artifact' and r.get('target',{}).get('name')==('umpire_backend_process' if direct else 'umpire_tla') for f in r['filenames'] if f.endswith('.rlib'))
suffix='-direct' if direct else ''
binary=root/f'.flow/tmp/fn2471/{phase}{suffix}-corpus'
source=root/'.flow/tmp/fn2471/corpus.rs'
if direct:
    source=root/'.flow/tmp/fn2471/direct-corpus.rs'
    source.write_text((root/'.flow/tmp/fn2471/corpus.rs').read_text().replace('extern crate umpire_tla as owner;','extern crate umpire_backend_process as owner;').replace('use owner::process::*;','use owner::*;'))
subprocess.run(['mise','exec','--','rustc','--edition=2024',str(source),'--extern',f'{"umpire_backend_process" if direct else "umpire_tla"}={rlib}','-L',f'dependency={pathlib.Path(rlib).parent / "deps"}','-o',str(binary)],cwd=root/'model',check=True)
r=subprocess.run([str(binary)],cwd=root,capture_output=True,check=True)
(root/f'.flow/tmp/fn2471/{phase}{suffix}-corpus.stdout').write_bytes(r.stdout)
(root/f'.flow/tmp/fn2471/{phase}{suffix}-corpus.stderr').write_bytes(r.stderr)
print(r.stdout.decode(),end='')
