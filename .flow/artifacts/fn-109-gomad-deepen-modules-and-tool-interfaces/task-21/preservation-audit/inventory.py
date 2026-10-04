"""Retain read-only baseline/current R18 source inventories and exact receipts."""
import collections
import datetime
import difflib
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
BASE = Path('/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3')
CURRENT = ROOT / 'tools/gomad3'
OUT = ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
ENV = dict(os.environ, GOTOOLCHAIN='local', GOWORK='off', GOFLAGS='')
receipts = []

def command(args, cwd, name, platform=None):
    start = datetime.datetime.now(datetime.timezone.utc).isoformat()
    tick = time.monotonic()
    env = dict(ENV)
    if platform:
        env['GOOS'], env['GOARCH'] = platform.split('/')
    run = subprocess.run(args, cwd=cwd, env=env, text=True, capture_output=True, timeout=60)
    output = run.stdout + run.stderr
    (OUT / name).write_text(output)
    receipts.append(dict(command=args, cwd=str(cwd), environment_overrides={k: env[k] for k in ('GOTOOLCHAIN','GOWORK','GOFLAGS','GOOS','GOARCH') if k in env}, start=start, end=datetime.datetime.now(datetime.timezone.utc).isoformat(), elapsed_seconds=time.monotonic()-tick, exit_code=run.returncode, log=name, log_sha256=hashlib.sha256(output.encode()).hexdigest()))
    return run

def files(root):
    return {str(p.relative_to(root)): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(root.rglob('*')) if p.is_file() and not {'.toolchain','.bin','.gomad','__pycache__'}.intersection(p.relative_to(root).parts)}

def document_packages(root, package):
    result = set()
    for p in (root / package).rglob('*.go'):
        rel = p.relative_to(root)
        if p.name.endswith('_test.go') or {'internal','testdata'}.intersection(rel.parts):
            continue
        if any((a / 'go.mod').exists() for a in p.parents if a != root and root in a.parents):
            continue
        result.add(str(p.parent.relative_to(root)))
    return result

