from pathlib import Path
import copy, hashlib, json, stat, tarfile

HERE = Path(__file__).resolve().parent

def digest(path):
    h=hashlib.sha256()
    with path.open('rb') as stream:
        for data in iter(lambda:stream.read(1024*1024), b''): h.update(data)
    return h.hexdigest()

def verify(manifest):
    assert len(manifest['objects']) == 1
    obj=manifest['objects'][0]; path=Path(obj['recovery_object'])
    assert not path.is_symlink() and path.is_file()
    assert path.stat().st_size == obj['size'] and digest(path) == obj['sha256']
    expected=manifest['archive_members']
    with tarfile.open(path,'r:gz') as archive:
        members=archive.getmembers(); assert len(members) == len(expected)
        assert len({m.name for m in members}) == len(members)
        for member,row in zip(members,expected):
            assert member.isfile() and not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
            assert member.name == row['path'] and member.size == row['size'] and oct(member.mode) == row['mode']
            h=hashlib.sha256(); stream=archive.extractfile(member)
            for data in iter(lambda:stream.read(1024*1024), b''): h.update(data)
            assert h.hexdigest() == row['sha256'], member.name
    return {'objects':1,'archive_members':len(expected),'original_source_reads':False}

def main():
    manifest=json.loads((HERE/'manifest.json').read_text()); result=verify(manifest)
    bad=copy.deepcopy(manifest); bad['objects'][0]['sha256']='0'*64
    try: verify(bad)
    except AssertionError: pass
    else: raise AssertionError('corrupted object accepted')
    bad=copy.deepcopy(manifest); bad['archive_members']=bad['archive_members'][:-1]
    try: verify(bad)
    except AssertionError: pass
    else: raise AssertionError('missing member accepted')
    result.update(retention_accepted=True,task_complete=False,cleanup_performed=False,
        corrupted_object_refused=True,missing_member_refused=True)
    (HERE/'verification.json').write_text(json.dumps(result,indent=2)+'\n')
    print(json.dumps(result))

if __name__ == '__main__': main()
