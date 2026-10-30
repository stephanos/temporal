#!/usr/bin/env python3
"""Retain and independently verify completed standalone proof; never rerun a gate or clean."""
import collections
import datetime
import gzip
import hashlib
import io
import json
import os
import pathlib
import shutil
import subprocess
import tarfile
import tempfile

HERE = pathlib.Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
STORE = TEMPORAL.parent / '.recovery/fn258-standalone-proof/objects'
PROOF = pathlib.Path('/home/agent/.cache/codex-build/tmp/umpire-standalone-Bnt8GY')
PARENT = pathlib.Path('/home/agent/.cache/codex-build/fn258-standalone-0547321')
SOURCE = pathlib.Path('/home/agent/.cache/codex-build/tmp/fn258-standalone-0547321/checkout')
REVISION = '05473210b48c3db862f22051eb7154ce98258f98'
TREE = '46829511b6d582b40f3fa5095f42bdf980c7d466'
records, checks = [], []


def sha(data):
    return hashlib.sha256(data).hexdigest()


def file_sha(path):
    h = hashlib.sha256()
    with path.open('rb') as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def read(path):
    return json.loads(path.read_bytes())


def write(name, data):
    (HERE/name).write_text(json.dumps(data, indent=2) + '\n')


def check(name, condition, detail=None):
    checks.append({'name': name, 'passed': bool(condition), 'detail': detail})


def preserve(path, logical, role, expected=None, readable=False):
    path = pathlib.Path(path)
    if not path.is_file() or path.is_symlink():
        raise RuntimeError('Required proof file unavailable/nonregular: ' + str(path))
    digest = file_sha(path)
    if expected is not None and digest != expected:
        raise RuntimeError('Sealed proof hash differs: ' + str(path))
    STORE.mkdir(parents=True, exist_ok=True)
    destination = STORE/digest
    if destination.exists():
        if file_sha(destination) != digest:
            raise RuntimeError('Existing recovery object is corrupt: ' + str(destination))
    else:
        with tempfile.NamedTemporaryFile(dir=STORE, delete=False) as temp:
            with path.open('rb') as source:
                shutil.copyfileobj(source, temp)
            temporary = pathlib.Path(temp.name)
        if file_sha(temporary) != digest:
            raise RuntimeError('Recovery copy hash differs: ' + str(path))
        temporary.replace(destination)
    record = {'source': str(path), 'destination': logical, 'role': role, 'size': path.stat().st_size,
              'mode': oct(path.stat().st_mode & 0o777), 'sha256': digest, 'recovery_object': str(destination)}
    records.append(record)
    if readable:
        target = HERE/logical
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(path, target)
        if file_sha(target) != digest:
            raise RuntimeError('Readable evidence copy differs')
    return record


def archive(paths, logical, role):
    # paths are explicit regular files or source directories already inspected above.
    with tempfile.NamedTemporaryFile(dir=STORE.parent, suffix='.tar.gz', delete=False) as temp:
        archive_path = pathlib.Path(temp.name)
    members = []
    with archive_path.open('wb') as raw:
        with gzip.GzipFile(filename='', fileobj=raw, mode='wb', mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode='w|') as output:
                for path, name in paths:
                    info = output.gettarinfo(str(path), arcname=name)
                    if not info.isfile():
                        raise RuntimeError('Expected regular archive member: ' + str(path))
                    info.mode &= 0o7777
                    info.uid = info.gid = info.mtime = 0
                    info.uname = info.gname = ''
                    with path.open('rb') as source:
                        output.addfile(info, source)
                    members.append({'path': name, 'size': info.size, 'mode': oct(info.mode), 'sha256': file_sha(path)})
    record = preserve(archive_path, logical, role)
    with tarfile.open(record['recovery_object'], 'r:gz') as retained:
        found = retained.getmembers()
        if [x.name for x in found] != [x['path'] for x in members]:
            raise RuntimeError('Archive member inventory differs')
        for actual, expected in zip(found, members):
            if sha(retained.extractfile(actual).read()) != expected['sha256']:
                raise RuntimeError('Archive member payload differs: ' + actual.name)
    archive_path.unlink()  # Only this script's verified temporary archive; CAS object remains.
    return {'object': record, 'verified_members': members}


