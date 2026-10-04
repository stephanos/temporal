"""Bind every public declaration difference to current source and Git provenance."""
import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time

ROOT=Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
MODULE=ROOT/'tools/gomad3'
OUT=ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-21/preservation-audit'
commands=[]
blames={}

def run(args,name):
    start=datetime.datetime.now(datetime.timezone.utc).isoformat();tick=time.monotonic()
    p=subprocess.run(args,cwd=ROOT,text=True,capture_output=True,timeout=60)
    output=p.stdout+p.stderr;(OUT/name).write_text(output)
    commands.append(dict(command=args,cwd=str(ROOT),start=start,end=datetime.datetime.now(datetime.timezone.utc).isoformat(),elapsed_seconds=time.monotonic()-tick,exit_code=p.returncode,log=name,log_sha256=hashlib.sha256(output.encode()).hexdigest()))
    assert p.returncode==0,(args,p.stderr)
    return p.stdout

def blame(path):
    if path in blames:return blames[path]
    text=run(['git','blame','--line-porcelain','--',path],'blame-'+path.replace('/','-')+'.log')
    result={};commit='';summary='';number=0
    for line in text.splitlines():
        m=re.match(r'^([0-9a-f]{40}) \d+ (\d+)(?: \d+)?$',line)
        if m:commit=m.group(1);number=int(m.group(2))
        elif line.startswith('summary '):summary=line[8:]
        elif line.startswith('\t'):result[number]={'commit':commit,'summary':summary,'source':line[1:]}
    blames[path]=result;return result

def parse(text):
    out={};key=None;mode='';enum=False
    for line in text.splitlines():
        if line.startswith('type '):
            name=line.split()[1];key='type '+name;out[key]=[line];mode='type';enum=False
        elif line.startswith('func '):
            m=re.match(r'func (?:\((.*?)\) )?(\w+)\(',line)
            if m:
                receiver=(m.group(1) or '').split()[-1:] or [''];owner=receiver[0].lstrip('*');key='func '+(owner+'.' if owner else '')+m.group(2);out[key]=[line];mode='func';enum=False
        elif line.startswith(('const (','var (')):mode=line.split()[0];enum=True;key=None
        elif line.startswith(('const ','var ')):
            key=line.split()[0]+' '+line.split()[1];out[key]=[line];mode='';enum=False
        elif enum and line.startswith('\t'):
            if line.lstrip().startswith('//'):continue
            key=mode+' '+line.strip().split()[0];out[key]=[line.strip()]
        elif mode=='type' and key and (line.startswith('\t') or line=='}'):
            if not line.lstrip().startswith('//'):out[key].append(line.split('//')[0].rstrip())
        elif line==')':enum=False
    return {k:'\n'.join(re.sub(r'\s+',' ',x).strip() for x in v) for k,v in out.items()}