before = {'baseline': files(BASE), 'current': files(CURRENT)}
command([GO,'version'], CURRENT, 'go-version.log')
command(['uname','-sm'], ROOT, 'host.log')
command(['sha256sum','--check','--quiet',str(ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/baseline-reconstruction/source.sha256')], BASE.parent.parent, 'baseline-source-check.log')
apis = []
all_diff = []
for surface in ('runner','artifact','target','deterministicio','record','choice','world','qualification'):
    for package in sorted(document_packages(BASE,surface) | document_packages(CURRENT,surface)):
        for platform in ('darwin/arm64','linux/amd64'):
            outputs=[]
            for role, tree in (('baseline',BASE),('current',CURRENT)):
                name=platform.replace('/','-')+'-'+role+'-'+package.replace('/','-')+'.godoc.txt'
                if (tree/package).exists():
                    run=command([GO,'doc','-all','./'+package],tree,name,platform)
                    if run.returncode:
                        raise RuntimeError('go doc failed: '+package)
                    outputs.append(run.stdout)
                else:
                    (OUT/name).write_text('')
                    outputs.append('')
            diff=''.join(difflib.unified_diff(outputs[0].splitlines(True),outputs[1].splitlines(True),fromfile=platform+'/baseline/'+package,tofile=platform+'/current/'+package))
            all_diff.append(diff)
            apis.append({'package':package,'platform':platform,'changed':bool(diff),'baseline_sha256':hashlib.sha256(outputs[0].encode()).hexdigest(),'current_sha256':hashlib.sha256(outputs[1].encode()).hexdigest()})
(OUT/'public-api.diff').write_text('\n'.join(all_diff))

def cli_inventory(tree, product):
    inventory=[]
    roots=[tree/'cmd'/product]
    if product=='gomadtool': roots.append(tree/'internal/gomadtool')
    for root in roots:
        for p in sorted(root.rglob('*.go')):
            if p.name.endswith('_test.go') or 'testdata' in p.parts: continue
            function=''; flagset=''
            for number,line in enumerate(p.read_text().splitlines(),1):
                if line.startswith('func '): function=line.strip();flagset=''
                if 'flag.NewFlagSet(' in line: flagset=line.strip()
                is_flag = re.search(r'\b(?:flags|flagset|fs)\.(?:String|Bool|Uint64|Int|Duration|Var|StringVar|BoolVar|Uint64Var|IntVar|DurationVar)\(',line)
                is_dispatch = 'cmd' in p.relative_to(tree).parts and re.match(r'\s*case "',line)
                if is_flag or is_dispatch:
                    inventory.append({'path':str(p.relative_to(tree)),'line':number,'function':function,'flagset':flagset,'source':line.strip()})
    return inventory

cli={role:{product:cli_inventory(tree,product) for product in ('gomad','gomadtool')} for role,tree in (('baseline',BASE),('current',CURRENT))}
(OUT/'cli-inventory.json').write_text(json.dumps(cli,indent=2)+'\n')
clidiff=[]
for product in ('gomad','gomadtool'):
    a=sorted(set(row['source'] for row in cli['baseline'][product]));b=sorted(set(row['source'] for row in cli['current'][product]))
    clidiff.extend(difflib.unified_diff(a,b,fromfile='baseline/'+product,tofile='current/'+product,lineterm=''))
(OUT/'cli-inventory.diff').write_text('\n'.join(clidiff)+'\n')
cli_doc=(CURRENT/'CLI.md').read_text()
flags={m.group(1) for m in re.finditer(r'--([a-z][a-z0-9-]*)',cli_doc)}
registered=set()
for row in cli['current']['gomad']:
    quoted=re.findall(r'"([a-z][a-z0-9-]*)"',row['source'])
    if quoted and 'case ' not in row['source']:registered.add(quoted[0])

current_comments=collections.defaultdict(list)
for p in CURRENT.rglob('*.go'):
    if '.toolchain' in p.parts:continue
    for number,line in enumerate(p.read_text().splitlines(),1):
        if line.strip().startswith('//'):current_comments[line.strip()].append(str(p.relative_to(CURRENT))+':'+str(number))
removed=[]
for p in BASE.rglob('*.go'):
    rel=p.relative_to(BASE);other=CURRENT/rel
    if not other.exists():continue
    old=p.read_text().splitlines();new=other.read_text().splitlines()
    for tag,a,z,b,y in difflib.SequenceMatcher(a=old,b=new,autojunk=False).get_opcodes():
        if tag not in ('delete','replace'):continue
        for index in range(a,z):
            comment=old[index].strip()
            if comment.startswith('//'):
                removed.append({'baseline':str(rel)+':'+str(index+1),'comment':comment,'current_exact_occurrences':current_comments.get(comment,[])})
(OUT/'deleted-comments.json').write_text(json.dumps(removed,indent=2)+'\n')

bound={}
for prefix in ('deterministicio/boundary/manifest.json','internal/compatibilitypack/packs','internal/compatibilitypack/requests'):
    keys=sorted({k for manifest in before.values() for k in manifest if k==prefix or k.startswith(prefix+'/')})
    bound[prefix]=[{'path':k,'baseline':before['baseline'].get(k),'current':before['current'].get(k)} for k in keys if before['baseline'].get(k)!=before['current'].get(k)]
retained=ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/round4-final-source.sha256'
reuse=[]
for line in retained.read_text().splitlines():
    expected,path=line.split(None,1);path=path.lstrip('*');p=ROOT/path
    actual=hashlib.sha256(p.read_bytes()).hexdigest() if p.is_file() else None
    reuse.append({'path':path,'expected':expected,'current':actual,'matches':actual==expected})
after={'baseline':files(BASE),'current':files(CURRENT)}
report={'baseline_root':str(BASE),'current_root':str(CURRENT),'source_before':before,'source_after':after,'sources_unchanged':before==after,'go_doc_packages':apis,'cli_doc_flags':sorted(flags),'current_registered_gomad_flags':sorted(registered),'registered_flags_absent_from_CLI_md':sorted(registered-flags),'boundary_pack_diffs':bound,'comment_removed_count':len(removed),'comments_without_exact_current_occurrence':[x for x in removed if not x['current_exact_occurrences']],'task19_manifest_verification':{'manifest':str(retained),'manifest_sha256':hashlib.sha256(retained.read_bytes()).hexdigest(),'files':len(reuse),'mismatches':[x for x in reuse if not x['matches']]},'commands':receipts,'native_qualification':False}
(OUT/'inventory.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({'api_packages':len(apis),'sources_unchanged':before==after,'comments_removed':len(removed),'comments_missing_exactly':len(report['comments_without_exact_current_occurrence']),'boundary_pack_diffs':{k:len(v) for k,v in bound.items()},'task19_mismatches':report['task19_manifest_verification']['mismatches'],'registered_flags_absent_from_CLI_md':report['registered_flags_absent_from_CLI_md'],'commands':len(receipts)},indent=2))