def inventory(path):
    return [{'path_hex': os.fsencode(str(p.relative_to(path))).hex(), 'sha256': file_sha(p),
             'executable': bool(p.stat().st_mode & 0o111)} for p in sorted(path.rglob('*')) if p.is_file()]


def observation_summary(directory):
    events = []
    for path in sorted(directory.glob('*.event'), key=lambda p: int(p.stem)):
        parts = path.read_text().rstrip('\n').split('\t', 2)
        if len(parts) != 3 or parts[0] != '1':
            raise RuntimeError('Malformed work observation: ' + str(path))
        events.append({'file': path.name, 'kind': parts[1], 'payload': parts[2], 'sha256': file_sha(path)})
    begins, finishes = {}, {}
    tlc_attempts, tlc_finishes = {}, {}
    for event in events:
        parts = event['payload'].split('\t')
        if event['kind'] == 'tests-begin':
            if parts[0] in begins:
                raise RuntimeError('Duplicate test producer')
            begins[parts[0]] = event
        elif event['kind'] == 'tests-finish':
            if parts[0] in finishes or len(parts) != 5:
                raise RuntimeError('Duplicate/malformed producer completion')
            finishes[parts[0]] = dict(zip(['discovered', 'executed', 'ignored', 'skipped'], map(int, parts[1:])))
        elif event['kind'] == 'tlc-attempt':
            tlc_attempts[parts[0]] = parts[1]
        elif event['kind'] == 'tlc-finish':
            tlc_finishes[parts[0]] = parts[1]
    check('test producer begin/finish identities complete: ' + str(directory), begins.keys() == finishes.keys())
    for counts in finishes.values():
        check('producer arithmetic', counts['discovered'] == counts['executed'] + counts['ignored'] + counts['skipped'], counts)
    return {'events': events, 'test_producers': finishes,
            'aggregate_tests': {k: sum(v[k] for v in finishes.values()) for k in ['discovered', 'executed', 'ignored', 'skipped']},
            'raw_tlc_attempts': len(tlc_attempts), 'raw_tlc_completions': len(tlc_finishes),
            'raw_tlc_confirmed_starts': sum(v == '1' for v in tlc_finishes.values()),
            'raw_tlc_attempts_without_completion': sorted(tlc_attempts.keys()-tlc_finishes.keys()),
            'tlc_authority_note': 'Raw observations retained separately. Published stage TLC counts are null and are not replaced by these derived diagnostic totals.'}


STORE.mkdir(parents=True, exist_ok=True)
result = read(PROOF/'result.json')
inputs = read(PROOF/'inputs.json')
parent_run = next((PARENT/'gate-receipts').iterdir())
child_run = next((PROOF/'target/gate-receipts').iterdir())
parent_rows = [json.loads(l) for l in (parent_run/'stages.jsonl').read_text().splitlines()]
child_rows = [json.loads(l) for l in (PROOF/'stages.jsonl').read_text().splitlines()]
expected = result['expected_stages']
check('source HEAD pinned', subprocess.check_output(['git','-C',str(SOURCE),'rev-parse','HEAD']).decode().strip() == REVISION)
check('source tree pinned', subprocess.check_output(['git','-C',str(SOURCE),'rev-parse','HEAD^{tree}']).decode().strip() == TREE)
status = subprocess.check_output(['git','-C',str(SOURCE),'status','--porcelain=v1'], env=dict(os.environ, GIT_OPTIONAL_LOCKS='0')).decode()
check('fresh source checkout tracked/nonignored cleanliness', not status, status)
check('successful complete cold standalone result', result['complete'] and result['child_exit'] == 0 and result['source_unchanged'] and result['inventory_error'] is None)
check('manifest seal', file_sha(PROOF/'inputs.json') == result['manifest_sha256'])
check('child stages seal', file_sha(PROOF/'stages.jsonl') == result['stages_sha256'])
check('child stage authority copy matches child run', (PROOF/'stages.jsonl').read_bytes() == (child_run/'stages.jsonl').read_bytes())
check('expected ordered child stages', [r['stage'] for r in child_rows] == expected == result['actual_stages'])
check('expected ordered parent stages', [r['stage'] for r in parent_rows] == expected + ['standalone'])
for label, rows, run in [('parent', parent_rows, parent_run), ('child', child_rows, child_run)]:
    for row in rows:
        check(label+' successful exclusive authority '+row['stage'], row['exit'] == 0 and row['state'] == 'success' and row['count_authority'] == 'exclusive-stage' and row['count_error'] is None)
        frame = read(run/'framework'/(''+row['stage']+'.json'))
        check(label+' individual stage equals ordered authority '+row['stage'], frame == row)
