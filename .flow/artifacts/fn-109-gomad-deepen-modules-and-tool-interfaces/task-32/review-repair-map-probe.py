import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile

OUT = Path(__file__).resolve().parent
ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'
GO = '/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go'
stage = sys.argv[1]
assert stage in ('candidate','baseline')
extra = r'''
func TestRepairReviewMapAlias(t *testing.T) {
 for _, test := range []errorProvenanceFixture{
  {name:"dirty", code:`v:=make(map[int]func());helper.Set(helper.Callbacks(v));v[0]()`, helper:`func Dirty(){Calls++;_=time.Now()};type Callbacks map[int]func();func Set(v Callbacks){v[0]=Dirty}`, calls:1, callback:"canonicaljson.Dirty"},
  {name:"clean", code:`v:=make(map[int]func());helper.Set(helper.Callbacks(v));v[0]()`, helper:`func Clean(){};type Callbacks map[int]func();func Set(v Callbacks){v[0]=Clean}`},
 } {t.Run(test.name,func(t *testing.T){checkErrorProvenance(t,test)})}
}
'''
with tempfile.TemporaryDirectory(prefix='task32-repair-map-review-') as temporary:
    scratch = Path(temporary)
    source = (ARCH / 'error_provenance_test.go').read_text() + extra
    test_path = scratch / 'error_provenance_test.go'
    test_path.write_text(source)
    replace = {str(ARCH / 'error_provenance_test.go'):str(test_path)}
    if stage == 'baseline':
        for name in ('effects.go','standard.go'):
            replace[str(ARCH/name)] = str(OUT/'sources/baseline'/name)
    overlay = scratch/'overlay.json'
    overlay.write_text(json.dumps({'Replace':replace}))
    print(json.dumps({'stage':stage,'test_source_sha256':hashlib.sha256(source.encode()).hexdigest(),'extra_fixture_source':extra,'overlay':replace}),flush=True)
    result = subprocess.run([GO,'test','-count=1','-tags','test_dep','-v','-overlay',str(overlay),'./internal/gomadtool/architecture','-run','^TestRepairReviewMapAlias$'],cwd=ROOT/'tools/gomad3')
sys.exit(result.returncode)
