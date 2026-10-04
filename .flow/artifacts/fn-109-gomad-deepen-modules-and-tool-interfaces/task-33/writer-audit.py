import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SOURCE = 'tools/gomad3/artifact/opened_test.go'
BASE = '1ee85b004cfaac41e178784701decbf0dd277968'

def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

assert subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip() == BASE
assert subprocess.check_output(['git', 'branch', '--show-current'], cwd=ROOT, text=True).strip() == 'gomad'
old = subprocess.check_output(['git', 'show', BASE + ':' + SOURCE], cwd=ROOT, text=True)
current = (ROOT / SOURCE).read_text()
restored = current[:current.index('func TestDeepCopyArrayReferences(')] + current[current.index('// populate fills every pointer, slice and map'):]
restored = restored.replace('\t"fmt"\n', '').replace('\t"unsafe"\n', '')
restored = restored.replace('populate(t, ', 'populate(').replace('func populate(t *testing.T, value reflect.Value) {\n\tt.Helper()', 'func populate(value reflect.Value) {')
restored = restored.replace('\tcase reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:\n\tcase reflect.Invalid, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:\n\t\tt.Fatalf("cannot populate a %s field", value.Kind())\n', '')
restored = restored.replace('\tcase reflect.Array:\n\t\tfor index := range original.Len() {\n\t\t\tassertNoSharedMemory(t, fmt.Sprintf("%s[%d]", path, index), clone.Index(index), original.Index(index))\n\t\t}\n', '')
restored = restored.replace('\tcase reflect.Bool, reflect.String,\n\t\treflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,\n\t\treflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,\n\t\treflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:\n\tcase reflect.Invalid, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:\n\t\tt.Fatalf("%s has unsupported kind %s", path, original.Kind())\n', '')
assert restored == old, 'unexpected old-source change outside admitted helper edits'
kinds = set('Invalid Bool Int Int8 Int16 Int32 Int64 Uint Uint8 Uint16 Uint32 Uint64 Uintptr Float32 Float64 Complex64 Complex128 Array Chan Func Interface Map Pointer Slice String Struct UnsafePointer'.split())
for helper in ('populate', 'assertNoSharedMemory'):
    body = re.search(r'^func ' + helper + r'\([^\n]+\).*?^}', current, re.M | re.S).group()
    cases = re.findall(r'^\tcase (.*?):', body, re.M | re.S)
    explicit = re.findall(r'reflect\.(\w+)', ' '.join(cases))
    assert set(explicit) == kinds and len(explicit) == 27, (helper, explicit)
    assert 'default:' not in body
    assert not re.search(r'\b(IsNil|IsValid)\(', body[:body.index('switch')])
assert sha(ROOT / 'tools/gomad3/artifact/manifest_copy.go') == 'f1dad686d9f2fc59ebde1f0f484bafa6dcfad7fb1f84d1d3fe7878eeb44d7dae'
assert sha(ROOT / 'tools/gomad3/artifact/publication.go') == '44d9d007b0b09397654e62e24596f8321e14366a6797a6f31fc63ec1fb827671'
names = ('baseline-package', 'baseline-lint', 'baseline-errortype', 'characterization', 'final-package', 'final-focused', 'final-boundaries', 'final-errortype', 'final-lint', 'final-gofmt', 'final-diff-check', 'final-validation')
receipts = {}
for name in names:
    item = json.loads((OUT / (name + '.json')).read_text())
    assert item['source_before'] == item['source_after'] and item['stable']
    assert item['source_before']['protected'] == json.loads((OUT / 'root-admission.json').read_text())['protected']
    assert sha(OUT / (name + '.log')) == item['log_sha256']
    assert item['exit_code'] == (1 if name.endswith('-lint') else 0), name
    if name.startswith('final-'):
        assert item['source_before']['source_sha256'] == sha(ROOT / SOURCE)
    receipts[name] = dict(receipt_sha256=sha(OUT / (name + '.json')), log_sha256=item['log_sha256'], exit_code=item['exit_code'], top_level_run=item['top_level_run'], top_level_pass=item['top_level_pass'])
for name, count in (('baseline-package', 46), ('characterization', 3), ('final-package', 49), ('final-focused', 23), ('final-boundaries', 5)):
    assert receipts[name]['top_level_run'] == receipts[name]['top_level_pass'] == count
assert (OUT / 'final-gofmt.log').stat().st_size == (OUT / 'final-diff-check.log').stat().st_size == 0
before = (OUT / 'baseline-lint.log').read_text()
after = (OUT / 'final-lint.log').read_text()
assert '4 issues:' in before and '2 issues:' in after
assert before.count('(exhaustive)') == 2 and '(exhaustive)' not in after
for path in ('manifest_copy.go', 'publication.go'):
    pattern = r'^tools/gomad3/artifact/' + re.escape(path) + r':[^\n]+\n(?:\t[^\n]*\n)+'
    assert re.search(pattern, before, re.M).group() == re.search(pattern, after, re.M).group()
assert len(re.findall(r'^tools/gomad3/artifact/', after, re.M)) == 2
result = dict(base_commit=BASE, source_sha256=sha(ROOT / SOURCE), original_source_restored_exactly=True, explicit_kinds_per_helper=27, baseline_lint=4, final_lint=2, resolved=2, introduced=0, receipts=receipts, proof_limits=['helper Fatalf branches not executed', 'rejection of deliberately shared array not executed', 'developmental stock linux/arm64 only; original qualification remains open'])
print(json.dumps(result, indent=2))