embedded = parent_rows[-1]['child_count_receipts']
check('exact child authority embedded once in parent', len(embedded) == 1 and embedded[0]['records'].encode() == (PROOF/'stages.jsonl').read_bytes() and embedded[0]['sha256'] == file_sha(PROOF/'stages.jsonl'))
source_errors = []
for row in inputs['inputs']:
    relative = pathlib.Path(os.fsdecode(bytes.fromhex(row['path_hex'])))
    for root in [SOURCE, PROOF/'framework']:
        path = root/relative
        if not path.is_file() or path.is_symlink() or file_sha(path) != row['sha256'] or bool(path.stat().st_mode & 0o111) != row['executable']:
            source_errors.append(str(path))
check('all input source bytes and executable modes unchanged in both copies', not source_errors, source_errors)
raw_tree = subprocess.check_output(['git','-C',str(SOURCE),'ls-tree','-rz',REVISION])
git_paths = {row.split(b'\t',1)[1] for row in raw_tree.split(b'\0') if row}
check('provenance input path set equals pinned Git source tree', git_paths == {bytes.fromhex(r['path_hex']) for r in inputs['inputs']})
observations = {}
for label, run, rows in [('parent', parent_run, parent_rows), ('child', child_run, child_rows)]:
    observations[label] = observation_summary(run/'work/framework/tests')
    stage = next(r for r in rows if r['stage'] == 'tests')
    check(label+' producer counts equal authoritative tests receipt', all(stage['counts'][k] == v for k,v in observations[label]['aggregate_tests'].items()))
    for name in ['admission.json','stages.jsonl','stdout.log','stderr.log']:
        preserve(run/name, label+'/'+name, 'ordered stage authority, admission or raw run log', readable=True)
    for frame in sorted((run/'framework').glob('*.json')):
        preserve(frame,label+'/stages/'+frame.name,'individual stage receipt',readable=True)
for name in ['inputs.json','result.json','controls.json','tools.json','setup.jsonl','stages.jsonl']:
    preserve(PROOF/name,'cold-proof/'+name,'native standalone proof receipt or source inventory',readable=True)
preserve(TEMPORAL/'.flow/tmp/fn258-standalone-0547321.log','make-validate.log','original complete make validate command log',readable=True)
for name, entry in {**result['binaries'], **result['artifacts']}.items():
    path = pathlib.Path(entry['path'])
    if path.is_dir():
        actual = inventory(path)
        expected_files = [{'path_hex':r['path_hex'],'sha256':r['sha256'],'executable':r['executable']} for r in entry['files']]
        check('sealed directory file inventory '+name, actual == expected_files)
        compact = json.dumps(actual, separators=(',',':'), ensure_ascii=False).encode()
        check('sealed directory aggregate '+name, sha(compact) == entry['sha256'])
        for file in actual:
            p=path/os.fsdecode(bytes.fromhex(file['path_hex']))
            preserve(p,'child/named/'+name+'/'+str(p.relative_to(path)),'named native parity payload',expected=file['sha256'])
    else:
        preserve(path,'child/named/'+name,'named proof binary/tool/guest artifact',expected=entry['sha256'])
    counterpart = PARENT/name
    if not counterpart.exists() and name.startswith(('.bin/','build/')):
        counterpart = SOURCE/name
    if counterpart.is_file():
        preserve(counterpart,'parent/named/'+name,'corresponding parent proof binary/tool/guest artifact')
    elif counterpart.is_dir():
        for file in inventory(counterpart):
            p=counterpart/os.fsdecode(bytes.fromhex(file['path_hex']))
            preserve(p,'parent/named/'+name+'/'+str(p.relative_to(counterpart)),'corresponding parent native parity payload')