def locate(package,key):
    kind,name=key.split(' ',1);owner,_,method=name.rpartition('.');found=[]
    for p in sorted((MODULE/package).glob('*.go')):
        if p.name.endswith('_test.go'):continue
        for n,line in enumerate(p.read_text().splitlines(),1):
            hit=False
            if kind=='type':hit=bool(re.match(r'^type '+re.escape(name)+r'\b',line))
            elif kind=='func':
                m=re.match(r'^func (?:\((.*?)\) )?(\w+)\(',line)
                if m:
                    receiver=(m.group(1) or '').split()[-1:] or [''];actual=receiver[0].lstrip('*');hit=m.group(2)==(method or name) and actual==owner
            else:hit=bool(re.match(r'^\s*(?:(?:var|const) )?'+re.escape(name)+r'\b',line))
            if hit:
                path=str(p.relative_to(ROOT));found.append({'path':path,'line':n,'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'declaration_blame':blame(path)[n]})
    return found

inventory=json.loads((OUT/'inventory.json').read_text());changes=[]
interface=(ROOT/'.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md').read_text()
for package in sorted({x['package'] for x in inventory['go_doc_packages']}):
    a=parse((OUT/('darwin-arm64-baseline-'+package.replace('/','-')+'.godoc.txt')).read_text());b=parse((OUT/('darwin-arm64-current-'+package.replace('/','-')+'.godoc.txt')).read_text())
    for key in sorted(a.keys()|b.keys()):
        if a.get(key)==b.get(key):continue
        row={'package':package,'declaration':key,'baseline':a.get(key),'current':b.get(key),'kind':'added' if key not in a else 'removed' if key not in b else 'changed','name_mentioned_in_inventory':bool(re.search(r'\b'+re.escape(key.split()[-1].split('.')[-1])+r'\b',interface)),'current_source':locate(package,key) if key in b else []}
        if key.startswith('type ') and key in a and key in b:
            added_lines=set(b[key].splitlines())-set(a[key].splitlines())
            matches=[]
            for location in row['current_source']:
                path=location['path'];lines=(ROOT/path).read_text().splitlines()
                start=location['line']-1
                stop=start+1
                if '{' in lines[start]:
                    for end in range(start+1,len(lines)):
                        if lines[end]=='}':stop=end+1;break
                for index in range(start,stop):
                    n=index+1;line=lines[index]
                    normalized=re.sub(r'\s+',' ',line.split('//')[0]).strip()
                    if normalized!='}' and normalized in added_lines:
                        matches.append({'path':path,'line':n,'blame':blame(path)[n]})
            row['added_or_changed_fields_blame']=matches
        if row['kind']=='added':
            for location in row['current_source']:
                source=location['declaration_blame']['source'].strip().split('{')[0].rstrip()
                name='history-'+package.replace('/','-')+'-'+re.sub(r'[^A-Za-z0-9.-]','-',key)+'.log'
                location['symbol_history_log']=name
                run(['git','log','--all','--format=%H%n%aI%n%s%n%b','-S',source,'--',location['path']],name)
        changes.append(row)

extra=[]
for path,expression in [
    ('tools/gomad3/choice/internal/wire/wire_generated.go',r'Profile\s*=|wireVersion\s*='),
    ('tools/gomad3/choice/trace.go',r'case SupersededProfile:|superseded choice trace profile'),
    ('tools/gomad3/runner/internal/exploration/choice/engine.go',r'breadth-first-rank-prefix/v3'),
    ('tools/gomad3/cmd/gomad/internal/cli/cli.go',r'flags\..*"(?:diagnostics|guide-regression|choice-start-ordinal)"'),
    ('tools/gomad3/cmd/gomad/internal/cli/qualify.go',r'flags\..*"diagnostics"'),
    ('tools/gomad3/cmd/gomad/internal/cli/resume.go',r'flags\..*"guide-regression"'),
    ('tools/gomad3/cmd/gomad/internal/cli/cli.go',r'flags\..*"resume"')]:
    p=ROOT/path
    for n,line in enumerate(p.read_text().splitlines(),1):
        if re.search(expression,line):extra.append({'path':path,'line':n,'source':line,'blame':blame(path)[n]})

commits=sorted({z['declaration_blame']['commit'] for x in changes for z in x['current_source']}|{x['blame']['commit'] for x in extra})
metadata={c:run(['git','show','-s','--format=%H%n%aI%n%s%n%b',c],'commit-'+c+'.log') for c in commits}
result={'base_source_manifest':inventory['baseline_root'],'declaration_changes':changes,'special_semantic_and_flag_changes':extra,'commit_metadata':metadata,'commands':commands,'limits':'Current declaration-line blame identifies the last introducing/editing commit. Full signatures and changed fields remain in public-api.diff. Lexical inventory-name mentions do not prove an intentional migration. Removed declarations require baseline provenance if ownership is disputed.'}
(OUT/'provenance.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps({'declaration_changes':len(changes),'unlocated_current':[x['declaration'] for x in changes if x['current'] and not x['current_source']],'commit_count':len(commits),'commands':len(commands)},indent=2))
