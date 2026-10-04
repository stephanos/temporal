"""Read-only independent receipt, preservation and characterization audit."""
import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SOURCE = ROOT / 'tools/gomad3/artifact/opened_test.go'
BASE = '1ee85b004cfaac41e178784701decbf0dd277968'


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def stable_inputs():
    paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
    values = {p: sha(ROOT / p) for p in paths if ROOT / p != SOURCE}
    protected = dict(files=len(values), aggregate_sha256=hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest())
    assert protected == json.loads((OUT / 'root-admission.json').read_text())['protected']
    return dict(source_sha256=sha(SOURCE), protected=protected)


before = stable_inputs()
current = SOURCE.read_text()
characterization = current.replace('populate(t, ', 'populate(')
characterization = characterization.replace('func populate(t *testing.T, value reflect.Value) {\n\tt.Helper()', 'func populate(value reflect.Value) {')
characterization = characterization.replace('\tcase reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:\n\tcase reflect.Invalid, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:\n\t\tt.Fatalf("cannot populate a %s field", value.Kind())\n', '')
characterization = characterization.replace('\tcase reflect.Array:\n\t\tfor index := range original.Len() {\n\t\t\tassertNoSharedMemory(t, fmt.Sprintf("%s[%d]", path, index), clone.Index(index), original.Index(index))\n\t\t}\n', '')
characterization = characterization.replace('\tcase reflect.Bool, reflect.String,\n\t\treflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,\n\t\treflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,\n\t\treflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:\n\tcase reflect.Invalid, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:\n\t\tt.Fatalf("%s has unsupported kind %s", path, original.Kind())\n', '')
characterization = characterization.replace('\t"fmt"\n', '')
characterization_sha = hashlib.sha256(characterization.encode()).hexdigest()
assert characterization_sha == json.loads((OUT / 'characterization.json').read_text())['source_before']['source_sha256']
restored = characterization[:characterization.index('func TestDeepCopyArrayReferences(')] + characterization[characterization.index('// populate fills every pointer, slice and map'):]
restored = restored.replace('\t"fmt"\n', '').replace('\t"unsafe"\n', '')
old = subprocess.check_output(['git', 'show', BASE + ':tools/gomad3/artifact/opened_test.go'], cwd=ROOT, text=True)
assert restored == old
expected = set('Invalid Bool Int Int8 Int16 Int32 Int64 Uint Uint8 Uint16 Uint32 Uint64 Uintptr Float32 Float64 Complex64 Complex128 Array Chan Func Interface Map Pointer Slice String Struct UnsafePointer'.split())
for helper in ('populate', 'assertNoSharedMemory'):
    body = re.search(r'^func ' + helper + r'\([^\n]+\).*?^}', current, re.M | re.S).group()
    cases = re.findall(r'^\tcase (.*?):', body, re.M | re.S)
    kinds = re.findall(r'reflect\.(\w+)', ' '.join(cases))
    assert set(kinds) == expected and len(kinds) == 27 and 'default:' not in body
    assert not re.search(r'\b(IsNil|IsValid)\(', body[:body.index('switch')])
names = ('package', 'focused', 'boundaries', 'lint', 'errortype', 'gofmt', 'diff-check', 'writer-audit', 'scope', 'validation')
receipts = {}
previous_end = None
for name in names:
    path = OUT / ('review-serial-' + name + '.json')
    value = json.loads(path.read_text())
    assert value['stable'] and value['input_before'] == value['input_after']
    assert value['input_before']['source_sha256'] == before['source_sha256']
    assert value['input_before']['protected'] == before['protected']
    assert value['input_before']['review_runner_sha256'] == sha(OUT / 'review-run.py')
    assert sha(OUT / ('review-serial-' + name + '.log')) == value['log_sha256']
    assert value['exit_code'] == (1 if name == 'lint' else 0)
    for tool, expected_sha in value['input_before']['tools_sha256'].items():
        assert sha(tool) == expected_sha
    start = datetime.datetime.fromisoformat(value['started_at'])
    if previous_end is not None:
        assert start >= previous_end
    previous_end = start + datetime.timedelta(seconds=value['elapsed_seconds'])
    if name in ('package', 'focused', 'boundaries'):
        assert value['top_level_run'] == value['top_level_pass'] == dict(package=49, focused=23, boundaries=5)[name]
    if name in ('gofmt', 'diff-check', 'errortype'):
        assert (OUT / ('review-serial-' + name + '.log')).stat().st_size == 0
    receipts[name] = dict(receipt=path.name, receipt_sha256=sha(path), log_sha256=value['log_sha256'], exit_code=value['exit_code'], elapsed_seconds=value['elapsed_seconds'], top_level_run=value['top_level_run'], top_level_pass=value['top_level_pass'])
for path in OUT.glob('review-*.json'):
    if path.name.startswith('review-serial-'):
        continue
    value = json.loads(path.read_text())
    assert value['stable'] and value['input_before'] == value['input_after']
    assert sha(path.with_suffix('.log')) == value['log_sha256']
lint = (OUT / 'review-serial-lint.log').read_bytes()
assert lint == (OUT / 'final-lint.log').read_bytes()
assert b'2 issues:' in lint and b'exhaustive' not in lint
after = stable_inputs()
assert before == after
print(json.dumps(dict(source_before=before, source_after=after, original_source_restored_exactly=True, reconstructed_characterization_sha256=characterization_sha, explicit_kinds_per_helper=27, review_commands_serialized=True, receipts=receipts, input_bindings=json.loads((OUT / 'review-serial-package.json').read_text())['input_before'], owned_files={p.name: sha(p) for p in sorted(OUT.glob('review-*')) if p.is_file()}, all_commands_terminal=True), indent=2))