for label, target in [('parent',PARENT),('child',PROOF/'target')]:
    for sub in ['guest-cache','guest-wrappers']:
        for path in sorted((target/sub).rglob('*')):
            if path.is_file() and (path.name.endswith('.umpire-id') or path.name.startswith('umpire-lock-') and path.suffix == '.json' or sub == 'guest-wrappers'):
                preserve(path,label+'/identity/'+str(path.relative_to(target)),'guest compile identity or actual compiler wrapper')
for name, tool in read(PROOF/'tools.json').items():
    check('successful pinned tool version '+name,tool['exit']==0)
    preserve(pathlib.Path(tool['executable']),'toolchain/'+name,'actual tool executable identity; toolchain shared caches are not cleanup candidates',expected=tool['executable_sha256'])
receipt_paths=[]
for label,run in [('parent',parent_run),('child',child_run)]:
    receipt_paths.extend((p,label+'/'+str(p.relative_to(run))) for p in sorted(run.rglob('*')) if p.is_file())
archives={'complete_stage_authority':archive(receipt_paths,'archives/stage-authority.tar.gz','complete ordered/individual receipts, claims, raw observations and logs'),
          'source_snapshot':archive([(SOURCE/os.fsdecode(bytes.fromhex(r['path_hex'])),os.fsdecode(bytes.fromhex(r['path_hex']))) for r in inputs['inputs']], 'archives/source-0547321.tar.gz','all exact source inputs, reconstructable without the build cache')}
# Final rehash makes later cleanup contingent on an independent recovery copy.
for record in records:
    check('recovery object verified '+record['destination'],file_sha(pathlib.Path(record['recovery_object']))==record['sha256'])
write('count-observations.json',observations)
write('archive-manifests.json',archives)
manifest={'source_commit':REVISION,'source_tree':TREE,'proof_root':str(PROOF),'parent_target':str(PARENT),'content_store':str(STORE),'records':records,'result':'pass' if all(c['passed'] for c in checks) else 'fail'}
write('recovery-manifest.json',manifest)
write('verification.json',{'observed_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'source_commit':REVISION,'source_tree':TREE,'checks':checks,'result':manifest['result'],'parent_stage_count':len(parent_rows),'child_stage_count':len(child_rows),'source_input_records':len(inputs['inputs']),'counts':{k:{a:b for a,b in v.items() if a!='events'} for k,v in observations.items()},'artifacts_retained':len(records),'unique_recovery_objects':len({r['sha256'] for r in records}),'unique_retained_bytes':sum(next(r['size'] for r in records if r['sha256']==digest) for digest in {r['sha256'] for r in records}),'limitations':['Dev required-TLC validation only; release workspace tests and paired/local-edit proof are separate evidence.','Apalache skipped, as recorded by the invoked gate.','Gate and edited-checker timings exceed existing soft budgets; no performance obligation closed.','Complete stage receipts prove stage success; original launcher process exit is conductor-reported, without a separate persisted exit-status receipt.','Shared toolchain executable identity is retained, not a self-contained Rust/Java/Go toolchain installation. No shared toolchain cache cleanup is authorized.','No cache cleanup is authorized before conductor review/closure and renewed process/lease audit.']})
print(json.dumps({'result':manifest['result'],'records':len(records),'checks':len(checks),'failed':[c for c in checks if not c['passed']]},indent=2))
if manifest['result']!='pass':
    raise SystemExit(1)
